package toolchains

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/totoual/groot/internal/helpers"
	"github.com/totoual/groot/internal/itoolchain"
)

func installArchiveIfNeeded(
	ic *itoolchain.InstallContext,
	binaryPath string,
	archiveURL string,
	archiveName string,
	installDir string,
	verify func(archivePath string) error,
) error {
	return installArchiveWithExtractorIfNeeded(ic, binaryPath, archiveURL, archiveName, installDir, verify, helpers.ExtractTarGz)
}

func installZipArchiveIfNeeded(
	ic *itoolchain.InstallContext,
	binaryPath string,
	archiveURL string,
	archiveName string,
	installDir string,
	verify func(archivePath string) error,
) error {
	return installArchiveWithExtractorIfNeeded(ic, binaryPath, archiveURL, archiveName, installDir, verify, helpers.ExtractZip)
}

func installArchiveWithExtractorIfNeeded(
	ic *itoolchain.InstallContext,
	binaryPath string,
	archiveURL string,
	archiveName string,
	installDir string,
	verify func(archivePath string) error,
	extract func(archive, dest string) error,
) error {
	if _, err := os.Stat(binaryPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat toolchain binary: %w", err)
	}

	archivePath := filepath.Join(ic.CacheDir, archiveName)
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		emitInstallStep("Downloading %s", archiveURL)
		if err := helpers.DownloadFile(archiveURL, archivePath); err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("stat cached archive: %w", err)
	}

	emitInstallStep("Verifying checksum")
	if err := verify(archivePath); err != nil {
		return err
	}

	installParent := filepath.Dir(installDir)
	if err := os.MkdirAll(installParent, 0o755); err != nil {
		return err
	}

	relBinaryPath, err := filepath.Rel(installDir, binaryPath)
	if err != nil {
		return fmt.Errorf("resolve staged binary path: %w", err)
	}
	if relBinaryPath == ".." || strings.HasPrefix(relBinaryPath, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("binary path %q escapes install dir %q", binaryPath, installDir)
	}

	stagingDir, err := os.MkdirTemp(installParent, filepath.Base(installDir)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create staging dir: %w", err)
	}
	defer func() {
		if stagingDir != "" {
			_ = os.RemoveAll(stagingDir)
		}
	}()

	emitInstallStep("Extracting %s", archivePath)
	if err := extract(archivePath, stagingDir); err != nil {
		return err
	}

	stagedBinaryPath := filepath.Join(stagingDir, relBinaryPath)
	if _, err := os.Stat(stagedBinaryPath); err != nil {
		return fmt.Errorf("toolchain installed but binary missing: %s", binaryPath)
	}

	if err := os.RemoveAll(installDir); err != nil {
		return fmt.Errorf("remove existing install dir: %w", err)
	}
	if err := os.Rename(stagingDir, installDir); err != nil {
		return fmt.Errorf("activate staged install: %w", err)
	}
	stagingDir = ""

	return nil
}
