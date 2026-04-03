package workspacecmds

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/totoual/groot/internal/app"
)

func TestCreateCmdRunCreatesWorkspace(t *testing.T) {
	a := app.NewApp(t.TempDir())

	output, err := captureStdout(func() error {
		return (&CreateCmd{}).Run(a, []string{"crawlly"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(output) != "" {
		t.Fatalf("expected create wrapper to stay quiet, got %q", output)
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

	output, err := captureStdout(func() error {
		return (&BindCmd{}).Run(a, []string{"crawlly", projectPath})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(output) != "" {
		t.Fatalf("expected bind wrapper to stay quiet, got %q", output)
	}

	manifest, err := loadManifest(filepath.Join(a.WorkspaceDir(), "crawlly"))
	if err != nil {
		t.Fatalf("loadManifest returned error: %v", err)
	}
	if manifest.ProjectPath != projectPath {
		t.Fatalf("ProjectPath = %q, want %q", manifest.ProjectPath, projectPath)
	}
}

func TestUnbindCmdRunClearsProjectPath(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	output, err := captureStdout(func() error {
		return (&UnbindCmd{}).Run(a, []string{"crawlly"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(output) != "" {
		t.Fatalf("expected unbind wrapper to stay quiet, got %q", output)
	}

	manifest, err := loadManifest(filepath.Join(a.WorkspaceDir(), "crawlly"))
	if err != nil {
		t.Fatalf("loadManifest returned error: %v", err)
	}
	if manifest.ProjectPath != "" {
		t.Fatalf("ProjectPath = %q, want empty", manifest.ProjectPath)
	}
}

func TestDeleteCmdRunDeletesWorkspace(t *testing.T) {
	a := app.NewApp(t.TempDir())
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	output, err := captureStdout(func() error {
		return (&DeleteCmd{}).Run(a, []string{"crawlly"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(output) != "" {
		t.Fatalf("expected delete wrapper to stay quiet, got %q", output)
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

	output, err := captureStdout(func() error {
		return (&AttachCmd{}).Run(a, []string{"crawlly", "go@1.25.0"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(output) != "" {
		t.Fatalf("expected attach wrapper to stay quiet, got %q", output)
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

func TestAttachCmdRunRejectsMalformedSpec(t *testing.T) {
	a := app.NewApp(t.TempDir())
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	err := (&AttachCmd{}).Run(a, []string{"crawlly", "go"})
	if err == nil {
		t.Fatal("expected Run to fail for malformed attach spec")
	}
	if !strings.Contains(err.Error(), "invalid tool spec") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAttachCmdRunRejectsUnknownToolchain(t *testing.T) {
	a := app.NewApp(t.TempDir())
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	err := (&AttachCmd{}).Run(a, []string{"crawlly", "ruby@3.4.0"})
	if err == nil {
		t.Fatal("expected Run to fail for unknown toolchain")
	}
	if !strings.Contains(err.Error(), `unsupported toolchain "ruby"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallCmdRunAcceptsEmptyWorkspace(t *testing.T) {
	a := app.NewApp(t.TempDir())
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	output, err := captureStdout(func() error {
		return (&InstallCmd{}).Run(a, []string{"crawlly"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(output) != "" {
		t.Fatalf("expected install wrapper to stay quiet, got %q", output)
	}
}

func TestGCCmdRunRemovesUnreferencedToolchains(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.AttachToWorkspace("crawlly", []string{"go@1.25.0"}); err != nil {
		t.Fatalf("AttachToWorkspace returned error: %v", err)
	}

	keepDir := filepath.Join(root, "toolchains", "go", "1.25.0")
	removeDir := filepath.Join(root, "toolchains", "go", "1.26.0")
	for _, dir := range []string{keepDir, removeDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll returned error: %v", err)
		}
	}

	output, err := captureStdout(func() error {
		return (&GCCmd{}).Run(a, nil)
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(output) != "" {
		t.Fatalf("expected gc wrapper to stay quiet, got %q", output)
	}

	if _, err := os.Stat(keepDir); err != nil {
		t.Fatalf("expected referenced toolchain dir to remain: %v", err)
	}
	if _, err := os.Stat(removeDir); !os.IsNotExist(err) {
		t.Fatalf("expected unreferenced toolchain dir to be removed, stat err=%v", err)
	}
}

func TestOpenCmdRunOpensWorkspaceWithSoftRuntime(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	hostHome := filepath.Join(root, "host-home")
	t.Setenv("HOME", hostHome)
	t.Setenv("PATH", "/usr/bin:/bin")

	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	scriptPath := filepath.Join(root, "open-capture.sh")
	script := "#!/bin/sh\npwd > open-pwd.txt\nprintf '%s' \"$HOME\" > open-home.txt\nprintf '%s' \"$1\" > open-arg.txt\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	output, err := captureStdout(func() error {
		return (&OpenCmd{}).Run(a, []string{"crawlly", "--ide", scriptPath})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(output) != "" {
		t.Fatalf("expected open wrapper to stay quiet, got %q", output)
	}

	wantProjectPath, err := filepath.EvalSymlinks(projectPath)
	if err != nil {
		t.Fatalf("EvalSymlinks returned error: %v", err)
	}

	gotPwd, err := os.ReadFile(filepath.Join(projectPath, "open-pwd.txt"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.TrimSpace(string(gotPwd)) != wantProjectPath {
		t.Fatalf("pwd = %q, want %q", strings.TrimSpace(string(gotPwd)), wantProjectPath)
	}

	gotHome, err := os.ReadFile(filepath.Join(projectPath, "open-home.txt"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.TrimSpace(string(gotHome)) != hostHome {
		t.Fatalf("HOME = %q, want %q", strings.TrimSpace(string(gotHome)), hostHome)
	}

	gotArg, err := os.ReadFile(filepath.Join(projectPath, "open-arg.txt"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.TrimSpace(string(gotArg)) != projectPath {
		t.Fatalf("arg = %q, want %q", strings.TrimSpace(string(gotArg)), projectPath)
	}
}

func TestOpenCmdRunUsesConfiguredDefaultIDE(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	hostHome := filepath.Join(root, "host-home")
	t.Setenv("HOME", hostHome)
	t.Setenv("PATH", "/usr/bin:/bin")

	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	scriptPath := filepath.Join(root, "default-open.sh")
	script := "#!/bin/sh\nprintf '%s' \"$1\" > default-open-arg.txt\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	t.Setenv("GROOT_IDE", scriptPath)

	output, err := captureStdout(func() error {
		return (&OpenCmd{}).Run(a, []string{"crawlly"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(output) != "" {
		t.Fatalf("expected open wrapper to stay quiet, got %q", output)
	}

	gotArg, err := os.ReadFile(filepath.Join(projectPath, "default-open-arg.txt"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.TrimSpace(string(gotArg)) != projectPath {
		t.Fatalf("arg = %q, want %q", strings.TrimSpace(string(gotArg)), projectPath)
	}
}

func TestExecCmdRunExecutesCommandInWorkspace(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	projectPath := filepath.Join(root, "repos", "crawlly")
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
	output, err := captureStdout(func() error {
		return (&ExecCmd{}).Run(a, []string{"crawlly", scriptPath, outFile})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(output) != "" {
		t.Fatalf("expected exec wrapper to stay quiet, got %q", output)
	}

	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	wantProjectPath, err := filepath.EvalSymlinks(projectPath)
	if err != nil {
		t.Fatalf("EvalSymlinks returned error: %v", err)
	}
	if gotPath := strings.TrimSpace(string(got)); gotPath != wantProjectPath {
		t.Fatalf("pwd = %q, want %q", gotPath, wantProjectPath)
	}
}

func TestEnvCmdRunPrintsWorkspaceExports(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	output, err := captureStdout(func() error {
		return (&EnvCmd{}).Run(a, []string{"crawlly"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(output, "export GROOT_WORKSPACE='crawlly'") {
		t.Fatalf("unexpected output: %q", output)
	}
	if !strings.Contains(output, "export GROOT_WORKDIR=") {
		t.Fatalf("expected GROOT_WORKDIR export, got %q", output)
	}
	if strings.Contains(output, "export PS1=") || strings.Contains(output, "export PROMPT=") {
		t.Fatalf("expected prompt vars to be omitted, got %q", output)
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
		{name: "env", cmd: &EnvCmd{}, args: nil},
		{name: "exec", cmd: &ExecCmd{}, args: []string{"crawlly"}},
		{name: "gc", cmd: &GCCmd{}, args: []string{"extra"}},
		{name: "open", cmd: &OpenCmd{}, args: nil},
		{name: "attach", cmd: &AttachCmd{}, args: []string{"crawlly"}},
		{name: "install", cmd: &InstallCmd{}, args: nil},
		{name: "shell", cmd: &ShellCmd{}, args: nil},
		{name: "unbind", cmd: &UnbindCmd{}, args: nil},
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

func captureStdout(fn func() error) (string, error) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	defer r.Close()

	os.Stdout = w
	runErr := fn()
	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		return "", err
	}

	return buf.String(), runErr
}
