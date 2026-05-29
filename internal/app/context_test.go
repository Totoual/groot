package app

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildContextPackDeterministicOrdering(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	projectPath, _ := setupIndexWorkspace(t, app, root)

	mustWriteFile(t, filepath.Join(projectPath, "cards.go"), "package demo\n\n// damage heat\nfunc DamagePerHeat() {}\n")
	mustWriteFile(t, filepath.Join(projectPath, "engine.go"), "package demo\n\n// damage heat\nfunc ResolveRound() {}\n")
	if _, err := app.UpdateIndex("crawlly"); err != nil {
		t.Fatalf("UpdateIndex returned error: %v", err)
	}
	if _, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeDecision,
		Title: "Engine logic remains deterministic",
		Body:  "Deterministic resolution is required.",
	}); err != nil {
		t.Fatalf("VaultAppend returned error: %v", err)
	}

	first, err := app.BuildContextPack("crawlly", "damage heat")
	if err != nil {
		t.Fatalf("BuildContextPack first returned error: %v", err)
	}
	second, err := app.BuildContextPack("crawlly", "damage heat")
	if err != nil {
		t.Fatalf("BuildContextPack second returned error: %v", err)
	}
	if got, want := first.Markdown(), second.Markdown(); got != want {
		t.Fatalf("expected deterministic markdown output\nfirst:\n%s\nsecond:\n%s", got, want)
	}
}

func TestBuildContextPackNarrowAppliesLimitsAndSkipsDocs(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	projectPath, _ := setupIndexWorkspace(t, app, root)

	for i := 0; i < 6; i++ {
		mustWriteFile(t, filepath.Join(projectPath, "pkg", "file_"+strings.Repeat("x", i+1)+".go"), "package demo\n\n// heat token\nfunc HeatToken"+strings.Repeat("X", i)+"() {}\n")
	}
	mustWriteFile(t, filepath.Join(projectPath, "docs", "heat.md"), "heat token docs\n")
	if _, err := app.UpdateIndex("crawlly"); err != nil {
		t.Fatalf("UpdateIndex returned error: %v", err)
	}
	for i := 0; i < 6; i++ {
		if _, err := app.VaultAppend("crawlly", VaultAppendSpec{
			Type:  VaultNodeTypeNote,
			Title: "Heat note " + strings.Repeat("x", i),
			Body:  "heat token context",
		}); err != nil {
			t.Fatalf("VaultAppend %d returned error: %v", i, err)
		}
		time.Sleep(1 * time.Millisecond)
	}

	pack, err := app.BuildContextPack("crawlly", "heat")
	if err != nil {
		t.Fatalf("BuildContextPack returned error: %v", err)
	}
	if pack.Mode != ContextModeNarrow {
		t.Fatalf("mode = %q, want %q", pack.Mode, ContextModeNarrow)
	}
	if len(pack.Files) > contextNarrowFileLimit {
		t.Fatalf("expected at most %d files, got %d", contextNarrowFileLimit, len(pack.Files))
	}
	if len(pack.Symbols) > contextNarrowSymbolLimit {
		t.Fatalf("expected at most %d symbols, got %d", contextNarrowSymbolLimit, len(pack.Symbols))
	}
	if len(pack.VaultEntries) > contextNarrowVaultLimit {
		t.Fatalf("expected at most %d vault entries, got %d", contextNarrowVaultLimit, len(pack.VaultEntries))
	}
	if len(pack.RecentActivity) != 0 {
		t.Fatalf("expected narrow mode to omit recent activity, got %#v", pack.RecentActivity)
	}
	for _, file := range pack.Files {
		if isContextDocLikePath(file.Path) {
			t.Fatalf("expected narrow mode to skip doc-like path, got %q", file.Path)
		}
	}
}

