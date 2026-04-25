package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/totoual/groot/internal/app"
)

func TestEventCmdZeroValueUsesDefaultSubcommands(t *testing.T) {
	var buf bytes.Buffer
	(&EventCmd{}).PrintHelp(&buf)

	if !strings.Contains(buf.String(), "list") {
		t.Fatalf("expected help to include list command, got %q", buf.String())
	}
}

func TestEventListCmdRunPrintsEvents(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)

	setupEventProject(t, a, root)
	workspaceName := "crawlly"
	task, err := a.StartTask("crawlly", app.TaskStartSpec{Name: "echo", Command: "/bin/sh", Args: []string{"-c", "printf ok"}})
	if err != nil {
		t.Fatalf("StartTask returned error: %v", err)
	}
	waitForTaskStateCmd(t, a, workspaceName, task.ID, app.TaskRunSucceeded)

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&eventListCmd{}).Run(a, []string{workspaceName})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, app.EventKindTaskStarted) || !strings.Contains(stdout, app.EventKindTaskExited) {
		t.Fatalf("expected task lifecycle events in output, got %q", stdout)
	}
}

func TestEventListCmdRunPrintsEmptyState(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)

	setupEventProject(t, a, root)
	workspaceName := "crawlly"
	stdout, stderr, err := captureCommandOutput(func() error {
		return (&eventListCmd{}).Run(a, []string{workspaceName})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if strings.TrimSpace(stdout) != "No events." {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
}

func setupEventProject(t *testing.T, a *app.App, root string) string {
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
