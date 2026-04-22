package app

import (
	"crypto/rand"
	"encoding/hex"
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

type TaskRunState string

const (
	TaskRunPending   TaskRunState = "pending"
	TaskRunRunning   TaskRunState = "running"
	TaskRunSucceeded TaskRunState = "succeeded"
	TaskRunFailed    TaskRunState = "failed"
	TaskRunCancelled TaskRunState = "cancelled"
)

type TaskRun struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Workspace    string       `json:"workspace"`
	Command      string       `json:"command"`
	Args         []string     `json:"args"`
	Cwd          string       `json:"cwd"`
	Declared     bool         `json:"declared"`
	State        TaskRunState `json:"state"`
	CreatedAt    time.Time    `json:"created_at"`
	StartedAt    time.Time    `json:"started_at"`
	FinishedAt   *time.Time   `json:"finished_at,omitempty"`
	ExitCode     *int         `json:"exit_code,omitempty"`
	PID          int          `json:"pid,omitempty"`
	StdoutLog    string       `json:"stdout_log"`
	StderrLog    string       `json:"stderr_log"`
	CancelReason string       `json:"cancel_reason,omitempty"`
}

type TaskStartSpec struct {
	Name    string
	Command string
	Args    []string
	Cwd     string
}

type TaskRunLogs struct {
	TaskID string `json:"task_id"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}

type taskRecord struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Workspace    string     `json:"workspace"`
	Command      string     `json:"command"`
	Args         []string   `json:"args"`
	Cwd          string     `json:"cwd"`
	Declared     bool       `json:"declared"`
	CreatedAt    time.Time  `json:"created_at"`
	StartedAt    time.Time  `json:"started_at"`
	PID          int        `json:"pid"`
	StdoutLog    string     `json:"stdout_log"`
	StderrLog    string     `json:"stderr_log"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	ExitCode     *int       `json:"exit_code,omitempty"`
	CancelReason string     `json:"cancel_reason,omitempty"`
}

func (a *App) StartTask(workspaceName string, spec TaskStartSpec) (TaskRun, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return TaskRun{}, err
	}
	env, workDir, err := a.workspaceRuntime(workspaceName)
	if err != nil {
		return TaskRun{}, err
	}

	command := strings.TrimSpace(spec.Command)
	if command == "" {
		return TaskRun{}, fmt.Errorf("task command required")
	}
	resolvedCommand, err := resolveCommandForEnv(command, env)
	if err != nil {
		return TaskRun{}, err
	}

	taskCwd, err := resolveTaskCwd(workDir, spec.Cwd)
	if err != nil {
		return TaskRun{}, err
	}

	taskID, err := newTaskID()
	if err != nil {
		return TaskRun{}, err
	}
	if strings.TrimSpace(spec.Name) == "" {
		spec.Name = filepath.Base(command)
	}

	taskDir := filepath.Join(wsPath, "state", "tasks", taskID)
	if err := os.MkdirAll(taskDir, 0o700); err != nil {
		return TaskRun{}, fmt.Errorf("mkdir task state dir: %w", err)
	}
	logDir := filepath.Join(wsPath, "logs", "tasks")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return TaskRun{}, fmt.Errorf("mkdir task logs dir: %w", err)
	}

	stdoutLog := filepath.Join(logDir, taskID+".stdout.log")
	stderrLog := filepath.Join(logDir, taskID+".stderr.log")
	exitCodePath := filepath.Join(taskDir, "exit_code")
	finishedAtPath := filepath.Join(taskDir, "finished_at")
	cancelReasonPath := filepath.Join(taskDir, "cancel_reason")
	supervisorPath := filepath.Join(taskDir, "supervise.sh")
	if err := os.WriteFile(supervisorPath, []byte(taskSupervisorScript(stdoutLog, stderrLog, exitCodePath, finishedAtPath, cancelReasonPath)), 0o700); err != nil {
		return TaskRun{}, fmt.Errorf("write task supervisor: %w", err)
	}

	createdAt := time.Now().UTC()
	record := taskRecord{
		ID:        taskID,
		Name:      spec.Name,
		Workspace: workspaceName,
		Command:   resolvedCommand,
		Args:      append([]string{}, spec.Args...),
		Cwd:       taskCwd,
		Declared:  false,
		CreatedAt: createdAt,
		StdoutLog: stdoutLog,
		StderrLog: stderrLog,
	}
	if err := a.writeTaskRecord(taskDir, record); err != nil {
		return TaskRun{}, err
	}

	cmdArgs := append([]string{supervisorPath, resolvedCommand}, spec.Args...)
	cmd := exec.Command("/bin/sh", cmdArgs...)
	cmd.Dir = taskCwd
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return TaskRun{}, fmt.Errorf("start task: %w", err)
	}

	record.StartedAt = time.Now().UTC()
	record.PID = cmd.Process.Pid
	if err := a.writeTaskRecord(taskDir, record); err != nil {
		_ = cmd.Process.Kill()
		return TaskRun{}, err
	}
	if err := cmd.Process.Release(); err != nil {
		return TaskRun{}, fmt.Errorf("release task process: %w", err)
	}

	return a.TaskStatus(workspaceName, taskID)
}

