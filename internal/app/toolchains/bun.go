package toolchains

import (
	"fmt"
	"path/filepath"

	"github.com/totoual/groot/internal/helpers"
	"github.com/totoual/groot/internal/itoolchain"
)

type BunInstaller struct{}

func (b BunInstaller) Name() string {
	return "bun"
}

func (b BunInstaller) archiveName(version, goos, goarch string) (string, error) {
	platform, err := b.platform(goos, goarch)
	if err != nil {
		return "", err
	}
	return "bun-" + platform + ".zip", nil
}

func (b BunInstaller) platform(goos, goarch string) (string, error) {
	switch {
	case goos == "darwin" && goarch == "arm64":
		return "darwin-aarch64", nil
	case goos == "darwin" && goarch == "amd64":
		return "darwin-x64", nil
	case goos == "linux" && goarch == "arm64":
		return "linux-aarch64", nil
	case goos == "linux" && goarch == "amd64":
		return "linux-x64", nil
	default:
		return "", fmt.Errorf("bun: unsupported platform %s/%s", goos, goarch)
	}
}

func (b BunInstaller) installDir(root, version string) string {
	return filepath.Join(root, "bun", version)
}

func (b BunInstaller) archiveRoot(root, version, goos, goarch string) (string, error) {
	archiveName, err := b.archiveName(version, goos, goarch)
	if err != nil {
		return "", err
	}
	return filepath.Join(b.installDir(root, version), archiveName[:len(archiveName)-len(".zip")]), nil
}

func (b BunInstaller) binaryPath(root, version, goos, goarch string) (string, error) {
	archiveRoot, err := b.archiveRoot(root, version, goos, goarch)
	if err != nil {
		return "", err
	}
	return filepath.Join(archiveRoot, "bun"), nil
}

func (b BunInstaller) downloadURL(version, goos, goarch string) (string, error) {
	archiveName, err := b.archiveName(version, goos, goarch)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://github.com/oven-sh/bun/releases/download/bun-v%s/%s", version, archiveName), nil
}

func (b BunInstaller) checksumURL(version string) string {
	return fmt.Sprintf("https://github.com/oven-sh/bun/releases/download/bun-v%s/SHASUMS256.txt", version)
}

func (b BunInstaller) EnsureInstalled(ic *itoolchain.InstallContext, version string) error {
	archiveName, err := b.archiveName(version, ic.GOOS, ic.GOARCH)
	if err != nil {
		return err
	}
	archiveURL, err := b.downloadURL(version, ic.GOOS, ic.GOARCH)
	if err != nil {
		return err
	}
	binaryPath, err := b.binaryPath(ic.ToolchainDir, version, ic.GOOS, ic.GOARCH)
	if err != nil {
		return err
	}

	return installZipArchiveIfNeeded(
		ic,
		binaryPath,
		archiveURL,
		archiveName,
		b.installDir(ic.ToolchainDir, version),
		func(archivePath string) error {
			return helpers.VerifyDownloadedArchiveFromChecksumList(archivePath, archiveName, b.checksumURL(version))
		},
	)
}

func (b BunInstaller) BinDir(ic *itoolchain.InstallContext, version string) (string, error) {
	archiveRoot, err := b.archiveRoot(ic.ToolchainDir, version, ic.GOOS, ic.GOARCH)
	if err != nil {
		return "", err
	}
	return archiveRoot, nil
}

func (b BunInstaller) Env(ic *itoolchain.InstallContext, version string) (map[string]string, error) {
	return nil, nil
}
