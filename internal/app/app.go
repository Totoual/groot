package app

import (
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

func (a *App) WorkspaceDir() string {
	return filepath.Join(a.Root, "workspaces")

}

func (a *App) StoreDir() string {
	return filepath.Join(a.Root, "store")
}
