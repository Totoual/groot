package toolchains

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/totoual/groot/internal/helpers"
	"github.com/totoual/groot/internal/itoolchain"
)

type PythonInstaller struct{}

type pythonSPDX struct {
	Packages []struct {
		PackageFileName string `json:"packageFileName"`
		Checksums       []struct {
			Algorithm     string `json:"algorithm"`
			ChecksumValue string `json:"checksumValue"`
		} `json:"checksums"`
	} `json:"packages"`
}

func (p PythonInstaller) Name() string {
	return "python"
}

func (p PythonInstaller) installDir(root, version string) string {
	return filepath.Join(root, "python", version)
}

func (p PythonInstaller) prefixDir(root, version string) string {
	return filepath.Join(p.installDir(root, version), "python")
}

func (p PythonInstaller) archiveName(version string) string {
	return fmt.Sprintf("Python-%s.tgz", version)
}

func (p PythonInstaller) archiveURL(version string) string {
	return fmt.Sprintf("https://www.python.org/ftp/python/%s/%s", version, p.archiveName(version))
}

func (p PythonInstaller) spdxURL(version string) string {
	return p.archiveURL(version) + ".spdx.json"
}

func (p PythonInstaller) sourceDir(root, version string) string {
	return filepath.Join(p.installDir(root, version), "src", fmt.Sprintf("Python-%s", version))
}

func (p PythonInstaller) BinDir(ic *itoolchain.InstallContext, version string) (string, error) {
	return filepath.Join(p.prefixDir(ic.ToolchainDir, version), "bin"), nil
}

func (p PythonInstaller) Env(ic *itoolchain.InstallContext, version string) (map[string]string, error) {
	return nil, nil
}

func (p PythonInstaller) EnsureInstalled(ic *itoolchain.InstallContext, version string) error {
	binDir, _ := p.BinDir(ic, version)
	pythonPath := filepath.Join(binDir, "python3")
	if _, err := os.Stat(pythonPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat python binary: %w", err)
	}

	archiveName := p.archiveName(version)
	archivePath := filepath.Join(ic.CacheDir, archiveName)
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		fmt.Println("Downloading", p.archiveURL(version))
		if err := helpers.DownloadFile(p.archiveURL(version), archivePath); err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("stat cached archive: %w", err)
	}

	checksum, err := p.expectedChecksum(version, archiveName)
	if err != nil {
		return err
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
	if _, err := os.Stat(p.sourceDir(ic.ToolchainDir, version)); os.IsNotExist(err) {
		fmt.Println("Extracting", archivePath)
		if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
			return err
		}
		if err := helpers.ExtractTarGz(archivePath, sourceRoot); err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("stat source tree: %w", err)
	}

	sourceDir := p.sourceDir(ic.ToolchainDir, version)
	prefixDir := p.prefixDir(ic.ToolchainDir, version)
	if err := os.MkdirAll(prefixDir, 0o755); err != nil {
		return err
	}

	fmt.Println("Building Python", version)
	if err := helpers.RunCommand("./configure", []string{"--prefix=" + prefixDir, "--without-ensurepip"}, sourceDir, nil); err != nil {
		return err
	}
	if err := helpers.RunCommand("make", []string{fmt.Sprintf("-j%d", runtime.NumCPU())}, sourceDir, nil); err != nil {
		return err
	}
	if err := helpers.RunCommand("make", []string{"install"}, sourceDir, nil); err != nil {
		return err
	}

	if _, err := os.Stat(pythonPath); err != nil {
		return fmt.Errorf("python installed but binary missing: %s", pythonPath)
	}

	return nil
}

func (p PythonInstaller) expectedChecksum(version, archiveName string) (string, error) {
	body, err := helpers.ReadURL(p.spdxURL(version))
	if err != nil {
		return "", err
	}

	var payload pythonSPDX
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}

	for _, pkg := range payload.Packages {
		if pkg.PackageFileName != archiveName {
			continue
		}
		for _, checksum := range pkg.Checksums {
			if checksum.Algorithm == "SHA256" {
				return checksum.ChecksumValue, nil
			}
		}
	}

	return "", fmt.Errorf("python checksum not found for %s", archiveName)
}
