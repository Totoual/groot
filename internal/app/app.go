package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type App struct {
	Root string
}

func NewApp(root string) *App {
	return &App{
		Root: root,
	}
}

func (a *App) Init() error {
	dirs := []string{
		a.Root,
		a.WorkspaceDir(),
		a.StoreDir(),
		a.BinDir(),
		a.ToolchainDir(),
		a.CacheDir(),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return fmt.Errorf("init mkdir %s: %w", d, err)
		}
	}
	return nil

}

func (a *App) WorkspaceDir() string {
	return filepath.Join(a.Root, "workspaces")

}

func (a *App) StoreDir() string {
	return filepath.Join(a.Root, "store")
}

func (a *App) BinDir() string {
	return filepath.Join(a.Root, "bin")
}

func (a *App) ToolchainDir() string {
	return filepath.Join(a.Root, "toolchains")
}

func (a *App) CacheDir() string {
	return filepath.Join(a.Root, "cache")
}

func (a *App) CreateNewWorkspace(name string) error {
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

	return nil
}

func (a *App) CreateManifest(name string) error {
	if err := a.Init(); err != nil {
		return err
	}
	wsPath := filepath.Join(a.WorkspaceDir(), name)
	if _, err := os.Stat(wsPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("workspace %q doesn't exist (run: groot ws create %s)", name, name)
		}
		return fmt.Errorf("stat workspace %s: %w", wsPath, err)
	}
	manifest := NewManifest(name)

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(wsPath, "manifest.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
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