func (a *App) StartDeclaredTask(workspaceName, taskName string) (TaskRun, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return TaskRun{}, err
	}
	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return TaskRun{}, err
	}

	var declared *TaskSpec
	for i := range manifest.Tasks {
		if manifest.Tasks[i].Name == taskName {
			declared = &manifest.Tasks[i]
			break
		}
	}
	if declared == nil {
		return TaskRun{}, fmt.Errorf("declared task %q not found", taskName)
	}
	if len(declared.Command) == 0 {
		return TaskRun{}, fmt.Errorf("declared task %q has no command", taskName)
	}

	task, err := a.StartTask(workspaceName, TaskStartSpec{
		Name:    declared.Name,
		Command: declared.Command[0],
		Args:    append([]string{}, declared.Command[1:]...),
		Cwd:     declared.Cwd,
	})
	if err != nil {
		return TaskRun{}, err
	}

	taskDir := filepath.Join(wsPath, "state", "tasks", task.ID)
	record, err := a.readTaskRecord(taskDir)
	if err != nil {
		return TaskRun{}, err
	}
	record.Declared = true
	if err := a.writeTaskRecord(taskDir, record); err != nil {
		return TaskRun{}, err
	}
	return a.TaskStatus(workspaceName, task.ID)
}

func (a *App) TaskStatus(workspaceName, taskID string) (TaskRun, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return TaskRun{}, err
	}
	taskDir := filepath.Join(wsPath, "state", "tasks", taskID)
	record, err := a.readTaskRecord(taskDir)
	if err != nil {
		return TaskRun{}, err
	}
	return a.taskFromRecord(taskDir, record)
}

func (a *App) TaskList(workspaceName string) ([]TaskRun, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return nil, err
	}

	taskRoot := filepath.Join(wsPath, "state", "tasks")
	entries, err := os.ReadDir(taskRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return []TaskRun{}, nil
		}
		return nil, fmt.Errorf("read tasks dir: %w", err)
	}

	tasks := make([]TaskRun, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		taskDir := filepath.Join(taskRoot, entry.Name())
		record, err := a.readTaskRecord(taskDir)
		if err != nil {
			return nil, err
		}
		task, err := a.taskFromRecord(taskDir, record)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	slices.SortFunc(tasks, func(a, b TaskRun) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})
	return tasks, nil
}

func (a *App) TaskLogs(workspaceName, taskID string) (TaskRunLogs, error) {
	task, err := a.TaskStatus(workspaceName, taskID)
	if err != nil {
		return TaskRunLogs{}, err
	}
	stdoutData, err := os.ReadFile(task.StdoutLog)
	if err != nil && !os.IsNotExist(err) {
		return TaskRunLogs{}, fmt.Errorf("read task stdout log: %w", err)
	}
	stderrData, err := os.ReadFile(task.StderrLog)
	if err != nil && !os.IsNotExist(err) {
		return TaskRunLogs{}, fmt.Errorf("read task stderr log: %w", err)
	}

	return TaskRunLogs{
		TaskID: task.ID,
		Stdout: string(stdoutData),
		Stderr: string(stderrData),
	}, nil
}

func (a *App) StopTask(workspaceName, taskID string) (TaskRun, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return TaskRun{}, err
	}
	taskDir := filepath.Join(wsPath, "state", "tasks", taskID)
	record, err := a.readTaskRecord(taskDir)
	if err != nil {
		return TaskRun{}, err
	}

	task, err := a.taskFromRecord(taskDir, record)
	if err != nil {
		return TaskRun{}, err
	}
	if task.State != TaskRunRunning && task.State != TaskRunPending {
		return task, nil
	}

	process, err := os.FindProcess(record.PID)
	if err != nil {
		return TaskRun{}, fmt.Errorf("find task process: %w", err)
	}
	if err := syscall.Kill(-record.PID, syscall.SIGTERM); err != nil {
		return TaskRun{}, fmt.Errorf("stop task: %w", err)
	}
	_ = process.Release()
	finishedAt := time.Now().UTC()
	exitCode := 143
	if err := writeStringMarker(filepath.Join(taskDir, "cancel_reason"), "requested stop"); err != nil {
		return TaskRun{}, err
	}
	if err := writeIntMarker(filepath.Join(taskDir, "exit_code"), exitCode); err != nil {
		return TaskRun{}, err
	}
	if err := writeTimeMarker(filepath.Join(taskDir, "finished_at"), finishedAt); err != nil {
		return TaskRun{}, err
	}
	return a.TaskStatus(workspaceName, taskID)
}

