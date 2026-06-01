package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/totoual/groot/internal/app"
)

func TestStatusCmdPrintsWorkspaceOverview(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := setupPorcelainWorkspace(t, a, root)

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&StatusCmd{}).Run(a, []string{"crawlly"})
	})
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	for _, want := range []string{
		"Workspace: crawlly",
		"Project Path: " + projectPath,
		"Runtime:",
		"Index: fresh",
		"Vault: 4 nodes, 3 edges, 7 changes",
		"Latest Task: Implement vault relationship queries [active]",
		"Latest Progress: Stopped after app and MCP read support [active]",
		"Counts:",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected status output to contain %q, got:\n%s", want, stdout)
		}
	}
}

func TestResumeCmdPrintsLatestTaskAndProgressContext(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	setupPorcelainWorkspace(t, a, root)

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&ResumeCmd{}).Run(a, []string{"crawlly"})
	})
	if err != nil {
		t.Fatalf("resume returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	for _, want := range []string{
		"# Groot Resume",
		"Active Task:",
		"Implement vault relationship queries",
		"Latest Progress:",
		"Stopped after app and MCP read support",
		"Completed Work: App.VaultQueryEdges and MCP vault_edge_query are implemented.",
		"Remaining Work: CLI query command and docs.",
		"Next Recommended Step: Add a CLI query command.",
		"Relevant Files:",
		"internal/app/vault.go",
		"Relevant Symbols:",
		"VaultQueryEdges",
		"Warnings:",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected resume output to contain %q, got:\n%s", want, stdout)
		}
	}
}

func TestResumeCmdOutputIsDeterministic(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	setupPorcelainWorkspace(t, a, root)

	first, firstErr, err := captureCommandOutput(func() error {
		return (&ResumeCmd{}).Run(a, []string{"crawlly"})
	})
	if err != nil {
		t.Fatalf("first resume returned error: %v", err)
	}
	if strings.TrimSpace(firstErr) != "" {
		t.Fatalf("expected first stderr to stay quiet, got %q", firstErr)
	}

	second, secondErr, err := captureCommandOutput(func() error {
		return (&ResumeCmd{}).Run(a, []string{"crawlly"})
	})
	if err != nil {
		t.Fatalf("second resume returned error: %v", err)
	}
	if strings.TrimSpace(secondErr) != "" {
		t.Fatalf("expected second stderr to stay quiet, got %q", secondErr)
	}
	if first != second {
		t.Fatalf("expected deterministic resume output\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestSearchCmdPrintsFilesAndSymbols(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	setupPorcelainWorkspace(t, a, root)

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&SearchCmd{}).Run(a, []string{"crawlly", "vault"})
	})
	if err != nil {
		t.Fatalf("search returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	for _, want := range []string{
		"# Groot Search",
		"Query: vault",
		"Files:",
		"internal/app/vault.go",
		"Symbols:",
		"VaultQueryEdges",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected search output to contain %q, got:\n%s", want, stdout)
		}
	}
}

func TestPorcelainCommandsPrintHelp(t *testing.T) {
	a := app.NewApp(t.TempDir())

	statusOut, _, err := captureCommandOutput(func() error {
		return (&StatusCmd{}).Run(a, []string{"-h"})
	})
	if err != nil {
		t.Fatalf("status help returned error: %v", err)
	}
	if !strings.Contains(statusOut, "usage: groot status <workspace-or-path> [--json]") {
		t.Fatalf("unexpected status help output: %q", statusOut)
	}

	resumeOut, _, err := captureCommandOutput(func() error {
		return (&ResumeCmd{}).Run(a, []string{"-h"})
	})
	if err != nil {
		t.Fatalf("resume help returned error: %v", err)
	}
	if !strings.Contains(resumeOut, "usage: groot resume <workspace>") {
		t.Fatalf("unexpected resume help output: %q", resumeOut)
	}

	searchOut, _, err := captureCommandOutput(func() error {
		return (&SearchCmd{}).Run(a, []string{"-h"})
	})
	if err != nil {
		t.Fatalf("search help returned error: %v", err)
	}
	if !strings.Contains(searchOut, "usage: groot search <workspace> <query>") {
		t.Fatalf("unexpected search help output: %q", searchOut)
	}

	syncOut, _, err := captureCommandOutput(func() error {
		return (&SyncCmd{}).Run(a, []string{"-h"})
	})
	if err != nil {
		t.Fatalf("sync help returned error: %v", err)
	}
	if !strings.Contains(syncOut, "usage: groot sync <workspace>") {
		t.Fatalf("unexpected sync help output: %q", syncOut)
	}
}

