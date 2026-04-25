package app

import (
	"os"
	"path/filepath"
	"strings"
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
