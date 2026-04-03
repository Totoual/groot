package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/totoual/groot/internal/app"
)

func TestExecCmdRunReusesWorkspaceBoundToProjectPath(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	t.Setenv("SHELL", "/bin/sh")
	t.Setenv("PATH", "/usr/bin:/bin")

	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	projectPath := filepath.Join(root, "repos", "goCrawl")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	scriptPath := filepath.Join(root, "capture.sh")
	script := "#!/bin/sh\npwd > \"$1\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	outFile := filepath.Join(root, "pwd.txt")
	output, err := captureCommandStdout(func() error {
		return (&ExecCmd{}).Run(a, []string{projectPath, scriptPath, outFile})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(output) != "" {
		t.Fatalf("expected reused-exec wrapper to stay quiet, got %q", output)
	}

	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	wantProjectPath, err := filepath.EvalSymlinks(projectPath)
	if err != nil {
		t.Fatalf("EvalSymlinks returned error: %v", err)
	}
	if strings.TrimSpace(string(got)) != wantProjectPath {
		t.Fatalf("pwd = %q, want %q", strings.TrimSpace(string(got)), wantProjectPath)
	}
}

func TestExecCmdRunCreatesWorkspaceForFirstSeenProjectPath(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	t.Setenv("SHELL", "/bin/sh")
	t.Setenv("PATH", "/usr/bin:/bin")

	projectPath := filepath.Join(root, "repos", "the_grime_tcg")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	scriptPath := filepath.Join(root, "capture.sh")
	script := "#!/bin/sh\nprintf '%s' \"$GROOT_WORKSPACE\" > \"$1\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	outFile := filepath.Join(root, "workspace.txt")
	output, err := captureCommandStdout(func() error {
		return (&ExecCmd{}).Run(a, []string{projectPath, scriptPath, outFile})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(output, `Created workspace "the_grime_tcg"`) {
		t.Fatalf("expected creation message, got %q", output)
	}

	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.TrimSpace(string(got)) != "the_grime_tcg" {
		t.Fatalf("GROOT_WORKSPACE = %q, want %q", strings.TrimSpace(string(got)), "the_grime_tcg")
	}
}

func TestEnterCmdRejectsMissingProjectPath(t *testing.T) {
	a := app.NewApp(t.TempDir())

	err := (&EnterCmd{}).Run(a, nil)
	if err == nil {
		t.Fatal("expected argument validation error")
	}
}

func TestExecCmdRejectsMissingPathOrCommand(t *testing.T) {
	a := app.NewApp(t.TempDir())

	err := (&ExecCmd{}).Run(a, []string{"only-path"})
	if err == nil {
		t.Fatal("expected argument validation error")
	}
}
