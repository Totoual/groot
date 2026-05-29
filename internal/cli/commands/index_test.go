package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/totoual/groot/internal/app"
)

func TestIndexCmdZeroValueUsesDefaultSubcommands(t *testing.T) {
	var buf bytes.Buffer
	(&IndexCmd{}).PrintHelp(&buf)

	output := buf.String()
	for _, want := range []string{"init", "update", "search", "symbols", "stats"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help to include %q, got %q", want, output)
		}
	}
}

func TestIndexInitUpdateSearchSymbolsAndStatsCmds(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := setupTaskProject(t, a, root)

	if err := osWriteFile(filepath.Join(projectPath, "engine.go"), `package demo

type Engine struct{}

func ResolveRound() {}

func (e *Engine) DamagePerHeat() {}
`); err != nil {
		t.Fatalf("osWriteFile returned error: %v", err)
	}
	if err := osWriteFile(filepath.Join(projectPath, "notes.txt"), "vault context and workspace scope\n"); err != nil {
		t.Fatalf("osWriteFile returned error: %v", err)
	}

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&indexInitCmd{}).Run(a, []string{"crawlly"})
	})
	if err != nil {
		t.Fatalf("index init returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, `Initialized index for workspace "crawlly".`) {
		t.Fatalf("unexpected init output: %q", stdout)
	}

	stdout, stderr, err = captureCommandOutput(func() error {
		return (&indexUpdateCmd{}).Run(a, []string{"crawlly"})
	})
	if err != nil {
		t.Fatalf("index update returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, `Indexed 2 files`) {
		t.Fatalf("unexpected update output: %q", stdout)
	}

	stdout, stderr, err = captureCommandOutput(func() error {
		return (&indexSearchCmd{}).Run(a, []string{"crawlly", "vault"})
	})
	if err != nil {
		t.Fatalf("index search returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, "notes.txt") {
		t.Fatalf("unexpected search output: %q", stdout)
	}

	stdout, stderr, err = captureCommandOutput(func() error {
		return (&indexSymbolsCmd{}).Run(a, []string{"crawlly", "DamagePerHeat"})
	})
	if err != nil {
		t.Fatalf("index symbols returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, "Engine.DamagePerHeat") {
		t.Fatalf("unexpected symbols output: %q", stdout)
	}

	stdout, stderr, err = captureCommandOutput(func() error {
		return (&indexStatsCmd{}).Run(a, []string{"crawlly"})
	})
	if err != nil {
		t.Fatalf("index stats returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	for _, want := range []string{"Indexed: true", "Indexed At:", "Fresh: true", "Status: fresh", "Workspace: crawlly", "Project Path: " + projectPath, "Files: 2", "Symbols:"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("unexpected stats output: %q", stdout)
		}
	}
}

func TestIndexStatsCmdHandlesMissingMetadata(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := setupTaskProject(t, a, root)
	if err := osWriteFile(filepath.Join(projectPath, "notes.txt"), "vault context and workspace scope\n"); err != nil {
		t.Fatalf("osWriteFile returned error: %v", err)
	}
	if err := (&indexInitCmd{}).Run(a, []string{"crawlly"}); err != nil {
		t.Fatalf("index init returned error: %v", err)
	}

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&indexStatsCmd{}).Run(a, []string{"crawlly"})
	})
	if err != nil {
		t.Fatalf("index stats returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	for _, want := range []string{"Indexed: false", "Indexed At: -", "Fresh: false", "Status: missing_metadata", "Workspace: crawlly", "Project Path: " + projectPath} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("unexpected stats output: %q", stdout)
		}
	}
	if !strings.Contains(stdout, "Files: 0") || !strings.Contains(stdout, "Symbols: 0") || !strings.Contains(stdout, "Terms: 0") {
		t.Fatalf("unexpected stats output: %q", stdout)
	}
}

func osWriteFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600)
}
