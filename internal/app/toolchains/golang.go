package toolchains

import (
	"fmt"
	"path/filepath"

	"github.com/totoual/groot/internal/helpers"
	"github.com/totoual/groot/internal/itoolchain"
)

type GoInstaller struct{}

func (g GoInstaller) Name() string {
	return "go"
}

func (g GoInstaller) archiveName(version, goos, goarch string) string {
	return fmt.Sprintf("go%s.%s-%s.tar.gz", version, goos, goarch)
}

func (g GoInstaller) downloadURL(version, goos, goarch string) string {
	return "https://go.dev/dl/" + g.archiveName(version, goos, goarch)
}

func (g GoInstaller) checksumURL(version, goos, goarch string) string {
	return "https://dl.google.com/go/" + g.archiveName(version, goos, goarch) + ".sha256"
}

func (g GoInstaller) installDir(root, version string) string {
	return filepath.Join(root, "go", version)
}

func (g GoInstaller) binaryPath(root, version string) string {
	return filepath.Join(root, "go", version, "go", "bin", "go")
}

func (g GoInstaller) EnsureInstalled(ic *itoolchain.InstallContext, version string) error {
	archiveName := g.archiveName(version, ic.GOOS, ic.GOARCH)
	return installArchiveIfNeeded(
		ic,
		g.binaryPath(ic.ToolchainDir, version),
		g.downloadURL(version, ic.GOOS, ic.GOARCH),
		archiveName,
		g.installDir(ic.ToolchainDir, version),
		func(archivePath string) error {
			return helpers.VerifyDownloadedArchive(
				archivePath,
				archiveName,
				g.checksumURL(version, ic.GOOS, ic.GOARCH),
			)
		},
	)
}

func (g GoInstaller) BinDir(ic *itoolchain.InstallContext, version string) (string, error) {
	return filepath.Dir(g.binaryPath(ic.ToolchainDir, version)), nil
}

func (g GoInstaller) Env(ic *itoolchain.InstallContext, version string) (map[string]string, error) {
	return nil, nil
}
