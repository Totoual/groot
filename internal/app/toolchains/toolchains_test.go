package toolchains

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/totoual/groot/internal/itoolchain"
)

func TestGoInstallerMetadata(t *testing.T) {
	g := GoInstaller{}
	ic := &itoolchain.InstallContext{ToolchainDir: "/toolchains", GOOS: "linux", GOARCH: "amd64"}

	if got := g.archiveName("1.25.0", "linux", "amd64"); got != "go1.25.0.linux-amd64.tar.gz" {
		t.Fatalf("archiveName = %q", got)
	}
	if got := g.downloadURL("1.25.0", "linux", "amd64"); got != "https://go.dev/dl/go1.25.0.linux-amd64.tar.gz" {
		t.Fatalf("downloadURL = %q", got)
	}
	if got := g.checksumURL("1.25.0", "linux", "amd64"); got != "https://dl.google.com/go/go1.25.0.linux-amd64.tar.gz.sha256" {
		t.Fatalf("checksumURL = %q", got)
	}
	if got := g.installDir("/toolchains", "1.25.0"); got != filepath.Join("/toolchains", "go", "1.25.0") {
		t.Fatalf("installDir = %q", got)
	}
	if got := g.binaryPath("/toolchains", "1.25.0"); got != filepath.Join("/toolchains", "go", "1.25.0", "go", "bin", "go") {
		t.Fatalf("binaryPath = %q", got)
	}

	binDir, err := g.BinDir(ic, "1.25.0")
	if err != nil {
		t.Fatalf("BinDir returned error: %v", err)
	}
	if want := filepath.Join("/toolchains", "go", "1.25.0", "go", "bin"); binDir != want {
		t.Fatalf("BinDir = %q, want %q", binDir, want)
	}
}

func TestNodeInstallerMetadataAndErrors(t *testing.T) {
	n := NodeInstaller{}
	ic := &itoolchain.InstallContext{ToolchainDir: "/toolchains", GOOS: "darwin", GOARCH: "arm64"}

	archiveName, err := n.archiveName("25.0.0", "darwin", "arm64")
	if err != nil {
		t.Fatalf("archiveName returned error: %v", err)
	}
	if archiveName != "node-v25.0.0-darwin-arm64.tar.gz" {
		t.Fatalf("archiveName = %q", archiveName)
	}

	binDir, err := n.BinDir(ic, "25.0.0")
	if err != nil {
		t.Fatalf("BinDir returned error: %v", err)
	}
	wantBinDir := filepath.Join("/toolchains", "node", "25.0.0", "node-v25.0.0-darwin-arm64", "bin")
	if binDir != wantBinDir {
		t.Fatalf("BinDir = %q, want %q", binDir, wantBinDir)
	}

	if _, err := n.platform("windows"); err == nil || !strings.Contains(err.Error(), "unsupported OS") {
		t.Fatalf("expected unsupported OS error, got %v", err)
	}
	if _, err := n.arch("386"); err == nil || !strings.Contains(err.Error(), "unsupported architecture") {
		t.Fatalf("expected unsupported architecture error, got %v", err)
	}
}

func TestDenoInstallerMetadataAndErrors(t *testing.T) {
	d := DenoInstaller{}
	ic := &itoolchain.InstallContext{ToolchainDir: "/toolchains", GOOS: "linux", GOARCH: "amd64"}

	archiveName, err := d.archiveName("2.7.5", "linux", "amd64")
	if err != nil {
		t.Fatalf("archiveName returned error: %v", err)
	}
	if archiveName != "deno-x86_64-unknown-linux-gnu.zip" {
		t.Fatalf("archiveName = %q", archiveName)
	}

	downloadURL, err := d.downloadURL("2.7.5", "linux", "amd64")
	if err != nil {
		t.Fatalf("downloadURL returned error: %v", err)
	}
	if downloadURL != "https://github.com/denoland/deno/releases/download/v2.7.5/deno-x86_64-unknown-linux-gnu.zip" {
		t.Fatalf("downloadURL = %q", downloadURL)
	}

	checksumURL, err := d.checksumURL("2.7.5", "linux", "amd64")
	if err != nil {
		t.Fatalf("checksumURL returned error: %v", err)
	}
	if checksumURL != "https://github.com/denoland/deno/releases/download/v2.7.5/deno-x86_64-unknown-linux-gnu.zip.sha256sum" {
		t.Fatalf("checksumURL = %q", checksumURL)
	}

	binDir, err := d.BinDir(ic, "2.7.5")
	if err != nil {
		t.Fatalf("BinDir returned error: %v", err)
	}
	if want := filepath.Join("/toolchains", "deno", "2.7.5"); binDir != want {
		t.Fatalf("BinDir = %q, want %q", binDir, want)
	}

	if _, err := d.targetTriple("windows", "amd64"); err == nil || !strings.Contains(err.Error(), "unsupported platform") {
		t.Fatalf("expected unsupported platform error, got %v", err)
	}
}

