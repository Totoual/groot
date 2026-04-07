package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/totoual/groot/internal/app"
)

func TestExportCmdRunPrintsWorkspaceExportJSON(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)

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
	if err := a.AttachToWorkspace("crawlly", []string{"go@1.25.4"}); err != nil {
		t.Fatalf("AttachToWorkspace returned error: %v", err)
	}
	backendDir := filepath.Join(projectPath, "backend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "go.mod"), []byte("module example.com/crawlly\n\ngo 1.25.4\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	seedInstalledGoToolchain(t, a, "1.25.4")

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&ExportCmd{}).Run(a, []string{projectPath})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}

	var exported struct {
		SchemaVersion int `json:"schema_version"`
		Workspace     struct {
			Name        string `json:"name"`
			ProjectPath string `json:"project_path"`
			Manifest    struct {
				Name string `json:"name"`
			} `json:"manifest"`
			Runtime struct {
				Status string `json:"status"`
			} `json:"runtime"`
		} `json:"workspace"`
	}
	if err := json.Unmarshal([]byte(stdout), &exported); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if exported.SchemaVersion != 1 {
		t.Fatalf("schema_version = %d, want %d", exported.SchemaVersion, 1)
	}
	if exported.Workspace.Name != "crawlly" {
		t.Fatalf("workspace.name = %q, want %q", exported.Workspace.Name, "crawlly")
	}
	if exported.Workspace.ProjectPath != projectPath {
		t.Fatalf("workspace.project_path = %q, want %q", exported.Workspace.ProjectPath, projectPath)
	}
	if exported.Workspace.Manifest.Name != "crawlly" {
		t.Fatalf("workspace.manifest.name = %q, want %q", exported.Workspace.Manifest.Name, "crawlly")
	}
	if exported.Workspace.Runtime.Status != "runtime owned by Groot" {
		t.Fatalf("workspace.runtime.status = %q, want %q", exported.Workspace.Runtime.Status, "runtime owned by Groot")
	}
}

func TestExportCmdRunWritesExportFile(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)

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

	outputPath := filepath.Join(root, "crawlly-export.json")
	stdout, stderr, err := captureCommandOutput(func() error {
		return (&ExportCmd{}).Run(a, []string{projectPath, "--output", outputPath})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected stdout to stay quiet, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(data), `"schema_version": 1`) {
		t.Fatalf("unexpected export file contents: %q", string(data))
	}
}

func TestExportCmdRunRejectsUnknownProjectPath(t *testing.T) {
	a := app.NewApp(t.TempDir())

	err := (&ExportCmd{}).Run(a, []string{"/tmp/does-not-exist"})
	if err == nil {
		t.Fatal("expected export to fail for unknown project path")
	}
}
