package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/totoual/groot/internal/app"
)

func TestImportCmdRunImportsWorkspaceFromFile(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	exported := app.WorkspaceExport{
		SchemaVersion: 1,
		Workspace: app.WorkspaceExportPayload{
			Name: "crawlly",
			Manifest: app.Manifest{
				SchemaVersion: 1,
				Name:          "crawlly",
				Packages:      []app.Component{{Name: "go", Version: "1.25.4"}},
			},
		},
	}
	data, err := json.Marshal(exported)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	exportPath := filepath.Join(root, "crawlly-export.json")
	if err := os.WriteFile(exportPath, data, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&ImportCmd{}).Run(a, []string{exportPath, "--project-path", projectPath})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout to stay quiet, got %q", stdout)
	}
	if !strings.Contains(stderr, `Imported workspace "crawlly"`) {
		t.Fatalf("expected import message on stderr, got %q", stderr)
	}

	manifest, err := loadManifest(filepath.Join(a.WorkspaceDir(), "crawlly"))
	if err != nil {
		t.Fatalf("loadManifest returned error: %v", err)
	}
	if manifest.ProjectPath != projectPath {
		t.Fatalf("ProjectPath = %q, want %q", manifest.ProjectPath, projectPath)
	}
}

func TestImportCmdRunReadsExportFromStdin(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	exported := app.WorkspaceExport{
		SchemaVersion: 1,
		Workspace: app.WorkspaceExportPayload{
			Name: "crawlly",
			Manifest: app.Manifest{
				SchemaVersion: 1,
				Name:          "crawlly",
			},
		},
	}
	data, err := json.Marshal(exported)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	oldStdin := os.Stdin
	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe returned error: %v", err)
	}
	if _, err := stdinW.Write(data); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	_ = stdinW.Close()
	os.Stdin = stdinR
	defer func() {
		os.Stdin = oldStdin
		_ = stdinR.Close()
	}()

	_, stderr, err := captureCommandOutput(func() error {
		return (&ImportCmd{}).Run(a, []string{"-", "--project-path", projectPath})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stderr, `Imported workspace "crawlly"`) {
		t.Fatalf("expected import message on stderr, got %q", stderr)
	}
}

func TestImportCmdRunSupportsWorkspaceNameOverride(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	existingProject := filepath.Join(root, "repos", "existing")
	importProject := filepath.Join(root, "repos", "imported")
	for _, projectPath := range []string{existingProject, importProject} {
		if err := os.MkdirAll(projectPath, 0o755); err != nil {
			t.Fatalf("MkdirAll returned error: %v", err)
		}
	}
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", existingProject); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	exported := app.WorkspaceExport{
		SchemaVersion: 1,
		Workspace: app.WorkspaceExportPayload{
			Name: "crawlly",
			Manifest: app.Manifest{
				SchemaVersion: 1,
				Name:          "crawlly",
			},
		},
	}
	data, err := json.Marshal(exported)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	exportPath := filepath.Join(root, "crawlly-export.json")
	if err := os.WriteFile(exportPath, data, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, stderr, err := captureCommandOutput(func() error {
		return (&ImportCmd{}).Run(a, []string{exportPath, "--project-path", importProject, "--workspace-name", "crawlly-imported"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stderr, `Imported workspace "crawlly-imported"`) {
		t.Fatalf("expected renamed import message on stderr, got %q", stderr)
	}
}
