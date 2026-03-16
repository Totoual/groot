package toolchains

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/totoual/groot/internal/helpers"
	"github.com/totoual/groot/internal/itoolchain"
)

type PHPInstaller struct{}

type phpRelease struct {
	Version string `json:"version"`
	Source  []struct {
		Filename string `json:"filename"`
		SHA256   string `json:"sha256"`
	} `json:"source"`
}

func (p PHPInstaller) Name() string {
	return "php"
}

func (p PHPInstaller) installDir(root, version string) string {
	return filepath.Join(root, "php", version)
}

func (p PHPInstaller) prefixDir(root, version string) string {
	return filepath.Join(p.installDir(root, version), "php")
}

func (p PHPInstaller) archiveName(version string) string {
	return "php-" + version + ".tar.gz"
}

func (p PHPInstaller) archiveURL(version string) string {
	return "https://www.php.net/distributions/" + p.archiveName(version)
}

func (p PHPInstaller) sourceDir(root, version string) string {
	return filepath.Join(p.installDir(root, version), "src", "php-"+version)
}

func (p PHPInstaller) BinDir(ic *itoolchain.InstallContext, version string) (string, error) {
	return filepath.Join(p.prefixDir(ic.ToolchainDir, version), "bin"), nil
}

func (p PHPInstaller) Env(ic *itoolchain.InstallContext, version string) (map[string]string, error) {
	return nil, nil
}

func (p PHPInstaller) EnsureInstalled(ic *itoolchain.InstallContext, version string) error {
	binDir, _ := p.BinDir(ic, version)
	phpPath := filepath.Join(binDir, "php")
	if _, err := os.Stat(phpPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat php binary: %w", err)
	}

	resolvedVersion, checksum, err := p.resolveRelease(version)
	if err != nil {
		return err
	}

	archiveName := p.archiveName(resolvedVersion)
	archivePath := filepath.Join(ic.CacheDir, archiveName)
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		fmt.Println("Downloading", p.archiveURL(resolvedVersion))
		if err := helpers.DownloadFile(p.archiveURL(resolvedVersion), archivePath); err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("stat cached archive: %w", err)
	}

	fmt.Println("Verifying checksum")
	if err := helpers.VerifyDownloadedArchiveWithExpectedSHA256(archivePath, archiveName, checksum); err != nil {
		return err
	}

	installDir := p.installDir(ic.ToolchainDir, version)
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return err
	}

	sourceRoot := filepath.Join(installDir, "src")
	sourceDir := p.sourceDir(ic.ToolchainDir, version)
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		fmt.Println("Extracting", archivePath)
		if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
			return err
		}
		if err := helpers.ExtractTarGz(archivePath, sourceRoot); err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("stat php source tree: %w", err)
	}

	prefixDir := p.prefixDir(ic.ToolchainDir, version)
	if err := os.MkdirAll(prefixDir, 0o755); err != nil {
		return err
	}

	fmt.Println("Building PHP", resolvedVersion)
	if err := helpers.RunCommand("./configure", []string{"--prefix=" + prefixDir, "--disable-all", "--enable-cli"}, sourceDir, nil); err != nil {
		return err
	}
	if err := helpers.RunCommand("make", []string{fmt.Sprintf("-j%d", runtime.NumCPU())}, sourceDir, nil); err != nil {
		return err
	}
	if err := helpers.RunCommand("make", []string{"install"}, sourceDir, nil); err != nil {
		return err
	}

	if _, err := os.Stat(phpPath); err != nil {
		return fmt.Errorf("php installed but binary missing: %s", phpPath)
	}

	return nil
}

func (p PHPInstaller) resolveRelease(version string) (string, string, error) {
	body, err := helpers.ReadURL("https://www.php.net/releases/index.php?json")
	if err != nil {
		return "", "", err
	}

	var payload map[string]phpRelease
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", "", err
	}

	if resolvedVersion, checksum := p.matchRelease(payload, version); resolvedVersion != "" {
		return resolvedVersion, checksum, nil
	}

	return "", "", fmt.Errorf("php release %q not found", version)
}

func (p PHPInstaller) matchRelease(payload map[string]phpRelease, version string) (string, string) {
	if checksum := p.checksumForVersion(payload, version); checksum != "" {
		return version, checksum
	}

	if strings.Count(version, ".") == 1 {
		prefix := version + "."
		for _, release := range payload {
			if strings.HasPrefix(release.Version, prefix) {
				if checksum := p.releaseChecksum(release); checksum != "" {
					return release.Version, checksum
				}
			}
		}
	}

	return "", ""
}

func (p PHPInstaller) checksumForVersion(payload map[string]phpRelease, version string) string {
	for _, release := range payload {
		if release.Version == version {
			return p.releaseChecksum(release)
		}
	}
	return ""
}

func (p PHPInstaller) releaseChecksum(release phpRelease) string {
	filename := p.archiveName(release.Version)
	for _, source := range release.Source {
		if source.Filename == filename {
			return source.SHA256
		}
	}
	return ""
}
