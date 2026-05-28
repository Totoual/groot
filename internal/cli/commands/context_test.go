package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/totoual/groot/internal/app"
)

func TestContextCmdZeroValueUsesDefaultSubcommands(t *testing.T) {
	var buf bytes.Buffer
	(&ContextCmd{}).PrintHelp(&buf)

	if !strings.Contains(buf.String(), "build") {
		t.Fatalf("expected help to include build command, got %q", buf.String())
	}
}

func TestContextBuildCmdRunPrintsMarkdownPack(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := setupTaskProject(t, a, root)

	if err := os.MkdirAll(filepath.Join(projectPath, "internal"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "internal", "engine.go"), []byte("package demo\n\nfunc DamagePerHeat() {}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if _, err := a.UpdateIndex("crawlly"); err != nil {
		t.Fatalf("UpdateIndex returned error: %v", err)
	}
	if _, err := a.VaultAppend("crawlly", app.VaultAppendSpec{
		Type:  app.VaultNodeTypeDecision,
		Title: "Vault is workspace-scoped",
		Body:  "Each workspace owns its own vault.",
		Tags:  []string{"vault"},
	}); err != nil {
		t.Fatalf("VaultAppend returned error: %v", err)
	}

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&contextBuildCmd{}).Run(a, []string{"crawlly", "vault", "damage"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	for _, want := range []string{
		"# Groot Context Pack",
		"Task:\nvault damage",
		"Relevant Vault Entries:",
		"Relevant Files:",
		"Relevant Symbols:",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, stdout)
		}
	}
}
