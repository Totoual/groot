package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInitVaultCreatesWorkspaceScopedFiles(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	if err := app.InitVault("crawlly"); err != nil {
		t.Fatalf("InitVault returned error: %v", err)
	}

	wsPath, err := app.EnsureWorkspace("crawlly")
	if err != nil {
		t.Fatalf("EnsureWorkspace returned error: %v", err)
	}
	for _, path := range []string{
		vaultNodesPath(wsPath),
		vaultEdgesPath(wsPath),
		vaultChangesPath(wsPath),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %q to exist: %v", path, err)
		}
	}
	meta, err := readJSONFile[VaultMetadata](vaultMetaPath(wsPath))
	if err != nil {
		t.Fatalf("readJSONFile vault metadata returned error: %v", err)
	}
	if meta.Workspace != "crawlly" {
		t.Fatalf("workspace = %q, want %q", meta.Workspace, "crawlly")
	}
	if meta.NodeCount != 0 || meta.EdgeCount != 0 || meta.ChangeCount != 0 {
		t.Fatalf("unexpected vault metadata after init: %#v", meta)
	}
	if meta.VaultUpdatedAt.IsZero() {
		t.Fatal("expected vault_updated_at to be set")
	}
}

func TestVaultAppendSearchRecentAndStats(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	first, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeDecision,
		Title: "Vault is workspace-scoped",
		Body:  "Each Groot workspace owns its own vault.",
		Tags:  []string{"vault", "design", "vault"},
	})
	if err != nil {
		t.Fatalf("VaultAppend first returned error: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	second, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeRule,
		Title: "Avoid arbitrary heat spending",
		Body:  "Cards should not allow arbitrary heat spending.",
		Tags:  []string{"rule", "combat"},
	})
	if err != nil {
		t.Fatalf("VaultAppend second returned error: %v", err)
	}

	if first.Source != "human" || first.Confidence != 1.0 || first.Status != "active" {
		t.Fatalf("unexpected defaults on first node: %#v", first)
	}
	if len(first.Tags) != 2 || first.Tags[0] != "design" || first.Tags[1] != "vault" {
		t.Fatalf("unexpected normalized tags: %#v", first.Tags)
	}

	hits, err := app.VaultSearch("crawlly", "vault", VaultSearchOptions{})
	if err != nil {
		t.Fatalf("VaultSearch returned error: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 search hit, got %d", len(hits))
	}
	if hits[0].Node.ID != first.ID {
		t.Fatalf("search hit = %q, want %q", hits[0].Node.ID, first.ID)
	}

	recent, err := app.VaultRecent("crawlly", VaultRecentOptions{Limit: 2})
	if err != nil {
		t.Fatalf("VaultRecent returned error: %v", err)
	}
	if len(recent) != 2 {
		t.Fatalf("expected 2 recent nodes, got %d", len(recent))
	}
	if recent[0].ID != second.ID {
		t.Fatalf("expected newest node first, got %q then %q", recent[0].ID, recent[1].ID)
	}

	stats, err := app.VaultStats("crawlly")
	if err != nil {
		t.Fatalf("VaultStats returned error: %v", err)
	}
	if stats.NodeCount != 2 || stats.EdgeCount != 0 || stats.ChangeCount != 2 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
	if stats.ByType[VaultNodeTypeDecision] != 1 || stats.ByType[VaultNodeTypeRule] != 1 {
		t.Fatalf("unexpected by-type stats: %#v", stats.ByType)
	}
	if stats.ByStatus["active"] != 2 {
		t.Fatalf("unexpected by-status stats: %#v", stats.ByStatus)
	}

	wsPath, err := app.EnsureWorkspace("crawlly")
	if err != nil {
		t.Fatalf("EnsureWorkspace returned error: %v", err)
	}
	changes, err := readJSONLRecords[VaultChange](filepath.Join(wsPath, "vault", "changes.jsonl"))
	if err != nil {
		t.Fatalf("readJSONLRecords changes returned error: %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}

	meta, err := readJSONFile[VaultMetadata](vaultMetaPath(wsPath))
	if err != nil {
		t.Fatalf("readJSONFile vault metadata returned error: %v", err)
	}
	if meta.NodeCount != 2 || meta.EdgeCount != 0 || meta.ChangeCount != 2 {
		t.Fatalf("unexpected vault metadata after append: %#v", meta)
	}
	if meta.VaultUpdatedAt.Before(second.CreatedAt) {
		t.Fatalf("expected vault_updated_at >= second created_at, got %#v", meta)
	}
}

func TestVaultAppendEdgeRecordsEdgeChangeAndStats(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	from, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeTask,
		Title: "Implement vault edges",
		Body:  "Add append-only vault edge support.",
	})
	if err != nil {
		t.Fatalf("VaultAppend from returned error: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	to, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeDecision,
		Title: "Keep edges directional",
		Body:  "Reject ambiguous edge shapes in the first pass.",
	})
	if err != nil {
		t.Fatalf("VaultAppend to returned error: %v", err)
	}

	edge, err := app.VaultAppendEdge("crawlly", VaultEdgeAppendSpec{
		FromID: from.ID,
		ToID:   to.ID,
		Type:   VaultEdgeTypeDependsOn,
	})
	if err != nil {
		t.Fatalf("VaultAppendEdge returned error: %v", err)
	}
	if edge.Type != VaultEdgeTypeDependsOn || edge.FromID != from.ID || edge.ToID != to.ID {
		t.Fatalf("unexpected appended edge: %#v", edge)
	}

	stats, err := app.VaultStats("crawlly")
	if err != nil {
		t.Fatalf("VaultStats returned error: %v", err)
	}
	if stats.NodeCount != 2 || stats.EdgeCount != 1 || stats.ChangeCount != 3 {
		t.Fatalf("unexpected stats after edge append: %#v", stats)
	}

	wsPath, err := app.EnsureWorkspace("crawlly")
	if err != nil {
		t.Fatalf("EnsureWorkspace returned error: %v", err)
	}
	edges, err := readJSONLRecords[VaultEdge](vaultEdgesPath(wsPath))
	if err != nil {
		t.Fatalf("readJSONLRecords edges returned error: %v", err)
	}
	if len(edges) != 1 || edges[0].ID != edge.ID {
		t.Fatalf("unexpected stored edges: %#v", edges)
	}

	changes, err := readJSONLRecords[VaultChange](vaultChangesPath(wsPath))
	if err != nil {
		t.Fatalf("readJSONLRecords changes returned error: %v", err)
	}
	if len(changes) != 3 {
		t.Fatalf("expected 3 changes, got %d", len(changes))
	}
	if changes[2].Kind != "edge.appended" {
		t.Fatalf("expected final change to record edge append, got %#v", changes[2])
	}
	if got := changes[2].Payload["edge_id"]; got != edge.ID {
		t.Fatalf("expected edge_id payload %q, got %#v", edge.ID, got)
	}

	meta, err := readJSONFile[VaultMetadata](vaultMetaPath(wsPath))
	if err != nil {
		t.Fatalf("readJSONFile vault metadata returned error: %v", err)
	}
	if meta.NodeCount != 2 || meta.EdgeCount != 1 || meta.ChangeCount != 3 {
		t.Fatalf("unexpected vault metadata after edge append: %#v", meta)
	}
	if meta.VaultUpdatedAt.Before(edge.CreatedAt) {
		t.Fatalf("expected vault_updated_at >= edge created_at, got %#v", meta)
	}
}

