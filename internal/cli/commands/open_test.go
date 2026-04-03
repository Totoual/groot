package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/totoual/groot/internal/app"
)

func TestOpenCmdRunReusesWorkspaceBoundToProjectPath(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	hostHome := filepath.Join(root, "host-home")
	t.Setenv("HOME", hostHome)
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

	scriptPath := filepath.Join(root, "open-capture.sh")
	script := "#!/bin/sh\nprintf '%s' \"$GROOT_WORKSPACE\" > open-workspace.txt\npwd > open-pwd.txt\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	output, err := captureCommandStdout(func() error {
		return (&OpenCmd{}).Run(a, []string{projectPath, "--ide", scriptPath})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(output) != "" {
		t.Fatalf("expected reused-open wrapper to stay quiet, got %q", output)
	}

	gotWorkspace, err := os.ReadFile(filepath.Join(projectPath, "open-workspace.txt"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.TrimSpace(string(gotWorkspace)) != "crawlly" {
		t.Fatalf("GROOT_WORKSPACE = %q, want %q", strings.TrimSpace(string(gotWorkspace)), "crawlly")
	}
}

func TestOpenCmdRunCreatesWorkspaceForFirstSeenProjectPath(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	hostHome := filepath.Join(root, "host-home")
	t.Setenv("HOME", hostHome)
	t.Setenv("PATH", "/usr/bin:/bin")

	projectPath := filepath.Join(root, "repos", "the_grime_tcg")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	scriptPath := filepath.Join(root, "open-capture.sh")
	script := "#!/bin/sh\nprintf '%s' \"$GROOT_WORKSPACE\" > open-workspace.txt\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	output, err := captureCommandStdout(func() error {
		return (&OpenCmd{}).Run(a, []string{projectPath, "--ide", scriptPath})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(output, `Created workspace "the_grime_tcg"`) {
		t.Fatalf("expected creation message, got %q", output)
	}

	gotWorkspace, err := os.ReadFile(filepath.Join(projectPath, "open-workspace.txt"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.TrimSpace(string(gotWorkspace)) != "the_grime_tcg" {
		t.Fatalf("GROOT_WORKSPACE = %q, want %q", strings.TrimSpace(string(gotWorkspace)), "the_grime_tcg")
	}

	manifest, err := loadManifest(filepath.Join(a.WorkspaceDir(), "the_grime_tcg"))
	if err != nil {
		t.Fatalf("loadManifest returned error: %v", err)
	}
	if manifest.ProjectPath != projectPath {
		t.Fatalf("ProjectPath = %q, want %q", manifest.ProjectPath, projectPath)
	}
}

func TestOpenCmdRejectsMissingProjectPath(t *testing.T) {
	a := app.NewApp(t.TempDir())

	err := (&OpenCmd{}).Run(a, nil)
	if err == nil {
		t.Fatal("expected argument validation error")
	}
}

func loadManifest(wsPath string) (app.Manifest, error) {
	data, err := os.ReadFile(filepath.Join(wsPath, "manifest.json"))
	if err != nil {
		return app.Manifest{}, err
	}

	var manifest app.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return app.Manifest{}, err
	}

	return manifest, nil
}
