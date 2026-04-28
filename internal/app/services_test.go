package app

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestStartServiceStatusLogsAndStop(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	projectPath := setupServiceProject(t, app, root, []ServiceSpec{
		{
			Name:    "api",
			Command: []string{"/bin/sh", "-c", "printf out; printf err >&2; sleep 30"},
		},
	})
	_ = projectPath

	service, err := app.StartService("crawlly", "api")
	if err != nil {
		t.Fatalf("StartService returned error: %v", err)
	}
	if service.State != ServiceRunning {
		t.Fatalf("service state = %q, want %q", service.State, ServiceRunning)
	}

	waitForServiceLogs(t, app, "crawlly", "api", "out", "err")
	logs, err := app.ServiceLogs("crawlly", "api")
	if err != nil {
		t.Fatalf("ServiceLogs returned error: %v", err)
	}
	if logs.Stdout != "out" {
		t.Fatalf("stdout = %q, want %q", logs.Stdout, "out")
	}
	if logs.Stderr != "err" {
		t.Fatalf("stderr = %q, want %q", logs.Stderr, "err")
	}

	service, err = app.StopService("crawlly", "api")
	if err != nil {
		t.Fatalf("StopService returned error: %v", err)
	}
	if service.State != ServiceStopped {
		t.Fatalf("service state = %q, want %q", service.State, ServiceStopped)
	}
	if service.StopReason != "requested stop" {
		t.Fatalf("stop reason = %q, want %q", service.StopReason, "requested stop")
	}

	events, err := app.EventList("crawlly", EventListOptions{})
	if err != nil {
		t.Fatalf("EventList returned error: %v", err)
	}
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %#v", events)
	}
	if events[0].Kind != EventKindServiceStopped {
		t.Fatalf("newest event kind = %q, want %q", events[0].Kind, EventKindServiceStopped)
	}
	if events[1].Kind != EventKindServiceStarted {
		t.Fatalf("second event kind = %q, want %q", events[1].Kind, EventKindServiceStarted)
	}
}

func TestServiceListIncludesDeclaredStoppedAndRunningServices(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	setupServiceProject(t, app, root, []ServiceSpec{
		{Name: "api", Command: []string{"/bin/sh", "-c", "sleep 30"}},
		{Name: "worker", Command: []string{"/bin/sh", "-c", "sleep 30"}},
	})

	if _, err := app.StartService("crawlly", "api"); err != nil {
		t.Fatalf("StartService returned error: %v", err)
	}

	services, err := app.ServiceList("crawlly")
	if err != nil {
		t.Fatalf("ServiceList returned error: %v", err)
	}
	if len(services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(services))
	}
	if services[0].Name != "api" || services[0].State != ServiceRunning {
		t.Fatalf("unexpected first service: %#v", services[0])
	}
	if services[1].Name != "worker" || services[1].State != ServiceStopped {
		t.Fatalf("unexpected second service: %#v", services[1])
	}
}

func TestServiceFailureEmitsFailedEventOnObservation(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	setupServiceProject(t, app, root, []ServiceSpec{
		{Name: "api", Command: []string{"/bin/sh", "-c", "exit 7"}},
	})

	if _, err := app.StartService("crawlly", "api"); err != nil {
		t.Fatalf("StartService returned error: %v", err)
	}
	service := waitForServiceState(t, app, "crawlly", "api", ServiceFailed)
	if service.ExitCode == nil || *service.ExitCode != 7 {
		t.Fatalf("unexpected exit code: %#v", service.ExitCode)
	}

	events, err := app.EventList("crawlly", EventListOptions{})
	if err != nil {
		t.Fatalf("EventList returned error: %v", err)
	}
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %#v", events)
	}
	if events[0].Kind != EventKindServiceFailed {
		t.Fatalf("newest event kind = %q, want %q", events[0].Kind, EventKindServiceFailed)
	}
}

func TestServiceStatusMarksKilledProcessFailed(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	setupServiceProject(t, app, root, []ServiceSpec{
		{Name: "api", Command: []string{"/bin/sh", "-c", "sleep 30"}},
	})

	service, err := app.StartService("crawlly", "api")
	if err != nil {
		t.Fatalf("StartService returned error: %v", err)
	}
	if err := syscall.Kill(-service.PID, syscall.SIGKILL); err != nil {
		t.Fatalf("Kill returned error: %v", err)
	}

	service = waitForServiceState(t, app, "crawlly", "api", ServiceFailed)
	if service.ExitCode == nil || *service.ExitCode != 1 {
		t.Fatalf("unexpected exit code after kill: %#v", service.ExitCode)
	}
	if service.StoppedAt == nil {
		t.Fatalf("expected finished timestamp after kill")
	}
}

