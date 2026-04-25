package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/totoual/groot/internal/app"
)

func TestServiceCmdZeroValueUsesDefaultSubcommands(t *testing.T) {
	var buf bytes.Buffer
	(&ServiceCmd{}).PrintHelp(&buf)

	output := buf.String()
	for _, want := range []string{"start", "status", "list", "logs", "stop"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help to include %q, got %q", want, output)
		}
	}
}

func TestServiceStartAndStatusCmdRun(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	setupServiceProjectCmd(t, a, root, []app.ServiceSpec{
		{Name: "api", Command: []string{"/bin/sh", "-c", "sleep 30"}},
	})
	workspaceName := "crawlly"

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&serviceStartCmd{}).Run(a, []string{workspaceName, "api"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, "State: running") {
		t.Fatalf("expected running state in output, got %q", stdout)
	}

	stdout, stderr, err = captureCommandOutput(func() error {
		return (&serviceStatusCmd{}).Run(a, []string{workspaceName, "api"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, "Service: api") || !strings.Contains(stdout, "State: running") {
		t.Fatalf("unexpected status output: %q", stdout)
	}
}

func TestServiceListCmdRunPrintsServices(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	setupServiceProjectCmd(t, a, root, []app.ServiceSpec{
		{Name: "api", Command: []string{"/bin/sh", "-c", "sleep 30"}},
		{Name: "worker", Command: []string{"/bin/sh", "-c", "sleep 30"}},
	})
	workspaceName := "crawlly"

	if _, err := a.StartService("crawlly", "api"); err != nil {
		t.Fatalf("StartService returned error: %v", err)
	}

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&serviceListCmd{}).Run(a, []string{workspaceName})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, "api\trunning") || !strings.Contains(stdout, "worker\tstopped") {
		t.Fatalf("unexpected list output: %q", stdout)
	}
}

func TestServiceLogsCmdRunPrintsStdoutAndStderr(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	setupServiceProjectCmd(t, a, root, []app.ServiceSpec{
		{Name: "api", Command: []string{"/bin/sh", "-c", "printf out; printf err >&2; sleep 30"}},
	})
	workspaceName := "crawlly"

	if _, err := a.StartService("crawlly", "api"); err != nil {
		t.Fatalf("StartService returned error: %v", err)
	}
	waitForServiceLogsCmd(t, a, workspaceName, "api", "out", "err")

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&serviceLogsCmd{}).Run(a, []string{workspaceName, "api"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if stdout != "out" {
		t.Fatalf("stdout = %q, want %q", stdout, "out")
	}
	if stderr != "err" {
		t.Fatalf("stderr = %q, want %q", stderr, "err")
	}
}

func TestServiceStopCmdRunStopsService(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	setupServiceProjectCmd(t, a, root, []app.ServiceSpec{
		{Name: "api", Command: []string{"/bin/sh", "-c", "sleep 30"}},
	})
	workspaceName := "crawlly"

	if _, err := a.StartService("crawlly", "api"); err != nil {
		t.Fatalf("StartService returned error: %v", err)
	}

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&serviceStopCmd{}).Run(a, []string{workspaceName, "api"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, "State: stopped") {
		t.Fatalf("expected stopped state in output, got %q", stdout)
	}
}

func TestServiceAddRemoveAndListDeclaredCmds(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)

	setupTaskProject(t, a, root)
	workspaceName := "crawlly"

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&serviceAddCmd{}).Run(a, []string{workspaceName, "api", "--cwd", ".", "--restart", "manual", "--", "go", "run", "./cmd/api"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, `Declared service "api"`) {
		t.Fatalf("unexpected add output: %q", stdout)
	}

	stdout, stderr, err = captureCommandOutput(func() error {
		return (&serviceListDeclaredCmd{}).Run(a, []string{workspaceName})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, "api\tgo run ./cmd/api") {
		t.Fatalf("unexpected list-declared output: %q", stdout)
	}

	stdout, stderr, err = captureCommandOutput(func() error {
		return (&serviceRemoveCmd{}).Run(a, []string{workspaceName, "api"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, `Removed service "api"`) {
		t.Fatalf("unexpected remove output: %q", stdout)
	}
}

func setupServiceProjectCmd(t *testing.T, a *app.App, root string, services []app.ServiceSpec) string {
	t.Helper()
	projectPath := setupTaskProject(t, a, root)
	wsPath, err := a.EnsureWorkspace("crawlly")
	if err != nil {
		t.Fatalf("EnsureWorkspace returned error: %v", err)
	}
	manifest := app.Manifest{
		SchemaVersion: 1,
		Name:          "crawlly",
		ProjectPath:   projectPath,
		Packages:      []app.PackageSpec{},
		Tasks:         []app.TaskSpec{},
		Services:      services,
		Env:           map[string]string{},
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsPath, "manifest.json"), data, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	return projectPath
}

func waitForServiceLogsCmd(t *testing.T, a *app.App, workspaceName, serviceName, wantStdout, wantStderr string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		logs, err := a.ServiceLogs(workspaceName, serviceName)
		if err != nil {
			t.Fatalf("ServiceLogs returned error: %v", err)
		}
		if strings.Contains(logs.Stdout, wantStdout) && strings.Contains(logs.Stderr, wantStderr) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for service %q logs", serviceName)
}
