package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInitIndexCreatesWorkspaceScopedFiles(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	if err := app.InitIndex("crawlly"); err != nil {
		t.Fatalf("InitIndex returned error: %v", err)
	}

	wsPath, err := app.EnsureWorkspace("crawlly")
	if err != nil {
		t.Fatalf("EnsureWorkspace returned error: %v", err)
	}
	for _, path := range []string{
		indexFilesPath(wsPath),
		indexSymbolsPath(wsPath),
		indexTermsPath(wsPath),
		indexForwardPath(wsPath),
		indexReversePath(wsPath),
		indexHashesPath(wsPath),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %q to exist: %v", path, err)
		}
	}
}

func TestUpdateIndexUsesBoundProjectPathAndSkipsWorkspaceAndIgnoredDirs(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	projectPath, wsPath := setupIndexWorkspace(t, app, root)

	mustWriteFile(t, filepath.Join(projectPath, "internal", "main.go"), "package main\n\nfunc main() { println(\"projecttoken\") }\n")
	mustWriteFile(t, filepath.Join(projectPath, "vendor", "ignored.go"), "package ignored\nconst VendorOnly = \"vendoronly\"\n")
	mustWriteFile(t, filepath.Join(projectPath, "dist", "bundle.txt"), "distonly\n")
	mustWriteFile(t, filepath.Join(projectPath, "node_modules", "lib", "index.js"), "nodeonly\n")
	mustWriteFile(t, filepath.Join(wsPath, "state", "tasks", "ignored.txt"), "stateonly\n")
	mustWriteFile(t, filepath.Join(wsPath, "logs", "ignored.log"), "logsonly\n")
	mustWriteFile(t, filepath.Join(wsPath, "vault", "nodes.jsonl"), "{\"body\":\"vaultonly\"}\n")
	mustWriteFile(t, filepath.Join(wsPath, "index", "files.jsonl"), "{\"path\":\"indexonly\"}\n")

	stats, err := app.UpdateIndex("crawlly")
	if err != nil {
		t.Fatalf("UpdateIndex returned error: %v", err)
	}
	if stats.FileCount != 1 {
		t.Fatalf("expected 1 indexed file, got %d", stats.FileCount)
	}

	files, err := app.indexFiles("crawlly")
	if err != nil {
		t.Fatalf("indexFiles returned error: %v", err)
	}
	if len(files) != 1 || files[0].Path != "internal/main.go" {
		t.Fatalf("unexpected indexed files: %#v", files)
	}

	for _, query := range []string{"vendoronly", "distonly", "nodeonly", "stateonly", "logsonly", "vaultonly", "indexonly"} {
		hits, err := app.IndexSearch("crawlly", query, IndexSearchOptions{})
		if err != nil {
			t.Fatalf("IndexSearch(%q) returned error: %v", query, err)
		}
		if len(hits) != 0 {
			t.Fatalf("expected no hits for %q, got %#v", query, hits)
		}
	}

	hits, err := app.IndexSearch("crawlly", "projecttoken", IndexSearchOptions{})
	if err != nil {
		t.Fatalf("IndexSearch(projecttoken) returned error: %v", err)
	}
	if len(hits) != 1 || hits[0].File.Path != "internal/main.go" {
		t.Fatalf("unexpected search hits: %#v", hits)
	}

	meta, err := readJSONFile[IndexMetadata](indexMetaPath(wsPath))
	if err != nil {
		t.Fatalf("readJSONFile index metadata returned error: %v", err)
	}
	if meta.Workspace != "crawlly" {
		t.Fatalf("workspace = %q, want %q", meta.Workspace, "crawlly")
	}
	if !meta.Indexed {
		t.Fatalf("expected indexed=true, got %#v", meta)
	}
	if meta.ProjectPath != projectPath {
		t.Fatalf("project_path = %q, want %q", meta.ProjectPath, projectPath)
	}
	if meta.FileCount != 1 || meta.SymbolCount != 2 || meta.TermCount == 0 {
		t.Fatalf("unexpected index metadata: %#v", meta)
	}
	if meta.IndexedAt.IsZero() {
		t.Fatal("expected indexed_at to be set")
	}
}