func TestBunInstallerMetadataAndErrors(t *testing.T) {
	b := BunInstaller{}
	ic := &itoolchain.InstallContext{ToolchainDir: "/toolchains", GOOS: "darwin", GOARCH: "arm64"}

	archiveName, err := b.archiveName("1.3.10", "darwin", "arm64")
	if err != nil {
		t.Fatalf("archiveName returned error: %v", err)
	}
	if archiveName != "bun-darwin-aarch64.zip" {
		t.Fatalf("archiveName = %q", archiveName)
	}

	downloadURL, err := b.downloadURL("1.3.10", "darwin", "arm64")
	if err != nil {
		t.Fatalf("downloadURL returned error: %v", err)
	}
	if downloadURL != "https://github.com/oven-sh/bun/releases/download/bun-v1.3.10/bun-darwin-aarch64.zip" {
		t.Fatalf("downloadURL = %q", downloadURL)
	}

	binDir, err := b.BinDir(ic, "1.3.10")
	if err != nil {
		t.Fatalf("BinDir returned error: %v", err)
	}
	if want := filepath.Join("/toolchains", "bun", "1.3.10", "bun-darwin-aarch64"); binDir != want {
		t.Fatalf("BinDir = %q, want %q", binDir, want)
	}

	if _, err := b.platform("windows", "amd64"); err == nil || !strings.Contains(err.Error(), "unsupported platform") {
		t.Fatalf("expected unsupported platform error, got %v", err)
	}
}

