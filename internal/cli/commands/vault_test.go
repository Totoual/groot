package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/totoual/groot/internal/app"
)

func TestVaultCmdZeroValueUsesDefaultSubcommands(t *testing.T) {
	var buf bytes.Buffer
	(&VaultCmd{}).PrintHelp(&buf)

	output := buf.String()
	for _, want := range []string{"init", "append", "edge", "search", "recent", "stats"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help to include %q, got %q", want, output)
		}
	}
}

func TestVaultInitAppendSearchRecentAndStatsCmds(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&vaultInitCmd{}).Run(a, []string{"crawlly"})
	})
	if err != nil {
		t.Fatalf("vault init returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, `Initialized vault for workspace "crawlly".`) {
		t.Fatalf("unexpected init output: %q", stdout)
	}

	stdout, stderr, err = captureCommandOutput(func() error {
		return (&vaultAppendCmd{}).Run(a, []string{"crawlly", "--type", "decision", "--title", "Vault is workspace-scoped", "--body", "Each Groot workspace owns its own vault.", "--tags", "vault,design"})
	})
	if err != nil {
		t.Fatalf("vault append returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, "\tdecision\tVault is workspace-scoped\t") {
		t.Fatalf("unexpected append output: %q", stdout)
	}
	firstNodeID := strings.SplitN(strings.TrimSpace(stdout), "\t", 2)[0]

	stdout, stderr, err = captureCommandOutput(func() error {
		return (&vaultAppendCmd{}).Run(a, []string{"crawlly", "--type", "task", "--title", "Wire edge command", "--body", "Connect the decision to the task."})
	})
	if err != nil {
		t.Fatalf("second vault append returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, "\ttask\tWire edge command\t") {
		t.Fatalf("unexpected second append output: %q", stdout)
	}
	secondNodeID := strings.SplitN(strings.TrimSpace(stdout), "\t", 2)[0]

	stdout, stderr, err = captureCommandOutput(func() error {
		return (&vaultEdgeCmd{}).Run(a, []string{"crawlly", "--from", secondNodeID, "--to", firstNodeID, "--type", "depends_on"})
	})
	if err != nil {
		t.Fatalf("vault edge returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, "\tdepends_on\t"+secondNodeID+"\t"+firstNodeID) {
		t.Fatalf("unexpected edge output: %q", stdout)
	}

	stdout, stderr, err = captureCommandOutput(func() error {
		return (&vaultSearchCmd{}).Run(a, []string{"crawlly", "--limit", "5", "vault"})
	})
	if err != nil {
		t.Fatalf("vault search returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, "[score=") || !strings.Contains(stdout, "Vault is workspace-scoped") {
		t.Fatalf("unexpected search output: %q", stdout)
	}

	stdout, stderr, err = captureCommandOutput(func() error {
		return (&vaultRecentCmd{}).Run(a, []string{"crawlly", "--limit", "5"})
	})
	if err != nil {
		t.Fatalf("vault recent returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, "Vault is workspace-scoped") {
		t.Fatalf("unexpected recent output: %q", stdout)
	}

	stdout, stderr, err = captureCommandOutput(func() error {
		return (&vaultStatsCmd{}).Run(a, []string{"crawlly"})
	})
	if err != nil {
		t.Fatalf("vault stats returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, "Nodes: 2") || !strings.Contains(stdout, "Edges: 1") || !strings.Contains(stdout, "Changes: 3") {
		t.Fatalf("unexpected stats output: %q", stdout)
	}
}

func TestVaultInitCreatesExpectedFiles(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	if err := (&vaultInitCmd{}).Run(a, []string{"crawlly"}); err != nil {
		t.Fatalf("vault init returned error: %v", err)
	}

	wsPath, err := a.EnsureWorkspace("crawlly")
	if err != nil {
		t.Fatalf("EnsureWorkspace returned error: %v", err)
	}
	for _, path := range []string{
		filepath.Join(wsPath, "vault", "nodes.jsonl"),
		filepath.Join(wsPath, "vault", "edges.jsonl"),
		filepath.Join(wsPath, "vault", "changes.jsonl"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %q to exist: %v", path, err)
		}
	}
}
