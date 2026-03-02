package app

import (
	"fmt"
	"os"
	"path/filepath"
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
