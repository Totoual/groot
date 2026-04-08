package toolchains

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/totoual/groot/internal/helpers"
	"github.com/totoual/groot/internal/itoolchain"
)

type JavaInstaller struct{}

type adoptiumAssetResponse struct {
	Binary struct {
		Package struct {
			Checksum string `json:"checksum"`
			Link     string `json:"link"`
			Name     string `json:"name"`
		} `json:"package"`
	} `json:"binary"`
}

func (j JavaInstaller) Name() string {
	return "java"
}

func (j JavaInstaller) apiOS(goos string) (string, error) {
	switch goos {
	case "darwin":
		return "mac", nil
	case "linux":
		return "linux", nil
	default:
		return "", fmt.Errorf("java: unsupported OS %q", goos)
	}
}

func (j JavaInstaller) apiArch(goarch string) (string, error) {
	switch goarch {
	case "amd64":
		return "x64", nil
	case "arm64":
		return "aarch64", nil
	default:
		return "", fmt.Errorf("java: unsupported architecture %q", goarch)
	}
}

func (j JavaInstaller) featureVersion(version string) string {
	if idx := strings.Index(version, "."); idx != -1 {
		return version[:idx]
	}
	if idx := strings.Index(version, "+"); idx != -1 {
		return version[:idx]
	}
	return version
}

func (j JavaInstaller) installDir(root, version string) string {
	return filepath.Join(root, "java", version)
}

func (j JavaInstaller) EnsureInstalled(ic *itoolchain.InstallContext, version string) error {
	if _, err := j.javaHome(ic, version); err == nil {
		return nil
	}

	asset, err := j.resolveAsset(ic, version)
	if err != nil {
		return err
	}

	archivePath := filepath.Join(ic.CacheDir, asset.Name)
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		emitInstallStep("Downloading %s", asset.Link)
		if err := helpers.DownloadFile(asset.Link, archivePath); err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("stat cached archive: %w", err)
	}

	emitInstallStep("Verifying checksum")
	if err := helpers.VerifyDownloadedArchiveWithExpectedSHA256(archivePath, asset.Name, asset.Checksum); err != nil {
		return err
	}

	installDir := j.installDir(ic.ToolchainDir, version)
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return err
	}

	emitInstallStep("Extracting %s", archivePath)
	if err := helpers.ExtractTarGz(archivePath, installDir); err != nil {
		return err
	}

	if _, err := j.javaHome(ic, version); err != nil {
		return err
	}

	return nil
}

func (j JavaInstaller) BinDir(ic *itoolchain.InstallContext, version string) (string, error) {
	homeDir, err := j.javaHome(ic, version)
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, "bin"), nil
}

func (j JavaInstaller) Env(ic *itoolchain.InstallContext, version string) (map[string]string, error) {
	homeDir, err := j.javaHome(ic, version)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"JAVA_HOME": homeDir,
	}, nil
}

func (j JavaInstaller) resolveAsset(ic *itoolchain.InstallContext, version string) (*struct {
	Checksum string
	Link     string
	Name     string
}, error) {
	apiOS, err := j.apiOS(ic.GOOS)
	if err != nil {
		return nil, err
	}
	apiArch, err := j.apiArch(ic.GOARCH)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(
		"https://api.adoptium.net/v3/assets/latest/%s/hotspot?architecture=%s&image_type=jdk&os=%s&vendor=eclipse",
		j.featureVersion(version),
		apiArch,
		apiOS,
	)

	var payload []adoptiumAssetResponse
	if err := helpers.ReadJSON(url, &payload); err != nil {
		return nil, err
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("java: no download found for version %q", version)
	}

	return &struct {
		Checksum string
		Link     string
		Name     string
	}{
		Checksum: payload[0].Binary.Package.Checksum,
		Link:     payload[0].Binary.Package.Link,
		Name:     payload[0].Binary.Package.Name,
	}, nil
}

func (j JavaInstaller) javaHome(ic *itoolchain.InstallContext, version string) (string, error) {
	installDir := j.installDir(ic.ToolchainDir, version)
	matches, err := filepath.Glob(filepath.Join(installDir, "*", "Contents", "Home", "bin", "java"))
	if err == nil && len(matches) > 0 {
		return filepath.Dir(filepath.Dir(matches[0])), nil
	}

	matches, err = filepath.Glob(filepath.Join(installDir, "*", "bin", "java"))
	if err == nil && len(matches) > 0 {
		return filepath.Dir(filepath.Dir(matches[0])), nil
	}

	return "", fmt.Errorf("java home not found under %s", installDir)
}
