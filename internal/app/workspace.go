package app

import (
	"encoding/json"
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
		filepath.Join(wsPath, "projects"),
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
	wsPath, err := a.EnsureWorkspace(name)
	if err != nil {
		return err
	}

	wsHome := filepath.Join(wsPath, "home")

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	args := []string{}
	base := filepath.Base(shell)
	if base == "bash" || base == "zsh" {
		args = append(args, "-i")
	}

	cmd := exec.Command(shell, args...)
	cmd.Dir = filepath.Join(wsPath, "projects")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	env := os.Environ()
	env = a.setEnv(env, "HOME", wsHome)
	env = a.setEnv(env, "XDG_CONFIG_HOME", filepath.Join(wsHome, ".config"))
	env = a.setEnv(env, "XDG_CACHE_HOME", filepath.Join(wsHome, ".cache"))
	env = a.setEnv(env, "XDG_DATA_HOME", filepath.Join(wsHome, ".local", "share"))
	env = a.setEnv(env, "GROOT_WORKSPACE", name)
	env = a.setEnv(env, "GROOT_WORKSPACE_DIR", wsPath)

	if base == "zsh" {
		p := fmt.Sprintf("(groot:%s) %%n@%%m %%1~ %%# ", name)
		env = a.setEnv(env, "PROMPT", p)
		env = a.setEnv(env, "PS1", p)
	} else {
		env = a.setEnv(env, "PS1", fmt.Sprintf("(groot:%s) ", name)+"$PS1")
	}

	cmd.Env = env

	return cmd.Run()
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
	components := a.createComponents(args)
	manifest.Packages = append(manifest.Packages, components...)

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	path := getManifestPath(wsPath)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return nil
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
	fmt.Println(manifest)
	if err := a.ensureToolchainInstalled(manifest.Packages[0]); err != nil {
		fmt.Errorf("%v", err)
		return err
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
