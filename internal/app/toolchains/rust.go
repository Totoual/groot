package toolchains

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/totoual/groot/internal/helpers"
	"github.com/totoual/groot/internal/itoolchain"
)

type RustInstaller struct{}

func (r RustInstaller) Name() string {
	return "rust"
}

func (r RustInstaller) targetTriple(goos, goarch string) (string, error) {
	switch {
	case goos == "darwin" && goarch == "arm64":
		return "aarch64-apple-darwin", nil
	case goos == "darwin" && goarch == "amd64":
		return "x86_64-apple-darwin", nil
	case goos == "linux" && goarch == "arm64":
		return "aarch64-unknown-linux-gnu", nil
	case goos == "linux" && goarch == "amd64":
		return "x86_64-unknown-linux-gnu", nil
	default:
		return "", fmt.Errorf("rust: unsupported platform %s/%s", goos, goarch)
	}
}

func (r RustInstaller) installDir(root, version string) string {
	return filepath.Join(root, "rust", version)
}

func (r RustInstaller) BinDir(ic *itoolchain.InstallContext, version string) (string, error) {
	return filepath.Join(r.installDir(ic.ToolchainDir, version), "cargo", "bin"), nil
}

func (r RustInstaller) Env(ic *itoolchain.InstallContext, version string) (map[string]string, error) {
	installDir := r.installDir(ic.ToolchainDir, version)
	return map[string]string{
		"CARGO_HOME":  filepath.Join(installDir, "cargo"),
		"RUSTUP_HOME": filepath.Join(installDir, "rustup"),
	}, nil
}

func (r RustInstaller) EnsureInstalled(ic *itoolchain.InstallContext, version string) error {
	binDir, _ := r.BinDir(ic, version)
	rustcPath := filepath.Join(binDir, "rustc")
	if _, err := os.Stat(rustcPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat rust toolchain binary: %w", err)
	}

	target, err := r.targetTriple(ic.GOOS, ic.GOARCH)
	if err != nil {
		return err
	}

	bootstrapURL := fmt.Sprintf("https://static.rust-lang.org/rustup/dist/%s/rustup-init", target)
	checksumURL := bootstrapURL + ".sha256"
	bootstrapPath := filepath.Join(ic.CacheDir, "rustup-init-"+target)

	if _, err := os.Stat(bootstrapPath); os.IsNotExist(err) {
		fmt.Println("Downloading", bootstrapURL)
		if err := helpers.DownloadFile(bootstrapURL, bootstrapPath); err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("stat rust bootstrap: %w", err)
	}

	fmt.Println("Verifying checksum")
	if err := helpers.VerifyDownloadedArchive(bootstrapPath, filepath.Base(bootstrapPath), checksumURL); err != nil {
		return err
	}

	if err := os.Chmod(bootstrapPath, 0o755); err != nil {
		return err
	}

	installDir := r.installDir(ic.ToolchainDir, version)
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return err
	}

	env := map[string]string{
		"CARGO_HOME":                 filepath.Join(installDir, "cargo"),
		"RUSTUP_HOME":                filepath.Join(installDir, "rustup"),
		"CARGO_INIT_SKIP_PATH_CHECK": "yes",
	}

	fmt.Println("Installing Rust toolchain", version)
	return helpers.RunCommand(
		bootstrapPath,
		[]string{"-y", "--profile", "minimal", "--no-modify-path", "--default-toolchain", version},
		installDir,
		env,
	)
}
