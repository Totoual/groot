package helpers

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveGrootHomeUsesDefaultHomeDirectory(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("GROOT_HOME", "")

	got, err := ResolveGrootHome()
	if err != nil {
		t.Fatalf("ResolveGrootHome returned error: %v", err)
	}

	want := filepath.Join(homeDir, ".groot")
	if got != want {
		t.Fatalf("ResolveGrootHome = %q, want %q", got, want)
	}
}

func TestResolveGrootHomeExpandsEnvPath(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("GROOT_HOME", "~/custom/groot")

	got, err := ResolveGrootHome()
	if err != nil {
		t.Fatalf("ResolveGrootHome returned error: %v", err)
	}

	want := filepath.Join(homeDir, "custom", "groot")
	if got != want {
		t.Fatalf("ResolveGrootHome = %q, want %q", got, want)
	}
}

func TestResolveGrootHomeReturnsAbsoluteCleanPath(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("GROOT_HOME", "./tmp-root/../tmp-root-a")

	got, err := ResolveGrootHome()
	if err != nil {
		t.Fatalf("ResolveGrootHome returned error: %v", err)
	}

	want, err := filepath.Abs("./tmp-root-a")
	if err != nil {
		t.Fatalf("filepath.Abs returned error: %v", err)
	}

	if got != filepath.Clean(want) {
		t.Fatalf("ResolveGrootHome = %q, want %q", got, filepath.Clean(want))
	}
}

func TestVerifyDownloadedArchiveWithExpectedSHA256(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "archive.tar.gz")
	contents := []byte("archive")
	if err := os.WriteFile(archivePath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	sum := sha256.Sum256(contents)
	expected := hex.EncodeToString(sum[:])

	if err := VerifyDownloadedArchiveWithExpectedSHA256(archivePath, "archive.tar.gz", expected); err != nil {
		t.Fatalf("VerifyDownloadedArchiveWithExpectedSHA256 returned error: %v", err)
	}
}

func TestVerifyDownloadedArchiveWithExpectedSHA256RejectsMismatch(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "archive.tar.gz")
	if err := os.WriteFile(archivePath, []byte("archive"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	err := VerifyDownloadedArchiveWithExpectedSHA256(archivePath, "archive.tar.gz", "deadbeef")
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}
}

func TestExtractTarGzRejectsPathTraversal(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "evil.tar.gz")
	dest := filepath.Join(root, "dest")

	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	gzw := gzip.NewWriter(file)
	tw := tar.NewWriter(gzw)
	if err := tw.WriteHeader(&tar.Header{Name: "../escape.txt", Mode: 0o600, Size: int64(len("evil"))}); err != nil {
		t.Fatalf("WriteHeader returned error: %v", err)
	}
	if _, err := tw.Write([]byte("evil")); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("Close tar returned error: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("Close gzip returned error: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close file returned error: %v", err)
	}

	err = ExtractTarGz(archivePath, dest)
	if err == nil {
		t.Fatal("expected path traversal error")
	}
	if _, err := os.Stat(filepath.Join(root, "escape.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected no escaped file, stat err=%v", err)
	}
}

func TestExtractZipRejectsPathTraversal(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "evil.zip")
	dest := filepath.Join(root, "dest")

	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	zw := zip.NewWriter(file)
	w, err := zw.Create("../escape.txt")
	if err != nil {
		t.Fatalf("Create zip entry returned error: %v", err)
	}
	if _, err := w.Write([]byte("evil")); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("Close zip returned error: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close file returned error: %v", err)
	}

	err = ExtractZip(archivePath, dest)
	if err == nil {
		t.Fatal("expected path traversal error")
	}
	if _, err := os.Stat(filepath.Join(root, "escape.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected no escaped file, stat err=%v", err)
	}
}

func TestChecksumFromListMatchesArchiveName(t *testing.T) {
	body := bytes.NewBufferString("abc123  ./archive.tar.gz\n")
	checksum, err := checksumFromList(body.String(), "archive.tar.gz")
	if err != nil {
		t.Fatalf("checksumFromList returned error: %v", err)
	}
	if checksum != "abc123" {
		t.Fatalf("checksumFromList = %q", checksum)
	}
}
