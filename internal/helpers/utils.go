package helpers

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func ResolveGrootHome() (string, error) {
	var root string

	if env := os.Getenv("GROOT_HOME"); env != "" {
		root = env
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(home, ".groot")
	}

	// Expand ~ manually if user passed it
	if strings.HasPrefix(root, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(home, root[1:])
	}

	// Convert to absolute path
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}

	return filepath.Clean(abs), nil
}

func DownloadFile(url, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return fmt.Errorf("create download dir: %w", err)
	}

	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	tmp := dest + ".tmp"

	out, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create temp file %s: %w", tmp, err)
	}

	defer func() {
		out.Close()
	}()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("write temp file %s: %w", tmp, err)
	}

	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close temp file %s: %w", tmp, err)
	}

	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename %s -> %s: %w", tmp, dest, err)
	}

	return nil
}

func ExtractTarGz(archive, dest string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)

		switch header.Typeflag {

		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			out, err := os.Create(target)
			if err != nil {
				return err
			}

			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}

			out.Close()
		}
	}

	return nil
}

func VerifyDownloadedArchive(archivePath, archiveName, checksumURL string) error {
	expected, err := fetchChecksum(checksumURL)
	if err != nil {
		return err
	}

	actual, err := computeSHA256(archivePath)
	if err != nil {
		return err
	}

	if actual != expected {
		return fmt.Errorf(
			"checksum mismatch for %s\nexpected: %s\nactual:   %s",
			archiveName,
			expected,
			actual,
		)
	}

	return nil
}

func fetchChecksum(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksum download failed: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	sum := strings.TrimSpace(string(data))

	return sum, nil
}

func computeSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	h := sha256.New()

	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
