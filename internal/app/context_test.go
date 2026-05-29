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
		Tags:  []string{"engine", "deterministic"},
	}); err != nil {
		t.Fatalf("VaultAppend first returned error: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	if _, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeRule,
		Title: "Heat spending stays bounded",
		Body:  "Do not allow arbitrary heat spending.",
		Tags:  []string{"heat"},
	}); err != nil {
		t.Fatalf("VaultAppend second returned error: %v", err)
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

func TestBuildContextPackDeduplicatesRecentAndSuggestedEntries(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	projectPath, _ := setupIndexWorkspace(t, app, root)

	mustWriteFile(t, filepath.Join(projectPath, "engine.go"), "package demo\n\ntype Engine struct{}\n\nfunc (e *Engine) DamagePerHeat() {}\n")
	if _, err := app.UpdateIndex("crawlly"); err != nil {
		t.Fatalf("UpdateIndex returned error: %v", err)
	}

	relevant, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeDecision,
		Title: "Vault is workspace-scoped",
		Body:  "Each workspace owns its own vault.",
		Tags:  []string{"vault"},
	})
	if err != nil {
		t.Fatalf("VaultAppend relevant returned error: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	if _, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeTask,
		Title: "Add vault append command",
		Body:  "Implemented vault append command.",
		Tags:  []string{"task"},
	}); err != nil {
		t.Fatalf("VaultAppend recent returned error: %v", err)
	}

	pack, err := app.BuildContextPack("crawlly", "vault")
	if err != nil {
		t.Fatalf("BuildContextPack returned error: %v", err)
	}

	for _, node := range pack.RecentActivity {
		if node.ID == relevant.ID {
			t.Fatalf("expected relevant node %q to be omitted from recent activity", relevant.ID)
		}
	}
	if len(pack.SuggestedReads) != 0 {
		t.Fatalf("expected suggested reads to omit files already present in relevant files, got %#v", pack.SuggestedReads)
	}
}

func TestBuildContextPackAppliesResultLimits(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	projectPath, _ := setupIndexWorkspace(t, app, root)

	for i := 0; i < 12; i++ {
		mustWriteFile(t, filepath.Join(projectPath, "pkg", "file_"+strings.Repeat("x", i+1)+".go"), "package demo\n\n// heat token\nfunc HeatToken"+strings.Repeat("X", i)+"() {}\n")
	}
	if _, err := app.UpdateIndex("crawlly"); err != nil {
		t.Fatalf("UpdateIndex returned error: %v", err)
	}
	for i := 0; i < 12; i++ {
		if _, err := app.VaultAppend("crawlly", VaultAppendSpec{
			Type:  VaultNodeTypeNote,
			Title: "Heat token note " + strings.Repeat("x", i),
			Body:  "heat token context",
			Tags:  []string{"heat"},
		}); err != nil {
			t.Fatalf("VaultAppend %d returned error: %v", i, err)
		}
		time.Sleep(1 * time.Millisecond)
	}

	pack, err := app.BuildContextPack("crawlly", "heat")
	if err != nil {
		t.Fatalf("BuildContextPack returned error: %v", err)
	}

	if len(pack.Files) > contextFileLimit {
		t.Fatalf("expected at most %d files, got %d", contextFileLimit, len(pack.Files))
	}
	if len(pack.Symbols) > contextSymbolLimit {
		t.Fatalf("expected at most %d symbols, got %d", contextSymbolLimit, len(pack.Symbols))
	}
	if len(pack.VaultEntries) > contextVaultLimit {
		t.Fatalf("expected at most %d vault entries, got %d", contextVaultLimit, len(pack.VaultEntries))
	}
	if len(pack.RecentActivity) > contextRecentLimit {
		t.Fatalf("expected at most %d recent entries, got %d", contextRecentLimit, len(pack.RecentActivity))
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

	pack, err := app.BuildContextPack("crawlly", "nonexistent query")
	if err != nil {
		t.Fatalf("BuildContextPack returned error: %v", err)
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

func TestBuildContextPackCombinesVaultIndexAndSymbols(t *testing.T) {
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
		Tags:  []string{"heat", "cards"},
	}); err != nil {
		t.Fatalf("VaultAppend rule returned error: %v", err)
	}
	if _, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeDecision,
		Title: "Engine logic remains deterministic",
		Body:  "Round resolution stays deterministic.",
		Tags:  []string{"engine"},
	}); err != nil {
		t.Fatalf("VaultAppend decision returned error: %v", err)
	}

	pack, err := app.BuildContextPack("crawlly", "add heat damage per round")
	if err != nil {
		t.Fatalf("BuildContextPack returned error: %v", err)
	}
	markdown := pack.Markdown()
	for _, want := range []string{
		"# Groot Context Pack",
		"Cards should not allow arbitrary heat spending",
		"internal/engine/effects.go",
		"DamagePerHeat (internal/engine/effects.go:8-8)",
		"ResolveRound (internal/engine/effects.go:6-6)",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("expected markdown to contain %q, got:\n%s", want, markdown)
		}
	}
}

func TestBuildContextPackIncludesTaskResumeCandidates(t *testing.T) {
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
		Body:  "CLI query command remains unfinished.",
	})
	if err != nil {
		t.Fatalf("VaultAppend progress returned error: %v", err)
	}
	if _, err := app.VaultAppendEdge("crawlly", VaultEdgeAppendSpec{
		FromID: progress.ID,
		ToID:   task.ID,
		Type:   VaultEdgeTypeForTask,
	}); err != nil {
		t.Fatalf("VaultAppendEdge returned error: %v", err)
	}

	pack, err := app.BuildContextPack("crawlly", "vault relationship queries")
	if err != nil {
		t.Fatalf("BuildContextPack returned error: %v", err)
	}
	if len(pack.TaskResumeCandidates) != 1 {
		t.Fatalf("expected 1 task resume candidate, got %#v", pack.TaskResumeCandidates)
	}
	if pack.TaskResumeCandidates[0].Task.ID != task.ID {
		t.Fatalf("unexpected task candidate: %#v", pack.TaskResumeCandidates[0])
	}
	if pack.TaskResumeCandidates[0].LatestProgress == nil || pack.TaskResumeCandidates[0].LatestProgress.ID != progress.ID {
		t.Fatalf("unexpected latest progress candidate: %#v", pack.TaskResumeCandidates[0])
	}
	markdown := pack.Markdown()
	for _, want := range []string{
		"Task Resume Candidates:",
		"Implement vault relationship queries - Stopped after app and MCP read support",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("expected markdown to contain %q, got:\n%s", want, markdown)
		}
	}
}