func TestServiceOnFailureRestartsOnObservation(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	projectPath := setupServiceProject(t, app, root, []ServiceSpec{
		{Name: "api", Command: []string{"/bin/sh", "service.sh"}, Restart: "on-failure"},
	})
	scriptPath := filepath.Join(projectPath, "service.sh")
	script := strings.Join([]string{
		"#!/bin/sh",
		`count_file=".service-count"`,
		`count=0`,
		`if [ -f "$count_file" ]; then count=$(cat "$count_file"); fi`,
		`count=$((count + 1))`,
		`printf '%s' "$count" > "$count_file"`,
		`if [ "$count" -eq 1 ]; then exit 7; fi`,
		`sleep 30`,
		"",
	}, "\n")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	if _, err := app.StartService("crawlly", "api"); err != nil {
		t.Fatalf("StartService returned error: %v", err)
	}
	waitForServiceStateWithOptions(t, app, "crawlly", "api", ServiceFailed, serviceStatusOptions{})

	service, err := app.ServiceStatus("crawlly", "api")
	if err != nil {
		t.Fatalf("ServiceStatus returned error: %v", err)
	}
	if service.State != ServiceRunning {
		t.Fatalf("service state = %q, want %q", service.State, ServiceRunning)
	}

	waitForFileContent(t, filepath.Join(projectPath, ".service-count"), "2")
	_ = stopServiceIfRunning(t, app, "crawlly", "api")

	events, err := app.EventList("crawlly", EventListOptions{})
	if err != nil {
		t.Fatalf("EventList returned error: %v", err)
	}
	var started, failed int
	for _, event := range events {
		switch event.Kind {
		case EventKindServiceStarted:
			started++
		case EventKindServiceFailed:
			failed++
		}
	}
	if started < 2 {
		t.Fatalf("expected at least 2 service.started events, got %#v", events)
	}
	if failed < 1 {
		t.Fatalf("expected at least 1 service.failed event, got %#v", events)
	}
}

func TestStopServiceDisablesOnFailureRestartAfterFailure(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	setupServiceProject(t, app, root, []ServiceSpec{
		{Name: "api", Command: []string{"/bin/sh", "-c", "exit 7"}, Restart: "on-failure"},
	})

	if _, err := app.StartService("crawlly", "api"); err != nil {
		t.Fatalf("StartService returned error: %v", err)
	}
	waitForServiceStateWithOptions(t, app, "crawlly", "api", ServiceFailed, serviceStatusOptions{})

	service, err := app.StopService("crawlly", "api")
	if err != nil {
		t.Fatalf("StopService returned error: %v", err)
	}
	if service.State != ServiceStopped {
		t.Fatalf("service state = %q, want %q", service.State, ServiceStopped)
	}
	if service.StopReason != "requested stop" {
		t.Fatalf("stop reason = %q, want %q", service.StopReason, "requested stop")
	}

	service, err = app.ServiceStatus("crawlly", "api")
	if err != nil {
		t.Fatalf("ServiceStatus returned error: %v", err)
	}
	if service.State != ServiceStopped {
		t.Fatalf("service state after stop = %q, want %q", service.State, ServiceStopped)
	}

	events, err := app.EventList("crawlly", EventListOptions{})
	if err != nil {
		t.Fatalf("EventList returned error: %v", err)
	}
	if len(events) < 3 {
		t.Fatalf("expected at least 3 events, got %#v", events)
	}
	if events[0].Kind != EventKindServiceStopped {
		t.Fatalf("newest event kind = %q, want %q", events[0].Kind, EventKindServiceStopped)
	}
}

