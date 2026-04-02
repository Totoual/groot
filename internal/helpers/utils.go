package helpers

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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

		target, err := safeExtractTarget(dest, header.Name)
		if err != nil {
			return err
		}

		switch header.Typeflag {

		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
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

func ExtractZip(archive, dest string) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer zr.Close()

	for _, file := range zr.File {
		target, err := safeExtractTarget(dest, file.Name)
		if err != nil {
			return err
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, file.Mode()); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		rc, err := file.Open()
		if err != nil {
			return err
		}

		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return err
		}

		if err := out.Close(); err != nil {
			rc.Close()
			return err
		}
		if err := rc.Close(); err != nil {
			return err
		}
	}

	return nil
}

func safeExtractTarget(dest, entryName string) (string, error) {
	clean := filepath.Clean(entryName)
	if clean == "." {
		return "", fmt.Errorf("invalid archive entry %q", entryName)
	}
	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("archive entry %q escapes destination", entryName)
	}

	target := filepath.Join(dest, clean)
	rel, err := filepath.Rel(dest, target)
	if err != nil {
		return "", fmt.Errorf("resolve archive entry %q: %w", entryName, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive entry %q escapes destination", entryName)
	}

	return target, nil
}

func VerifyDownloadedArchive(archivePath, archiveName, checksumURL string) error {
	expected, err := fetchChecksum(checksumURL)
	if err != nil {
		return err
	}

	return VerifyDownloadedArchiveWithExpectedSHA256(archivePath, archiveName, expected)
}

func VerifyDownloadedArchiveWithExpectedSHA256(archivePath, archiveName, expected string) error {
	actual, err := computeSHA256(archivePath)
	if err != nil {
		return err
	}

	if actual != strings.TrimSpace(expected) {
		return fmt.Errorf(
			"checksum mismatch for %s\nexpected: %s\nactual:   %s",
			archiveName,
			strings.TrimSpace(expected),
			actual,
		)
	}

	return nil
}

func VerifyDownloadedArchiveFromChecksumList(archivePath, archiveName, checksumURL string) error {
	body, err := ReadURL(checksumURL)
	if err != nil {
		return err
	}

	expected, err := checksumFromList(string(body), archiveName)
	if err != nil {
		return err
	}

	return VerifyDownloadedArchiveWithExpectedSHA256(archivePath, archiveName, expected)
}

func ReadURL(url string) ([]byte, error) {
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func ReadJSON(url string, dst any) error {
	body, err := ReadURL(url)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, dst); err != nil {
		return err
	}

	return nil
}

func RunCommand(name string, args []string, dir string, env map[string]string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if len(env) > 0 {
		cmdEnv := os.Environ()
		for key, value := range env {
			cmdEnv = setEnv(cmdEnv, key, value)
		}
		cmd.Env = cmdEnv
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run %s: %w", name, err)
	}

	return nil
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i := range env {
		if strings.HasPrefix(env[i], prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func checksumFromList(contents, archiveName string) (string, error) {
	for _, line := range strings.Split(contents, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		if filepath.Base(name) == archiveName {
			return fields[0], nil
		}
	}

	return "", fmt.Errorf("checksum for %s not found", archiveName)
}

func fetchChecksum(url string) (string, error) {
	body, err := ReadURL(url)
	if err != nil {
		return "", err
	}

	sum := strings.TrimSpace(string(body))
	fields := strings.Fields(sum)
	if len(fields) > 0 {
		return fields[0], nil
	}

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
