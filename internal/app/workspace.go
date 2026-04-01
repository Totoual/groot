package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (a *App) CreateNewWorkspace(name string) error {
	if name == "" || name == "." || name == ".." || strings.Contains(name, "/") {
		return fmt.Errorf("invalid workspace name %q", name)
	}
	if err := a.Init(); err != nil {
		return err
	}
	wsPath := filepath.Join(a.WorkspaceDir(), name)

	if _, err := os.Stat(wsPath); err == nil {
		return fmt.Errorf("workspace %q already exists", name)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("workspace stat %s: %w", wsPath, err)
	}

	for _, d := range []string{
		wsPath,
		filepath.Join(wsPath, "home"),
		filepath.Join(wsPath, "state"),
		filepath.Join(wsPath, "logs"),
	} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}

	err := a.CreateManifest(name)
	if err != nil {
		return err
	}
	return nil
}

func (a *App) DeleteWorkspace(name string) error {
	if name == "" || name == "." || name == ".." || strings.Contains(name, "/") {
		return fmt.Errorf("invalid workspace name %q", name)
	}

	wsPath := filepath.Join(a.WorkspaceDir(), name)

	if _, err := os.Stat(wsPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("workspace %q doesn't exist", name)
		}
		return fmt.Errorf("stat workspace %s: %w", wsPath, err)
	}

	if err := os.RemoveAll(wsPath); err != nil {
		return fmt.Errorf("remove workspace %s: %w", wsPath, err)
	}

	return nil
}

func (a *App) WorkspaceShell(name string) error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	args := []string{}
	base := filepath.Base(shell)
	if base == "bash" || base == "zsh" {
		args = append(args, "-i")
	}

	return a.ExecWorkspace(name, shell, args)
}

func (a *App) ExecWorkspace(name, command string, args []string) error {
	env, workDir, err := a.workspaceRuntime(name)
	if err != nil {
		return err
	}

	cmd := exec.Command(command, args...)
	cmd.Dir = workDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = env

	return cmd.Run()
}

func (a *App) workspaceRuntime(name string) ([]string, string, error) {
	wsPath, err := a.EnsureWorkspace(name)
	if err != nil {
		return nil, "", err
	}
	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return nil, "", err
	}

	toolchainBinDirs := make([]string, 0, len(manifest.Packages))
	toolchainEnv := make(map[string]string)
	seenBinDirs := make(map[string]struct{}, len(manifest.Packages))
	for _, pkg := range manifest.Packages {
		if err := a.ensureToolchainInstalled(pkg); err != nil {
			return nil, "", err
		}

		binDir, err := a.toolchainBinDir(pkg)
		if err != nil {
			return nil, "", err
		}
		if _, ok := seenBinDirs[binDir]; ok {
			continue
		}
		seenBinDirs[binDir] = struct{}{}
		toolchainBinDirs = append(toolchainBinDirs, binDir)

		extraEnv, err := a.toolchainEnv(pkg)
		if err != nil {
			return nil, "", err
		}
		for key, value := range extraEnv {
			toolchainEnv[key] = value
		}
	}

	wsHome := filepath.Join(wsPath, "home")
	workDir := wsPath
	if manifest.ProjectPath != "" {
		workDir = manifest.ProjectPath
	}

	env := os.Environ()
	env = a.setEnv(env, "HOME", wsHome)
	env = a.setEnv(env, "XDG_CONFIG_HOME", filepath.Join(wsHome, ".config"))
	env = a.setEnv(env, "XDG_CACHE_HOME", filepath.Join(wsHome, ".cache"))
	env = a.setEnv(env, "XDG_DATA_HOME", filepath.Join(wsHome, ".local", "share"))
	env = a.setEnv(env, "GROOT_WORKSPACE", name)
	env = a.setEnv(env, "GROOT_WORKSPACE_DIR", wsPath)
	for key, value := range toolchainEnv {
		env = a.setEnv(env, key, value)
	}
	if len(toolchainBinDirs) > 0 {
		pathParts := append([]string{}, toolchainBinDirs...)
		if currentPath := os.Getenv("PATH"); currentPath != "" {
			pathParts = append(pathParts, currentPath)
		}
		env = a.setEnv(env, "PATH", strings.Join(pathParts, string(os.PathListSeparator)))
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	base := filepath.Base(shell)
	if base == "zsh" {
		p := fmt.Sprintf("(groot:%s) %%n@%%m %%1~ %%# ", name)
		env = a.setEnv(env, "PROMPT", p)
		env = a.setEnv(env, "PS1", p)
	} else {
		env = a.setEnv(env, "PS1", fmt.Sprintf("(groot:%s) ", name)+"$PS1")
	}

	return env, workDir, nil
}

func (a *App) AttachToWorkspace(name string, args []string) error {
	wsPath, err := a.EnsureWorkspace(name)
	if err != nil {
		return err
	}
	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return err
	}
	components, err := a.parseComponents(args)
	if err != nil {
		return err
	}
	for _, comp := range components {
		updated := false
		for i := range manifest.Packages {
			if manifest.Packages[i].Name == comp.Name {
				manifest.Packages[i].Version = comp.Version
				updated = true
				break
			}
		}
		if !updated {
			manifest.Packages = append(manifest.Packages, comp)
		}
	}

	return a.writeManifest(wsPath, manifest)
}

func (a *App) BindWorkspace(name, projectPath string) error {
	wsPath, err := a.EnsureWorkspace(name)
	if err != nil {
		return err
	}

	if projectPath == "" {
		return fmt.Errorf("project path required")
	}

	if strings.HasPrefix(projectPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve home directory: %w", err)
		}
		projectPath = filepath.Join(home, strings.TrimPrefix(projectPath, "~"))
	}

	cleanPath := filepath.Clean(projectPath)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("resolve project path %q: %w", projectPath, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("project path %q does not exist", absPath)
		}
		return fmt.Errorf("stat project path %q: %w", absPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("project path %q is not a directory", absPath)
	}

	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return err
	}
	manifest.ProjectPath = absPath

	return a.writeManifest(wsPath, manifest)
}

func (a *App) InstallToWorkspace(name string) error {
	wsPath, err := a.EnsureWorkspace(name)
	if err != nil {
		return err
	}
	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return err
	}
	if len(manifest.Packages) == 0 {
		return nil
	}
	for _, pkg := range manifest.Packages {
		if err := a.ensureToolchainInstalled(pkg); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) EnsureWorkspace(name string) (string, error) {
	if name == "" || name == "." || name == ".." || strings.Contains(name, "/") {
		return "", fmt.Errorf("invalid workspace name %q", name)
	}
	if err := a.Init(); err != nil {
		return "", err
	}

	wsPath := filepath.Join(a.WorkspaceDir(), name)

	if _, err := os.Stat(wsPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("workspace %q doesn't exist (run: groot ws create %s)", name, name)
		}
		return "", fmt.Errorf("stat workspace %s: %w", wsPath, err)
	}

	return wsPath, nil
}
