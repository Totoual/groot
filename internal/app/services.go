package app

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"
)

type ServiceState string

const (
	ServiceStarting ServiceState = "starting"
	ServiceRunning  ServiceState = "running"
	ServiceStopped  ServiceState = "stopped"
	ServiceFailed   ServiceState = "failed"
)

type ServiceStatus struct {
	Name          string       `json:"name"`
	Workspace     string       `json:"workspace"`
	Command       string       `json:"command,omitempty"`
	Args          []string     `json:"args,omitempty"`
	Cwd           string       `json:"cwd"`
	RestartPolicy string       `json:"restart_policy,omitempty"`
	Version       string       `json:"version,omitempty"`
	State         ServiceState `json:"state"`
	StartedAt     *time.Time   `json:"started_at,omitempty"`
	StoppedAt     *time.Time   `json:"stopped_at,omitempty"`
	ExitCode      *int         `json:"exit_code,omitempty"`
	PID           int          `json:"pid,omitempty"`
	StdoutLog     string       `json:"stdout_log"`
	StderrLog     string       `json:"stderr_log"`
	StopReason    string       `json:"stop_reason,omitempty"`
}

type ServiceLogs struct {
	Service string `json:"service"`
	Stdout  string `json:"stdout"`
	Stderr  string `json:"stderr"`
}

