package itoolchain

// InstallContext captures the shared runtime-owned paths and target platform
// information available to installers.
type InstallContext struct {
	ToolchainDir string
	CacheDir     string
	GOOS         string
	GOARCH       string
}

// ToolchainInstaller is the stable installer contract used by the app runtime.
// Installers own toolchain-specific installation, layout, and environment
// details; the app runtime owns workspace orchestration and PATH/env injection.
type ToolchainInstaller interface {
	Name() string
	EnsureInstalled(ic *InstallContext, version string) error
	BinDir(ic *InstallContext, version string) (string, error)
	Env(ic *InstallContext, version string) (map[string]string, error)
}
