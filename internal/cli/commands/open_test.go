package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
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

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&OpenCmd{}).Run(a, []string{projectPath, "--ide", scriptPath})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected reused-open stdout to stay quiet, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected reused-open stderr to stay quiet, got %q", stderr)
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
	backendDir := filepath.Join(projectPath, "backend")
	frontendDir := filepath.Join(projectPath, "frontend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(frontendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "go.mod"), []byte("module example.com/tcg\n\ngo 1.25.4\n"), 0o600); err != nil {
		t.Fatalf("WriteFile go.mod returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "package.json"), []byte(`{"engines":{"node":"25.8.1"}}`), 0o600); err != nil {
		t.Fatalf("WriteFile package.json returned error: %v", err)
	}

	scriptPath := filepath.Join(root, "open-capture.sh")
	script := "#!/bin/sh\nprintf '%s' \"$GROOT_WORKSPACE\" > open-workspace.txt\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&OpenCmd{}).Run(a, []string{projectPath, "--ide", scriptPath})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected first-open stdout to stay quiet, got %q", stdout)
	}
	if !strings.Contains(stderr, `Created workspace "the_grime_tcg"`) {
		t.Fatalf("expected creation message on stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, `Detected likely runtimes for workspace "the_grime_tcg": go@1.25.4, node@25.8.1`) {
		t.Fatalf("expected detected runtimes message on stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, `First-open behavior is warn-only for now`) {
		t.Fatalf("expected warn-only message on stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, `Workspace "the_grime_tcg" does not declare detected runtimes: go@1.25.4, node@25.8.1`) {
		t.Fatalf("expected missing-runtime warning on stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, `Commands may fall back to host toolchains until these are attached and installed.`) {
		t.Fatalf("expected host fallback warning on stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, `groot ws attach the_grime_tcg go@1.25.4 node@25.8.1`) {
		t.Fatalf("expected attach suggestion on stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, `groot ws install the_grime_tcg`) {
		t.Fatalf("expected install suggestion on stderr, got %q", stderr)
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

func TestOpenCmdRunAttachDetectedAutoAttachesConcreteVersions(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	hostHome := filepath.Join(root, "host-home")
	t.Setenv("HOME", hostHome)
	t.Setenv("PATH", "/usr/bin:/bin")

	projectPath := filepath.Join(root, "repos", "the_grime_tcg")
	backendDir := filepath.Join(projectPath, "backend")
	frontendDir := filepath.Join(projectPath, "frontend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(frontendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "go.mod"), []byte("module example.com/tcg\n\ngo 1.25.4\n"), 0o600); err != nil {
		t.Fatalf("WriteFile go.mod returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "package.json"), []byte(`{"engines":{"node":"25.8.1"}}`), 0o600); err != nil {
		t.Fatalf("WriteFile package.json returned error: %v", err)
	}

	scriptPath := filepath.Join(root, "open-capture.sh")
	script := "#!/bin/sh\nprintf '%s' \"$GROOT_WORKSPACE\" > open-workspace.txt\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	seedInstalledGoToolchain(t, a, "1.25.4")
	seedInstalledNodeToolchain(t, a, "25.8.1")

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&OpenCmd{}).Run(a, []string{projectPath, "--attach-detected", "--ide", scriptPath})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected first-open stdout to stay quiet, got %q", stdout)
	}
	if !strings.Contains(stderr, `Auto-attached detected runtimes for workspace "the_grime_tcg": go@1.25.4, node@25.8.1`) {
		t.Fatalf("expected auto-attach message on stderr, got %q", stderr)
	}
	if strings.Contains(stderr, `Workspace "the_grime_tcg" does not declare detected runtimes`) {
		t.Fatalf("did not expect undeclared-runtime warning after auto-attach, got %q", stderr)
	}
	if strings.Contains(stderr, `First-open behavior is warn-only for now`) {
		t.Fatalf("did not expect warn-only message when --attach-detected is set, got %q", stderr)
	}

	manifest, err := loadManifest(filepath.Join(a.WorkspaceDir(), "the_grime_tcg"))
	if err != nil {
		t.Fatalf("loadManifest returned error: %v", err)
	}
	if len(manifest.Packages) != 2 {
		t.Fatalf("expected 2 attached packages, got %d", len(manifest.Packages))
	}
	if manifest.Packages[0] != (app.Component{Name: "go", Version: "1.25.4"}) {
		t.Fatalf("unexpected first package: %#v", manifest.Packages[0])
	}
	if manifest.Packages[1] != (app.Component{Name: "node", Version: "25.8.1"}) {
		t.Fatalf("unexpected second package: %#v", manifest.Packages[1])
	}
}

func TestOpenCmdRunAttachDetectedSkipsVersionlessRuntimes(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	hostHome := filepath.Join(root, "host-home")
	t.Setenv("HOME", hostHome)
	t.Setenv("PATH", "/usr/bin:/bin")

	projectPath := filepath.Join(root, "repos", "the_grime_tcg")
	backendDir := filepath.Join(projectPath, "backend")
	frontendDir := filepath.Join(projectPath, "frontend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(frontendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "go.mod"), []byte("module example.com/tcg\n\ngo 1.25.4\n"), 0o600); err != nil {
		t.Fatalf("WriteFile go.mod returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "package.json"), []byte(`{"name":"tcg-ui"}`), 0o600); err != nil {
		t.Fatalf("WriteFile package.json returned error: %v", err)
	}

	scriptPath := filepath.Join(root, "open-capture.sh")
	script := "#!/bin/sh\nprintf '%s' \"$GROOT_WORKSPACE\" > open-workspace.txt\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	seedInstalledGoToolchain(t, a, "1.25.4")

	_, stderr, err := captureCommandOutput(func() error {
		return (&OpenCmd{}).Run(a, []string{projectPath, "--attach-detected", "--ide", scriptPath})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stderr, `Auto-attached detected runtimes for workspace "the_grime_tcg": go@1.25.4`) {
		t.Fatalf("expected partial auto-attach message on stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, `Skipped detected runtimes without a concrete version for workspace "the_grime_tcg": node`) {
		t.Fatalf("expected skipped versionless message on stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, `Workspace "the_grime_tcg" does not declare detected runtimes: node`) {
		t.Fatalf("expected undeclared warning for remaining versionless runtime, got %q", stderr)
	}
}

func TestOpenCmdRunSetupAliasInstallsConcreteVersions(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	hostHome := filepath.Join(root, "host-home")
	t.Setenv("HOME", hostHome)
	t.Setenv("PATH", "/usr/bin:/bin")

	projectPath := filepath.Join(root, "repos", "the_grime_tcg")
	backendDir := filepath.Join(projectPath, "backend")
	frontendDir := filepath.Join(projectPath, "frontend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(frontendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "go.mod"), []byte("module example.com/tcg\n\ngo 1.25.4\n"), 0o600); err != nil {
		t.Fatalf("WriteFile go.mod returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "package.json"), []byte(`{"engines":{"node":"25.8.1"}}`), 0o600); err != nil {
		t.Fatalf("WriteFile package.json returned error: %v", err)
	}

	scriptPath := filepath.Join(root, "open-capture.sh")
	script := "#!/bin/sh\nprintf '%s' \"$GROOT_WORKSPACE\" > open-workspace.txt\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	seedInstalledGoToolchain(t, a, "1.25.4")
	seedInstalledNodeToolchain(t, a, "25.8.1")

	_, stderr, err := captureCommandOutput(func() error {
		return (&OpenCmd{}).Run(a, []string{projectPath, "--setup", "--ide", scriptPath})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stderr, `Auto-attached detected runtimes for workspace "the_grime_tcg": go@1.25.4, node@25.8.1`) {
		t.Fatalf("expected auto-attach message on stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, `Installed detected runtimes for workspace "the_grime_tcg": go@1.25.4, node@25.8.1`) {
		t.Fatalf("expected installed message on stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, `First-open summary: Groot attached and installed the detected runtimes`) {
		t.Fatalf("expected setup summary on stderr, got %q", stderr)
	}
	if strings.Contains(stderr, `Workspace "the_grime_tcg" does not declare detected runtimes`) {
		t.Fatalf("did not expect undeclared warning after setup, got %q", stderr)
	}
}

func TestOpenCmdRejectsMissingProjectPath(t *testing.T) {
	a := app.NewApp(t.TempDir())

	err := (&OpenCmd{}).Run(a, nil)
	if err == nil {
		t.Fatal("expected argument validation error")
	}
}

func TestOpenCmdWarnsWhenExistingWorkspaceStillReliesOnHostToolchains(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	hostHome := filepath.Join(root, "host-home")
	t.Setenv("HOME", hostHome)
	t.Setenv("PATH", "/usr/bin:/bin")

	if err := a.CreateNewWorkspace("tcg"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	projectPath := filepath.Join(root, "repos", "the_grime_tcg")
	backendDir := filepath.Join(projectPath, "backend")
	frontendDir := filepath.Join(projectPath, "frontend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(frontendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "go.mod"), []byte("module example.com/tcg\n\ngo 1.25.4\n"), 0o600); err != nil {
		t.Fatalf("WriteFile go.mod returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "package.json"), []byte(`{"engines":{"node":"25.8.1"}}`), 0o600); err != nil {
		t.Fatalf("WriteFile package.json returned error: %v", err)
	}
	if err := a.BindWorkspace("tcg", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}
	scriptPath := filepath.Join(root, "open-capture.sh")
	script := "#!/bin/sh\nprintf '%s' \"$GROOT_WORKSPACE\" > open-workspace.txt\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&OpenCmd{}).Run(a, []string{projectPath, "--ide", scriptPath})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected reused-open stdout to stay quiet, got %q", stdout)
	}
	if strings.Contains(stderr, `First-open behavior is warn-only for now`) {
		t.Fatalf("did not expect first-open warning for existing workspace, got %q", stderr)
	}
	if !strings.Contains(stderr, `Workspace "tcg" does not declare detected runtimes: go@1.25.4, node@25.8.1`) {
		t.Fatalf("expected missing-runtime warning for existing workspace, got %q", stderr)
	}
	if !strings.Contains(stderr, `groot ws attach tcg go@1.25.4 node@25.8.1`) {
		t.Fatalf("expected attach suggestion for missing runtimes, got %q", stderr)
	}
}

func TestStatusCmdPrintsRuntimeOwnershipSummary(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	t.Setenv("HOME", filepath.Join(root, "host-home"))
	t.Setenv("PATH", "/usr/bin:/bin")

	projectPath := filepath.Join(root, "repos", "the_grime_tcg")
	backendDir := filepath.Join(projectPath, "backend")
	frontendDir := filepath.Join(projectPath, "frontend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(frontendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "go.mod"), []byte("module example.com/tcg\n\ngo 1.25.4\n"), 0o600); err != nil {
		t.Fatalf("WriteFile go.mod returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "package.json"), []byte(`{"engines":{"node":"25.8.1"}}`), 0o600); err != nil {
		t.Fatalf("WriteFile package.json returned error: %v", err)
	}

	if err := a.CreateNewWorkspace("the_grime_tcg"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("the_grime_tcg", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}
	if err := a.AttachToWorkspace("the_grime_tcg", []string{"go@1.25.4"}); err != nil {
		t.Fatalf("AttachToWorkspace returned error: %v", err)
	}
	seedInstalledGoToolchain(t, a, "1.25.4")

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&StatusCmd{}).Run(a, []string{projectPath})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, "Workspace: the_grime_tcg") {
		t.Fatalf("expected workspace name in stdout, got %q", stdout)
	}
	if !strings.Contains(stdout, "Detected: go@1.25.4, node@25.8.1") {
		t.Fatalf("expected detected runtimes in stdout, got %q", stdout)
	}
	if !strings.Contains(stdout, "Groot-Managed: go@1.25.4") {
		t.Fatalf("expected installed Groot-managed runtime in stdout, got %q", stdout)
	}
	if !strings.Contains(stdout, "Host Fallback Risk: node@25.8.1") {
		t.Fatalf("expected host fallback risk in stdout, got %q", stdout)
	}
	if !strings.Contains(stdout, "Status: partial runtime ownership") {
		t.Fatalf("expected partial ownership status in stdout, got %q", stdout)
	}
}

func TestStatusCmdPrintsRuntimeOwnershipJSON(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	t.Setenv("HOME", filepath.Join(root, "host-home"))
	t.Setenv("PATH", "/usr/bin:/bin")

	projectPath := filepath.Join(root, "repos", "the_grime_tcg")
	backendDir := filepath.Join(projectPath, "backend")
	frontendDir := filepath.Join(projectPath, "frontend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(frontendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "go.mod"), []byte("module example.com/tcg\n\ngo 1.25.4\n"), 0o600); err != nil {
		t.Fatalf("WriteFile go.mod returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "package.json"), []byte(`{"engines":{"node":"25.8.1"}}`), 0o600); err != nil {
		t.Fatalf("WriteFile package.json returned error: %v", err)
	}

	if err := a.CreateNewWorkspace("the_grime_tcg"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("the_grime_tcg", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}
	if err := a.AttachToWorkspace("the_grime_tcg", []string{"go@1.25.4"}); err != nil {
		t.Fatalf("AttachToWorkspace returned error: %v", err)
	}
	seedInstalledGoToolchain(t, a, "1.25.4")

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&StatusCmd{}).Run(a, []string{"--json", projectPath})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}

	var output struct {
		WorkspaceName       string                  `json:"workspace_name"`
		ProjectPath         string                  `json:"project_path"`
		Status              string                  `json:"status"`
		Detected            []app.DetectedToolchain `json:"detected"`
		Attached            []app.Component         `json:"attached"`
		Installed           []app.Component         `json:"installed"`
		AttachedUninstalled []app.Component         `json:"attached_uninstalled"`
		Missing             []app.DetectedToolchain `json:"missing"`
	}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("Unmarshal returned error: %v\nstdout=%s", err, stdout)
	}
	if output.WorkspaceName != "the_grime_tcg" {
		t.Fatalf("WorkspaceName = %q, want %q", output.WorkspaceName, "the_grime_tcg")
	}
	if output.ProjectPath != projectPath {
		t.Fatalf("ProjectPath = %q, want %q", output.ProjectPath, projectPath)
	}
	if output.Status != "partial runtime ownership" {
		t.Fatalf("Status = %q, want %q", output.Status, "partial runtime ownership")
	}
	if len(output.Detected) != 2 {
		t.Fatalf("expected 2 detected runtimes, got %#v", output.Detected)
	}
	if len(output.Attached) != 1 || output.Attached[0] != (app.Component{Name: "go", Version: "1.25.4"}) {
		t.Fatalf("unexpected attached runtimes: %#v", output.Attached)
	}
	if len(output.Installed) != 1 || output.Installed[0] != (app.Component{Name: "go", Version: "1.25.4"}) {
		t.Fatalf("unexpected installed runtimes: %#v", output.Installed)
	}
	if len(output.AttachedUninstalled) != 0 {
		t.Fatalf("expected no attached-but-uninstalled runtimes, got %#v", output.AttachedUninstalled)
	}
	if len(output.Missing) != 1 || output.Missing[0].Name != "node" || output.Missing[0].Version != "25.8.1" {
		t.Fatalf("unexpected missing runtimes: %#v", output.Missing)
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

func seedInstalledGoToolchain(t *testing.T, a *app.App, version string) {
	t.Helper()

	binaryPath := filepath.Join(a.ToolchainDir(), "go", version, "go", "bin", "go")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}

func seedInstalledNodeToolchain(t *testing.T, a *app.App, version string) {
	t.Helper()

	platform := runtime.GOOS
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		arch = "x64"
	case "arm64":
		arch = "arm64"
	default:
		t.Fatalf("unsupported test architecture %q", runtime.GOARCH)
	}
	if platform != "darwin" && platform != "linux" {
		t.Fatalf("unsupported test platform %q", runtime.GOOS)
	}

	binaryPath := filepath.Join(a.ToolchainDir(), "node", version, "node-v"+version+"-"+platform+"-"+arch, "bin", "node")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}
