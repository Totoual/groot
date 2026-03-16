package itoolchain

type InstallContext struct {
	ToolchainDir string
	CacheDir     string
	GOOS         string
	GOARCH       string
}

type ToolchainInstaller interface {
	Name() string
	EnsureInstalled(ic *InstallContext, version string) error
	BinDir(ic *InstallContext, version string) (string, error)
	Env(ic *InstallContext, version string) (map[string]string, error)
}
