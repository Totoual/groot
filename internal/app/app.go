package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/totoual/groot/internal/app/toolchains"
	"github.com/totoual/groot/internal/helpers"
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

	binaryPath := installer.BinaryPath(a.ToolchainDir(), tc.Version)

	// already installed?
	if _, err := os.Stat(binaryPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat toolchain binary: %w", err)
	}

	goos := runtime.GOOS
	goarch := runtime.GOARCH

	url, err := installer.DownloadURL(tc.Version, goos, goarch)
	if err != nil {
		return err
	}

	archiveName := installer.ArchiveName(tc.Version, goos, goarch)
	archivePath := filepath.Join(a.CacheDir(), archiveName)

	installDir := installer.InstallDir(a.ToolchainDir(), tc.Version)

	// 1️⃣ Download archive if missing
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		fmt.Println("Downloading", url)

		if err := helpers.DownloadFile(url, archivePath); err != nil {
			return err
		}
	}

	// 2️⃣ Verify checksum
	checksumURL, err := installer.ChecksumURL(tc.Version)
	if err != nil {
		return err
	}

	fmt.Println("Verifying checksum")

	if err := helpers.VerifyDownloadedArchive(
		archivePath,
		archiveName,
		checksumURL,
	); err != nil {
		return fmt.Errorf("checksum verification failed: %w", err)
	}

	// 3️⃣ Extract archive
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return err
	}

	fmt.Println("Extracting", archivePath)

	if err := helpers.ExtractTarGz(archivePath, installDir); err != nil {
		return err
	}

	// 4️⃣ Sanity check binary exists
	if _, err := os.Stat(binaryPath); err != nil {
		return fmt.Errorf("toolchain installed but binary missing: %s", binaryPath)
	}

	return nil
}
