package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/totoual/groot/internal/itoolchain"
)

type stubInstaller struct {
	name        string
	binDir      string
	env         map[string]string
	ensureCalls []string
}

func (s *stubInstaller) Name() string { return s.name }

func (s *stubInstaller) EnsureInstalled(_ *itoolchain.InstallContext, version string) error {
	s.ensureCalls = append(s.ensureCalls, version)
	return nil
}

func (s *stubInstaller) BinDir(_ *itoolchain.InstallContext, _ string) (string, error) {
	return s.binDir, nil
}

func (s *stubInstaller) Env(_ *itoolchain.InstallContext, _ string) (map[string]string, error) {
	return s.env, nil
}

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

func TestWriteManifestRoundTrip(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)

	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	wsPath := filepath.Join(root, "workspaces", "crawlly")
	want := Manifest{
		SchemaVersion: 1,
		Name:          "crawlly",
		ProjectPath:   filepath.Join(root, "repos", "crawlly"),
		Packages: []Component{
			{Name: "go", Version: "1.25.0"},
		},
		Services: []Component{
			{Name: "redis", Version: "7"},
		},
		Env: map[string]string{
			"APP_ENV": "dev",
		},
	}

	if err := app.writeManifest(wsPath, want); err != nil {
		t.Fatalf("writeManifest returned error: %v", err)
	}

	got, err := app.getManifest(wsPath)
	if err != nil {
		t.Fatalf("getManifest returned error: %v", err)
	}

	if got.SchemaVersion != want.SchemaVersion || got.Name != want.Name || got.ProjectPath != want.ProjectPath {
		t.Fatalf("unexpected manifest round-trip: %#v", got)
	}
	if len(got.Packages) != 1 || got.Packages[0] != want.Packages[0] {
		t.Fatalf("unexpected packages: %#v", got.Packages)
	}
	if len(got.Services) != 1 || got.Services[0] != want.Services[0] {
		t.Fatalf("unexpected services: %#v", got.Services)
	}
	if got.Env["APP_ENV"] != "dev" {
		t.Fatalf("unexpected env map: %#v", got.Env)
	}
}

func TestWorkspaceRuntimeUsesWorkspaceRootWhenUnbound(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	t.Setenv("PATH", "/usr/bin")
	t.Setenv("SHELL", "/bin/bash")

	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	env, workDir, err := app.workspaceRuntime("crawlly")
	if err != nil {
		t.Fatalf("workspaceRuntime returned error: %v", err)
	}

	wsPath := filepath.Join(root, "workspaces", "crawlly")
	if workDir != wsPath {
		t.Fatalf("workDir = %q, want %q", workDir, wsPath)
	}

	envMap := envSliceToMap(env)
	if envMap["HOME"] != filepath.Join(wsPath, "home") {
		t.Fatalf("HOME = %q", envMap["HOME"])
	}
	if envMap["XDG_CONFIG_HOME"] != filepath.Join(wsPath, "home", ".config") {
		t.Fatalf("XDG_CONFIG_HOME = %q", envMap["XDG_CONFIG_HOME"])
	}
	if envMap["GROOT_WORKSPACE"] != "crawlly" {
		t.Fatalf("GROOT_WORKSPACE = %q", envMap["GROOT_WORKSPACE"])
	}
	if envMap["GROOT_WORKSPACE_DIR"] != wsPath {
		t.Fatalf("GROOT_WORKSPACE_DIR = %q", envMap["GROOT_WORKSPACE_DIR"])
	}
	if envMap["PS1"] != "(groot:crawlly) $PS1" {
		t.Fatalf("PS1 = %q", envMap["PS1"])
	}
}