func TestBuildContextPackHandoffPutsTaskResumeFirst(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	projectPath, _ := setupIndexWorkspace(t, app, root)

	mustWriteFile(t, filepath.Join(projectPath, "internal", "vault", "query.go"), "package vault\n\nfunc VaultQueryEdges() {}\n")
	if _, err := app.UpdateIndex("crawlly"); err != nil {
		t.Fatalf("UpdateIndex returned error: %v", err)
	}
	task, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeTask,
		Title: "Implement vault relationship queries",
		Body:  "Add deterministic relationship queries over workspace vault edges.",
	})
	if err != nil {
		t.Fatalf("VaultAppend task returned error: %v", err)
	}
	progress, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeProgress,
		Title: "Stopped after app and MCP read support",
		Body:  "Remaining work: CLI query command and docs.",
	})
	if err != nil {
		t.Fatalf("VaultAppend progress returned error: %v", err)
	}
	decision, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeDecision,
		Title: "Keep relationship queries node-centric",
		Body:  "Do not expand into graph traversal in the first slice.",
	})
	if err != nil {
		t.Fatalf("VaultAppend decision returned error: %v", err)
	}
	if _, err := app.VaultAppendEdge("crawlly", VaultEdgeAppendSpec{FromID: progress.ID, ToID: task.ID, Type: VaultEdgeTypeForTask}); err != nil {
		t.Fatalf("VaultAppendEdge progress returned error: %v", err)
	}
	if _, err := app.VaultAppendEdge("crawlly", VaultEdgeAppendSpec{FromID: decision.ID, ToID: task.ID, Type: VaultEdgeTypeSupports}); err != nil {
		t.Fatalf("VaultAppendEdge decision returned error: %v", err)
	}

	pack, err := app.BuildContextPackWithOptions("crawlly", "vault relationship queries", ContextBuildOptions{Mode: ContextModeHandoff})
	if err != nil {
		t.Fatalf("BuildContextPackWithOptions returned error: %v", err)
	}
	if pack.TaskResume == nil || pack.TaskResume.Task.ID != task.ID {
		t.Fatalf("expected handoff task resume, got %#v", pack.TaskResume)
	}
	markdown := pack.Markdown()
	taskResumeIdx := strings.Index(markdown, "\nTask Resume:\n")
	filesIdx := strings.Index(markdown, "\nRelevant Files:\n")
	if taskResumeIdx == -1 || filesIdx == -1 || taskResumeIdx > filesIdx {
		t.Fatalf("expected Task Resume section before Relevant Files, got:\n%s", markdown)
	}
	for _, want := range []string{
		"Latest Progress: Stopped after app and MCP read support. Remaining work: CLI query command and docs.",
		"Decisions:\n- Keep relationship queries node-centric. Do not expand into graph traversal in the first slice.",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("expected markdown to contain %q, got:\n%s", want, markdown)
		}
	}
}

func TestBuildContextPackBroadPreservesCurrentSections(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	projectPath, _ := setupIndexWorkspace(t, app, root)

	mustWriteFile(t, filepath.Join(projectPath, "internal", "engine", "effects.go"), `package engine

// heat damage per round
type EffectType struct{}

func ResolveRound() {}

func DamagePerHeat() {}
`)
	mustWriteFile(t, filepath.Join(projectPath, "internal", "engine", "effects_test.go"), "package engine\n")
	if _, err := app.UpdateIndex("crawlly"); err != nil {
		t.Fatalf("UpdateIndex returned error: %v", err)
	}
	if _, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeRule,
		Title: "Cards should not allow arbitrary heat spending",
		Body:  "Heat spending must remain bounded.",
	}); err != nil {
		t.Fatalf("VaultAppend rule returned error: %v", err)
	}
	if _, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeDecision,
		Title: "Engine logic remains deterministic",
		Body:  "Round resolution stays deterministic.",
	}); err != nil {
		t.Fatalf("VaultAppend decision returned error: %v", err)
	}

	pack, err := app.BuildContextPackWithOptions("crawlly", "add heat damage per round", ContextBuildOptions{Mode: ContextModeBroad})
	if err != nil {
		t.Fatalf("BuildContextPackWithOptions returned error: %v", err)
	}
	markdown := pack.Markdown()
	for _, want := range []string{
		"# Groot Context Pack",
		"Relevant Vault Entries:",
		"Relevant Files:",
		"Relevant Symbols:",
		"Recent Vault Activity:",
		"DamagePerHeat (internal/engine/effects.go:8-8)",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("expected markdown to contain %q, got:\n%s", want, markdown)
		}
	}
}

func TestBuildContextPackHandlesEmptyResults(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	projectPath := filepath.Join(root, "repos", "crawlly")
	mustWriteFile(t, filepath.Join(projectPath, "README.md"), "workspace docs\n")
	if err := app.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}
	if _, err := app.UpdateIndex("crawlly"); err != nil {
		t.Fatalf("UpdateIndex returned error: %v", err)
	}

	pack, err := app.BuildContextPackWithOptions("crawlly", "nonexistent query", ContextBuildOptions{Mode: ContextModeBroad})
	if err != nil {
		t.Fatalf("BuildContextPackWithOptions returned error: %v", err)
	}
	markdown := pack.Markdown()
	for _, want := range []string{
		"Relevant Vault Entries:\n- none",
		"Relevant Files:\n- none",
		"Relevant Symbols:\n- none",
		"Recent Vault Activity:\n- none",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("expected markdown to contain %q, got:\n%s", want, markdown)
		}
	}
}