func TestUpdateIndexSkipsWorkspaceConfiguredIgnoredDirs(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	projectPath, wsPath := setupIndexWorkspace(t, app, root)

	mustWriteFile(t, filepath.Join(projectPath, "internal", "main.go"), "package main\n\nfunc main() { println(\"projecttoken\") }\n")
	mustWriteFile(t, filepath.Join(projectPath, "generated", "assets.txt"), "generatedtoken\n")
	mustWriteFile(t, filepath.Join(projectPath, "tmp", "cache.txt"), "tmptoken\n")

	manifest, err := app.getManifest(wsPath)
	if err != nil {
		t.Fatalf("getManifest returned error: %v", err)
	}
	manifest.Index.Ignore = []string{"generated", " tmp "}
	if err := app.writeManifest(wsPath, manifest); err != nil {
		t.Fatalf("writeManifest returned error: %v", err)
	}

	stats, err := app.UpdateIndex("crawlly")
	if err != nil {
		t.Fatalf("UpdateIndex returned error: %v", err)
	}
	if stats.FileCount != 1 {
		t.Fatalf("expected 1 indexed file, got %d", stats.FileCount)
	}

	files, err := app.indexFiles("crawlly")
	if err != nil {
		t.Fatalf("indexFiles returned error: %v", err)
	}
	if len(files) != 1 || files[0].Path != "internal/main.go" {
		t.Fatalf("unexpected indexed files: %#v", files)
	}

	for _, query := range []string{"generatedtoken", "tmptoken"} {
		hits, err := app.IndexSearch("crawlly", query, IndexSearchOptions{})
		if err != nil {
			t.Fatalf("IndexSearch(%q) returned error: %v", query, err)
		}
		if len(hits) != 0 {
			t.Fatalf("expected no hits for %q, got %#v", query, hits)
		}
	}
}

func TestUpdateIndexExtractsGoSymbols(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	projectPath, _ := setupIndexWorkspace(t, app, root)

	mustWriteFile(t, filepath.Join(projectPath, "engine.go"), `package demo

import "fmt"

type Engine struct{}

func ResolveRound() {}

func (e *Engine) DamagePerHeat() {
	fmt.Println("heat")
}
`)

	if _, err := app.UpdateIndex("crawlly"); err != nil {
		t.Fatalf("UpdateIndex returned error: %v", err)
	}

	symbols, err := app.indexSymbols("crawlly")
	if err != nil {
		t.Fatalf("indexSymbols returned error: %v", err)
	}

	seen := map[string]string{}
	for _, symbol := range symbols {
		seen[symbol.QualifiedName] = symbol.Kind
	}
	for want, kind := range map[string]string{
		"demo":                 "package",
		"fmt":                  "import",
		"Engine":               "type",
		"ResolveRound":         "func",
		"Engine.DamagePerHeat": "method",
	} {
		if seen[want] != kind {
			t.Fatalf("expected symbol %q kind %q, got %#v", want, kind, seen)
		}
	}

	gotLines := map[string][2]int{}
	for _, symbol := range symbols {
		gotLines[symbol.QualifiedName] = [2]int{symbol.LineStart, symbol.LineEnd}
	}
	for want, lines := range map[string][2]int{
		"demo":                 {1, 1},
		"fmt":                  {3, 3},
		"Engine":               {5, 5},
		"ResolveRound":         {7, 7},
		"Engine.DamagePerHeat": {9, 11},
	} {
		if gotLines[want] != lines {
			t.Fatalf("expected symbol %q lines %v, got %v", want, lines, gotLines[want])
		}
	}
}

func TestIndexSearchReturnsMatchingFiles(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	projectPath, _ := setupIndexWorkspace(t, app, root)

	mustWriteFile(t, filepath.Join(projectPath, "docs", "vault.txt"), "vault scope workspace memory\n")
	mustWriteFile(t, filepath.Join(projectPath, "docs", "rules.txt"), "heat spending rules\n")

	if _, err := app.UpdateIndex("crawlly"); err != nil {
		t.Fatalf("UpdateIndex returned error: %v", err)
	}

	hits, err := app.IndexSearch("crawlly", "vault", IndexSearchOptions{})
	if err != nil {
		t.Fatalf("IndexSearch returned error: %v", err)
	}
	if len(hits) != 1 || hits[0].File.Path != "docs/vault.txt" {
		t.Fatalf("unexpected search hits: %#v", hits)
	}
}

