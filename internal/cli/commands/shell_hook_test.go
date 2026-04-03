package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/totoual/groot/internal/app"
)

func TestShellHookCmdRunPrintsExportsForCurrentWorkspace(t *testing.T) {
	a := app.NewApp(t.TempDir())
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("PATH", "/usr/bin:/bin")
	t.Setenv("GROOT_WORKSPACE", "crawlly")

	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	output, err := captureCommandStdout(func() error {
		return (&ShellHookCmd{}).Run(a, nil)
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(output, "export GROOT_WORKSPACE='crawlly'") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestShellHookCmdRunAllowsEmptyContext(t *testing.T) {
	a := app.NewApp(t.TempDir())

	output, err := captureCommandStdout(func() error {
		return (&ShellHookCmd{}).Run(a, nil)
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if output != "" {
		t.Fatalf("expected empty output, got %q", output)
	}
}

func TestShellHookCmdRejectsArguments(t *testing.T) {
	a := app.NewApp(t.TempDir())

	err := (&ShellHookCmd{}).Run(a, []string{"extra"})
	if err == nil {
		t.Fatal("expected argument validation error")
	}
}

func captureCommandStdout(fn func() error) (string, error) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	defer r.Close()

	os.Stdout = w
	runErr := fn()
	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		return "", err
	}
	return buf.String(), runErr
}
