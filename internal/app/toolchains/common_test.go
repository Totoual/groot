package toolchains

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/totoual/groot/internal/itoolchain"
)

func TestInstallArchiveWithExtractorIfNeededSkipsWhenBinaryExists(t *testing.T) {
	root := t.TempDir()
	binaryPath := filepath.Join(root, "toolchains", "go", "bin", "go")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(binaryPath, []byte("go"), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	verifyCalled := false
	extractCalled := false

	err := installArchiveWithExtractorIfNeeded(
		&itoolchain.InstallContext{CacheDir: filepath.Join(root, "cache")},
		binaryPath,
		"https://example.com/go.tar.gz",
		"go.tar.gz",
		filepath.Join(root, "install"),
		func(string) error {
			verifyCalled = true
			return nil
		},
		func(string, string) error {
			extractCalled = true
			return nil
		},
	)
	if err != nil {
		t.Fatalf("installArchiveWithExtractorIfNeeded returned error: %v", err)
	}
	if verifyCalled || extractCalled {
		t.Fatalf("expected installer to skip verify/extract, got verify=%v extract=%v", verifyCalled, extractCalled)
	}
}

func TestInstallArchiveWithExtractorIfNeededUsesCachedArchiveAndInstallsBinary(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(root, "cache")
	installDir := filepath.Join(root, "install")
	binaryPath := filepath.Join(installDir, "go", "bin", "go")
	archivePath := filepath.Join(cacheDir, "go.tar.gz")

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(archivePath, []byte("archive"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	verifyCalled := false
	extractCalled := false

	err := installArchiveWithExtractorIfNeeded(
		&itoolchain.InstallContext{CacheDir: cacheDir},
		binaryPath,
		"https://example.com/go.tar.gz",
		"go.tar.gz",
		installDir,
		func(path string) error {
			verifyCalled = path == archivePath
			return nil
		},
		func(path, dest string) error {
			extractCalled = path == archivePath && strings.HasPrefix(dest, installDir+".tmp-")
			stagedBinaryPath := filepath.Join(dest, "go", "bin", "go")
			if err := os.MkdirAll(filepath.Dir(stagedBinaryPath), 0o755); err != nil {
				return err
			}
			return os.WriteFile(stagedBinaryPath, []byte("go"), 0o755)
		},
	)
	if err != nil {
		t.Fatalf("installArchiveWithExtractorIfNeeded returned error: %v", err)
	}
	if !verifyCalled || !extractCalled {
		t.Fatalf("expected verify and extract to be called, got verify=%v extract=%v", verifyCalled, extractCalled)
	}
	if _, err := os.Stat(binaryPath); err != nil {
		t.Fatalf("expected binary to exist after install: %v", err)
	}
}

func TestInstallArchiveWithExtractorIfNeededFailsWhenBinaryMissingAfterExtract(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(root, "cache")
	installDir := filepath.Join(root, "install")
	binaryPath := filepath.Join(installDir, "go", "bin", "go")
	archivePath := filepath.Join(cacheDir, "go.tar.gz")

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(archivePath, []byte("archive"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	err := installArchiveWithExtractorIfNeeded(
		&itoolchain.InstallContext{CacheDir: cacheDir},
		binaryPath,
		"https://example.com/go.tar.gz",
		"go.tar.gz",
		installDir,
		func(string) error { return nil },
		func(string, string) error {
			return nil
		},
	)
	if err == nil {
		t.Fatal("expected missing binary error")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "binary missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallArchiveWithExtractorIfNeededReplacesExistingInstallAtomically(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(root, "cache")
	installDir := filepath.Join(root, "install")
	binaryPath := filepath.Join(installDir, "go", "bin", "go")
	archivePath := filepath.Join(cacheDir, "go.tar.gz")

	if err := os.MkdirAll(filepath.Join(installDir, "stale"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "stale", "old.txt"), []byte("old"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(archivePath, []byte("archive"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	err := installArchiveWithExtractorIfNeeded(
		&itoolchain.InstallContext{CacheDir: cacheDir},
		binaryPath,
		"https://example.com/go.tar.gz",
		"go.tar.gz",
		installDir,
		func(string) error { return nil },
		func(_ string, dest string) error {
			if err := os.MkdirAll(filepath.Join(dest, "go", "bin"), 0o755); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(dest, "go", "bin", "go"), []byte("go"), 0o755)
		},
	)
	if err != nil {
		t.Fatalf("installArchiveWithExtractorIfNeeded returned error: %v", err)
	}

	if _, err := os.Stat(binaryPath); err != nil {
		t.Fatalf("expected binary to exist after install: %v", err)
	}
	if _, err := os.Stat(filepath.Join(installDir, "stale", "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected stale install contents to be replaced, stat err=%v", err)
	}

	entries, err := os.ReadDir(filepath.Dir(installDir))
	if err != nil {
		t.Fatalf("ReadDir returned error: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), filepath.Base(installDir)+".tmp-") {
			t.Fatalf("expected no leftover staging dir, found %q", entry.Name())
		}
	}
}