func TestJavaInstallerHelpersAndEnv(t *testing.T) {
	j := JavaInstaller{}
	root := t.TempDir()
	ic := &itoolchain.InstallContext{ToolchainDir: root, GOOS: "linux", GOARCH: "amd64"}

	if got := j.featureVersion("21.0.2+13"); got != "21" {
		t.Fatalf("featureVersion = %q", got)
	}
	if got, err := j.apiOS("darwin"); err != nil || got != "mac" {
		t.Fatalf("apiOS = %q, err=%v", got, err)
	}
	if got, err := j.apiArch("amd64"); err != nil || got != "x64" {
		t.Fatalf("apiArch = %q, err=%v", got, err)
	}
	if _, err := j.apiOS("windows"); err == nil || !strings.Contains(err.Error(), "unsupported OS") {
		t.Fatalf("expected unsupported OS error, got %v", err)
	}
	if _, err := j.apiArch("386"); err == nil || !strings.Contains(err.Error(), "unsupported architecture") {
		t.Fatalf("expected unsupported architecture error, got %v", err)
	}

	javaPath := filepath.Join(root, "java", "21", "jdk-21.0.2", "bin", "java")
	if err := os.MkdirAll(filepath.Dir(javaPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(javaPath, []byte(""), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	homeDir, err := j.javaHome(ic, "21")
	if err != nil {
		t.Fatalf("javaHome returned error: %v", err)
	}
	if want := filepath.Join(root, "java", "21", "jdk-21.0.2"); homeDir != want {
		t.Fatalf("javaHome = %q, want %q", homeDir, want)
	}

	binDir, err := j.BinDir(ic, "21")
	if err != nil {
		t.Fatalf("BinDir returned error: %v", err)
	}
	if want := filepath.Join(root, "java", "21", "jdk-21.0.2", "bin"); binDir != want {
		t.Fatalf("BinDir = %q, want %q", binDir, want)
	}

	env, err := j.Env(ic, "21")
	if err != nil {
		t.Fatalf("Env returned error: %v", err)
	}
	if env["JAVA_HOME"] != filepath.Join(root, "java", "21", "jdk-21.0.2") {
		t.Fatalf("JAVA_HOME = %q", env["JAVA_HOME"])
	}
}

func TestPHPInstallerHelpers(t *testing.T) {
	p := PHPInstaller{}
	ic := &itoolchain.InstallContext{ToolchainDir: "/toolchains"}

	if got := p.archiveName("8.5.4"); got != "php-8.5.4.tar.gz" {
		t.Fatalf("archiveName = %q", got)
	}
	if got := p.archiveURL("8.5.4"); got != "https://www.php.net/distributions/php-8.5.4.tar.gz" {
		t.Fatalf("archiveURL = %q", got)
	}
	if got := p.installDir("/toolchains", "8.5.4"); got != filepath.Join("/toolchains", "php", "8.5.4") {
		t.Fatalf("installDir = %q", got)
	}
	if got := p.prefixDir("/toolchains", "8.5.4"); got != filepath.Join("/toolchains", "php", "8.5.4", "php") {
		t.Fatalf("prefixDir = %q", got)
	}
	if got := p.sourceDir("/toolchains", "8.5.4"); got != filepath.Join("/toolchains", "php", "8.5.4", "src", "php-8.5.4") {
		t.Fatalf("sourceDir = %q", got)
	}

	binDir, err := p.BinDir(ic, "8.5.4")
	if err != nil {
		t.Fatalf("BinDir returned error: %v", err)
	}
	if want := filepath.Join("/toolchains", "php", "8.5.4", "php", "bin"); binDir != want {
		t.Fatalf("BinDir = %q, want %q", binDir, want)
	}

	release := phpRelease{
		Version: "8.5.4",
		Source: []struct {
			Filename string `json:"filename"`
			SHA256   string `json:"sha256"`
		}{
			{Filename: "php-8.5.4.tar.gz", SHA256: "abc123"},
		},
	}
	if got := p.releaseChecksum(release); got != "abc123" {
		t.Fatalf("releaseChecksum = %q", got)
	}
}

func TestPythonInstallerHelpers(t *testing.T) {
	p := PythonInstaller{}
	root := t.TempDir()
	ic := &itoolchain.InstallContext{ToolchainDir: root}

	if got := p.archiveName("3.14.2"); got != "Python-3.14.2.tgz" {
		t.Fatalf("archiveName = %q", got)
	}
	if got := p.archiveURL("3.14.2"); got != "https://www.python.org/ftp/python/3.14.2/Python-3.14.2.tgz" {
		t.Fatalf("archiveURL = %q", got)
	}
	if got := p.spdxURL("3.14.2"); got != "https://www.python.org/ftp/python/3.14.2/Python-3.14.2.tgz.spdx.json" {
		t.Fatalf("spdxURL = %q", got)
	}

	binDir, err := p.BinDir(ic, "3.14.2")
	if err != nil {
		t.Fatalf("BinDir returned error: %v", err)
	}
	if want := filepath.Join(root, "python", "3.14.2", "python", "bin"); binDir != want {
		t.Fatalf("BinDir = %q, want %q", binDir, want)
	}

	for _, version := range []string{"3.14.0", "3.14.2", "3.13.7"} {
		path := filepath.Join(root, "python", version)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("MkdirAll returned error: %v", err)
		}
	}

	got, err := p.latestInstalledVersion(root, "3.14")
	if err != nil {
		t.Fatalf("latestInstalledVersion returned error: %v", err)
	}
	if got != "3.14.2" {
		t.Fatalf("latestInstalledVersion = %q", got)
	}
}

func TestRustInstallerHelpersAndEnv(t *testing.T) {
	r := RustInstaller{}
	ic := &itoolchain.InstallContext{ToolchainDir: "/toolchains", GOOS: "linux", GOARCH: "arm64"}

	target, err := r.targetTriple("linux", "arm64")
	if err != nil {
		t.Fatalf("targetTriple returned error: %v", err)
	}
	if target != "aarch64-unknown-linux-gnu" {
		t.Fatalf("targetTriple = %q", target)
	}

	binDir, err := r.BinDir(ic, "stable")
	if err != nil {
		t.Fatalf("BinDir returned error: %v", err)
	}
	if want := filepath.Join("/toolchains", "rust", "stable", "cargo", "bin"); binDir != want {
		t.Fatalf("BinDir = %q, want %q", binDir, want)
	}

	env, err := r.Env(ic, "stable")
	if err != nil {
		t.Fatalf("Env returned error: %v", err)
	}
	if env["CARGO_HOME"] != filepath.Join("/toolchains", "rust", "stable", "cargo") {
		t.Fatalf("CARGO_HOME = %q", env["CARGO_HOME"])
	}
	if env["RUSTUP_HOME"] != filepath.Join("/toolchains", "rust", "stable", "rustup") {
		t.Fatalf("RUSTUP_HOME = %q", env["RUSTUP_HOME"])
	}

	if _, err := r.targetTriple("windows", "amd64"); err == nil || !strings.Contains(err.Error(), "unsupported platform") {
		t.Fatalf("expected unsupported platform error, got %v", err)
	}
}
