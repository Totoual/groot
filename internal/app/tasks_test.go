package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStartTaskPersistsRecordAndLogs(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)

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

	task, err := app.StartTask("crawlly", TaskStartSpec{
		Name:    "echo",
		Command: "/bin/sh",
		Args:    []string{"-c", "printf hello; printf boom >&2"},
	})
	if err != nil {
		t.Fatalf("StartTask returned error: %v", err)
	}
	if task.ID == "" {
		t.Fatal("expected task id to be set")
	}
	if task.State != WorkspaceTaskRunning {
		t.Fatalf("expected task to start running, got %q", task.State)
	}

	task = waitForTaskState(t, app, "crawlly", task.ID, WorkspaceTaskSucceeded)
	if task.ExitCode == nil || *task.ExitCode != 0 {
		t.Fatalf("unexpected exit code: %#v", task.ExitCode)
	}

	logs, err := app.TaskLogs("crawlly", task.ID)
	if err != nil {
		t.Fatalf("TaskLogs returned error: %v", err)
	}
	if logs.Stdout != "hello" {
		t.Fatalf("stdout = %q, want %q", logs.Stdout, "hello")
	}
	if logs.Stderr != "boom" {
		t.Fatalf("stderr = %q, want %q", logs.Stderr, "boom")
	}
}

func TestStartDeclaredTaskRunsManifestTask(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)

	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(filepath.Join(projectPath, "subdir"), 0o755); err != nil {
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
	manifest.Tasks = []TaskSpec{
		{
			Name:    "print",
			Command: []string{"/bin/sh", "-c", "pwd"},
			Cwd:     "subdir",
		},
	}
	if err := app.writeManifest(wsPath, manifest); err != nil {
		t.Fatalf("writeManifest returned error: %v", err)
	}

	task, err := app.StartDeclaredTask("crawlly", "print")
	if err != nil {
		t.Fatalf("StartDeclaredTask returned error: %v", err)
	}
	if !task.Declared {
		t.Fatal("expected declared task to be marked declared")
	}

	task = waitForTaskState(t, app, "crawlly", task.ID, WorkspaceTaskSucceeded)
	logs, err := app.TaskLogs("crawlly", task.ID)
	if err != nil {
		t.Fatalf("TaskLogs returned error: %v", err)
	}
	gotPath := strings.TrimSpace(logs.Stdout)
	wantPath, err := filepath.EvalSymlinks(filepath.Join(projectPath, "subdir"))
	if err != nil {
		t.Fatalf("EvalSymlinks returned error: %v", err)
	}
	if gotPath != wantPath {
		t.Fatalf("stdout = %q, want %q", gotPath, wantPath)
	}
}

func TestTaskListReturnsNewestFirst(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)

	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	first, err := app.StartTask("crawlly", TaskStartSpec{Name: "first", Command: "/bin/sh", Args: []string{"-c", "printf first"}})
	if err != nil {
		t.Fatalf("StartTask first returned error: %v", err)
	}
	second, err := app.StartTask("crawlly", TaskStartSpec{Name: "second", Command: "/bin/sh", Args: []string{"-c", "printf second"}})
	if err != nil {
		t.Fatalf("StartTask second returned error: %v", err)
	}

	waitForTaskState(t, app, "crawlly", first.ID, WorkspaceTaskSucceeded)
	waitForTaskState(t, app, "crawlly", second.ID, WorkspaceTaskSucceeded)

	tasks, err := app.TaskList("crawlly")
	if err != nil {
		t.Fatalf("TaskList returned error: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].ID != second.ID {
		t.Fatalf("expected newest task first, got %q then %q", tasks[0].ID, tasks[1].ID)
	}
}

func TestStopTaskCancelsRunningTask(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)

	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	task, err := app.StartTask("crawlly", TaskStartSpec{
		Name:    "sleep",
		Command: "/bin/sh",
		Args:    []string{"-c", "sleep 30"},
	})
	if err != nil {
		t.Fatalf("StartTask returned error: %v", err)
	}

	if _, err := app.StopTask("crawlly", task.ID); err != nil {
		t.Fatalf("StopTask returned error: %v", err)
	}

	task = waitForTaskState(t, app, "crawlly", task.ID, WorkspaceTaskCancelled)
	if task.CancelReason == "" {
		t.Fatal("expected cancel reason to be recorded")
	}
}

func waitForTaskState(t *testing.T, app *App, workspaceName, taskID string, want WorkspaceTaskState) WorkspaceTask {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		task, err := app.TaskStatus(workspaceName, taskID)
		if err != nil {
			t.Fatalf("TaskStatus returned error: %v", err)
		}
		if task.State == want {
			return task
		}
		time.Sleep(50 * time.Millisecond)
	}

	task, err := app.TaskStatus(workspaceName, taskID)
	if err != nil {
		t.Fatalf("TaskStatus returned error: %v", err)
	}
	t.Fatalf("timed out waiting for task %q to reach state %q, got %q", taskID, want, task.State)
	return WorkspaceTask{}
}
