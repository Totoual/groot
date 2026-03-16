package toolchains

import (
	"fmt"
	"path/filepath"
)

type GoInstaller struct{}

func (g GoInstaller) Name() string {
	return "go"
}

func (g GoInstaller) ArchiveName(version, goos, goarch string) string {
	return fmt.Sprintf("go%s.%s-%s.tar.gz", version, goos, goarch)
}

func (g GoInstaller) DownloadURL(version, goos, goarch string) (string, error) {
	name := g.ArchiveName(version, goos, goarch)
	return "https://go.dev/dl/" + name, nil
}

func (g GoInstaller) InstallDir(root, version string) string {
	return filepath.Join(root, "go", version)
}

func (g GoInstaller) BinaryPath(root, version string) string {
	return filepath.Join(root, "go", version, "go", "bin", "go")
}

func (g GoInstaller) ChecksumURL(version string) (string, error) {
	return fmt.Sprintf("https://go.dev/dl/go%s.sha256", version), nil
}
