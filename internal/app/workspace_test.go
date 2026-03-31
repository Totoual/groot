package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateNewWorkspaceOmitsProjectsDirAndInitializesManifest(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)

	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	wsPath := filepath.Join(root, "workspaces", "crawlly")

	for _, path := range []string{
		wsPath,
		filepath.Join(wsPath, "home"),
		filepath.Join(wsPath, "state"),
		filepath.Join(wsPath, "logs"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected path %s to exist: %v", path, err)
		}
	}

	if _, err := os.Stat(filepath.Join(wsPath, "projects")); !os.IsNotExist(err) {
		t.Fatalf("expected projects dir to be absent, stat err=%v", err)
	}

	manifest, err := app.getManifest(wsPath)
	if err != nil {
		t.Fatalf("getManifest returned error: %v", err)
	}
	if manifest.ProjectPath != "" {
		t.Fatalf("expected empty ProjectPath, got %q", manifest.ProjectPath)
	}
}

func TestBindWorkspaceStoresAbsoluteProjectPath(t *testing.T) {
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

	manifest, err := app.getManifest(filepath.Join(root, "workspaces", "crawlly"))
	if err != nil {
		t.Fatalf("getManifest returned error: %v", err)
	}
	if manifest.ProjectPath != projectPath {
		t.Fatalf("expected ProjectPath %q, got %q", projectPath, manifest.ProjectPath)
	}
}