func TestSyncCmdUpdatesMissingIndexAndPrintsBeforeAfter(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := setupTaskProject(t, a, root)
	if err := os.MkdirAll(filepath.Join(projectPath, "internal", "app"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "internal", "app", "vault.go"), []byte("package app\n\nfunc VaultQueryEdges() {}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
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
		return (&SyncCmd{}).Run(a, []string{"crawlly"})
	})
	if err != nil {
		t.Fatalf("sync returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	for _, want := range []string{
		"Workspace: crawlly",
		"Project Path: " + projectPath,
		"Index Before: missing_metadata",
		"Index Action: updated",
		"Index After: fresh",
		"Vault: 2 nodes, 1 edges, 3 changes",
		"Latest Task: Implement vault relationship queries [active]",
		"Latest Progress: Stopped after app and MCP read support [active]",
		"Counts:",
		"Files: 0 -> 1",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected sync output to contain %q, got:\n%s", want, stdout)
		}
	}
}

func TestSyncCmdSkipsFreshIndexUpdate(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	setupPorcelainWorkspace(t, a, root)

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&SyncCmd{}).Run(a, []string{"crawlly"})
	})
	if err != nil {
		t.Fatalf("sync returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	for _, want := range []string{
		"Index Before: fresh",
		"Index Action: no update needed",
		"Index After: fresh",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected fresh sync output to contain %q, got:\n%s", want, stdout)
		}
	}
}

func setupPorcelainWorkspace(t *testing.T, a *app.App, root string) string {
	t.Helper()

	projectPath := setupTaskProject(t, a, root)
	for _, dir := range []string{
		filepath.Join(projectPath, "internal", "app"),
		filepath.Join(projectPath, "internal", "mcp"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll returned error: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(projectPath, "internal", "app", "vault.go"), []byte("package app\n\nfunc VaultQueryEdges() {}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile vault.go returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "internal", "mcp", "server.go"), []byte("package mcp\n\nfunc vault_edge_query() {}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile server.go returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "README.md"), []byte("# Crawlly\n"), 0o600); err != nil {
		t.Fatalf("WriteFile README returned error: %v", err)
	}
	if _, err := a.UpdateIndex("crawlly"); err != nil {
		t.Fatalf("UpdateIndex returned error: %v", err)
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
		Body:  "Completed Work: App.VaultQueryEdges and MCP vault_edge_query are implemented.\nRemaining Work: CLI query command and docs.\nNext Recommended Step: Add a CLI query command.",
	})
	if err != nil {
		t.Fatalf("VaultAppend progress returned error: %v", err)
	}
	decision, err := a.VaultAppend("crawlly", app.VaultAppendSpec{
		Type:  app.VaultNodeTypeDecision,
		Title: "Start with app and MCP query support",
		Body:  "Build read-only edge queries first and add user-facing consumers later.",
	})
	if err != nil {
		t.Fatalf("VaultAppend decision returned error: %v", err)
	}
	failure, err := a.VaultAppend("crawlly", app.VaultAppendSpec{
		Type:  app.VaultNodeTypeFailure,
		Title: "Graph semantics can sprawl early",
		Body:  "Keep traversal shallow and deterministic.",
	})
	if err != nil {
		t.Fatalf("VaultAppend failure returned error: %v", err)
	}
	for _, edge := range []app.VaultEdgeAppendSpec{
		{FromID: progress.ID, ToID: task.ID, Type: app.VaultEdgeTypeForTask},
		{FromID: decision.ID, ToID: task.ID, Type: app.VaultEdgeTypeSupports},
		{FromID: failure.ID, ToID: task.ID, Type: app.VaultEdgeTypeSupports},
	} {
		if _, err := a.VaultAppendEdge("crawlly", edge); err != nil {
			t.Fatalf("VaultAppendEdge returned error: %v", err)
		}
	}

	return projectPath
}