func TestVaultAppendEdgeValidatesInputs(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	from, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeTask,
		Title: "Task",
		Body:  "body",
	})
	if err != nil {
		t.Fatalf("VaultAppend from returned error: %v", err)
	}
	to, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeDecision,
		Title: "Decision",
		Body:  "body",
	})
	if err != nil {
		t.Fatalf("VaultAppend to returned error: %v", err)
	}

	for _, tc := range []struct {
		name string
		spec VaultEdgeAppendSpec
		want string
	}{
		{
			name: "missing from",
			spec: VaultEdgeAppendSpec{ToID: to.ID, Type: VaultEdgeTypeDependsOn},
			want: "from_id required",
		},
		{
			name: "missing to",
			spec: VaultEdgeAppendSpec{FromID: from.ID, Type: VaultEdgeTypeDependsOn},
			want: "to_id required",
		},
		{
			name: "same node",
			spec: VaultEdgeAppendSpec{FromID: from.ID, ToID: from.ID, Type: VaultEdgeTypeDependsOn},
			want: "distinct from_id and to_id",
		},
		{
			name: "missing node",
			spec: VaultEdgeAppendSpec{FromID: from.ID, ToID: "node-missing", Type: VaultEdgeTypeDependsOn},
			want: `to_id "node-missing" not found`,
		},
		{
			name: "unsupported type",
			spec: VaultEdgeAppendSpec{FromID: from.ID, ToID: to.ID, Type: "related_to"},
			want: `unsupported vault edge type "related_to"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := app.VaultAppendEdge("crawlly", tc.spec)
			if err == nil {
				t.Fatal("expected VaultAppendEdge to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}

	if _, err := app.VaultAppendEdge("crawlly", VaultEdgeAppendSpec{
		FromID: from.ID,
		ToID:   to.ID,
		Type:   VaultEdgeTypeDependsOn,
	}); err != nil {
		t.Fatalf("VaultAppendEdge first returned error: %v", err)
	}
	_, err = app.VaultAppendEdge("crawlly", VaultEdgeAppendSpec{
		FromID: from.ID,
		ToID:   to.ID,
		Type:   VaultEdgeTypeDependsOn,
	})
	if err == nil {
		t.Fatal("expected duplicate edge append to fail")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("duplicate edge error = %q, want duplicate message", err.Error())
	}
}

func TestVaultRecentReturnsEmptyForNewWorkspace(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	nodes, err := app.VaultRecent("crawlly", VaultRecentOptions{Limit: 5})
	if err != nil {
		t.Fatalf("VaultRecent returned error: %v", err)
	}
	if len(nodes) != 0 {
		t.Fatalf("expected no recent nodes, got %#v", nodes)
	}
}
