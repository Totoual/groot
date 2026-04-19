package app

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestImportWorkspaceCreatesWorkspaceFromExport(t *testing.T) {
	root := t.TempDir()
	a := NewApp(root)
	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := osMkdirAll(projectPath); err != nil {
		t.Fatalf("osMkdirAll returned error: %v", err)
	}

	exported := WorkspaceExport{
		SchemaVersion: 1,
		Workspace: WorkspaceExportPayload{
			Name: "crawlly",
			Manifest: Manifest{
				SchemaVersion: 1,
				Name:          "crawlly",
				Packages:      []Component{{Name: "go", Version: "1.25.4"}},
				Services:      []ServiceSpec{{Name: "postgres", Version: "16"}},
				Env:           map[string]string{"APP_ENV": "dev"},
			},
		},
	}

	result, err := a.ImportWorkspace(exported, projectPath, false)
	if err != nil {
		t.Fatalf("ImportWorkspace returned error: %v", err)
	}
	if !result.Created {
		t.Fatal("expected import to create a workspace")
	}
	if result.WorkspaceName != "crawlly" {
		t.Fatalf("WorkspaceName = %q, want %q", result.WorkspaceName, "crawlly")
	}

	manifest, err := a.getManifest(filepath.Join(root, "workspaces", "crawlly"))
	if err != nil {
		t.Fatalf("getManifest returned error: %v", err)
	}
	if manifest.ProjectPath != projectPath {
		t.Fatalf("ProjectPath = %q, want %q", manifest.ProjectPath, projectPath)
	}
	if len(manifest.Packages) != 1 || manifest.Packages[0] != (Component{Name: "go", Version: "1.25.4"}) {
		t.Fatalf("unexpected packages: %#v", manifest.Packages)
	}
	if len(manifest.Services) != 1 || !reflect.DeepEqual(manifest.Services[0], ServiceSpec{Name: "postgres", Version: "16"}) {
		t.Fatalf("unexpected services: %#v", manifest.Services)
	}
	if manifest.Env["APP_ENV"] != "dev" {
		t.Fatalf("Env[APP_ENV] = %q, want %q", manifest.Env["APP_ENV"], "dev")
	}
}

func TestImportWorkspaceUpdatesExistingMatchingBinding(t *testing.T) {
	root := t.TempDir()
	a := NewApp(root)
	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := osMkdirAll(projectPath); err != nil {
		t.Fatalf("osMkdirAll returned error: %v", err)
	}
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	exported := WorkspaceExport{
		SchemaVersion: 1,
		Workspace: WorkspaceExportPayload{
			Name: "crawlly",
			Manifest: Manifest{
				SchemaVersion: 1,
				Name:          "crawlly",
				Packages:      []Component{{Name: "go", Version: "1.25.4"}},
				Env:           map[string]string{"APP_ENV": "dev"},
			},
		},
	}

	result, err := a.ImportWorkspace(exported, projectPath, false)
	if err != nil {
		t.Fatalf("ImportWorkspace returned error: %v", err)
	}
	if result.Created {
		t.Fatal("expected import to reuse existing workspace")
	}

	manifest, err := a.getManifest(filepath.Join(root, "workspaces", "crawlly"))
	if err != nil {
		t.Fatalf("getManifest returned error: %v", err)
	}
	if len(manifest.Packages) != 1 || manifest.Packages[0] != (Component{Name: "go", Version: "1.25.4"}) {
		t.Fatalf("unexpected packages: %#v", manifest.Packages)
	}
}

func TestImportWorkspaceRejectsConflictingWorkspaceBinding(t *testing.T) {
	root := t.TempDir()
	a := NewApp(root)
	existingProject := filepath.Join(root, "repos", "existing")
	importProject := filepath.Join(root, "repos", "imported")
	if err := osMkdirAll(existingProject); err != nil {
		t.Fatalf("osMkdirAll returned error: %v", err)
	}
	if err := osMkdirAll(importProject); err != nil {
		t.Fatalf("osMkdirAll returned error: %v", err)
	}
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", existingProject); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	exported := WorkspaceExport{
		SchemaVersion: 1,
		Workspace: WorkspaceExportPayload{
			Name: "crawlly",
			Manifest: Manifest{
				SchemaVersion: 1,
				Name:          "crawlly",
			},
		},
	}

	_, err := a.ImportWorkspace(exported, importProject, false)
	if err == nil {
		t.Fatal("expected import to fail for conflicting workspace binding")
	}
}

func TestImportWorkspaceAsAllowsWorkspaceNameOverrideOnCollision(t *testing.T) {
	root := t.TempDir()
	a := NewApp(root)
	existingProject := filepath.Join(root, "repos", "existing")
	importProject := filepath.Join(root, "repos", "imported")
	if err := osMkdirAll(existingProject); err != nil {
		t.Fatalf("osMkdirAll returned error: %v", err)
	}
	if err := osMkdirAll(importProject); err != nil {
		t.Fatalf("osMkdirAll returned error: %v", err)
	}
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", existingProject); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	exported := WorkspaceExport{
		SchemaVersion: 1,
		Workspace: WorkspaceExportPayload{
			Name: "crawlly",
			Manifest: Manifest{
				SchemaVersion: 1,
				Name:          "crawlly",
				Packages:      []Component{{Name: "go", Version: "1.25.4"}},
			},
		},
	}

	result, err := a.ImportWorkspaceAs(exported, importProject, "crawlly-imported", false)
	if err != nil {
		t.Fatalf("ImportWorkspaceAs returned error: %v", err)
	}
	if result.WorkspaceName != "crawlly-imported" {
		t.Fatalf("WorkspaceName = %q, want %q", result.WorkspaceName, "crawlly-imported")
	}
	manifest, err := a.getManifest(filepath.Join(root, "workspaces", "crawlly-imported"))
	if err != nil {
		t.Fatalf("getManifest returned error: %v", err)
	}
	if manifest.Name != "crawlly-imported" {
		t.Fatalf("manifest.Name = %q, want %q", manifest.Name, "crawlly-imported")
	}
	if manifest.ProjectPath != importProject {
		t.Fatalf("manifest.ProjectPath = %q, want %q", manifest.ProjectPath, importProject)
	}
}

func osMkdirAll(path string) error {
	return os.MkdirAll(path, 0o755)
}
