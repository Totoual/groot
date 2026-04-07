package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/totoual/groot/internal/app"
)

func TestMCPCmdHelp(t *testing.T) {
	a := app.NewApp(t.TempDir())

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&MCPCmd{}).Run(a, []string{"-h"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, "usage: groot mcp") {
		t.Fatalf("expected usage in stdout, got %q", stdout)
	}
	if !strings.Contains(stdout, "--project") || !strings.Contains(stdout, "--workspace") {
		t.Fatalf("expected scoping flags in help output, got %q", stdout)
	}
}

func TestMCPAllowedProjectsIncludesNormalizedProjectAndWorkspaceScopes(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	got, err := mcpAllowedProjects(a, []string{filepath.Join(projectPath, "..", "crawlly")}, []string{"crawlly"})
	if err != nil {
		t.Fatalf("mcpAllowedProjects returned error: %v", err)
	}
	normalized, err := app.NormalizeProjectPath(projectPath)
	if err != nil {
		t.Fatalf("NormalizeProjectPath returned error: %v", err)
	}
	if len(got) != 1 || got[0] != normalized {
		t.Fatalf("allowed projects = %#v, want [%q]", got, normalized)
	}
}

func TestMCPAllowedProjectsRejectsUnboundWorkspaceScope(t *testing.T) {
	a := app.NewApp(t.TempDir())
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	_, err := mcpAllowedProjects(a, nil, []string{"crawlly"})
	if err == nil {
		t.Fatal("expected unbound workspace scope to fail")
	}
	if !strings.Contains(err.Error(), `workspace "crawlly" is not bound to a project path`) {
		t.Fatalf("unexpected error: %v", err)
	}
}
