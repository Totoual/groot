package toolchains

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"

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

var pythonReleaseDirPattern = regexp.MustCompile(`href="([0-9]+\.[0-9]+\.[0-9]+)/"`)

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
	resolvedVersion, err := p.resolveVersion(ic, version)
	if err != nil {
		return "", err
	}
	return filepath.Join(p.prefixDir(ic.ToolchainDir, resolvedVersion), "bin"), nil
}

func (p PythonInstaller) Env(ic *itoolchain.InstallContext, version string) (map[string]string, error) {
	return nil, nil
}

func (p PythonInstaller) EnsureInstalled(ic *itoolchain.InstallContext, version string) error {
	resolvedVersion, err := p.resolveVersion(ic, version)
	if err != nil {
		return err
	}

	binDir := filepath.Join(p.prefixDir(ic.ToolchainDir, resolvedVersion), "bin")
	pythonPath := filepath.Join(binDir, "python3")
	if _, err := os.Stat(pythonPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat python binary: %w", err)
	}

	archiveName := p.archiveName(resolvedVersion)
	archivePath := filepath.Join(ic.CacheDir, archiveName)
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		emitInstallStep("Downloading %s", p.archiveURL(resolvedVersion))
		if err := helpers.DownloadFile(p.archiveURL(resolvedVersion), archivePath); err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("stat cached archive: %w", err)
	}

	checksum, err := p.expectedChecksum(resolvedVersion, archiveName)
	if err != nil {
		return err
	}

	emitInstallStep("Verifying checksum")
	if err := helpers.VerifyDownloadedArchiveWithExpectedSHA256(archivePath, archiveName, checksum); err != nil {
		return err
	}

	installDir := p.installDir(ic.ToolchainDir, resolvedVersion)
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return err
	}

	sourceRoot := filepath.Join(installDir, "src")
	if _, err := os.Stat(p.sourceDir(ic.ToolchainDir, resolvedVersion)); os.IsNotExist(err) {
		emitInstallStep("Extracting %s", archivePath)
		if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
			return err
		}
		if err := helpers.ExtractTarGz(archivePath, sourceRoot); err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("stat source tree: %w", err)
	}

	sourceDir := p.sourceDir(ic.ToolchainDir, resolvedVersion)
	prefixDir := p.prefixDir(ic.ToolchainDir, resolvedVersion)
	if err := os.MkdirAll(prefixDir, 0o755); err != nil {
		return err
	}

	emitInstallStep("Building Python %s", resolvedVersion)
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

func (p PythonInstaller) resolveVersion(ic *itoolchain.InstallContext, version string) (string, error) {
	if !isPythonMinorSeries(version) {
		return version, nil
	}

	resolvedVersion, err := p.resolveLatestPublishedVersion(version)
	if err == nil {
		return resolvedVersion, nil
	}

	localVersion, localErr := p.latestInstalledVersion(ic.ToolchainDir, version)
	if localErr == nil {
		return localVersion, nil
	}

	return "", fmt.Errorf("resolve python version %s: %w", version, err)
}

func (p PythonInstaller) resolveLatestPublishedVersion(series string) (string, error) {
	body, err := helpers.ReadURL("https://www.python.org/ftp/python/")
	if err != nil {
		return "", err
	}

	return latestPythonPatchVersion(string(body), series)
}

func (p PythonInstaller) latestInstalledVersion(root, series string) (string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "python"))
	if err != nil {
		return "", err
	}

	candidates := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, series+".") && isExactPythonVersion(name) {
			candidates = append(candidates, name)
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no installed python release found for %s", series)
	}

	slices.SortFunc(candidates, comparePythonVersions)
	return candidates[len(candidates)-1], nil
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

func latestPythonPatchVersion(listing, series string) (string, error) {
	matches := pythonReleaseDirPattern.FindAllStringSubmatch(listing, -1)
	candidates := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))

	for _, match := range matches {
		version := match[1]
		if !strings.HasPrefix(version, series+".") {
			continue
		}
		if !isExactPythonVersion(version) {
			continue
		}
		if _, ok := seen[version]; ok {
			continue
		}
		seen[version] = struct{}{}
		candidates = append(candidates, version)
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no Python patch release found for %s", series)
	}

	slices.SortFunc(candidates, comparePythonVersions)
	return candidates[len(candidates)-1], nil
}

func isPythonMinorSeries(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) != 2 {
		return false
	}

	for _, part := range parts {
		if part == "" {
			return false
		}
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}

	return true
}

func isExactPythonVersion(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}

	for _, part := range parts {
		if part == "" {
			return false
		}
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}

	return true
}

func comparePythonVersions(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	for i := 0; i < 3; i++ {
		aNum, _ := strconv.Atoi(aParts[i])
		bNum, _ := strconv.Atoi(bParts[i])
		if aNum < bNum {
			return -1
		}
		if aNum > bNum {
			return 1
		}
	}

	return 0
}