func TestWorkspaceRuntimeUsesBoundProjectPathAndInjectsToolchainEnv(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	t.Setenv("PATH", "/usr/bin:/bin")
	t.Setenv("SHELL", "/bin/zsh")

	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := app.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	stub := &stubInstaller{
		name:   "stub",
		binDir: "/toolchains/stub/1.0/bin",
		env: map[string]string{
			"STUB_HOME": "/toolchains/stub/1.0",
		},
	}
	app.toolchains = map[string]itoolchain.ToolchainInstaller{
		"stub": stub,
	}

	if err := app.AttachToWorkspace("crawlly", []string{"stub@1.0", "stub@1.0"}); err != nil {
		t.Fatalf("AttachToWorkspace returned error: %v", err)
	}

	env, workDir, err := app.workspaceRuntime("crawlly")
	if err != nil {
		t.Fatalf("workspaceRuntime returned error: %v", err)
	}

	if workDir != projectPath {
		t.Fatalf("workDir = %q, want %q", workDir, projectPath)
	}

	if len(stub.ensureCalls) != 2 {
		t.Fatalf("expected 2 ensure calls, got %d", len(stub.ensureCalls))
	}

	envMap := envSliceToMap(env)
	if envMap["STUB_HOME"] != "/toolchains/stub/1.0" {
		t.Fatalf("STUB_HOME = %q", envMap["STUB_HOME"])
	}
	if !strings.HasPrefix(envMap["PATH"], "/toolchains/stub/1.0/bin:") {
		t.Fatalf("PATH = %q", envMap["PATH"])
	}
	if strings.Count(envMap["PATH"], "/toolchains/stub/1.0/bin") != 1 {
		t.Fatalf("expected deduped PATH entry, got %q", envMap["PATH"])
	}
	expectedPrompt := "(groot:crawlly) %n@%m %1~ %# "
	if envMap["PROMPT"] != expectedPrompt || envMap["PS1"] != expectedPrompt {
		t.Fatalf("unexpected zsh prompt values: PROMPT=%q PS1=%q", envMap["PROMPT"], envMap["PS1"])
	}
}

func TestInstallToWorkspaceEnsuresAttachedToolchains(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)

	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	stub := &stubInstaller{name: "stub", binDir: "/toolchains/stub/1.0/bin"}
	app.toolchains = map[string]itoolchain.ToolchainInstaller{
		"stub": stub,
	}

	if err := app.AttachToWorkspace("crawlly", []string{"stub@1.0", "stub@2.0"}); err != nil {
		t.Fatalf("AttachToWorkspace returned error: %v", err)
	}

	if err := app.InstallToWorkspace("crawlly"); err != nil {
		t.Fatalf("InstallToWorkspace returned error: %v", err)
	}

	if len(stub.ensureCalls) != 2 {
		t.Fatalf("expected 2 ensure calls, got %d", len(stub.ensureCalls))
	}
	if stub.ensureCalls[0] != "1.0" || stub.ensureCalls[1] != "2.0" {
		t.Fatalf("unexpected ensure calls: %#v", stub.ensureCalls)
	}
}

func TestExecWorkspaceRunsCommandInWorkspaceRuntime(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	t.Setenv("SHELL", "/bin/sh")
	t.Setenv("PATH", "/usr/bin:/bin")

	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := app.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	scriptPath := filepath.Join(root, "capture.sh")
	script := "#!/bin/sh\npwd > \"$1\"\nprintf '%s' \"$GROOT_WORKSPACE\" > \"$2\"\nprintf '%s' \"$HOME\" > \"$3\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	pwdFile := filepath.Join(root, "pwd.txt")
	wsFile := filepath.Join(root, "workspace.txt")
	homeFile := filepath.Join(root, "home.txt")

	if err := app.ExecWorkspace("crawlly", scriptPath, []string{pwdFile, wsFile, homeFile}); err != nil {
		t.Fatalf("ExecWorkspace returned error: %v", err)
	}

	gotPwd, err := os.ReadFile(pwdFile)
	if err != nil {
		t.Fatalf("ReadFile pwd returned error: %v", err)
	}
	wantProjectPath, err := filepath.EvalSymlinks(projectPath)
	if err != nil {
		t.Fatalf("EvalSymlinks returned error: %v", err)
	}
	if strings.TrimSpace(string(gotPwd)) != wantProjectPath {
		t.Fatalf("pwd = %q, want %q", strings.TrimSpace(string(gotPwd)), wantProjectPath)
	}

	gotWorkspace, err := os.ReadFile(wsFile)
	if err != nil {
		t.Fatalf("ReadFile workspace returned error: %v", err)
	}
	if string(gotWorkspace) != "crawlly" {
		t.Fatalf("GROOT_WORKSPACE = %q", string(gotWorkspace))
	}

	gotHome, err := os.ReadFile(homeFile)
	if err != nil {
		t.Fatalf("ReadFile home returned error: %v", err)
	}
	wantHome := filepath.Join(root, "workspaces", "crawlly", "home")
	if string(gotHome) != wantHome {
		t.Fatalf("HOME = %q, want %q", string(gotHome), wantHome)
	}
}

func envSliceToMap(env []string) map[string]string {
	result := make(map[string]string, len(env))
	for _, entry := range env {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		result[parts[0]] = parts[1]
	}
	return result
}
