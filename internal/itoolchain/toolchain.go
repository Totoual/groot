package itoolchain

type ToolchainInstaller interface {
	Name() string
	DownloadURL(version, goos, goarch string) (string, error)
	ArchiveName(version, goos, goarch string) string

	ChecksumURL(version string) (string, error)

	InstallDir(root, version string) string
	BinaryPath(root, version string) string
}
