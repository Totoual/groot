package app

import (
	"os"
	"path/filepath"
	"strings"
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
	if manifest.Name != "crawlly" {
		t.Fatalf("expected manifest name %q, got %q", "crawlly", manifest.Name)
	}
	if manifest.SchemaVersion != 1 {
		t.Fatalf("expected schema version 1, got %d", manifest.SchemaVersion)
	}
	if len(manifest.Packages) != 0 {
		t.Fatalf("expected no packages, got %d", len(manifest.Packages))
	}
	if len(manifest.Services) != 0 {
		t.Fatalf("expected no services, got %d", len(manifest.Services))
	}
	if manifest.Env == nil {
		t.Fatal("expected manifest env map to be initialized")
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

func TestBindWorkspaceExpandsTildePath(t *testing.T) {
	root := t.TempDir()
	homeDir := filepath.Join(root, "home")
	projectPath := filepath.Join(homeDir, "dev", "crawlly")

	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	t.Setenv("HOME", homeDir)

	app := NewApp(root)
	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	if err := app.BindWorkspace("crawlly", "~/dev/crawlly"); err != nil {
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

func TestBindWorkspaceRejectsMissingPath(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)

	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	err := app.BindWorkspace("crawlly", filepath.Join(root, "missing"))
	if err == nil {
		t.Fatal("expected BindWorkspace to fail for missing path")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected missing path error, got %v", err)
	}
}

func TestBindWorkspaceRejectsFilePath(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)

	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	filePath := filepath.Join(root, "repo.txt")
	if err := os.WriteFile(filePath, []byte("not a dir"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	err := app.BindWorkspace("crawlly", filePath)
	if err == nil {
		t.Fatal("expected BindWorkspace to fail for file path")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("expected not-a-directory error, got %v", err)
	}
}

func TestAttachToWorkspacePersistsPackages(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)

	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	if err := app.AttachToWorkspace("crawlly", []string{"go@1.25.0", "node@25.0.0"}); err != nil {
		t.Fatalf("AttachToWorkspace returned error: %v", err)
	}

	manifest, err := app.getManifest(filepath.Join(root, "workspaces", "crawlly"))
	if err != nil {
		t.Fatalf("getManifest returned error: %v", err)
	}

	if len(manifest.Packages) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(manifest.Packages))
	}
	if manifest.Packages[0] != (Component{Name: "go", Version: "1.25.0"}) {
		t.Fatalf("unexpected first package: %#v", manifest.Packages[0])
	}
	if manifest.Packages[1] != (Component{Name: "node", Version: "25.0.0"}) {
		t.Fatalf("unexpected second package: %#v", manifest.Packages[1])
	}
}

func TestDeleteWorkspaceRemovesWorkspaceDirectory(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)

	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	if err := app.DeleteWorkspace("crawlly"); err != nil {
		t.Fatalf("DeleteWorkspace returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "workspaces", "crawlly")); !os.IsNotExist(err) {
		t.Fatalf("expected workspace directory to be removed, stat err=%v", err)
	}
}

func TestDeleteWorkspaceRejectsMissingWorkspace(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)

	err := app.DeleteWorkspace("crawlly")
	if err == nil {
		t.Fatal("expected DeleteWorkspace to fail for missing workspace")
	}
	if !strings.Contains(err.Error(), "doesn't exist") {
		t.Fatalf("expected missing workspace error, got %v", err)
	}
}
