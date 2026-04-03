package commands

import (
	"bytes"
	"os"
	"path/filepath"
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

func TestShellHookCmdInstallWritesManagedBlockToZshrc(t *testing.T) {
	a := app.NewApp(t.TempDir())
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("SHELL", "/bin/zsh")

	output, err := captureCommandStdout(func() error {
		return (&ShellHookCmd{}).Run(a, []string{"install"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(output, "Installed Groot shell hook") {
		t.Fatalf("unexpected output: %q", output)
	}

	data, err := os.ReadFile(filepath.Join(homeDir, ".zshrc"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, shellHookStartMarker) {
		t.Fatalf("expected start marker, got %q", content)
	}
	if !strings.Contains(content, shellHookLine) {
		t.Fatalf("expected shell hook line, got %q", content)
	}
	if !strings.Contains(content, shellHookEndMarker) {
		t.Fatalf("expected end marker, got %q", content)
	}
}

func TestShellHookCmdInstallIsIdempotent(t *testing.T) {
	a := app.NewApp(t.TempDir())
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("SHELL", "/bin/bash")

	if _, err := captureCommandStdout(func() error {
		return (&ShellHookCmd{}).Run(a, []string{"install"})
	}); err != nil {
		t.Fatalf("first install returned error: %v", err)
	}

	output, err := captureCommandStdout(func() error {
		return (&ShellHookCmd{}).Run(a, []string{"install"})
	})
	if err != nil {
		t.Fatalf("second install returned error: %v", err)
	}
	if !strings.Contains(output, "already installed") {
		t.Fatalf("unexpected output: %q", output)
	}

	data, err := os.ReadFile(filepath.Join(homeDir, ".bashrc"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.Count(string(data), shellHookLine) != 1 {
		t.Fatalf("expected one shell hook line, got %q", string(data))
	}
}

func TestShellHookCmdInstallRejectsUnsupportedShell(t *testing.T) {
	a := app.NewApp(t.TempDir())
	t.Setenv("SHELL", "/bin/fish")

	err := (&ShellHookCmd{}).Run(a, []string{"install"})
	if err == nil {
		t.Fatal("expected unsupported shell error")
	}
	if !strings.Contains(err.Error(), `unsupported shell "fish"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func captureCommandStdout(fn func() error) (string, error) {
	stdout, _, err := captureCommandOutput(fn)
	return stdout, err
}

func captureCommandOutput(fn func() error) (string, string, error) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return "", "", err
	}
	defer stdoutR.Close()

	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		_ = stdoutW.Close()
		return "", "", err
	}
	defer stderrR.Close()

	os.Stdout = stdoutW
	os.Stderr = stderrW
	runErr := fn()
	_ = stdoutW.Close()
	_ = stderrW.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var stdoutBuf bytes.Buffer
	if _, err := stdoutBuf.ReadFrom(stdoutR); err != nil {
		return "", "", err
	}
	var stderrBuf bytes.Buffer
	if _, err := stderrBuf.ReadFrom(stderrR); err != nil {
		return "", "", err
	}

	return stdoutBuf.String(), stderrBuf.String(), runErr
}