type serviceRecord struct {
	Name          string     `json:"name"`
	Workspace     string     `json:"workspace"`
	Command       string     `json:"command,omitempty"`
	Args          []string   `json:"args,omitempty"`
	Cwd           string     `json:"cwd"`
	RestartPolicy string     `json:"restart_policy,omitempty"`
	Version       string     `json:"version,omitempty"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	PID           int        `json:"pid,omitempty"`
	StdoutLog     string     `json:"stdout_log"`
	StderrLog     string     `json:"stderr_log"`
}

func (a *App) StartService(workspaceName, serviceName string) (ServiceStatus, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return ServiceStatus{}, err
	}
	spec, err := a.serviceSpec(workspaceName, serviceName)
	if err != nil {
		return ServiceStatus{}, err
	}

	current, err := a.ServiceStatus(workspaceName, serviceName)
	if err == nil && current.State == ServiceRunning {
		return current, nil
	}

	env, workDir, err := a.workspaceRuntime(workspaceName)
	if err != nil {
		return ServiceStatus{}, err
	}
	serviceCwd, err := resolveTaskCwd(workDir, spec.Cwd)
	if err != nil {
		return ServiceStatus{}, err
	}
	if len(spec.Command) == 0 {
		return ServiceStatus{}, fmt.Errorf("service %q has no command", serviceName)
	}
	command := strings.TrimSpace(spec.Command[0])
	if command == "" {
		return ServiceStatus{}, fmt.Errorf("service %q has no command", serviceName)
	}
	resolvedCommand, err := resolveCommandForEnv(command, env)
	if err != nil {
		return ServiceStatus{}, err
	}

	serviceDir, stdoutLog, stderrLog, err := a.ensureServicePaths(wsPath, serviceName)
	if err != nil {
		return ServiceStatus{}, err
	}
	if err := clearServiceRuntimeMarkers(serviceDir); err != nil {
		return ServiceStatus{}, err
	}

	supervisorPath := filepath.Join(serviceDir, "supervise.sh")
	exitCodePath := filepath.Join(serviceDir, "exit_code")
	finishedAtPath := filepath.Join(serviceDir, "finished_at")
	stopReasonPath := filepath.Join(serviceDir, "stop_reason")
	if err := os.WriteFile(supervisorPath, []byte(taskSupervisorScript(stdoutLog, stderrLog, exitCodePath, finishedAtPath, stopReasonPath)), 0o700); err != nil {
		return ServiceStatus{}, fmt.Errorf("write service supervisor: %w", err)
	}

	startedAt := time.Now().UTC()
	record := serviceRecord{
		Name:          spec.Name,
		Workspace:     workspaceName,
		Command:       resolvedCommand,
		Args:          append([]string{}, spec.Command[1:]...),
		Cwd:           serviceCwd,
		RestartPolicy: spec.Restart,
		Version:       spec.Version,
		StartedAt:     &startedAt,
		StdoutLog:     stdoutLog,
		StderrLog:     stderrLog,
	}
	if err := a.writeServiceRecord(serviceDir, record); err != nil {
		return ServiceStatus{}, err
	}

	cmdArgs := append([]string{supervisorPath, resolvedCommand}, record.Args...)
	cmd := exec.Command("/bin/sh", cmdArgs...)
	cmd.Dir = serviceCwd
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return ServiceStatus{}, fmt.Errorf("start service: %w", err)
	}
	record.PID = cmd.Process.Pid
	if err := a.writeServiceRecord(serviceDir, record); err != nil {
		_ = cmd.Process.Kill()
		return ServiceStatus{}, err
	}
	if err := cmd.Process.Release(); err != nil {
		return ServiceStatus{}, fmt.Errorf("release service process: %w", err)
	}

	service := serviceStatusFromRecord(record)
	if err := a.emitServiceStartedEvent(service); err != nil {
		return ServiceStatus{}, err
	}
	return a.ServiceStatus(workspaceName, serviceName)
}

func (a *App) StopService(workspaceName, serviceName string) (ServiceStatus, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return ServiceStatus{}, err
	}
	service, err := a.ServiceStatus(workspaceName, serviceName)
	if err != nil {
		return ServiceStatus{}, err
	}
	if service.State != ServiceRunning && service.State != ServiceStarting {
		return service, nil
	}

	if _, err := os.FindProcess(service.PID); err != nil {
		return ServiceStatus{}, fmt.Errorf("find service process: %w", err)
	}
	if err := syscall.Kill(-service.PID, syscall.SIGTERM); err != nil && err != syscall.ESRCH {
		return ServiceStatus{}, fmt.Errorf("stop service: %w", err)
	}

	serviceDir := serviceStateDir(wsPath, serviceName)
	finishedAt := time.Now().UTC()
	exitCode := 143
	if err := writeStringMarker(filepath.Join(serviceDir, "stop_reason"), "requested stop"); err != nil {
		return ServiceStatus{}, err
	}
	if err := writeIntMarker(filepath.Join(serviceDir, "exit_code"), exitCode); err != nil {
		return ServiceStatus{}, err
	}
	if err := writeTimeMarker(filepath.Join(serviceDir, "finished_at"), finishedAt); err != nil {
		return ServiceStatus{}, err
	}

	service, err = a.ServiceStatus(workspaceName, serviceName)
	if err != nil {
		return ServiceStatus{}, err
	}
	if err := a.emitServiceStoppedEvent(service); err != nil {
		return ServiceStatus{}, err
	}
	if err := writeServiceTerminalMarker(serviceDir, EventKindServiceStopped); err != nil {
		return ServiceStatus{}, err
	}
	return service, nil
}

func (a *App) ServiceStatus(workspaceName, serviceName string) (ServiceStatus, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return ServiceStatus{}, err
	}
	spec, err := a.serviceSpec(workspaceName, serviceName)
	if err != nil {
		return ServiceStatus{}, err
	}
	_, workDir, err := a.workspaceRuntime(workspaceName)
	if err != nil {
		return ServiceStatus{}, err
	}

	serviceDir := serviceStateDir(wsPath, serviceName)
	record, err := a.readServiceRecord(serviceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return serviceStatusFromSpec(workspaceName, workDir, spec, wsPath), nil
		}
		return ServiceStatus{}, err
	}
	service, err := a.serviceStatusFromRecord(serviceDir, record, spec)
	if err != nil {
		return ServiceStatus{}, err
	}
	return service, nil
}

func (a *App) ServiceList(workspaceName string) ([]ServiceStatus, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return nil, err
	}
	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return nil, err
	}
	services := make([]ServiceStatus, 0, len(manifest.Services))
	for _, spec := range manifest.Services {
		service, err := a.ServiceStatus(workspaceName, spec.Name)
		if err != nil {
			return nil, err
		}
		services = append(services, service)
	}
	slices.SortFunc(services, func(a, b ServiceStatus) int {
		return strings.Compare(a.Name, b.Name)
	})
	return services, nil
}

func (a *App) ServiceLogs(workspaceName, serviceName string) (ServiceLogs, error) {
	service, err := a.ServiceStatus(workspaceName, serviceName)
	if err != nil {
		return ServiceLogs{}, err
	}
	stdoutData, err := os.ReadFile(service.StdoutLog)
	if err != nil && !os.IsNotExist(err) {
		return ServiceLogs{}, fmt.Errorf("read service stdout log: %w", err)
	}
	stderrData, err := os.ReadFile(service.StderrLog)
	if err != nil && !os.IsNotExist(err) {
		return ServiceLogs{}, fmt.Errorf("read service stderr log: %w", err)
	}
	return ServiceLogs{
		Service: service.Name,
		Stdout:  string(stdoutData),
		Stderr:  string(stderrData),
	}, nil
}

func (a *App) serviceSpec(workspaceName, serviceName string) (ServiceSpec, error) {
	if err := validateServiceName(serviceName); err != nil {
		return ServiceSpec{}, err
	}
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return ServiceSpec{}, err
	}
	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return ServiceSpec{}, err
	}
	for _, spec := range manifest.Services {
		if spec.Name == serviceName {
			return spec, nil
		}
	}
	return ServiceSpec{}, fmt.Errorf("declared service %q not found", serviceName)
}

func (a *App) ensureServicePaths(wsPath, serviceName string) (string, string, string, error) {
	serviceDir := serviceStateDir(wsPath, serviceName)
	if err := os.MkdirAll(serviceDir, 0o700); err != nil {
		return "", "", "", fmt.Errorf("mkdir service state dir: %w", err)
	}
	logDir := filepath.Join(wsPath, "logs", "services")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return "", "", "", fmt.Errorf("mkdir service logs dir: %w", err)
	}
	stdoutLog := filepath.Join(logDir, serviceName+".stdout.log")
	stderrLog := filepath.Join(logDir, serviceName+".stderr.log")
	return serviceDir, stdoutLog, stderrLog, nil
}

func clearServiceRuntimeMarkers(serviceDir string) error {
	for _, name := range []string{"exit_code", "finished_at", "stop_reason", "terminal_event"} {
		if err := os.Remove(filepath.Join(serviceDir, name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (a *App) readServiceRecord(serviceDir string) (serviceRecord, error) {
	data, err := os.ReadFile(filepath.Join(serviceDir, "service.json"))
	if err != nil {
		return serviceRecord{}, err
	}
	var record serviceRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return serviceRecord{}, err
	}
	return record, nil
}

func (a *App) writeServiceRecord(serviceDir string, record serviceRecord) error {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(serviceDir, "service.json")
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func (a *App) serviceStatusFromRecord(serviceDir string, record serviceRecord, spec ServiceSpec) (ServiceStatus, error) {
	service := serviceStatusFromRecord(record)
	service.RestartPolicy = spec.Restart
	service.Version = spec.Version

	if finishedAt, ok, err := readTimeMarker(filepath.Join(serviceDir, "finished_at")); err != nil {
		return ServiceStatus{}, err
	} else if ok {
		service.StoppedAt = &finishedAt
	}
	if exitCode, ok, err := readIntMarker(filepath.Join(serviceDir, "exit_code")); err != nil {
		return ServiceStatus{}, err
	} else if ok {
		service.ExitCode = &exitCode
	}
	if stopReason, ok, err := readStringMarker(filepath.Join(serviceDir, "stop_reason")); err != nil {
		return ServiceStatus{}, err
	} else if ok {
		service.StopReason = stopReason
	}

	switch {
	case service.StoppedAt != nil && service.StopReason != "":
		service.State = ServiceStopped
	case service.StoppedAt != nil:
		service.State = ServiceFailed
	case record.PID == 0:
		service.State = ServiceStopped
	default:
		service.State = ServiceRunning
	}

	if err := a.emitServiceFailureEventIfNeeded(serviceDir, service); err != nil {
		return ServiceStatus{}, err
	}
	return service, nil
}

func serviceStatusFromSpec(workspaceName, workDir string, spec ServiceSpec, wsPath string) ServiceStatus {
	command := ""
	args := []string(nil)
	if len(spec.Command) > 0 {
		command = spec.Command[0]
		args = append([]string{}, spec.Command[1:]...)
	}
	return ServiceStatus{
		Name:          spec.Name,
		Workspace:     workspaceName,
		Command:       command,
		Args:          args,
		Cwd:           previewRuntimeCwd(workDir, spec.Cwd),
		RestartPolicy: spec.Restart,
		Version:       spec.Version,
		State:         ServiceStopped,
		StdoutLog:     filepath.Join(wsPath, "logs", "services", spec.Name+".stdout.log"),
		StderrLog:     filepath.Join(wsPath, "logs", "services", spec.Name+".stderr.log"),
	}
}

func serviceStatusFromRecord(record serviceRecord) ServiceStatus {
	state := ServiceRunning
	if record.PID == 0 {
		state = ServiceStopped
	}
	return ServiceStatus{
		Name:          record.Name,
		Workspace:     record.Workspace,
		Command:       record.Command,
		Args:          append([]string{}, record.Args...),
		Cwd:           record.Cwd,
		RestartPolicy: record.RestartPolicy,
		Version:       record.Version,
		State:         state,
		StartedAt:     record.StartedAt,
		PID:           record.PID,
		StdoutLog:     record.StdoutLog,
		StderrLog:     record.StderrLog,
	}
}

func previewRuntimeCwd(baseDir, requested string) string {
	requested = strings.TrimSpace(requested)
	if requested == "" || requested == "." {
		return baseDir
	}
	if filepath.IsAbs(requested) {
		return requested
	}
	return filepath.Join(baseDir, requested)
}

func serviceStateDir(wsPath, serviceName string) string {
	return filepath.Join(wsPath, "state", "services", serviceName)
}

func validateServiceName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." || strings.Contains(name, "/") {
		return fmt.Errorf("invalid service name %q", name)
	}
	return nil
}

func (a *App) emitServiceStartedEvent(service ServiceStatus) error {
	return a.emitEvent(service.Workspace, RuntimeEvent{
		Kind:         EventKindServiceStarted,
		ResourceType: "service",
		ResourceID:   service.Name,
		Message:      fmt.Sprintf("Service %q started.", service.Name),
		Payload:      serviceEventPayload(service),
	})
}

func (a *App) emitServiceStoppedEvent(service ServiceStatus) error {
	return a.emitEvent(service.Workspace, RuntimeEvent{
		Kind:         EventKindServiceStopped,
		ResourceType: "service",
		ResourceID:   service.Name,
		Message:      fmt.Sprintf("Service %q stopped.", service.Name),
		Payload:      serviceEventPayload(service),
	})
}

func (a *App) emitServiceFailureEventIfNeeded(serviceDir string, service ServiceStatus) error {
	if service.State != ServiceFailed {
		return nil
	}
	if serviceTerminalEventEmitted(serviceDir) {
		return nil
	}
	if err := a.emitEvent(service.Workspace, RuntimeEvent{
		Kind:         EventKindServiceFailed,
		ResourceType: "service",
		ResourceID:   service.Name,
		Message:      fmt.Sprintf("Service %q failed.", service.Name),
		Payload:      serviceEventPayload(service),
	}); err != nil {
		return err
	}
	return writeServiceTerminalMarker(serviceDir, EventKindServiceFailed)
}

func serviceEventPayload(service ServiceStatus) map[string]any {
	payload := map[string]any{
		"name":           service.Name,
		"state":          service.State,
		"command":        service.Command,
		"args":           append([]string{}, service.Args...),
		"cwd":            service.Cwd,
		"restart_policy": service.RestartPolicy,
		"pid":            service.PID,
	}
	if service.Version != "" {
		payload["version"] = service.Version
	}
	if service.StartedAt != nil {
		payload["started_at"] = service.StartedAt.UTC().Format(time.RFC3339)
	}
	if service.StoppedAt != nil {
		payload["finished_at"] = service.StoppedAt.UTC().Format(time.RFC3339)
	}
	if service.ExitCode != nil {
		payload["exit_code"] = *service.ExitCode
	}
	if service.StopReason != "" {
		payload["stop_reason"] = service.StopReason
	}
	return payload
}

func serviceTerminalEventMarkerPath(serviceDir string) string {
	return filepath.Join(serviceDir, "terminal_event")
}

func serviceTerminalEventEmitted(serviceDir string) bool {
	_, err := os.Stat(serviceTerminalEventMarkerPath(serviceDir))
	return err == nil
}

func writeServiceTerminalMarker(serviceDir, kind string) error {
	return os.WriteFile(serviceTerminalEventMarkerPath(serviceDir), []byte(kind), 0o600)
}
