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

func TestTaskCmdZeroValueUsesDefaultSubcommands(t *testing.T) {
	var buf bytes.Buffer
	(&TaskCmd{}).PrintHelp(&buf)

	output := buf.String()
	for _, want := range []string{"start", "status", "list", "logs", "stop"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help to include %q, got %q", want, output)
		}
	}
}

func TestTaskStartAndStatusCmdRun(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)

	projectPath := setupTaskProject(t, a, root)

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&taskStartCmd{}).Run(a, []string{projectPath, "--name", "echo", "/bin/sh", "-c", "printf hello"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	taskID := extractTaskID(t, stdout)
	waitForTaskStateCmd(t, a, projectPath, taskID, app.TaskRunSucceeded)

	stdout, stderr, err = captureCommandOutput(func() error {
		return (&taskStatusCmd{}).Run(a, []string{projectPath, taskID})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, "State: succeeded") {
		t.Fatalf("expected succeeded state in output, got %q", stdout)
	}
}

func TestTaskStartCmdRunStartsDeclaredTask(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)

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
		Tasks: []app.TaskSpec{
			{Name: "print", Command: []string{"/bin/sh", "-c", "printf declared"}},
		},
		Services: []app.ServiceSpec{},
		Env:      map[string]string{},
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsPath, "manifest.json"), data, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&taskStartCmd{}).Run(a, []string{projectPath, "--task", "print"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	taskID := extractTaskID(t, stdout)
	waitForTaskStateCmd(t, a, projectPath, taskID, app.TaskRunSucceeded)
	if !strings.Contains(stdout, "Declared: yes") {
		t.Fatalf("expected declared task output, got %q", stdout)
	}
}

func TestTaskListCmdRunPrintsTasks(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)

	projectPath := setupTaskProject(t, a, root)
	task, err := a.StartTask("crawlly", app.TaskStartSpec{Name: "echo", Command: "/bin/sh", Args: []string{"-c", "printf hello"}})
	if err != nil {
		t.Fatalf("StartTask returned error: %v", err)
	}
	waitForTaskStateCmd(t, a, projectPath, task.ID, app.TaskRunSucceeded)

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&taskListCmd{}).Run(a, []string{projectPath})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, task.ID) || !strings.Contains(stdout, "succeeded") || !strings.Contains(stdout, "echo") {
		t.Fatalf("unexpected list output: %q", stdout)
	}
}

func TestTaskLogsCmdRunPrintsStdoutAndStderr(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)

	projectPath := setupTaskProject(t, a, root)
	task, err := a.StartTask("crawlly", app.TaskStartSpec{Name: "logs", Command: "/bin/sh", Args: []string{"-c", "printf out; printf err >&2"}})
	if err != nil {
		t.Fatalf("StartTask returned error: %v", err)
	}
	waitForTaskStateCmd(t, a, projectPath, task.ID, app.TaskRunSucceeded)

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&taskLogsCmd{}).Run(a, []string{projectPath, task.ID})
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

func TestTaskStopCmdRunCancelsTask(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)

	projectPath := setupTaskProject(t, a, root)
	task, err := a.StartTask("crawlly", app.TaskStartSpec{Name: "sleep", Command: "/bin/sh", Args: []string{"-c", "sleep 30"}})
	if err != nil {
		t.Fatalf("StartTask returned error: %v", err)
	}

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&taskStopCmd{}).Run(a, []string{projectPath, task.ID})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, "State: cancelled") {
		t.Fatalf("expected cancelled state in output, got %q", stdout)
	}
}

func setupTaskProject(t *testing.T, a *app.App, root string) string {
	t.Helper()

	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}
	return projectPath
}

func extractTaskID(t *testing.T, output string) string {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "Task: ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Task: "))
		}
	}
	t.Fatalf("task id not found in output %q", output)
	return ""
}

func waitForTaskStateCmd(t *testing.T, a *app.App, projectPath, taskID string, want app.TaskRunState) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		workspaceName, _, err := a.ResolveOrCreateWorkspaceByProjectPath(projectPath)
		if err != nil {
			t.Fatalf("ResolveOrCreateWorkspaceByProjectPath returned error: %v", err)
		}
		task, err := a.TaskStatus(workspaceName, taskID)
		if err != nil {
			t.Fatalf("TaskStatus returned error: %v", err)
		}
		if task.State == want {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for task %q to reach %q", taskID, want)
}