func (a *App) readTaskRecord(taskDir string) (taskRecord, error) {
	data, err := os.ReadFile(filepath.Join(taskDir, "task.json"))
	if err != nil {
		return taskRecord{}, err
	}
	var record taskRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return taskRecord{}, err
	}
	return record, nil
}

func (a *App) writeTaskRecord(taskDir string, record taskRecord) error {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(taskDir, "task.json")
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

func (a *App) taskFromRecord(taskDir string, record taskRecord) (TaskRun, error) {
	task := TaskRun{
		ID:           record.ID,
		Name:         record.Name,
		Workspace:    record.Workspace,
		Command:      record.Command,
		Args:         append([]string{}, record.Args...),
		Cwd:          record.Cwd,
		Declared:     record.Declared,
		CreatedAt:    record.CreatedAt,
		StartedAt:    record.StartedAt,
		PID:          record.PID,
		StdoutLog:    record.StdoutLog,
		StderrLog:    record.StderrLog,
		CancelReason: record.CancelReason,
	}

	if finishedAt, ok, err := readTimeMarker(filepath.Join(taskDir, "finished_at")); err != nil {
		return TaskRun{}, err
	} else if ok {
		task.FinishedAt = &finishedAt
	}
	if exitCode, ok, err := readIntMarker(filepath.Join(taskDir, "exit_code")); err != nil {
		return TaskRun{}, err
	} else if ok {
		task.ExitCode = &exitCode
	}
	if cancelReason, ok, err := readStringMarker(filepath.Join(taskDir, "cancel_reason")); err != nil {
		return TaskRun{}, err
	} else if ok {
		task.CancelReason = cancelReason
	}

	switch {
	case task.FinishedAt != nil && task.CancelReason != "":
		task.State = TaskRunCancelled
	case task.FinishedAt != nil && task.ExitCode != nil && *task.ExitCode == 0:
		task.State = TaskRunSucceeded
	case task.FinishedAt != nil:
		task.State = TaskRunFailed
	case record.PID == 0:
		task.State = TaskRunPending
	default:
		task.State = TaskRunRunning
	}

	return task, nil
}

func resolveTaskCwd(baseDir, requested string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" || requested == "." {
		return baseDir, nil
	}
	if filepath.IsAbs(requested) {
		info, err := os.Stat(requested)
		if err != nil {
			return "", fmt.Errorf("task cwd %q: %w", requested, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("task cwd %q is not a directory", requested)
		}
		return requested, nil
	}
	cwd := filepath.Join(baseDir, requested)
	info, err := os.Stat(cwd)
	if err != nil {
		return "", fmt.Errorf("task cwd %q: %w", cwd, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("task cwd %q is not a directory", cwd)
	}
	return cwd, nil
}

func newTaskID() (string, error) {
	var suffix [6]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", fmt.Errorf("generate task id: %w", err)
	}
	return fmt.Sprintf("%d-%s", time.Now().UTC().UnixMilli(), hex.EncodeToString(suffix[:])), nil
}

func taskSupervisorScript(stdoutLog, stderrLog, exitCodePath, finishedAtPath, cancelReasonPath string) string {
	return strings.Join([]string{
		"#!/bin/sh",
		`set +e`,
		`rm -f ` + shellQuote(cancelReasonPath),
		`"$@" > ` + shellQuote(stdoutLog) + ` 2> ` + shellQuote(stderrLog),
		`status=$?`,
		`printf '%s' "$status" > ` + shellQuote(exitCodePath),
		`date -u +"%Y-%m-%dT%H:%M:%SZ" > ` + shellQuote(finishedAtPath),
		`exit 0`,
		"",
	}, "\n")
}

func readTimeMarker(path string) (time.Time, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, err
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return time.Time{}, false, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false, err
	}
	return parsed, true, nil
}

func readIntMarker(path string) (int, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, false, nil
		}
		return 0, false, err
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return 0, false, nil
	}
	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
		return 0, false, err
	}
	return result, true, nil
}

func readStringMarker(path string) (string, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return "", false, nil
	}
	return value, true, nil
}

func writeTimeMarker(path string, value time.Time) error {
	return os.WriteFile(path, []byte(value.Format(time.RFC3339)), 0o600)
}

func writeIntMarker(path string, value int) error {
	return os.WriteFile(path, []byte(fmt.Sprintf("%d", value)), 0o600)
}

func writeStringMarker(path, value string) error {
	return os.WriteFile(path, []byte(value), 0o600)
}