func TestDeclareServiceUpdatesManifest(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)

	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	if err := app.DeclareService("crawlly", ServiceSpec{
		Name:    "api",
		Command: []string{"go", "run", "./cmd/api"},
		Cwd:     ".",
		Restart: "manual",
	}); err != nil {
		t.Fatalf("DeclareService returned error: %v", err)
	}

	services, err := app.DeclaredServices("crawlly")
	if err != nil {
		t.Fatalf("DeclaredServices returned error: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}
	if services[0].Name != "api" || strings.Join(services[0].Command, " ") != "go run ./cmd/api" || services[0].Restart != "manual" {
		t.Fatalf("unexpected declared service: %#v", services[0])
	}

	if err := app.DeclareService("crawlly", ServiceSpec{
		Name:    "api",
		Command: []string{"go", "run", "./cmd/server"},
		Cwd:     "backend",
		Restart: "on-failure",
	}); err != nil {
		t.Fatalf("DeclareService update returned error: %v", err)
	}

	services, err = app.DeclaredServices("crawlly")
	if err != nil {
		t.Fatalf("DeclaredServices returned error: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("expected 1 service after update, got %d", len(services))
	}
	if strings.Join(services[0].Command, " ") != "go run ./cmd/server" || services[0].Cwd != "backend" || services[0].Restart != "on-failure" {
		t.Fatalf("unexpected updated service: %#v", services[0])
	}
}

func TestDeleteServiceRemovesDeclaration(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)

	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := app.DeclareService("crawlly", ServiceSpec{Name: "api", Command: []string{"go", "run", "./cmd/api"}}); err != nil {
		t.Fatalf("DeclareService returned error: %v", err)
	}

	if err := app.DeleteService("crawlly", "api"); err != nil {
		t.Fatalf("DeleteService returned error: %v", err)
	}

	services, err := app.DeclaredServices("crawlly")
	if err != nil {
		t.Fatalf("DeclaredServices returned error: %v", err)
	}
	if len(services) != 0 {
		t.Fatalf("expected no services, got %#v", services)
	}
}

func setupServiceProject(t *testing.T, app *App, root string, services []ServiceSpec) string {
	t.Helper()

	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := app.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}
	wsPath, err := app.EnsureWorkspace("crawlly")
	if err != nil {
		t.Fatalf("EnsureWorkspace returned error: %v", err)
	}
	manifest, err := app.getManifest(wsPath)
	if err != nil {
		t.Fatalf("getManifest returned error: %v", err)
	}
	manifest.Services = services
	if err := app.writeManifest(wsPath, manifest); err != nil {
		t.Fatalf("writeManifest returned error: %v", err)
	}
	return projectPath
}

func waitForServiceState(t *testing.T, app *App, workspaceName, serviceName string, want ServiceState) ServiceStatus {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		service, err := app.ServiceStatus(workspaceName, serviceName)
		if err != nil {
			t.Fatalf("ServiceStatus returned error: %v", err)
		}
		if service.State == want {
			return service
		}
		time.Sleep(50 * time.Millisecond)
	}
	service, err := app.ServiceStatus(workspaceName, serviceName)
	if err != nil {
		t.Fatalf("ServiceStatus returned error: %v", err)
	}
	t.Fatalf("timed out waiting for service %q to reach state %q, got %q", serviceName, want, service.State)
	return ServiceStatus{}
}

func waitForServiceStateWithOptions(t *testing.T, app *App, workspaceName, serviceName string, want ServiceState, opts serviceStatusOptions) ServiceStatus {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		service, err := app.serviceStatus(workspaceName, serviceName, opts)
		if err != nil {
			t.Fatalf("serviceStatus returned error: %v", err)
		}
		if service.State == want {
			return service
		}
		time.Sleep(50 * time.Millisecond)
	}
	service, err := app.serviceStatus(workspaceName, serviceName, opts)
	if err != nil {
		t.Fatalf("serviceStatus returned error: %v", err)
	}
	t.Fatalf("timed out waiting for service %q to reach state %q, got %q", serviceName, want, service.State)
	return ServiceStatus{}
}

func waitForServiceLogs(t *testing.T, app *App, workspaceName, serviceName, wantStdout, wantStderr string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		logs, err := app.ServiceLogs(workspaceName, serviceName)
		if err != nil {
			t.Fatalf("ServiceLogs returned error: %v", err)
		}
		if strings.Contains(logs.Stdout, wantStdout) && strings.Contains(logs.Stderr, wantStderr) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	logs, err := app.ServiceLogs(workspaceName, serviceName)
	if err != nil {
		t.Fatalf("ServiceLogs returned error: %v", err)
	}
	t.Fatalf("timed out waiting for service logs, got stdout=%q stderr=%q", logs.Stdout, logs.Stderr)
}

func waitForFileContent(t *testing.T, path, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil && strings.TrimSpace(string(data)) == want {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	t.Fatalf("timed out waiting for %s to contain %q, got %q", path, want, strings.TrimSpace(string(data)))
}

func stopServiceIfRunning(t *testing.T, app *App, workspaceName, serviceName string) ServiceStatus {
	t.Helper()
	service, err := app.serviceStatus(workspaceName, serviceName, serviceStatusOptions{})
	if err != nil {
		t.Fatalf("serviceStatus returned error: %v", err)
	}
	if service.State != ServiceRunning && service.State != ServiceStarting {
		return service
	}
	service, err = app.StopService(workspaceName, serviceName)
	if err != nil {
		t.Fatalf("StopService returned error: %v", err)
	}
	return service
}
