package toolchains

import (
	"fmt"
	"path/filepath"

	"github.com/totoual/groot/internal/helpers"
	"github.com/totoual/groot/internal/itoolchain"
)

type DenoInstaller struct{}

func (d DenoInstaller) Name() string {
	return "deno"
}

func (d DenoInstaller) archiveName(version, goos, goarch string) (string, error) {
	target, err := d.targetTriple(goos, goarch)
	if err != nil {
		return "", err
	}
	return "deno-" + target + ".zip", nil
}

func (d DenoInstaller) targetTriple(goos, goarch string) (string, error) {
	switch {
	case goos == "darwin" && goarch == "arm64":
		return "aarch64-apple-darwin", nil
	case goos == "darwin" && goarch == "amd64":
		return "x86_64-apple-darwin", nil
	case goos == "linux" && goarch == "arm64":
		return "aarch64-unknown-linux-gnu", nil
	case goos == "linux" && goarch == "amd64":
		return "x86_64-unknown-linux-gnu", nil
	default:
		return "", fmt.Errorf("deno: unsupported platform %s/%s", goos, goarch)
	}
}

func (d DenoInstaller) installDir(root, version string) string {
	return filepath.Join(root, "deno", version)
}

func (d DenoInstaller) binaryPath(root, version string) string {
	return filepath.Join(root, "deno", version, "deno")
}

func (d DenoInstaller) downloadURL(version, goos, goarch string) (string, error) {
	archiveName, err := d.archiveName(version, goos, goarch)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://github.com/denoland/deno/releases/download/v%s/%s", version, archiveName), nil
}

func (d DenoInstaller) checksumURL(version, goos, goarch string) (string, error) {
	archiveName, err := d.archiveName(version, goos, goarch)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://github.com/denoland/deno/releases/download/v%s/%s.sha256sum", version, archiveName), nil
}

func (d DenoInstaller) EnsureInstalled(ic *itoolchain.InstallContext, version string) error {
	archiveName, err := d.archiveName(version, ic.GOOS, ic.GOARCH)
	if err != nil {
		return err
	}
	archiveURL, err := d.downloadURL(version, ic.GOOS, ic.GOARCH)
	if err != nil {
		return err
	}
	checksumURL, err := d.checksumURL(version, ic.GOOS, ic.GOARCH)
	if err != nil {
		return err
	}

	return installZipArchiveIfNeeded(
		ic,
		d.binaryPath(ic.ToolchainDir, version),
		archiveURL,
		archiveName,
		d.installDir(ic.ToolchainDir, version),
		func(archivePath string) error {
			return helpers.VerifyDownloadedArchive(archivePath, archiveName, checksumURL)
		},
	)
}

func (d DenoInstaller) BinDir(ic *itoolchain.InstallContext, version string) (string, error) {
	return d.installDir(ic.ToolchainDir, version), nil
}

func (d DenoInstaller) Env(ic *itoolchain.InstallContext, version string) (map[string]string, error) {
	return nil, nil
}
