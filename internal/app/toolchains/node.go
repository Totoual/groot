package toolchains

import (
	"fmt"
	"path/filepath"

	"github.com/totoual/groot/internal/helpers"
	"github.com/totoual/groot/internal/itoolchain"
)

type NodeInstaller struct{}

func (n NodeInstaller) Name() string {
	return "node"
}

func (n NodeInstaller) platform(goos string) (string, error) {
	switch goos {
	case "darwin":
		return "darwin", nil
	case "linux":
		return "linux", nil
	default:
		return "", fmt.Errorf("node: unsupported OS %q", goos)
	}
}

func (n NodeInstaller) arch(goarch string) (string, error) {
	switch goarch {
	case "amd64":
		return "x64", nil
	case "arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf("node: unsupported architecture %q", goarch)
	}
}

func (n NodeInstaller) archiveName(version, goos, goarch string) (string, error) {
	platform, err := n.platform(goos)
	if err != nil {
		return "", err
	}
	arch, err := n.arch(goarch)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("node-v%s-%s-%s.tar.gz", version, platform, arch), nil
}

func (n NodeInstaller) installDir(root, version string) string {
	return filepath.Join(root, "node", version)
}

func (n NodeInstaller) archiveRoot(root, version, goos, goarch string) (string, error) {
	archiveName, err := n.archiveName(version, goos, goarch)
	if err != nil {
		return "", err
	}
	return filepath.Join(n.installDir(root, version), archiveName[:len(archiveName)-len(".tar.gz")]), nil
}

func (n NodeInstaller) EnsureInstalled(ic *itoolchain.InstallContext, version string) error {
	archiveName, err := n.archiveName(version, ic.GOOS, ic.GOARCH)
	if err != nil {
		return err
	}
	archiveRoot, err := n.archiveRoot(ic.ToolchainDir, version, ic.GOOS, ic.GOARCH)
	if err != nil {
		return err
	}
	archiveURL := fmt.Sprintf("https://nodejs.org/dist/v%s/%s", version, archiveName)
	checksumURL := fmt.Sprintf("https://nodejs.org/dist/v%s/SHASUMS256.txt", version)
	return installArchiveIfNeeded(
		ic,
		filepath.Join(archiveRoot, "bin", "node"),
		archiveURL,
		archiveName,
		n.installDir(ic.ToolchainDir, version),
		func(archivePath string) error {
			return helpers.VerifyDownloadedArchiveFromChecksumList(archivePath, archiveName, checksumURL)
		},
	)
}

func (n NodeInstaller) BinDir(ic *itoolchain.InstallContext, version string) (string, error) {
	archiveRoot, err := n.archiveRoot(ic.ToolchainDir, version, ic.GOOS, ic.GOARCH)
	if err != nil {
		return "", err
	}
	return filepath.Join(archiveRoot, "bin"), nil
}

func (n NodeInstaller) Env(ic *itoolchain.InstallContext, version string) (map[string]string, error) {
	return nil, nil
}
