package versioninfo

import (
	"os"
	"runtime/debug"
)

// ReleaseVersion is the product-facing version shown by `groot version`.
// Release builds can override it with -ldflags.
var ReleaseVersion = "dev"

type Info struct {
	Version       string `json:"version"`
	ModulePath    string `json:"module_path"`
	ModuleVersion string `json:"module_version,omitempty"`
	GoVersion     string `json:"go_version"`
	VCSRevision   string `json:"vcs_revision,omitempty"`
	VCSTime       string `json:"vcs_time,omitempty"`
	VCSModified   string `json:"vcs_modified,omitempty"`
	BinaryPath    string `json:"binary_path,omitempty"`
	BuildMode     string `json:"build_mode,omitempty"`
	Compiler      string `json:"compiler,omitempty"`
	CGOEnabled    string `json:"cgo_enabled,omitempty"`
	BuildMissing  bool   `json:"build_missing,omitempty"`
}

func Current() Info {
	info := Info{
		Version: ReleaseVersion,
	}
	if info.Version == "" {
		info.Version = "dev"
	}

	if executable, err := os.Executable(); err == nil {
		info.BinaryPath = executable
	}

	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		info.BuildMissing = true
		return info
	}

	info.ModulePath = buildInfo.Main.Path
	info.GoVersion = buildInfo.GoVersion
	if buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
		info.ModuleVersion = buildInfo.Main.Version
	}

	for _, setting := range buildInfo.Settings {
		switch setting.Key {
		case "vcs.revision":
			info.VCSRevision = setting.Value
		case "vcs.time":
			info.VCSTime = setting.Value
		case "vcs.modified":
			info.VCSModified = setting.Value
		case "-buildmode":
			info.BuildMode = setting.Value
		case "-compiler":
			info.Compiler = setting.Value
		case "CGO_ENABLED":
			info.CGOEnabled = setting.Value
		}
	}

	return info
}
