package workspacecmds

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/totoual/groot/internal/app"
)

func TestCreateCmdRunCreatesWorkspace(t *testing.T) {
	a := app.NewApp(t.TempDir())

	if err := (&CreateCmd{}).Run(a, []string{"crawlly"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(a.WorkspaceDir(), "crawlly")); err != nil {
		t.Fatalf("expected workspace to exist: %v", err)
	}
}

func TestBindCmdRunStoresProjectPath(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	if err := (&BindCmd{}).Run(a, []string{"crawlly", projectPath}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	manifest, err := loadManifest(filepath.Join(a.WorkspaceDir(), "crawlly"))
	if err != nil {
		t.Fatalf("loadManifest returned error: %v", err)
	}
	if manifest.ProjectPath != projectPath {
		t.Fatalf("ProjectPath = %q, want %q", manifest.ProjectPath, projectPath)
	}
}

func TestDeleteCmdRunDeletesWorkspace(t *testing.T) {
	a := app.NewApp(t.TempDir())
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	if err := (&DeleteCmd{}).Run(a, []string{"crawlly"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(a.WorkspaceDir(), "crawlly")); !os.IsNotExist(err) {
		t.Fatalf("expected workspace to be deleted, stat err=%v", err)
	}
}

func TestAttachCmdRunPersistsPackages(t *testing.T) {
	a := app.NewApp(t.TempDir())
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	if err := (&AttachCmd{}).Run(a, []string{"crawlly", "go@1.25.0"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	wsPath, err := a.EnsureWorkspace("crawlly")
	if err != nil {
		t.Fatalf("EnsureWorkspace returned error: %v", err)
	}
	manifest, err := loadManifest(wsPath)
	if err != nil {
		t.Fatalf("loadManifest returned error: %v", err)
	}
	if len(manifest.Packages) != 1 || manifest.Packages[0].Name != "go" {
		t.Fatalf("unexpected packages: %#v", manifest.Packages)
	}
}

func TestInstallCmdRunAcceptsEmptyWorkspace(t *testing.T) {
	a := app.NewApp(t.TempDir())
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	if err := (&InstallCmd{}).Run(a, []string{"crawlly"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestWorkspaceCmdsRequireExpectedArgs(t *testing.T) {
	a := app.NewApp(t.TempDir())

	tests := []struct {
		name string
		cmd  interface {
			Run(*app.App, []string) error
		}
		args []string
	}{
		{name: "create", cmd: &CreateCmd{}, args: nil},
		{name: "bind", cmd: &BindCmd{}, args: []string{"crawlly"}},
		{name: "delete", cmd: &DeleteCmd{}, args: nil},
		{name: "attach", cmd: &AttachCmd{}, args: []string{"crawlly"}},
		{name: "install", cmd: &InstallCmd{}, args: nil},
		{name: "shell", cmd: &ShellCmd{}, args: nil},
	}

	for _, tt := range tests {
		if err := tt.cmd.Run(a, tt.args); err == nil {
			t.Fatalf("%s: expected argument validation error", tt.name)
		}
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
