package toolchains

import (
	"fmt"
	"os"
	"path/filepath"

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
		fmt.Println("Downloading", archiveURL)
		if err := helpers.DownloadFile(archiveURL, archivePath); err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("stat cached archive: %w", err)
	}

	fmt.Println("Verifying checksum")
	if err := verify(archivePath); err != nil {
		return err
	}

	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return err
	}

	fmt.Println("Extracting", archivePath)
	if err := extract(archivePath, installDir); err != nil {
		return err
	}

	if _, err := os.Stat(binaryPath); err != nil {
		return fmt.Errorf("toolchain installed but binary missing: %s", binaryPath)
	}

	return nil
}