func TestIndexSymbolsReturnsMatchingSymbols(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	projectPath, _ := setupIndexWorkspace(t, app, root)

	mustWriteFile(t, filepath.Join(projectPath, "engine.go"), `package demo

type Engine struct{}

func (e *Engine) DamagePerHeat() {}
`)

	if _, err := app.UpdateIndex("crawlly"); err != nil {
		t.Fatalf("UpdateIndex returned error: %v", err)
	}

	hits, err := app.IndexSymbols("crawlly", "DamagePerHeat", IndexSearchOptions{})
	if err != nil {
		t.Fatalf("IndexSymbols returned error: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("expected symbol hits")
	}
	if hits[0].Symbol.QualifiedName != "Engine.DamagePerHeat" {
		t.Fatalf("unexpected top symbol hit: %#v", hits[0])
	}
}

func TestIndexMetadataReturnsEmptyFreshnessWhenMetadataMissing(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	projectPath, _ := setupIndexWorkspace(t, app, root)

	mustWriteFile(t, filepath.Join(projectPath, "notes.txt"), "workspace note\n")
	if err := app.InitIndex("crawlly"); err != nil {
		t.Fatalf("InitIndex returned error: %v", err)
	}

	meta, err := app.IndexMetadata("crawlly")
	if err != nil {
		t.Fatalf("IndexMetadata returned error: %v", err)
	}
	if meta.Indexed {
		t.Fatalf("expected indexed=false, got %#v", meta)
	}
	if !meta.IndexedAt.IsZero() {
		t.Fatalf("expected zero indexed_at, got %#v", meta)
	}
	if meta.Workspace != "crawlly" || meta.ProjectPath != projectPath {
		t.Fatalf("unexpected metadata identity: %#v", meta)
	}
	if meta.FileCount != 0 || meta.SymbolCount != 0 || meta.TermCount != 0 {
		t.Fatalf("expected zero counts without update, got %#v", meta)
	}
}

func TestIndexStatusReturnsMissingMetadataState(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	projectPath, _ := setupIndexWorkspace(t, app, root)

	status, err := app.indexStatusAt("crawlly", time.Unix(200, 0).UTC(), DefaultIndexFreshnessMaxAge)
	if err != nil {
		t.Fatalf("indexStatusAt returned error: %v", err)
	}
	if status.Fresh || !status.Stale || status.Reason != "missing_metadata" {
		t.Fatalf("unexpected status: %#v", status)
	}
	if status.Workspace != "crawlly" || status.ProjectPath != projectPath {
		t.Fatalf("unexpected status identity: %#v", status)
	}
}

func TestIndexStatusReturnsFreshStateWithinMaxAge(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	projectPath, wsPath := setupIndexWorkspace(t, app, root)

	mustWriteFile(t, filepath.Join(projectPath, "notes.txt"), "workspace note\n")
	if _, err := app.UpdateIndex("crawlly"); err != nil {
		t.Fatalf("UpdateIndex returned error: %v", err)
	}

	meta, err := readJSONFile[IndexMetadata](indexMetaPath(wsPath))
	if err != nil {
		t.Fatalf("readJSONFile index metadata returned error: %v", err)
	}
	status, err := app.indexStatusAt("crawlly", meta.IndexedAt.Add(2*time.Hour), 24*time.Hour)
	if err != nil {
		t.Fatalf("indexStatusAt returned error: %v", err)
	}
	if !status.Fresh || status.Stale || status.Reason != "fresh" {
		t.Fatalf("unexpected status: %#v", status)
	}
	if status.AgeSeconds != int64((2*time.Hour)/time.Second) {
		t.Fatalf("unexpected age_seconds: %#v", status)
	}
}

func TestIndexStatusReturnsStaleStateWhenOlderThanMaxAge(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	projectPath, wsPath := setupIndexWorkspace(t, app, root)

	mustWriteFile(t, filepath.Join(projectPath, "notes.txt"), "workspace note\n")
	if _, err := app.UpdateIndex("crawlly"); err != nil {
		t.Fatalf("UpdateIndex returned error: %v", err)
	}

	meta, err := readJSONFile[IndexMetadata](indexMetaPath(wsPath))
	if err != nil {
		t.Fatalf("readJSONFile index metadata returned error: %v", err)
	}
	status, err := app.indexStatusAt("crawlly", meta.IndexedAt.Add(48*time.Hour), 24*time.Hour)
	if err != nil {
		t.Fatalf("indexStatusAt returned error: %v", err)
	}
	if status.Fresh || !status.Stale || status.Reason != "stale" {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func setupIndexWorkspace(t *testing.T, app *App, root string) (string, string) {
	t.Helper()

	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := app.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}
	wsPath, err := app.EnsureWorkspace("crawlly")
	if err != nil {
		t.Fatalf("EnsureWorkspace returned error: %v", err)
	}
	return projectPath, wsPath
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) returned error: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) returned error: %v", path, err)
	}
}

func TestWriteJSONLAtomicReplacesContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index", "files.jsonl")
	if err := writeJSONLAtomic(path, []IndexFileRecord{{ID: 1, Path: "old.txt"}}); err != nil {
		t.Fatalf("writeJSONLAtomic first returned error: %v", err)
	}
	if err := writeJSONLAtomic(path, []IndexFileRecord{{ID: 2, Path: "new.txt"}}); err != nil {
		t.Fatalf("writeJSONLAtomic second returned error: %v", err)
	}

	records, err := readJSONLRecords[IndexFileRecord](path)
	if err != nil {
		t.Fatalf("readJSONLRecords returned error: %v", err)
	}
	if len(records) != 1 || records[0].Path != "new.txt" {
		t.Fatalf("unexpected records after atomic replace: %#v", records)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.Contains(string(data), "old.txt") {
		t.Fatalf("expected old content to be replaced, got %q", string(data))
	}
}
