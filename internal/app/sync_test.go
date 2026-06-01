package app

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSyncWorkspaceUpdatesMissingIndexAndReportsVaultState(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	projectPath, wsPath := setupIndexWorkspace(t, app, root)

	mustWriteFile(t, filepath.Join(projectPath, "internal", "main.go"), "package main\n\nfunc main() { println(\"projecttoken\") }\n")
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
	if _, err := app.VaultAppendEdge("crawlly", VaultEdgeAppendSpec{
		FromID: progress.ID,
		ToID:   task.ID,
		Type:   VaultEdgeTypeForTask,
	}); err != nil {
		t.Fatalf("VaultAppendEdge returned error: %v", err)
	}

	report, err := app.SyncWorkspace("crawlly")
	if err != nil {
		t.Fatalf("SyncWorkspace returned error: %v", err)
	}
	if report.WorkspaceName != "crawlly" || report.ProjectPath != projectPath {
		t.Fatalf("unexpected identity in report: %#v", report)
	}
	if report.Before.Reason != "missing_metadata" {
		t.Fatalf("before.reason = %q, want %q", report.Before.Reason, "missing_metadata")
	}
	if !report.UpdatedIndex {
		t.Fatal("expected sync to update missing index")
	}
	if !report.After.Fresh || report.After.Reason != "fresh" {
		t.Fatalf("expected fresh after sync, got %#v", report.After)
	}
	if report.After.FileCount != 1 || report.After.SymbolCount == 0 || report.After.TermCount == 0 {
		t.Fatalf("unexpected after counts: %#v", report.After)
	}
	if report.LatestTask == nil || report.LatestTask.ID != task.ID {
		t.Fatalf("unexpected latest task: %#v", report.LatestTask)
	}
	if report.LatestProgress == nil || report.LatestProgress.ID != progress.ID {
		t.Fatalf("unexpected latest progress: %#v", report.LatestProgress)
	}
	if report.VaultStats.NodeCount != 2 || report.VaultStats.EdgeCount != 1 || report.VaultStats.ChangeCount != 3 {
		t.Fatalf("unexpected vault stats: %#v", report.VaultStats)
	}

	meta, err := readJSONFile[IndexMetadata](indexMetaPath(wsPath))
	if err != nil {
		t.Fatalf("readJSONFile returned error: %v", err)
	}
	if !meta.Indexed || meta.IndexedAt.IsZero() {
		t.Fatalf("expected index metadata to be written, got %#v", meta)
	}
}

func TestSyncWorkspaceDoesNotUpdateFreshIndex(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	projectPath, wsPath := setupIndexWorkspace(t, app, root)

	mustWriteFile(t, filepath.Join(projectPath, "internal", "main.go"), "package main\n\nfunc main() { println(\"projecttoken\") }\n")
	if _, err := app.UpdateIndex("crawlly"); err != nil {
		t.Fatalf("UpdateIndex returned error: %v", err)
	}
	beforeMeta, err := readJSONFile[IndexMetadata](indexMetaPath(wsPath))
	if err != nil {
		t.Fatalf("readJSONFile before returned error: %v", err)
	}

	report, err := app.SyncWorkspace("crawlly")
	if err != nil {
		t.Fatalf("SyncWorkspace returned error: %v", err)
	}
	if report.ProjectPath != projectPath {
		t.Fatalf("report.ProjectPath = %q, want %q", report.ProjectPath, projectPath)
	}
	if report.UpdatedIndex {
		t.Fatal("expected sync to skip fresh index")
	}
	if report.Before.Reason != "fresh" || report.After.Reason != "fresh" {
		t.Fatalf("expected fresh before/after, got before=%#v after=%#v", report.Before, report.After)
	}
	if !report.Before.IndexedAt.Equal(report.After.IndexedAt) {
		t.Fatalf("expected indexed_at to stay stable, got before=%s after=%s", report.Before.IndexedAt, report.After.IndexedAt)
	}
	afterMeta, err := readJSONFile[IndexMetadata](indexMetaPath(wsPath))
	if err != nil {
		t.Fatalf("readJSONFile after returned error: %v", err)
	}
	if !beforeMeta.IndexedAt.Equal(afterMeta.IndexedAt) {
		t.Fatalf("expected metadata indexed_at to stay stable, got before=%s after=%s", beforeMeta.IndexedAt, afterMeta.IndexedAt)
	}
}

func TestSyncWorkspaceUpdatesStaleIndex(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	projectPath, wsPath := setupIndexWorkspace(t, app, root)

	mustWriteFile(t, filepath.Join(projectPath, "internal", "main.go"), "package main\n\nfunc main() { println(\"projecttoken\") }\n")
	if _, err := app.UpdateIndex("crawlly"); err != nil {
		t.Fatalf("UpdateIndex returned error: %v", err)
	}
	stats, err := app.IndexStats("crawlly")
	if err != nil {
		t.Fatalf("IndexStats returned error: %v", err)
	}
	staleAt := time.Now().UTC().Add(-48 * time.Hour)
	if err := app.writeIndexMetadata("crawlly", wsPath, staleAt, stats); err != nil {
		t.Fatalf("writeIndexMetadata returned error: %v", err)
	}

	report, err := app.SyncWorkspace("crawlly")
	if err != nil {
		t.Fatalf("SyncWorkspace returned error: %v", err)
	}
	if report.Before.Reason != "stale" {
		t.Fatalf("before.reason = %q, want %q", report.Before.Reason, "stale")
	}
	if !report.UpdatedIndex {
		t.Fatal("expected stale index to be refreshed")
	}
	if !report.After.Fresh || !report.After.IndexedAt.After(staleAt) {
		t.Fatalf("expected refreshed index after sync, got %#v", report.After)
	}
}
