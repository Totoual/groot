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
	for _, want := range []string{"init", "append", "edge", "search", "recent", "resume", "stats"} {
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

func TestVaultAppendAndEdgeCmdsSupportProgressForTask(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	taskOut, taskErr, err := captureCommandOutput(func() error {
		return (&vaultAppendCmd{}).Run(a, []string{"crawlly", "--type", "task", "--title", "Implement vault relationship queries", "--body", "Add deterministic vault edge query support in app and MCP."})
	})
	if err != nil {
		t.Fatalf("vault append task returned error: %v", err)
	}
	if strings.TrimSpace(taskErr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", taskErr)
	}
	if !strings.Contains(taskOut, "\ttask\tImplement vault relationship queries\t") {
		t.Fatalf("unexpected task append output: %q", taskOut)
	}
	taskNodeID := strings.SplitN(strings.TrimSpace(taskOut), "\t", 2)[0]

	progressOut, progressErr, err := captureCommandOutput(func() error {
		return (&vaultAppendCmd{}).Run(a, []string{"crawlly", "--type", "progress", "--title", "Stopped after app and MCP read support", "--body", "CLI query command and context integration remain unfinished."})
	})
	if err != nil {
		t.Fatalf("vault append progress returned error: %v", err)
	}
	if strings.TrimSpace(progressErr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", progressErr)
	}
	if !strings.Contains(progressOut, "\tprogress\tStopped after app and MCP read support\t") {
		t.Fatalf("unexpected progress append output: %q", progressOut)
	}
	progressNodeID := strings.SplitN(strings.TrimSpace(progressOut), "\t", 2)[0]

	edgeOut, edgeErr, err := captureCommandOutput(func() error {
		return (&vaultEdgeCmd{}).Run(a, []string{"crawlly", "--from", progressNodeID, "--to", taskNodeID, "--type", "for_task"})
	})
	if err != nil {
		t.Fatalf("vault edge returned error: %v", err)
	}
	if strings.TrimSpace(edgeErr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", edgeErr)
	}
	if !strings.Contains(edgeOut, "\tfor_task\t"+progressNodeID+"\t"+taskNodeID) {
		t.Fatalf("unexpected progress edge output: %q", edgeOut)
	}
}

func TestVaultResumeCmdOutputsTaskResume(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	task, err := a.VaultAppend("crawlly", app.VaultAppendSpec{
		Type:  app.VaultNodeTypeTask,
		Title: "Implement vault relationship queries",
		Body:  "Add deterministic relationship queries over workspace vault edges.",
	})
	if err != nil {
		t.Fatalf("VaultAppend task returned error: %v", err)
	}
	progress, err := a.VaultAppend("crawlly", app.VaultAppendSpec{
		Type:  app.VaultNodeTypeProgress,
		Title: "Stopped after app and MCP read support",
		Body:  "Remaining work: CLI query command and docs.",
	})
	if err != nil {
		t.Fatalf("VaultAppend progress returned error: %v", err)
	}
	if _, err := a.VaultAppendEdge("crawlly", app.VaultEdgeAppendSpec{
		FromID: progress.ID,
		ToID:   task.ID,
		Type:   app.VaultEdgeTypeForTask,
	}); err != nil {
		t.Fatalf("VaultAppendEdge returned error: %v", err)
	}

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&vaultResumeCmd{}).Run(a, []string{"crawlly", "vault relationship queries"})
	})
	if err != nil {
		t.Fatalf("vault resume returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	for _, want := range []string{
		"# Groot Task Resume",
		"Implement vault relationship queries",
		"Stopped after app and MCP read support",
		"for_task: " + progress.ID + " -> " + task.ID,
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected resume output to contain %q, got:\n%s", want, stdout)
		}
	}
}
