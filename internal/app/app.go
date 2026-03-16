package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/totoual/groot/internal/app/toolchains"
	"github.com/totoual/groot/internal/itoolchain"
)

type App struct {
	Root       string
	toolchains map[string]itoolchain.ToolchainInstaller
}

func NewApp(root string) *App {
	a := &App{
		Root:       root,
		toolchains: map[string]itoolchain.ToolchainInstaller{},
	}

	a.registerToolchain(toolchains.GoInstaller{})
	a.registerToolchain(toolchains.NodeInstaller{})
	a.registerToolchain(toolchains.DenoInstaller{})
	a.registerToolchain(toolchains.BunInstaller{})
	a.registerToolchain(toolchains.PHPInstaller{})
	a.registerToolchain(toolchains.JavaInstaller{})
	a.registerToolchain(toolchains.PythonInstaller{})
	a.registerToolchain(toolchains.RustInstaller{})
	return a
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

func (a *App) registerToolchain(t itoolchain.ToolchainInstaller) {
	a.toolchains[t.Name()] = t
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

func (a *App) setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i := range env {
		if strings.HasPrefix(env[i], prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func (a *App) ensureToolchainInstalled(tc Component) error {
	installer, ok := a.toolchains[tc.Name]
	if !ok {
		return fmt.Errorf("unsupported toolchain %q", tc.Name)
	}
	return installer.EnsureInstalled(a.installContext(), tc.Version)
}

func (a *App) toolchainBinDir(tc Component) (string, error) {
	installer, ok := a.toolchains[tc.Name]
	if !ok {
		return "", fmt.Errorf("unsupported toolchain %q", tc.Name)
	}

	return installer.BinDir(a.installContext(), tc.Version)
}

func (a *App) toolchainEnv(tc Component) (map[string]string, error) {
	installer, ok := a.toolchains[tc.Name]
	if !ok {
		return nil, fmt.Errorf("unsupported toolchain %q", tc.Name)
	}

	return installer.Env(a.installContext(), tc.Version)
}

func (a *App) installContext() *itoolchain.InstallContext {
	return &itoolchain.InstallContext{
		ToolchainDir: a.ToolchainDir(),
		CacheDir:     a.CacheDir(),
		GOOS:         runtime.GOOS,
		GOARCH:       runtime.GOARCH,
	}
}
