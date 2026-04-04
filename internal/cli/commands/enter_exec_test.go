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
	stdout, stderr, err := captureCommandOutput(func() error {
		return (&ExecCmd{}).Run(a, []string{projectPath, scriptPath, outFile})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected reused-exec stdout to stay quiet, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected reused-exec stderr to stay quiet, got %q", stderr)
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
	stdout, stderr, err := captureCommandOutput(func() error {
		return (&ExecCmd{}).Run(a, []string{projectPath, scriptPath, outFile})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected first-exec stdout to stay quiet, got %q", stdout)
	}
	if !strings.Contains(stderr, `Created workspace "the_grime_tcg"`) {
		t.Fatalf("expected creation message on stderr, got %q", stderr)
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

func TestExecCmdStrictRuntimeRejectsUndeclaredToolchains(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	t.Setenv("SHELL", "/bin/sh")
	t.Setenv("PATH", "/usr/bin:/bin")
	t.Setenv("GROOT_STRICT_RUNTIME", "1")

	projectPath := filepath.Join(root, "repos", "the_grime_tcg")
	backendDir := filepath.Join(projectPath, "backend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "go.mod"), []byte("module example.com/tcg\n\ngo 1.25.4\n"), 0o600); err != nil {
		t.Fatalf("WriteFile go.mod returned error: %v", err)
	}

	scriptPath := filepath.Join(root, "capture.sh")
	script := "#!/bin/sh\nprintf 'should-not-run' > \"$1\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	outFile := filepath.Join(root, "out.txt")
	stdout, stderr, err := captureCommandOutput(func() error {
		return (&ExecCmd{}).Run(a, []string{projectPath, scriptPath, outFile})
	})
	if err == nil {
		t.Fatal("expected strict runtime mode to reject undeclared toolchains")
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout to stay quiet, got %q", stdout)
	}
	if !strings.Contains(err.Error(), `strict runtime mode rejected undeclared detected runtimes`) {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr, `Workspace "the_grime_tcg" does not declare detected runtimes: go@1.25.4`) {
		t.Fatalf("expected strict warning on stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, `Strict runtime mode is enabled via GROOT_STRICT_RUNTIME`) {
		t.Fatalf("expected strict mode note on stderr, got %q", stderr)
	}
	if _, statErr := os.Stat(outFile); !os.IsNotExist(statErr) {
		t.Fatalf("expected command not to run, stat err=%v", statErr)
	}
}
