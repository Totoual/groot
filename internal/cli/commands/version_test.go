package commands

import (
	"strings"
	"testing"

	"github.com/totoual/groot/internal/versioninfo"
)

func TestVersionCmdRunPrintsShortVersion(t *testing.T) {
	old := currentVersionInfo
	currentVersionInfo = func() versioninfo.Info {
		return versioninfo.Info{
			Version:       "v0.1.0",
			ModulePath:    "github.com/totoual/groot/cmd/groot",
			ModuleVersion: "v0.0.0-20260425152153-7dfb8af32701",
			GoVersion:     "go1.25.4",
			VCSRevision:   "abc123",
			VCSTime:       "2026-04-25T16:00:00Z",
			VCSModified:   "false",
			BinaryPath:    "/tmp/groot",
		}
	}
	defer func() { currentVersionInfo = old }()

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&VersionCmd{}).Run(nil, nil)
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if strings.TrimSpace(stdout) != "v0.1.0" {
		t.Fatalf("stdout = %q, want %q", strings.TrimSpace(stdout), "v0.1.0")
	}
}

func TestVersionCmdRunPrintsVerboseOutput(t *testing.T) {
	old := currentVersionInfo
	currentVersionInfo = func() versioninfo.Info {
		return versioninfo.Info{
			Version:       "v0.1.0",
			ModulePath:    "github.com/totoual/groot/cmd/groot",
			ModuleVersion: "v0.0.0-20260425152153-7dfb8af32701",
			GoVersion:     "go1.25.4",
			VCSRevision:   "abc123",
			VCSTime:       "2026-04-25T16:00:00Z",
			VCSModified:   "false",
			BinaryPath:    "/tmp/groot",
		}
	}
	defer func() { currentVersionInfo = old }()

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&VersionCmd{}).Run(nil, []string{"--verbose"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	for _, want := range []string{
		"Version: v0.1.0",
		"Module: github.com/totoual/groot/cmd/groot",
		"Module Version: v0.0.0-20260425152153-7dfb8af32701",
		"Go: go1.25.4",
		"Revision: abc123",
		"Build Time: 2026-04-25T16:00:00Z",
		"Modified: false",
		"Binary: /tmp/groot",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected %q in output, got %q", want, stdout)
		}
	}
}

func TestVersionCmdRunPrintsJSON(t *testing.T) {
	old := currentVersionInfo
	currentVersionInfo = func() versioninfo.Info {
		return versioninfo.Info{
			Version:     "v0.1.0",
			GoVersion:   "go1.25.4",
			VCSRevision: "abc123",
		}
	}
	defer func() { currentVersionInfo = old }()

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&VersionCmd{}).Run(nil, []string{"--json"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	for _, want := range []string{`"version": "v0.1.0"`, `"go_version": "go1.25.4"`, `"vcs_revision": "abc123"`} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected %q in JSON output, got %q", want, stdout)
		}
	}
}
