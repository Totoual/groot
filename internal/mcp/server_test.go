package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/totoual/groot/internal/app"
)

func TestServerHandleInitializeAndListTools(t *testing.T) {
	server := NewServer(app.NewApp(t.TempDir()))

	response, err := server.HandleMessage([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`))
	if err != nil {
		t.Fatalf("HandleMessage initialize returned error: %v", err)
	}

	var initResponse struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
			Capabilities    struct {
				Resources map[string]any `json:"resources"`
			} `json:"capabilities"`
			ServerInfo struct {
				Name string `json:"name"`
			} `json:"serverInfo"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &initResponse); err != nil {
		t.Fatalf("Unmarshal initialize response returned error: %v", err)
	}
	if initResponse.Result.ProtocolVersion != ProtocolVersion {
		t.Fatalf("protocolVersion = %q, want %q", initResponse.Result.ProtocolVersion, ProtocolVersion)
	}
	if initResponse.Result.Capabilities.Resources == nil {
		t.Fatal("expected resources capability to be advertised")
	}
	if initResponse.Result.ServerInfo.Name != "groot" {
		t.Fatalf("serverInfo.name = %q, want %q", initResponse.Result.ServerInfo.Name, "groot")
	}

	response, err = server.HandleMessage([]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`))
	if err != nil {
		t.Fatalf("HandleMessage tools/list returned error: %v", err)
	}

	var listResponse struct {
		Result struct {
			Tools []struct {
				Name         string         `json:"name"`
				OutputSchema map[string]any `json:"outputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &listResponse); err != nil {
		t.Fatalf("Unmarshal tools/list response returned error: %v", err)
	}
	if len(listResponse.Result.Tools) != 40 {
		t.Fatalf("len(tools) = %d, want %d", len(listResponse.Result.Tools), 40)
	}
	names := make([]string, 0, len(listResponse.Result.Tools))
	for _, tool := range listResponse.Result.Tools {
		names = append(names, tool.Name)
	}
	for _, want := range []string{"task_start", "task_declare", "task_delete", "task_list_declared", "task_status", "task_list", "task_logs", "task_stop", "service_start", "service_restart", "service_declare", "service_delete", "service_list_declared", "service_status", "service_list", "service_logs", "service_stop", "event_list", "index_update", "index_stats", "index_search", "index_symbols", "vault_init", "vault_recent", "vault_search", "vault_append", "vault_edge_append", "vault_edge_query", "vault_task_resume", "context_build"} {
		if !slicesContainsString(names, want) {
			t.Fatalf("missing tool %q in %#v", want, names)
		}
	}
	for _, want := range []struct {
		name     string
		required []string
	}{
		{name: "index_update", required: []string{"created", "stats", "meta", "status"}},
		{name: "index_stats", required: []string{"created", "stats", "meta", "status"}},
	} {
		schema := map[string]any(nil)
		for _, tool := range listResponse.Result.Tools {
			if tool.Name == want.name {
				schema = tool.OutputSchema
				break
			}
		}
		if schema == nil {
			t.Fatalf("missing output schema for %q", want.name)
		}
		required, ok := schema["required"].([]any)
		if !ok {
			t.Fatalf("expected required array in %q output schema, got %#v", want.name, schema)
		}
		for _, field := range want.required {
			if !slicesContainsAnyString(required, field) {
				t.Fatalf("expected %q output schema to require %q, got %#v", want.name, field, required)
			}
		}
	}
}

func TestServerHandleShutdownReturnsEmptyObject(t *testing.T) {
	server := NewServer(app.NewApp(t.TempDir()))

	response, err := server.HandleMessage([]byte(`{"jsonrpc":"2.0","id":1,"method":"shutdown"}`))
	if err != nil {
		t.Fatalf("HandleMessage shutdown returned error: %v", err)
	}

	var rpc struct {
		Result map[string]any `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal shutdown response returned error: %v", err)
	}
	if rpc.Result == nil || len(rpc.Result) != 0 {
		t.Fatalf("unexpected shutdown result: %#v", rpc.Result)
	}
}

func TestServerIndexUpdateToolReturnsStructuredContent(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := setupMCPIndexedWorkspace(t, a, root)

	server := NewServer(a)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"index_update","arguments":{"path":"` + projectPath + `"}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage index_update returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Created bool `json:"created"`
				Stats   struct {
					FileCount   int `json:"file_count"`
					SymbolCount int `json:"symbol_count"`
				} `json:"stats"`
				Meta struct {
					Indexed     bool   `json:"indexed"`
					IndexedAt   string `json:"indexed_at"`
					Workspace   string `json:"workspace"`
					ProjectPath string `json:"project_path"`
				} `json:"meta"`
				Status struct {
					Fresh      bool   `json:"fresh"`
					Stale      bool   `json:"stale"`
					Reason     string `json:"reason"`
					IndexedAt  string `json:"indexed_at"`
					Workspace  string `json:"workspace"`
					AgeSeconds int64  `json:"age_seconds"`
				} `json:"status"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal index_update returned error: %v", err)
	}
	if rpc.Result.IsError {
		t.Fatal("expected index_update success result")
	}
	if rpc.Result.StructuredContent.Stats.FileCount == 0 {
		t.Fatal("expected index_update to return indexed files")
	}
	if rpc.Result.StructuredContent.Stats.SymbolCount == 0 {
		t.Fatal("expected index_update to return extracted symbols")
	}
	if !rpc.Result.StructuredContent.Meta.Indexed || rpc.Result.StructuredContent.Meta.IndexedAt == "" {
		t.Fatalf("expected index_update to return metadata, got %#v", rpc.Result.StructuredContent.Meta)
	}
	if rpc.Result.StructuredContent.Meta.Workspace != "crawlly" || rpc.Result.StructuredContent.Meta.ProjectPath != projectPath {
		t.Fatalf("unexpected index_update meta identity: %#v", rpc.Result.StructuredContent.Meta)
	}
	if !rpc.Result.StructuredContent.Status.Fresh || rpc.Result.StructuredContent.Status.Stale || rpc.Result.StructuredContent.Status.Reason != "fresh" {
		t.Fatalf("expected index_update to return fresh status, got %#v", rpc.Result.StructuredContent.Status)
	}
	if rpc.Result.StructuredContent.Status.Workspace != "crawlly" || rpc.Result.StructuredContent.Status.IndexedAt == "" {
		t.Fatalf("unexpected index_update status identity: %#v", rpc.Result.StructuredContent.Status)
	}
}

func TestServerIndexStatsToolReturnsStructuredContent(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := setupMCPIndexedWorkspace(t, a, root)

	server := NewServer(a)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"index_stats","arguments":{"path":"` + projectPath + `"}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage index_stats returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Stats struct {
					FileCount   int `json:"file_count"`
					SymbolCount int `json:"symbol_count"`
					TermCount   int `json:"term_count"`
				} `json:"stats"`
				Meta struct {
					Indexed     bool   `json:"indexed"`
					IndexedAt   string `json:"indexed_at"`
					Workspace   string `json:"workspace"`
					ProjectPath string `json:"project_path"`
					FileCount   int    `json:"file_count"`
					SymbolCount int    `json:"symbol_count"`
					TermCount   int    `json:"term_count"`
				} `json:"meta"`
				Status struct {
					Fresh         bool   `json:"fresh"`
					Stale         bool   `json:"stale"`
					Reason        string `json:"reason"`
					IndexedAt     string `json:"indexed_at"`
					Workspace     string `json:"workspace"`
					ProjectPath   string `json:"project_path"`
					FileCount     int    `json:"file_count"`
					SymbolCount   int    `json:"symbol_count"`
					TermCount     int    `json:"term_count"`
					MaxAgeSeconds int64  `json:"max_age_seconds"`
				} `json:"status"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal index_stats returned error: %v", err)
	}
	if rpc.Result.IsError {
		t.Fatal("expected index_stats success result")
	}
	if rpc.Result.StructuredContent.Stats.FileCount == 0 || rpc.Result.StructuredContent.Stats.SymbolCount == 0 || rpc.Result.StructuredContent.Stats.TermCount == 0 {
		t.Fatalf("unexpected index_stats result: %#v", rpc.Result.StructuredContent.Stats)
	}
	if !rpc.Result.StructuredContent.Meta.Indexed || rpc.Result.StructuredContent.Meta.IndexedAt == "" {
		t.Fatalf("expected freshness metadata in index_stats, got %#v", rpc.Result.StructuredContent.Meta)
	}
	if rpc.Result.StructuredContent.Meta.Workspace != "crawlly" || rpc.Result.StructuredContent.Meta.ProjectPath != projectPath {
		t.Fatalf("unexpected index_stats meta identity: %#v", rpc.Result.StructuredContent.Meta)
	}
	if rpc.Result.StructuredContent.Meta.FileCount != rpc.Result.StructuredContent.Stats.FileCount ||
		rpc.Result.StructuredContent.Meta.SymbolCount != rpc.Result.StructuredContent.Stats.SymbolCount ||
		rpc.Result.StructuredContent.Meta.TermCount != rpc.Result.StructuredContent.Stats.TermCount {
		t.Fatalf("expected meta counts to match stats, got meta=%#v stats=%#v", rpc.Result.StructuredContent.Meta, rpc.Result.StructuredContent.Stats)
	}
	if !rpc.Result.StructuredContent.Status.Fresh || rpc.Result.StructuredContent.Status.Stale || rpc.Result.StructuredContent.Status.Reason != "fresh" {
		t.Fatalf("expected fresh index status, got %#v", rpc.Result.StructuredContent.Status)
	}
	if rpc.Result.StructuredContent.Status.IndexedAt != rpc.Result.StructuredContent.Meta.IndexedAt {
		t.Fatalf("expected status indexed_at to match meta indexed_at, got status=%#v meta=%#v", rpc.Result.StructuredContent.Status, rpc.Result.StructuredContent.Meta)
	}
	if rpc.Result.StructuredContent.Status.Workspace != rpc.Result.StructuredContent.Meta.Workspace ||
		rpc.Result.StructuredContent.Status.ProjectPath != rpc.Result.StructuredContent.Meta.ProjectPath {
		t.Fatalf("expected status identity to match meta, got status=%#v meta=%#v", rpc.Result.StructuredContent.Status, rpc.Result.StructuredContent.Meta)
	}
	if rpc.Result.StructuredContent.Status.FileCount != rpc.Result.StructuredContent.Meta.FileCount ||
		rpc.Result.StructuredContent.Status.SymbolCount != rpc.Result.StructuredContent.Meta.SymbolCount ||
		rpc.Result.StructuredContent.Status.TermCount != rpc.Result.StructuredContent.Meta.TermCount {
		t.Fatalf("expected status counts to match meta, got status=%#v meta=%#v", rpc.Result.StructuredContent.Status, rpc.Result.StructuredContent.Meta)
	}
	if rpc.Result.StructuredContent.Status.MaxAgeSeconds <= 0 {
		t.Fatalf("expected positive max_age_seconds, got %#v", rpc.Result.StructuredContent.Status)
	}
}

func TestServerIndexSearchAndSymbolsToolsReturnStructuredContent(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := setupMCPIndexedWorkspace(t, a, root)

	server := NewServer(a)
	searchRequest := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"index_search","arguments":{"path":"` + projectPath + `","query":"vault"}}}`
	response, err := server.HandleMessage([]byte(searchRequest))
	if err != nil {
		t.Fatalf("HandleMessage index_search returned error: %v", err)
	}

	var searchRPC struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Created bool `json:"created"`
				Files   []struct {
					File struct {
						Path string `json:"path"`
					} `json:"file"`
				} `json:"files"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &searchRPC); err != nil {
		t.Fatalf("Unmarshal index_search returned error: %v", err)
	}
	if searchRPC.Result.IsError {
		t.Fatal("expected index_search success result")
	}
	if len(searchRPC.Result.StructuredContent.Files) == 0 {
		t.Fatal("expected index_search to return files")
	}
	foundNotes := false
	for _, file := range searchRPC.Result.StructuredContent.Files {
		if file.File.Path == "notes.txt" {
			foundNotes = true
			break
		}
	}
	if !foundNotes {
		t.Fatalf("unexpected index_search files: %#v", searchRPC.Result.StructuredContent.Files)
	}

	symbolsRequest := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"index_symbols","arguments":{"path":"` + projectPath + `","query":"DamagePerHeat"}}}`
	response, err = server.HandleMessage([]byte(symbolsRequest))
	if err != nil {
		t.Fatalf("HandleMessage index_symbols returned error: %v", err)
	}

	var symbolsRPC struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Symbols []struct {
					Symbol struct {
						QualifiedName string `json:"qualified_name"`
					} `json:"symbol"`
				} `json:"symbols"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &symbolsRPC); err != nil {
		t.Fatalf("Unmarshal index_symbols returned error: %v", err)
	}
	if symbolsRPC.Result.IsError {
		t.Fatal("expected index_symbols success result")
	}
	if len(symbolsRPC.Result.StructuredContent.Symbols) == 0 || symbolsRPC.Result.StructuredContent.Symbols[0].Symbol.QualifiedName != "Engine.DamagePerHeat" {
		t.Fatalf("unexpected index_symbols result: %#v", symbolsRPC.Result.StructuredContent.Symbols)
	}
}

func TestServerVaultSearchAndAppendToolsReturnStructuredContent(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := setupMCPIndexedWorkspace(t, a, root)

	server := NewServer(a)
	appendRequest := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"vault_append","arguments":{"path":"` + projectPath + `","type":"decision","title":"Vault is workspace-scoped","body":"Each workspace owns its own vault.","tags":["vault","design"]}}}`
	response, err := server.HandleMessage([]byte(appendRequest))
	if err != nil {
		t.Fatalf("HandleMessage vault_append returned error: %v", err)
	}

	var appendRPC struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Node struct {
					Type  string   `json:"type"`
					Title string   `json:"title"`
					Tags  []string `json:"tags"`
				} `json:"node"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &appendRPC); err != nil {
		t.Fatalf("Unmarshal vault_append returned error: %v", err)
	}
	if appendRPC.Result.IsError {
		t.Fatal("expected vault_append success result")
	}
	if appendRPC.Result.StructuredContent.Node.Type != "decision" || appendRPC.Result.StructuredContent.Node.Title != "Vault is workspace-scoped" {
		t.Fatalf("unexpected appended node: %#v", appendRPC.Result.StructuredContent.Node)
	}

	searchRequest := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"vault_search","arguments":{"path":"` + projectPath + `","query":"vault"}}}`
	response, err = server.HandleMessage([]byte(searchRequest))
	if err != nil {
		t.Fatalf("HandleMessage vault_search returned error: %v", err)
	}

	var searchRPC struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Nodes []struct {
					Node struct {
						Title string `json:"title"`
					} `json:"node"`
				} `json:"nodes"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &searchRPC); err != nil {
		t.Fatalf("Unmarshal vault_search returned error: %v", err)
	}
	if searchRPC.Result.IsError {
		t.Fatal("expected vault_search success result")
	}
	if len(searchRPC.Result.StructuredContent.Nodes) == 0 || searchRPC.Result.StructuredContent.Nodes[0].Node.Title != "Vault is workspace-scoped" {
		t.Fatalf("unexpected vault_search nodes: %#v", searchRPC.Result.StructuredContent.Nodes)
	}
}

func TestServerVaultEdgeAppendToolReturnsStructuredContent(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := setupMCPIndexedWorkspace(t, a, root)

	from, err := a.VaultAppend("crawlly", app.VaultAppendSpec{
		Type:  app.VaultNodeTypeTask,
		Title: "Implement edge tool",
		Body:  "Wire through MCP.",
	})
	if err != nil {
		t.Fatalf("VaultAppend from returned error: %v", err)
	}
	to, err := a.VaultAppend("crawlly", app.VaultAppendSpec{
		Type:  app.VaultNodeTypeDecision,
		Title: "Keep edges minimal",
		Body:  "Append-only and directional.",
	})
	if err != nil {
		t.Fatalf("VaultAppend to returned error: %v", err)
	}

	server := NewServer(a)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"vault_edge_append","arguments":{"path":"` + projectPath + `","from_id":"` + from.ID + `","to_id":"` + to.ID + `","type":"depends_on"}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage vault_edge_append returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Edge struct {
					Type   string `json:"type"`
					FromID string `json:"from_id"`
					ToID   string `json:"to_id"`
				} `json:"edge"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal vault_edge_append returned error: %v", err)
	}
	if rpc.Result.IsError {
		t.Fatal("expected vault_edge_append success result")
	}
	if rpc.Result.StructuredContent.Edge.Type != app.VaultEdgeTypeDependsOn ||
		rpc.Result.StructuredContent.Edge.FromID != from.ID ||
		rpc.Result.StructuredContent.Edge.ToID != to.ID {
		t.Fatalf("unexpected appended edge: %#v", rpc.Result.StructuredContent.Edge)
	}

	stats, err := a.VaultStats("crawlly")
	if err != nil {
		t.Fatalf("VaultStats returned error: %v", err)
	}
	if stats.EdgeCount != 1 || stats.ChangeCount != 3 {
		t.Fatalf("unexpected stats after MCP edge append: %#v", stats)
	}
}

func TestServerVaultAppendAndEdgeSupportProgressForTask(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := setupMCPIndexedWorkspace(t, a, root)

	server := NewServer(a)
	taskRequest := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"vault_append","arguments":{"path":"` + projectPath + `","type":"task","title":"Implement vault relationship queries","body":"Add deterministic vault edge query support in app and MCP."}}}`
	taskResponse, err := server.HandleMessage([]byte(taskRequest))
	if err != nil {
		t.Fatalf("HandleMessage task vault_append returned error: %v", err)
	}

	var taskRPC struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Node struct {
					ID   string `json:"id"`
					Type string `json:"type"`
				} `json:"node"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(taskResponse, &taskRPC); err != nil {
		t.Fatalf("Unmarshal task vault_append returned error: %v", err)
	}
	if taskRPC.Result.IsError || taskRPC.Result.StructuredContent.Node.Type != app.VaultNodeTypeTask {
		t.Fatalf("unexpected task append response: %#v", taskRPC)
	}

	progressRequest := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"vault_append","arguments":{"path":"` + projectPath + `","type":"progress","title":"Stopped after app and MCP read support","body":"CLI query command and context integration remain unfinished."}}}`
	progressResponse, err := server.HandleMessage([]byte(progressRequest))
	if err != nil {
		t.Fatalf("HandleMessage progress vault_append returned error: %v", err)
	}

	var progressRPC struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Node struct {
					ID   string `json:"id"`
					Type string `json:"type"`
				} `json:"node"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(progressResponse, &progressRPC); err != nil {
		t.Fatalf("Unmarshal progress vault_append returned error: %v", err)
	}
	if progressRPC.Result.IsError || progressRPC.Result.StructuredContent.Node.Type != app.VaultNodeTypeProgress {
		t.Fatalf("unexpected progress append response: %#v", progressRPC)
	}

	edgeRequest := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"vault_edge_append","arguments":{"path":"` + projectPath + `","from_id":"` + progressRPC.Result.StructuredContent.Node.ID + `","to_id":"` + taskRPC.Result.StructuredContent.Node.ID + `","type":"for_task"}}}`
	edgeResponse, err := server.HandleMessage([]byte(edgeRequest))
	if err != nil {
		t.Fatalf("HandleMessage progress vault_edge_append returned error: %v", err)
	}

	var edgeRPC struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Edge struct {
					Type   string `json:"type"`
					FromID string `json:"from_id"`
					ToID   string `json:"to_id"`
				} `json:"edge"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(edgeResponse, &edgeRPC); err != nil {
		t.Fatalf("Unmarshal progress vault_edge_append returned error: %v", err)
	}
	if edgeRPC.Result.IsError {
		t.Fatal("expected progress vault_edge_append success result")
	}
	if edgeRPC.Result.StructuredContent.Edge.Type != app.VaultEdgeTypeForTask ||
		edgeRPC.Result.StructuredContent.Edge.FromID != progressRPC.Result.StructuredContent.Node.ID ||
		edgeRPC.Result.StructuredContent.Edge.ToID != taskRPC.Result.StructuredContent.Node.ID {
		t.Fatalf("unexpected progress edge response: %#v", edgeRPC.Result.StructuredContent.Edge)
	}
}

func TestServerVaultEdgeQueryToolReturnsStructuredContent(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := setupMCPIndexedWorkspace(t, a, root)

	task, err := a.VaultAppend("crawlly", app.VaultAppendSpec{
		Type:  app.VaultNodeTypeTask,
		Title: "Task node",
		Body:  "body",
	})
	if err != nil {
		t.Fatalf("VaultAppend task returned error: %v", err)
	}
	decision, err := a.VaultAppend("crawlly", app.VaultAppendSpec{
		Type:  app.VaultNodeTypeDecision,
		Title: "Decision node",
		Body:  "body",
	})
	if err != nil {
		t.Fatalf("VaultAppend decision returned error: %v", err)
	}
	rule, err := a.VaultAppend("crawlly", app.VaultAppendSpec{
		Type:  app.VaultNodeTypeRule,
		Title: "Rule node",
		Body:  "body",
	})
	if err != nil {
		t.Fatalf("VaultAppend rule returned error: %v", err)
	}
	if _, err := a.VaultAppendEdge("crawlly", app.VaultEdgeAppendSpec{
		FromID: task.ID,
		ToID:   decision.ID,
		Type:   app.VaultEdgeTypeDependsOn,
	}); err != nil {
		t.Fatalf("VaultAppendEdge outgoing returned error: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	incoming, err := a.VaultAppendEdge("crawlly", app.VaultEdgeAppendSpec{
		FromID: rule.ID,
		ToID:   task.ID,
		Type:   app.VaultEdgeTypeSupports,
	})
	if err != nil {
		t.Fatalf("VaultAppendEdge incoming returned error: %v", err)
	}

	server := NewServer(a)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"vault_edge_query","arguments":{"path":"` + projectPath + `","node_id":"` + task.ID + `","direction":"incoming","type":"supports","limit":1}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage vault_edge_query returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Edges []struct {
					ID     string `json:"id"`
					Type   string `json:"type"`
					FromID string `json:"from_id"`
					ToID   string `json:"to_id"`
				} `json:"edges"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal vault_edge_query returned error: %v", err)
	}
	if rpc.Result.IsError {
		t.Fatal("expected vault_edge_query success result")
	}
	if len(rpc.Result.StructuredContent.Edges) != 1 {
		t.Fatalf("expected 1 queried edge, got %#v", rpc.Result.StructuredContent.Edges)
	}
	if rpc.Result.StructuredContent.Edges[0].ID != incoming.ID ||
		rpc.Result.StructuredContent.Edges[0].Type != app.VaultEdgeTypeSupports ||
		rpc.Result.StructuredContent.Edges[0].ToID != task.ID {
		t.Fatalf("unexpected queried edge: %#v", rpc.Result.StructuredContent.Edges[0])
	}
}

func TestServerVaultTaskResumeToolReturnsStructuredContent(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := setupMCPIndexedWorkspace(t, a, root)

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
	decision, err := a.VaultAppend("crawlly", app.VaultAppendSpec{
		Type:  app.VaultNodeTypeDecision,
		Title: "Keep relationship queries node-centric",
		Body:  "Do not expand into graph traversal in the first slice.",
	})
	if err != nil {
		t.Fatalf("VaultAppend decision returned error: %v", err)
	}
	if _, err := a.VaultAppendEdge("crawlly", app.VaultEdgeAppendSpec{
		FromID: progress.ID,
		ToID:   task.ID,
		Type:   app.VaultEdgeTypeForTask,
	}); err != nil {
		t.Fatalf("VaultAppendEdge progress returned error: %v", err)
	}
	if _, err := a.VaultAppendEdge("crawlly", app.VaultEdgeAppendSpec{
		FromID: decision.ID,
		ToID:   task.ID,
		Type:   app.VaultEdgeTypeSupports,
	}); err != nil {
		t.Fatalf("VaultAppendEdge decision returned error: %v", err)
	}

	server := NewServer(a)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"vault_task_resume","arguments":{"path":"` + projectPath + `","query":"vault relationship queries"}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage vault_task_resume returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			StructuredContent struct {
				Resume struct {
					Task struct {
						ID    string `json:"id"`
						Title string `json:"title"`
					} `json:"task"`
					LatestProgress *struct {
						ID    string `json:"id"`
						Title string `json:"title"`
					} `json:"latest_progress"`
					Decisions []struct {
						ID string `json:"id"`
					} `json:"decisions"`
				} `json:"resume"`
				Markdown string `json:"markdown"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal vault_task_resume returned error: %v", err)
	}
	if rpc.Result.IsError {
		t.Fatal("expected vault_task_resume success result")
	}
	if rpc.Result.StructuredContent.Resume.Task.ID != task.ID || rpc.Result.StructuredContent.Resume.Task.Title != task.Title {
		t.Fatalf("unexpected resumed task: %#v", rpc.Result.StructuredContent.Resume.Task)
	}
	if rpc.Result.StructuredContent.Resume.LatestProgress == nil || rpc.Result.StructuredContent.Resume.LatestProgress.ID != progress.ID {
		t.Fatalf("unexpected latest progress: %#v", rpc.Result.StructuredContent.Resume.LatestProgress)
	}
	if len(rpc.Result.StructuredContent.Resume.Decisions) != 1 || rpc.Result.StructuredContent.Resume.Decisions[0].ID != decision.ID {
		t.Fatalf("unexpected decisions: %#v", rpc.Result.StructuredContent.Resume.Decisions)
	}
	if len(rpc.Result.Content) == 0 || !strings.Contains(rpc.Result.Content[0].Text, "# Groot Task Resume") {
		t.Fatalf("expected markdown text content, got %#v", rpc.Result.Content)
	}
	if !strings.Contains(rpc.Result.StructuredContent.Markdown, "Stopped after app and MCP read support") {
		t.Fatalf("unexpected task resume markdown: %q", rpc.Result.StructuredContent.Markdown)
	}
}

func TestServerVaultInitToolReturnsStructuredContent(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "project")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll project returned error: %v", err)
	}

	server := NewServer(a)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"vault_init","arguments":{"path":"` + projectPath + `"}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage vault_init returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Created bool `json:"created"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal vault_init returned error: %v", err)
	}
	if rpc.Result.IsError {
		t.Fatal("expected vault_init success result")
	}

	workspaceName, _, err := a.ResolveOrCreateWorkspaceByProjectPath(projectPath)
	if err != nil {
		t.Fatalf("ResolveOrCreateWorkspaceByProjectPath returned error: %v", err)
	}
	for _, name := range []string{"nodes.jsonl", "edges.jsonl", "changes.jsonl"} {
		path := filepath.Join(root, "workspaces", workspaceName, "vault", name)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("Stat %s returned error: %v", path, err)
		}
	}
}

func TestServerVaultRecentToolReturnsStructuredContent(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := setupMCPIndexedWorkspace(t, a, root)
	first, err := a.VaultAppend("crawlly", app.VaultAppendSpec{
		Type:  app.VaultNodeTypeDecision,
		Title: "First vault node",
		Body:  "first body",
	})
	if err != nil {
		t.Fatalf("VaultAppend first returned error: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	second, err := a.VaultAppend("crawlly", app.VaultAppendSpec{
		Type:  app.VaultNodeTypeTask,
		Title: "Second vault node",
		Body:  "second body",
	})
	if err != nil {
		t.Fatalf("VaultAppend second returned error: %v", err)
	}

	server := NewServer(a)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"vault_recent","arguments":{"path":"` + projectPath + `","limit":2}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage vault_recent returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Nodes []struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				} `json:"nodes"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal vault_recent returned error: %v", err)
	}
	if rpc.Result.IsError {
		t.Fatal("expected vault_recent success result")
	}
	if len(rpc.Result.StructuredContent.Nodes) != 2 {
		t.Fatalf("expected 2 recent nodes, got %#v", rpc.Result.StructuredContent.Nodes)
	}
	if rpc.Result.StructuredContent.Nodes[0].ID != second.ID || rpc.Result.StructuredContent.Nodes[1].ID != first.ID {
		t.Fatalf("unexpected vault_recent ordering: %#v", rpc.Result.StructuredContent.Nodes)
	}
}

func TestServerContextBuildToolReturnsMarkdownWithoutChangingOutput(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := setupMCPIndexedWorkspace(t, a, root)
	if _, err := a.VaultAppend("crawlly", app.VaultAppendSpec{
		Type:  app.VaultNodeTypeDecision,
		Title: "Engine logic remains deterministic",
		Body:  "Round resolution stays deterministic.",
		Tags:  []string{"engine"},
	}); err != nil {
		t.Fatalf("VaultAppend returned error: %v", err)
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

	server := NewServer(a)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"context_build","arguments":{"path":"` + projectPath + `","task":"vault damage"}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage context_build returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Markdown string `json:"markdown"`
				Context  struct {
					Task string `json:"task"`
					Mode string `json:"mode"`
				} `json:"context"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal context_build returned error: %v", err)
	}
	if rpc.Result.IsError {
		t.Fatal("expected context_build success result")
	}
	if rpc.Result.StructuredContent.Context.Task != "vault damage" {
		t.Fatalf("task = %q, want %q", rpc.Result.StructuredContent.Context.Task, "vault damage")
	}
	if rpc.Result.StructuredContent.Context.Mode != string(app.ContextModeNarrow) {
		t.Fatalf("mode = %q, want %q", rpc.Result.StructuredContent.Context.Mode, app.ContextModeNarrow)
	}
	for _, want := range []string{"# Groot Context Pack", "Relevant Vault Entries:", "Relevant Files:", "Relevant Symbols:", "Engine.DamagePerHeat (engine.go:6-6)"} {
		if !strings.Contains(rpc.Result.StructuredContent.Markdown, want) {
			t.Fatalf("expected markdown to contain %q, got:\n%s", want, rpc.Result.StructuredContent.Markdown)
		}
	}

	handoffRequest := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"context_build","arguments":{"path":"` + projectPath + `","task":"vault relationship queries","mode":"handoff"}}}`
	response, err = server.HandleMessage([]byte(handoffRequest))
	if err != nil {
		t.Fatalf("HandleMessage handoff context_build returned error: %v", err)
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal handoff context_build returned error: %v", err)
	}
	if rpc.Result.IsError {
		t.Fatal("expected handoff context_build success result")
	}
	if rpc.Result.StructuredContent.Context.Mode != string(app.ContextModeHandoff) {
		t.Fatalf("handoff mode = %q, want %q", rpc.Result.StructuredContent.Context.Mode, app.ContextModeHandoff)
	}
	if !strings.Contains(rpc.Result.StructuredContent.Markdown, "Task Resume:") {
		t.Fatalf("expected handoff markdown to contain Task Resume, got:\n%s", rpc.Result.StructuredContent.Markdown)
	}
}

func TestServerResourcesListReturnsManifestAndMetadataForActiveWorkspace(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "the_grime_tcg")
	if err := os.MkdirAll(filepath.Join(projectPath, "backend"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "backend", "go.mod"), []byte("module example.com/tcg\n\ngo 1.25.4\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	server := NewServer(a)
	activate := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"workspace_activate","arguments":{"path":"` + projectPath + `"}}}`
	if _, err := server.HandleMessage([]byte(activate)); err != nil {
		t.Fatalf("HandleMessage activate returned error: %v", err)
	}

	response, err := server.HandleMessage([]byte(`{"jsonrpc":"2.0","id":2,"method":"resources/list"}`))
	if err != nil {
		t.Fatalf("HandleMessage resources/list returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			Resources []struct {
				URI  string `json:"uri"`
				Name string `json:"name"`
			} `json:"resources"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if len(rpc.Result.Resources) != 2 {
		t.Fatalf("len(resources) = %d, want %d", len(rpc.Result.Resources), 2)
	}
	uris := []string{rpc.Result.Resources[0].URI, rpc.Result.Resources[1].URI}
	if !slicesContainsString(uris, "groot://workspace/the_grime_tcg/manifest") {
		t.Fatalf("missing manifest resource in %#v", uris)
	}
	if !slicesContainsString(uris, "groot://workspace/the_grime_tcg/metadata") {
		t.Fatalf("missing metadata resource in %#v", uris)
	}
}

func TestServerResourcesReadReturnsManifestJSON(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "the_grime_tcg")
	if err := os.MkdirAll(filepath.Join(projectPath, "backend"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := a.CreateNewWorkspace("the_grime_tcg"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("the_grime_tcg", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	server := NewServer(a)
	activate := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"workspace_activate","arguments":{"workspace":"the_grime_tcg"}}}`
	if _, err := server.HandleMessage([]byte(activate)); err != nil {
		t.Fatalf("HandleMessage activate returned error: %v", err)
	}

	request := `{"jsonrpc":"2.0","id":2,"method":"resources/read","params":{"uri":"groot://workspace/the_grime_tcg/manifest"}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage resources/read returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			Contents []struct {
				URI  string `json:"uri"`
				Text string `json:"text"`
			} `json:"contents"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if len(rpc.Result.Contents) != 1 {
		t.Fatalf("len(contents) = %d, want %d", len(rpc.Result.Contents), 1)
	}
	if rpc.Result.Contents[0].URI != "groot://workspace/the_grime_tcg/manifest" {
		t.Fatalf("uri = %q", rpc.Result.Contents[0].URI)
	}

	var manifest app.Manifest
	if err := json.Unmarshal([]byte(rpc.Result.Contents[0].Text), &manifest); err != nil {
		t.Fatalf("Unmarshal manifest text returned error: %v", err)
	}
	if manifest.Name != "the_grime_tcg" {
		t.Fatalf("manifest.name = %q, want %q", manifest.Name, "the_grime_tcg")
	}
}

func TestServerResourcesReadRespectsScope(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	allowedPath := filepath.Join(root, "repos", "crawlly")
	otherPath := filepath.Join(root, "repos", "the_grime_tcg")
	for _, projectPath := range []string{allowedPath, otherPath} {
		if err := os.MkdirAll(projectPath, 0o755); err != nil {
			t.Fatalf("MkdirAll returned error: %v", err)
		}
	}
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", allowedPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}
	if err := a.CreateNewWorkspace("the_grime_tcg"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("the_grime_tcg", otherPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	server := NewScopedServer(a, []string{allowedPath})
	request := `{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"groot://workspace/the_grime_tcg/metadata"}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage resources/read returned error: %v", err)
	}

	var rpc struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if rpc.Error == nil {
		t.Fatal("expected resources/read to return an RPC error for out-of-scope resource")
	}
	if !strings.Contains(rpc.Error.Message, "outside the MCP scope") {
		t.Fatalf("unexpected error message %q", rpc.Error.Message)
	}
}

func TestServerWorkspaceStatusToolReturnsStructuredContent(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "the_grime_tcg")
	if err := os.MkdirAll(filepath.Join(projectPath, "backend"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "backend", "go.mod"), []byte("module example.com/tcg\n\ngo 1.25.4\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	server := NewServer(a)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"workspace_status","arguments":{"path":"` + projectPath + `"}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			StructuredContent struct {
				Created bool `json:"created"`
				Status  struct {
					WorkspaceName string `json:"workspace_name"`
					Status        string `json:"status"`
				} `json:"status"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if !rpc.Result.StructuredContent.Created {
		t.Fatal("expected workspace_status to report created=true on first use")
	}
	if rpc.Result.StructuredContent.Status.WorkspaceName != "the_grime_tcg" {
		t.Fatalf("workspace_name = %q, want %q", rpc.Result.StructuredContent.Status.WorkspaceName, "the_grime_tcg")
	}
	if rpc.Result.StructuredContent.Status.Status != "partial runtime ownership" {
		t.Fatalf("status = %q, want %q", rpc.Result.StructuredContent.Status.Status, "partial runtime ownership")
	}
}

func TestServerWorkspaceActivateToolSetsSessionScopeFromProjectPath(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	crawllyPath := filepath.Join(root, "repos", "crawlly")
	tcgPath := filepath.Join(root, "repos", "the_grime_tcg")
	for _, projectPath := range []string{crawllyPath, tcgPath} {
		if err := os.MkdirAll(filepath.Join(projectPath, "backend"), 0o755); err != nil {
			t.Fatalf("MkdirAll returned error: %v", err)
		}
		if err := os.WriteFile(filepath.Join(projectPath, "backend", "go.mod"), []byte("module example.com/test\n\ngo 1.25.4\n"), 0o600); err != nil {
			t.Fatalf("WriteFile returned error: %v", err)
		}
	}

	server := NewServer(a)
	activate := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"workspace_activate","arguments":{"path":"` + crawllyPath + `"}}}`
	response, err := server.HandleMessage([]byte(activate))
	if err != nil {
		t.Fatalf("HandleMessage activate returned error: %v", err)
	}

	var activateRPC struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				ActiveProject string `json:"active_project"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &activateRPC); err != nil {
		t.Fatalf("Unmarshal activate response returned error: %v", err)
	}
	if activateRPC.Result.IsError {
		t.Fatal("expected workspace_activate success result")
	}
	if activateRPC.Result.StructuredContent.ActiveProject != crawllyPath {
		t.Fatalf("active_project = %q, want %q", activateRPC.Result.StructuredContent.ActiveProject, crawllyPath)
	}

	reject := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"workspace_status","arguments":{"path":"` + tcgPath + `"}}}`
	response, err = server.HandleMessage([]byte(reject))
	if err != nil {
		t.Fatalf("HandleMessage status returned error: %v", err)
	}

	var rejectRPC struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rejectRPC); err != nil {
		t.Fatalf("Unmarshal reject response returned error: %v", err)
	}
	if !rejectRPC.Result.IsError {
		t.Fatal("expected workspace_status to be rejected after activation")
	}
	if len(rejectRPC.Result.Content) == 0 || !strings.Contains(rejectRPC.Result.Content[0].Text, "outside the MCP scope") {
		t.Fatalf("unexpected reject content: %#v", rejectRPC.Result.Content)
	}
}

func TestServerWorkspaceActivateToolCanSwitchProjectsInUnscopedSession(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	crawllyPath := filepath.Join(root, "repos", "crawlly")
	tcgPath := filepath.Join(root, "repos", "the_grime_tcg")
	for _, projectPath := range []string{crawllyPath, tcgPath} {
		if err := os.MkdirAll(filepath.Join(projectPath, "backend"), 0o755); err != nil {
			t.Fatalf("MkdirAll returned error: %v", err)
		}
		if err := os.WriteFile(filepath.Join(projectPath, "backend", "go.mod"), []byte("module example.com/test\n\ngo 1.25.4\n"), 0o600); err != nil {
			t.Fatalf("WriteFile returned error: %v", err)
		}
	}

	server := NewServer(a)
	for _, request := range []string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"workspace_activate","arguments":{"path":"` + crawllyPath + `"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"workspace_activate","arguments":{"path":"` + tcgPath + `"}}}`,
	} {
		response, err := server.HandleMessage([]byte(request))
		if err != nil {
			t.Fatalf("HandleMessage returned error: %v", err)
		}
		var rpc struct {
			Result struct {
				IsError bool `json:"isError"`
			} `json:"result"`
		}
		if err := json.Unmarshal(response, &rpc); err != nil {
			t.Fatalf("Unmarshal returned error: %v", err)
		}
		if rpc.Result.IsError {
			t.Fatalf("expected activation request %q to succeed", request)
		}
	}

	rejectOld := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"workspace_status","arguments":{"path":"` + crawllyPath + `"}}}`
	response, err := server.HandleMessage([]byte(rejectOld))
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}
	var rejectRPC struct {
		Result struct {
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rejectRPC); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if !rejectRPC.Result.IsError {
		t.Fatal("expected previous active project to be rejected after switching activation")
	}

	allowNew := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"workspace_status","arguments":{"path":"` + tcgPath + `"}}}`
	response, err = server.HandleMessage([]byte(allowNew))
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}
	var allowRPC struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Status struct {
					WorkspaceName string `json:"workspace_name"`
				} `json:"status"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &allowRPC); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if allowRPC.Result.IsError {
		t.Fatal("expected current active project to stay allowed after switching activation")
	}
	if allowRPC.Result.StructuredContent.Status.WorkspaceName != "the_grime_tcg" {
		t.Fatalf("workspace_name = %q, want %q", allowRPC.Result.StructuredContent.Status.WorkspaceName, "the_grime_tcg")
	}
}

func TestServerWorkspaceActivateToolSupportsBoundWorkspaceName(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	server := NewServer(a)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"workspace_activate","arguments":{"workspace":"crawlly"}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				ActiveProject string `json:"active_project"`
				WorkspaceName string `json:"workspace_name"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if rpc.Result.IsError {
		t.Fatal("expected workspace_activate success result")
	}
	if rpc.Result.StructuredContent.ActiveProject != projectPath {
		t.Fatalf("active_project = %q, want %q", rpc.Result.StructuredContent.ActiveProject, projectPath)
	}
	if rpc.Result.StructuredContent.WorkspaceName != "crawlly" {
		t.Fatalf("workspace_name = %q, want %q", rpc.Result.StructuredContent.WorkspaceName, "crawlly")
	}
}

func TestScopedServerRejectsProjectPathOutsideAllowedScope(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	allowedPath := filepath.Join(root, "repos", "crawlly")
	otherPath := filepath.Join(root, "repos", "the_grime_tcg")
	for _, projectPath := range []string{allowedPath, otherPath} {
		if err := os.MkdirAll(filepath.Join(projectPath, "backend"), 0o755); err != nil {
			t.Fatalf("MkdirAll returned error: %v", err)
		}
		if err := os.WriteFile(filepath.Join(projectPath, "backend", "go.mod"), []byte("module example.com/test\n\ngo 1.25.4\n"), 0o600); err != nil {
			t.Fatalf("WriteFile returned error: %v", err)
		}
	}

	server := NewScopedServer(a, []string{allowedPath})
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"workspace_status","arguments":{"path":"` + otherPath + `"}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if !rpc.Result.IsError {
		t.Fatal("expected scoped server to reject out-of-scope path")
	}
	if len(rpc.Result.Content) == 0 || !strings.Contains(rpc.Result.Content[0].Text, "outside the MCP scope") {
		t.Fatalf("unexpected error content: %#v", rpc.Result.Content)
	}
}

func TestScopedServerActivateCannotEscapeStartupScope(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	allowedPath := filepath.Join(root, "repos", "crawlly")
	otherPath := filepath.Join(root, "repos", "the_grime_tcg")
	for _, projectPath := range []string{allowedPath, otherPath} {
		if err := os.MkdirAll(projectPath, 0o755); err != nil {
			t.Fatalf("MkdirAll returned error: %v", err)
		}
	}

	server := NewScopedServer(a, []string{allowedPath})
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"workspace_activate","arguments":{"path":"` + otherPath + `"}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if !rpc.Result.IsError {
		t.Fatal("expected workspace_activate to stay inside startup scope")
	}
	if len(rpc.Result.Content) == 0 || !strings.Contains(rpc.Result.Content[0].Text, "outside the MCP scope") {
		t.Fatalf("unexpected error content: %#v", rpc.Result.Content)
	}
}

func TestScopedServerAllowsEquivalentProjectPathWithinScope(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(filepath.Join(projectPath, "backend"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "backend", "go.mod"), []byte("module example.com/crawlly\n\ngo 1.25.4\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	server := NewScopedServer(a, []string{projectPath})
	messyPath := filepath.Join(projectPath, "..", "crawlly")
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"workspace_status","arguments":{"path":"` + messyPath + `"}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Status struct {
					WorkspaceName string `json:"workspace_name"`
				} `json:"status"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if rpc.Result.IsError {
		t.Fatal("expected scoped server to allow equivalent project path")
	}
	if rpc.Result.StructuredContent.Status.WorkspaceName != "crawlly" {
		t.Fatalf("workspace_name = %q, want %q", rpc.Result.StructuredContent.Status.WorkspaceName, "crawlly")
	}
}

func TestServerWorkspaceSetupToolSupportsWarnOnlyOptions(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "the_grime_tcg")
	if err := os.MkdirAll(filepath.Join(projectPath, "backend"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "backend", "go.mod"), []byte("module example.com/tcg\n\ngo 1.25.4\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	server := NewServer(a)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"workspace_setup","arguments":{"path":"` + projectPath + `","attach_detected":false,"install_detected":false}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			StructuredContent struct {
				Plan struct {
					AttachRequested bool `json:"attach_requested"`
					Missing         []struct {
						Name string `json:"name"`
					} `json:"missing"`
				} `json:"plan"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if rpc.Result.StructuredContent.Plan.AttachRequested {
		t.Fatal("expected attach_requested=false in warn-only setup")
	}
	if len(rpc.Result.StructuredContent.Plan.Missing) != 1 || rpc.Result.StructuredContent.Plan.Missing[0].Name != "go" {
		t.Fatalf("unexpected missing toolchains: %#v", rpc.Result.StructuredContent.Plan.Missing)
	}
}

func TestServerWorkspaceExecToolCapturesOutput(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "empty-project")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	t.Setenv("PATH", "/usr/bin:/bin")

	server := NewServer(a)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"workspace_exec","arguments":{"path":"` + projectPath + `","command":"/bin/sh","args":["-c","printf hello"]}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				ExitCode int    `json:"exit_code"`
				Stdout   string `json:"stdout"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if rpc.Result.IsError {
		t.Fatal("expected workspace_exec success result")
	}
	if rpc.Result.StructuredContent.ExitCode != 0 {
		t.Fatalf("exit_code = %d, want %d", rpc.Result.StructuredContent.ExitCode, 0)
	}
	if rpc.Result.StructuredContent.Stdout != "hello" {
		t.Fatalf("stdout = %q, want %q", rpc.Result.StructuredContent.Stdout, "hello")
	}
}

func TestServerWorkspaceInspectToolReturnsManifestAndPaths(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "the_grime_tcg")
	if err := os.MkdirAll(filepath.Join(projectPath, "backend"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "backend", "go.mod"), []byte("module example.com/tcg\n\ngo 1.25.4\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	server := NewServer(a)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"workspace_inspect","arguments":{"path":"` + projectPath + `"}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			StructuredContent struct {
				Created bool `json:"created"`
				Inspect struct {
					WorkspaceName string `json:"workspace_name"`
					WorkspaceDir  string `json:"workspace_dir"`
					ManifestPath  string `json:"manifest_path"`
					Manifest      struct {
						Name        string `json:"name"`
						ProjectPath string `json:"project_path"`
					} `json:"manifest"`
					Runtime struct {
						Status string `json:"status"`
					} `json:"runtime"`
				} `json:"inspect"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if !rpc.Result.StructuredContent.Created {
		t.Fatal("expected workspace_inspect to report created=true on first use")
	}
	if rpc.Result.StructuredContent.Inspect.Manifest.Name != "the_grime_tcg" {
		t.Fatalf("manifest.name = %q, want %q", rpc.Result.StructuredContent.Inspect.Manifest.Name, "the_grime_tcg")
	}
	if rpc.Result.StructuredContent.Inspect.Manifest.ProjectPath != projectPath {
		t.Fatalf("manifest.project_path = %q, want %q", rpc.Result.StructuredContent.Inspect.Manifest.ProjectPath, projectPath)
	}
	if !strings.HasSuffix(rpc.Result.StructuredContent.Inspect.ManifestPath, filepath.Join("the_grime_tcg", "manifest.json")) {
		t.Fatalf("unexpected manifest path: %q", rpc.Result.StructuredContent.Inspect.ManifestPath)
	}
	if rpc.Result.StructuredContent.Inspect.Runtime.Status != "partial runtime ownership" {
		t.Fatalf("runtime.status = %q, want %q", rpc.Result.StructuredContent.Inspect.Runtime.Status, "partial runtime ownership")
	}
}

func TestServerWorkspaceEnvToolReturnsStructuredEnv(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "the_grime_tcg")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	t.Setenv("PATH", "/usr/bin:/bin")
	t.Setenv("SHELL", "/bin/zsh")

	server := NewServer(a)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"workspace_env","arguments":{"path":"` + projectPath + `"}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			StructuredContent struct {
				Created       bool              `json:"created"`
				WorkspaceName string            `json:"workspace_name"`
				WorkDir       string            `json:"workdir"`
				Env           map[string]string `json:"env"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if !rpc.Result.StructuredContent.Created {
		t.Fatal("expected workspace_env to report created=true on first use")
	}
	if rpc.Result.StructuredContent.WorkspaceName != "the_grime_tcg" {
		t.Fatalf("workspace_name = %q, want %q", rpc.Result.StructuredContent.WorkspaceName, "the_grime_tcg")
	}
	if rpc.Result.StructuredContent.WorkDir != projectPath {
		t.Fatalf("workdir = %q, want %q", rpc.Result.StructuredContent.WorkDir, projectPath)
	}
	if rpc.Result.StructuredContent.Env["GROOT_WORKSPACE"] != "the_grime_tcg" {
		t.Fatalf("GROOT_WORKSPACE = %q, want %q", rpc.Result.StructuredContent.Env["GROOT_WORKSPACE"], "the_grime_tcg")
	}
	if rpc.Result.StructuredContent.Env["GROOT_WORKDIR"] != projectPath {
		t.Fatalf("GROOT_WORKDIR = %q, want %q", rpc.Result.StructuredContent.Env["GROOT_WORKDIR"], projectPath)
	}
	if _, ok := rpc.Result.StructuredContent.Env["TERM"]; ok {
		t.Fatalf("expected TERM to be omitted, got %#v", rpc.Result.StructuredContent.Env)
	}
}

func TestServerWorkspaceAttachToolAttachesManifestComponents(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "the_grime_tcg")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	server := NewServer(a)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"workspace_attach","arguments":{"path":"` + projectPath + `","toolchains":["go@1.25.4","node@25.8.1"]}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Attached []struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				} `json:"attached"`
				Status struct {
					Attached []struct {
						Name    string `json:"name"`
						Version string `json:"version"`
					} `json:"attached"`
				} `json:"status"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if rpc.Result.IsError {
		t.Fatal("expected workspace_attach success result")
	}
	if len(rpc.Result.StructuredContent.Attached) != 2 {
		t.Fatalf("len(attached) = %d, want %d", len(rpc.Result.StructuredContent.Attached), 2)
	}
	if len(rpc.Result.StructuredContent.Status.Attached) != 2 {
		t.Fatalf("len(status.attached) = %d, want %d", len(rpc.Result.StructuredContent.Status.Attached), 2)
	}
}

func TestServerWorkspaceInstallToolInstallsAttachedToolchains(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "the_grime_tcg")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := a.CreateNewWorkspace("the_grime_tcg"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("the_grime_tcg", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}
	if err := a.AttachToWorkspace("the_grime_tcg", []string{"go@1.25.4"}); err != nil {
		t.Fatalf("AttachToWorkspace returned error: %v", err)
	}
	binPath := filepath.Join(root, "toolchains", "go", "1.25.4", "go", "bin", "go")
	if err := os.MkdirAll(filepath.Dir(binPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	server := NewServer(a)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"workspace_install","arguments":{"path":"` + projectPath + `"}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Installed []struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				} `json:"installed"`
				Status struct {
					Status string `json:"status"`
				} `json:"status"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if rpc.Result.IsError {
		t.Fatal("expected workspace_install success result")
	}
	if len(rpc.Result.StructuredContent.Installed) != 1 {
		t.Fatalf("len(installed) = %d, want %d", len(rpc.Result.StructuredContent.Installed), 1)
	}
	if rpc.Result.StructuredContent.Status.Status != "runtime owned by Groot" &&
		rpc.Result.StructuredContent.Status.Status != "no runtimes detected" &&
		rpc.Result.StructuredContent.Status.Status != "partial runtime ownership" &&
		rpc.Result.StructuredContent.Status.Status != "workspace runtime available, but no project runtimes detected" {
		t.Fatalf("unexpected status %q", rpc.Result.StructuredContent.Status.Status)
	}
}

func TestServerWorkspaceExportToolReturnsPortableWorkspaceContract(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "the_grime_tcg")
	backendDir := filepath.Join(projectPath, "backend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "go.mod"), []byte("module example.com/tcg\n\ngo 1.25.4\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := a.CreateNewWorkspace("the_grime_tcg"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("the_grime_tcg", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}
	if err := a.AttachToWorkspace("the_grime_tcg", []string{"go@1.25.4"}); err != nil {
		t.Fatalf("AttachToWorkspace returned error: %v", err)
	}
	binPath := filepath.Join(root, "toolchains", "go", "1.25.4", "go", "bin", "go")
	if err := os.MkdirAll(filepath.Dir(binPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	server := NewServer(a)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"workspace_export","arguments":{"path":"` + projectPath + `"}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Export struct {
					SchemaVersion int `json:"schema_version"`
					Workspace     struct {
						Name        string `json:"name"`
						ProjectPath string `json:"project_path"`
						Manifest    struct {
							Name        string `json:"name"`
							ProjectPath string `json:"project_path"`
						} `json:"manifest"`
						Runtime struct {
							Status string `json:"status"`
						} `json:"runtime"`
					} `json:"workspace"`
				} `json:"export"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if rpc.Result.IsError {
		t.Fatal("expected workspace_export success result")
	}
	if rpc.Result.StructuredContent.Export.SchemaVersion != 1 {
		t.Fatalf("export.schema_version = %d, want %d", rpc.Result.StructuredContent.Export.SchemaVersion, 1)
	}
	if rpc.Result.StructuredContent.Export.Workspace.Name != "the_grime_tcg" {
		t.Fatalf("export.workspace.name = %q, want %q", rpc.Result.StructuredContent.Export.Workspace.Name, "the_grime_tcg")
	}
	if rpc.Result.StructuredContent.Export.Workspace.ProjectPath != projectPath {
		t.Fatalf("export.workspace.project_path = %q, want %q", rpc.Result.StructuredContent.Export.Workspace.ProjectPath, projectPath)
	}
	if rpc.Result.StructuredContent.Export.Workspace.Manifest.Name != "the_grime_tcg" {
		t.Fatalf("export.workspace.manifest.name = %q, want %q", rpc.Result.StructuredContent.Export.Workspace.Manifest.Name, "the_grime_tcg")
	}
	if rpc.Result.StructuredContent.Export.Workspace.Manifest.ProjectPath != projectPath {
		t.Fatalf("export.workspace.manifest.project_path = %q, want %q", rpc.Result.StructuredContent.Export.Workspace.Manifest.ProjectPath, projectPath)
	}
	if rpc.Result.StructuredContent.Export.Workspace.Runtime.Status != "runtime owned by Groot" {
		t.Fatalf("export.workspace.runtime.status = %q, want %q", rpc.Result.StructuredContent.Export.Workspace.Runtime.Status, "runtime owned by Groot")
	}
}

func TestServerWorkspaceImportToolImportsPortableWorkspaceContract(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "crawlly")
	backendDir := filepath.Join(projectPath, "backend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "go.mod"), []byte("module example.com/crawlly\n\ngo 1.25.4\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	exported := app.WorkspaceExport{
		SchemaVersion: 1,
		Workspace: app.WorkspaceExportPayload{
			Name: "crawlly",
			Manifest: app.Manifest{
				SchemaVersion: 1,
				Name:          "crawlly",
				Packages:      []app.Component{{Name: "go", Version: "1.25.4"}},
			},
		},
	}
	exportJSON, err := json.Marshal(exported)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	server := NewServer(a)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"workspace_import","arguments":{"path":"` + projectPath + `","export":` + string(exportJSON) + `}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Created       bool   `json:"created"`
				WorkspaceName string `json:"workspace_name"`
				ProjectPath   string `json:"project_path"`
				Status        struct {
					WorkspaceName string `json:"workspace_name"`
				} `json:"status"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if rpc.Result.IsError {
		t.Fatal("expected workspace_import success result")
	}
	if !rpc.Result.StructuredContent.Created {
		t.Fatal("expected workspace_import to create the workspace")
	}
	if rpc.Result.StructuredContent.WorkspaceName != "crawlly" {
		t.Fatalf("workspace_name = %q, want %q", rpc.Result.StructuredContent.WorkspaceName, "crawlly")
	}
	if rpc.Result.StructuredContent.ProjectPath != projectPath {
		t.Fatalf("project_path = %q, want %q", rpc.Result.StructuredContent.ProjectPath, projectPath)
	}
}

func TestServerWorkspaceImportToolSupportsWorkspaceNameOverride(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	existingPath := filepath.Join(root, "repos", "existing")
	importPath := filepath.Join(root, "repos", "imported")
	for _, projectPath := range []string{existingPath, importPath} {
		if err := os.MkdirAll(projectPath, 0o755); err != nil {
			t.Fatalf("MkdirAll returned error: %v", err)
		}
	}
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", existingPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	exported := app.WorkspaceExport{
		SchemaVersion: 1,
		Workspace: app.WorkspaceExportPayload{
			Name: "crawlly",
			Manifest: app.Manifest{
				SchemaVersion: 1,
				Name:          "crawlly",
			},
		},
	}
	exportJSON, err := json.Marshal(exported)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	server := NewServer(a)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"workspace_import","arguments":{"path":"` + importPath + `","workspace_name":"crawlly-imported","export":` + string(exportJSON) + `}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}

	var rpc struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				WorkspaceName string `json:"workspace_name"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if rpc.Result.IsError {
		t.Fatal("expected workspace_import success result")
	}
	if rpc.Result.StructuredContent.WorkspaceName != "crawlly-imported" {
		t.Fatalf("workspace_name = %q, want %q", rpc.Result.StructuredContent.WorkspaceName, "crawlly-imported")
	}
}

func TestWorkspaceExportFromArgAcceptsLegacyPayloadShape(t *testing.T) {
	exported, err := workspaceExportFromArg(map[string]any{
		"name":         "crawlly",
		"project_path": "/tmp/crawlly",
		"manifest": map[string]any{
			"name":           "crawlly",
			"schema_version": 1,
		},
		"runtime": map[string]any{
			"status": "no runtimes detected",
		},
	})
	if err != nil {
		t.Fatalf("workspaceExportFromArg returned error: %v", err)
	}
	if exported.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want %d", exported.SchemaVersion, 1)
	}
	if exported.Workspace.Name != "crawlly" {
		t.Fatalf("Workspace.Name = %q, want %q", exported.Workspace.Name, "crawlly")
	}
}

func TestServerTaskToolsStartStatusListAndLogs(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	server := NewServer(a)
	start := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"task_start","arguments":{"path":"` + projectPath + `","name":"echo","command":"/bin/sh","args":["-c","printf out; printf err >&2"]}}}`
	response, err := server.HandleMessage([]byte(start))
	if err != nil {
		t.Fatalf("HandleMessage task_start returned error: %v", err)
	}
	taskID := decodeTaskRunResult(t, response).Task.ID
	if taskID == "" {
		t.Fatal("expected task_start to return task id")
	}

	task := waitForMCPTaskState(t, server, projectPath, taskID, app.TaskRunSucceeded)
	if task.ExitCode == nil || *task.ExitCode != 0 {
		t.Fatalf("unexpected exit code: %#v", task.ExitCode)
	}

	list := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"task_list","arguments":{"path":"` + projectPath + `"}}}`
	response, err = server.HandleMessage([]byte(list))
	if err != nil {
		t.Fatalf("HandleMessage task_list returned error: %v", err)
	}
	var listRPC struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Tasks []app.TaskRun `json:"tasks"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &listRPC); err != nil {
		t.Fatalf("Unmarshal task_list returned error: %v", err)
	}
	if listRPC.Result.IsError {
		t.Fatal("expected task_list success result")
	}
	if len(listRPC.Result.StructuredContent.Tasks) != 1 || listRPC.Result.StructuredContent.Tasks[0].ID != taskID {
		t.Fatalf("unexpected task_list result: %#v", listRPC.Result.StructuredContent.Tasks)
	}

	logs := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"task_logs","arguments":{"path":"` + projectPath + `","task_id":"` + taskID + `"}}}`
	response, err = server.HandleMessage([]byte(logs))
	if err != nil {
		t.Fatalf("HandleMessage task_logs returned error: %v", err)
	}
	var logsRPC struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Logs app.TaskRunLogs `json:"logs"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &logsRPC); err != nil {
		t.Fatalf("Unmarshal task_logs returned error: %v", err)
	}
	if logsRPC.Result.IsError {
		t.Fatal("expected task_logs success result")
	}
	if logsRPC.Result.StructuredContent.Logs.Stdout != "out" {
		t.Fatalf("stdout = %q, want %q", logsRPC.Result.StructuredContent.Logs.Stdout, "out")
	}
	if logsRPC.Result.StructuredContent.Logs.Stderr != "err" {
		t.Fatalf("stderr = %q, want %q", logsRPC.Result.StructuredContent.Logs.Stderr, "err")
	}
}

func TestServerTaskToolsUseActiveProjectScopeWhenPathOmitted(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	server := NewServer(a)
	activate := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"workspace_activate","arguments":{"workspace":"crawlly"}}}`
	if _, err := server.HandleMessage([]byte(activate)); err != nil {
		t.Fatalf("HandleMessage workspace_activate returned error: %v", err)
	}

	start := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"task_start","arguments":{"name":"echo","command":"/bin/sh","args":["-c","printf ok"]}}}`
	response, err := server.HandleMessage([]byte(start))
	if err != nil {
		t.Fatalf("HandleMessage task_start returned error: %v", err)
	}
	taskID := decodeTaskRunResult(t, response).Task.ID
	task := waitForMCPTaskState(t, server, projectPath, taskID, app.TaskRunSucceeded)
	if task.State != app.TaskRunSucceeded {
		t.Fatalf("task state = %q, want %q", task.State, app.TaskRunSucceeded)
	}
}

func TestServerTaskDeclarationTools(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	server := NewServer(a)
	activate := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"workspace_activate","arguments":{"workspace":"crawlly"}}}`
	if _, err := server.HandleMessage([]byte(activate)); err != nil {
		t.Fatalf("HandleMessage workspace_activate returned error: %v", err)
	}

	declare := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"task_declare","arguments":{"name":"test","command":["go","test","./..."],"cwd":"."}}}`
	if _, err := server.HandleMessage([]byte(declare)); err != nil {
		t.Fatalf("HandleMessage task_declare returned error: %v", err)
	}

	list := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"task_list_declared","arguments":{}}}`
	response, err := server.HandleMessage([]byte(list))
	if err != nil {
		t.Fatalf("HandleMessage task_list_declared returned error: %v", err)
	}
	var listRPC struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Tasks []app.TaskSpec `json:"tasks"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &listRPC); err != nil {
		t.Fatalf("Unmarshal task_list_declared returned error: %v", err)
	}
	if listRPC.Result.IsError {
		t.Fatal("expected task_list_declared success result")
	}
	if len(listRPC.Result.StructuredContent.Tasks) != 1 || listRPC.Result.StructuredContent.Tasks[0].Name != "test" {
		t.Fatalf("unexpected declared tasks: %#v", listRPC.Result.StructuredContent.Tasks)
	}

	deleteReq := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"task_delete","arguments":{"name":"test"}}}`
	if _, err := server.HandleMessage([]byte(deleteReq)); err != nil {
		t.Fatalf("HandleMessage task_delete returned error: %v", err)
	}
}

func TestServerTaskStopCancelsRunningTask(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	server := NewServer(a)
	start := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"task_start","arguments":{"path":"` + projectPath + `","name":"sleep","command":"/bin/sh","args":["-c","sleep 30"]}}}`
	response, err := server.HandleMessage([]byte(start))
	if err != nil {
		t.Fatalf("HandleMessage task_start returned error: %v", err)
	}
	taskID := decodeTaskRunResult(t, response).Task.ID

	stop := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"task_stop","arguments":{"path":"` + projectPath + `","task_id":"` + taskID + `"}}}`
	response, err = server.HandleMessage([]byte(stop))
	if err != nil {
		t.Fatalf("HandleMessage task_stop returned error: %v", err)
	}
	task := decodeTaskRunResult(t, response).Task
	if task.State != app.TaskRunCancelled {
		t.Fatalf("task state = %q, want %q", task.State, app.TaskRunCancelled)
	}
}

func TestServerServiceToolsStartStatusListLogsAndStop(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}
	writeServiceManifestForMCPTest(t, a, projectPath, []app.ServiceSpec{
		{Name: "api", Command: []string{"/bin/sh", "-c", "printf out; printf err >&2; sleep 30"}},
	})

	server := NewServer(a)
	start := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"service_start","arguments":{"path":"` + projectPath + `","name":"api"}}}`
	response, err := server.HandleMessage([]byte(start))
	if err != nil {
		t.Fatalf("HandleMessage service_start returned error: %v", err)
	}
	service := decodeServiceStatusResult(t, response).Service
	if service.State != app.ServiceRunning {
		t.Fatalf("service state = %q, want %q", service.State, app.ServiceRunning)
	}

	status := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"service_status","arguments":{"path":"` + projectPath + `","name":"api"}}}`
	response, err = server.HandleMessage([]byte(status))
	if err != nil {
		t.Fatalf("HandleMessage service_status returned error: %v", err)
	}
	service = decodeServiceStatusResult(t, response).Service
	if service.State != app.ServiceRunning {
		t.Fatalf("service state = %q, want %q", service.State, app.ServiceRunning)
	}
	initialPID := service.PID

	restart := `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"service_restart","arguments":{"path":"` + projectPath + `","name":"api"}}}`
	response, err = server.HandleMessage([]byte(restart))
	if err != nil {
		t.Fatalf("HandleMessage service_restart returned error: %v", err)
	}
	service = decodeServiceStatusResult(t, response).Service
	if service.State != app.ServiceRunning {
		t.Fatalf("service state after restart = %q, want %q", service.State, app.ServiceRunning)
	}
	if service.PID == initialPID {
		t.Fatalf("expected restart to replace pid %d, got %d", initialPID, service.PID)
	}

	waitForMCPServiceLogs(t, server, projectPath, "api", "out", "err")
	logs := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"service_logs","arguments":{"path":"` + projectPath + `","name":"api"}}}`
	response, err = server.HandleMessage([]byte(logs))
	if err != nil {
		t.Fatalf("HandleMessage service_logs returned error: %v", err)
	}
	var logsRPC struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Logs app.ServiceLogs `json:"logs"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &logsRPC); err != nil {
		t.Fatalf("Unmarshal service_logs returned error: %v", err)
	}
	if logsRPC.Result.IsError {
		t.Fatal("expected service_logs success result")
	}
	if logsRPC.Result.StructuredContent.Logs.Stdout != "out" || logsRPC.Result.StructuredContent.Logs.Stderr != "err" {
		t.Fatalf("unexpected service logs: %#v", logsRPC.Result.StructuredContent.Logs)
	}

	list := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"service_list","arguments":{"path":"` + projectPath + `"}}}`
	response, err = server.HandleMessage([]byte(list))
	if err != nil {
		t.Fatalf("HandleMessage service_list returned error: %v", err)
	}
	var listRPC struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Services []app.ServiceStatus `json:"services"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &listRPC); err != nil {
		t.Fatalf("Unmarshal service_list returned error: %v", err)
	}
	if listRPC.Result.IsError {
		t.Fatal("expected service_list success result")
	}
	if len(listRPC.Result.StructuredContent.Services) != 1 || listRPC.Result.StructuredContent.Services[0].Name != "api" {
		t.Fatalf("unexpected service_list result: %#v", listRPC.Result.StructuredContent.Services)
	}

	stop := `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"service_stop","arguments":{"path":"` + projectPath + `","name":"api"}}}`
	response, err = server.HandleMessage([]byte(stop))
	if err != nil {
		t.Fatalf("HandleMessage service_stop returned error: %v", err)
	}
	service = decodeServiceStatusResult(t, response).Service
	if service.State != app.ServiceStopped {
		t.Fatalf("service state = %q, want %q", service.State, app.ServiceStopped)
	}
}

func TestServerServiceDeclarationTools(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	server := NewServer(a)
	activate := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"workspace_activate","arguments":{"workspace":"crawlly"}}}`
	if _, err := server.HandleMessage([]byte(activate)); err != nil {
		t.Fatalf("HandleMessage workspace_activate returned error: %v", err)
	}

	declare := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"service_declare","arguments":{"name":"api","command":["go","run","./cmd/api"],"cwd":".","restart":"manual"}}}`
	if _, err := server.HandleMessage([]byte(declare)); err != nil {
		t.Fatalf("HandleMessage service_declare returned error: %v", err)
	}

	list := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"service_list_declared","arguments":{}}}`
	response, err := server.HandleMessage([]byte(list))
	if err != nil {
		t.Fatalf("HandleMessage service_list_declared returned error: %v", err)
	}
	var listRPC struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Services []app.ServiceSpec `json:"services"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &listRPC); err != nil {
		t.Fatalf("Unmarshal service_list_declared returned error: %v", err)
	}
	if listRPC.Result.IsError {
		t.Fatal("expected service_list_declared success result")
	}
	if len(listRPC.Result.StructuredContent.Services) != 1 || listRPC.Result.StructuredContent.Services[0].Name != "api" {
		t.Fatalf("unexpected declared services: %#v", listRPC.Result.StructuredContent.Services)
	}

	deleteReq := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"service_delete","arguments":{"name":"api"}}}`
	if _, err := server.HandleMessage([]byte(deleteReq)); err != nil {
		t.Fatalf("HandleMessage service_delete returned error: %v", err)
	}
}

func TestServerServiceToolsRespectScope(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	allowedPath := filepath.Join(root, "repos", "allowed")
	otherPath := filepath.Join(root, "repos", "other")
	if err := os.MkdirAll(allowedPath, 0o755); err != nil {
		t.Fatalf("MkdirAll allowed returned error: %v", err)
	}
	if err := os.MkdirAll(otherPath, 0o755); err != nil {
		t.Fatalf("MkdirAll other returned error: %v", err)
	}

	server := NewScopedServer(a, []string{allowedPath})
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"service_list","arguments":{"path":"` + otherPath + `"}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}
	var rpc struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if !rpc.Result.IsError {
		t.Fatal("expected service_list to fail outside scope")
	}
	if len(rpc.Result.Content) == 0 || !strings.Contains(rpc.Result.Content[0].Text, "outside the MCP scope") {
		t.Fatalf("unexpected error content: %#v", rpc.Result.Content)
	}
}

func TestServerEventListReturnsTaskLifecycleEvents(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	server := NewServer(a)
	start := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"task_start","arguments":{"path":"` + projectPath + `","name":"echo","command":"/bin/sh","args":["-c","printf ok"]}}}`
	response, err := server.HandleMessage([]byte(start))
	if err != nil {
		t.Fatalf("HandleMessage task_start returned error: %v", err)
	}
	taskID := decodeTaskRunResult(t, response).Task.ID
	waitForMCPTaskState(t, server, projectPath, taskID, app.TaskRunSucceeded)

	list := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"event_list","arguments":{"path":"` + projectPath + `"}}}`
	response, err = server.HandleMessage([]byte(list))
	if err != nil {
		t.Fatalf("HandleMessage event_list returned error: %v", err)
	}
	var rpc struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Events []app.RuntimeEvent `json:"events"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal event_list returned error: %v", err)
	}
	if rpc.Result.IsError {
		t.Fatal("expected event_list success result")
	}
	if len(rpc.Result.StructuredContent.Events) != 2 {
		t.Fatalf("expected 2 events, got %#v", rpc.Result.StructuredContent.Events)
	}
	if rpc.Result.StructuredContent.Events[0].Kind != app.EventKindTaskExited {
		t.Fatalf("newest event kind = %q, want %q", rpc.Result.StructuredContent.Events[0].Kind, app.EventKindTaskExited)
	}
}

func TestServerEventListReturnsServiceLifecycleEvents(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}
	writeServiceManifestForMCPTest(t, a, projectPath, []app.ServiceSpec{
		{Name: "api", Command: []string{"/bin/sh", "-c", "sleep 30"}},
	})

	server := NewServer(a)
	start := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"service_start","arguments":{"path":"` + projectPath + `","name":"api"}}}`
	if _, err := server.HandleMessage([]byte(start)); err != nil {
		t.Fatalf("HandleMessage service_start returned error: %v", err)
	}
	stop := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"service_stop","arguments":{"path":"` + projectPath + `","name":"api"}}}`
	if _, err := server.HandleMessage([]byte(stop)); err != nil {
		t.Fatalf("HandleMessage service_stop returned error: %v", err)
	}
	list := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"event_list","arguments":{"path":"` + projectPath + `"}}}`
	response, err := server.HandleMessage([]byte(list))
	if err != nil {
		t.Fatalf("HandleMessage event_list returned error: %v", err)
	}
	var rpc struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Events []app.RuntimeEvent `json:"events"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal event_list returned error: %v", err)
	}
	if rpc.Result.IsError {
		t.Fatal("expected event_list success result")
	}
	if len(rpc.Result.StructuredContent.Events) < 2 {
		t.Fatalf("expected at least 2 events, got %#v", rpc.Result.StructuredContent.Events)
	}
	if rpc.Result.StructuredContent.Events[0].Kind != app.EventKindServiceStopped {
		t.Fatalf("newest event kind = %q, want %q", rpc.Result.StructuredContent.Events[0].Kind, app.EventKindServiceStopped)
	}
}

func TestServerTaskToolsRespectScope(t *testing.T) {
	root := t.TempDir()
	a := app.NewApp(root)
	allowedPath := filepath.Join(root, "repos", "allowed")
	otherPath := filepath.Join(root, "repos", "other")
	if err := os.MkdirAll(allowedPath, 0o755); err != nil {
		t.Fatalf("MkdirAll allowed returned error: %v", err)
	}
	if err := os.MkdirAll(otherPath, 0o755); err != nil {
		t.Fatalf("MkdirAll other returned error: %v", err)
	}

	server := NewScopedServer(a, []string{allowedPath})
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"task_start","arguments":{"path":"` + otherPath + `","command":"/bin/sh","args":["-c","printf nope"]}}}`
	response, err := server.HandleMessage([]byte(request))
	if err != nil {
		t.Fatalf("HandleMessage returned error: %v", err)
	}
	var rpc struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if !rpc.Result.IsError {
		t.Fatal("expected task_start to fail outside scope")
	}
	if len(rpc.Result.Content) == 0 || !strings.Contains(rpc.Result.Content[0].Text, "outside the MCP scope") {
		t.Fatalf("unexpected error content: %#v", rpc.Result.Content)
	}
}

func TestServerServeUsesNewlineDelimitedMessages(t *testing.T) {
	server := NewServer(app.NewApp(t.TempDir()))

	var in bytes.Buffer
	in.WriteString(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}` + "\n")
	in.WriteString(`{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n")
	in.WriteString(`{"jsonrpc":"2.0","id":2,"method":"shutdown"}` + "\n")
	in.WriteString(`{"jsonrpc":"2.0","method":"notifications/exit"}` + "\n")
	in.WriteString(`{"jsonrpc":"2.0","id":3,"method":"ping"}` + "\n")

	var out bytes.Buffer
	if err := server.Serve(&in, &out); err != nil {
		t.Fatalf("Serve returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 response lines, got %d: %q", len(lines), out.String())
	}
}

func TestHTTPHandlerAcceptsJSONRPCPost(t *testing.T) {
	server := NewServer(app.NewApp(t.TempDir()))
	handler := NewHTTPHandler(server, "/mcp")

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", res.Code, http.StatusOK, res.Body.String())
	}
	if got := res.Header().Get("MCP-Protocol-Version"); got != ProtocolVersion {
		t.Fatalf("MCP-Protocol-Version = %q, want %q", got, ProtocolVersion)
	}

	var rpc struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &rpc); err != nil {
		t.Fatalf("Unmarshal HTTP initialize response returned error: %v", err)
	}
	if rpc.Result.ProtocolVersion != ProtocolVersion {
		t.Fatalf("protocolVersion = %q, want %q", rpc.Result.ProtocolVersion, ProtocolVersion)
	}
}

func TestHTTPHandlerReturnsAcceptedForNotification(t *testing.T) {
	server := NewServer(app.NewApp(t.TempDir()))
	handler := NewHTTPHandler(server, "/mcp")

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d: %s", res.Code, http.StatusAccepted, res.Body.String())
	}
	if body := strings.TrimSpace(res.Body.String()); body != "" {
		t.Fatalf("body = %q, want empty", body)
	}
}

func TestHTTPHandlerRejectsGETStream(t *testing.T) {
	server := NewServer(app.NewApp(t.TempDir()))
	handler := NewHTTPHandler(server, "/mcp")

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusMethodNotAllowed)
	}
}

func decodeTaskRunResult(t *testing.T, response []byte) taskRunResult {
	t.Helper()
	var rpc struct {
		Result struct {
			IsError           bool          `json:"isError"`
			StructuredContent taskRunResult `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal task run response returned error: %v", err)
	}
	if rpc.Result.IsError {
		t.Fatalf("expected task tool success response: %s", response)
	}
	return rpc.Result.StructuredContent
}

func decodeServiceStatusResult(t *testing.T, response []byte) serviceStatusResult {
	t.Helper()
	var rpc struct {
		Result struct {
			IsError           bool                `json:"isError"`
			StructuredContent serviceStatusResult `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &rpc); err != nil {
		t.Fatalf("Unmarshal service response returned error: %v", err)
	}
	if rpc.Result.IsError {
		t.Fatalf("expected service tool success response: %s", response)
	}
	return rpc.Result.StructuredContent
}

func waitForMCPTaskState(t *testing.T, server *Server, projectPath, taskID string, want app.TaskRunState) app.TaskRun {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		status := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"task_status","arguments":{"path":"` + projectPath + `","task_id":"` + taskID + `"}}}`
		response, err := server.HandleMessage([]byte(status))
		if err != nil {
			t.Fatalf("HandleMessage task_status returned error: %v", err)
		}
		task := decodeTaskRunResult(t, response).Task
		if task.State == want {
			return task
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for task %q to reach %q", taskID, want)
	return app.TaskRun{}
}

func waitForMCPServiceLogs(t *testing.T, server *Server, projectPath, serviceName, wantStdout, wantStderr string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		logs := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"service_logs","arguments":{"path":"` + projectPath + `","name":"` + serviceName + `"}}}`
		response, err := server.HandleMessage([]byte(logs))
		if err != nil {
			t.Fatalf("HandleMessage service_logs returned error: %v", err)
		}
		var rpc struct {
			Result struct {
				IsError           bool `json:"isError"`
				StructuredContent struct {
					Logs app.ServiceLogs `json:"logs"`
				} `json:"structuredContent"`
			} `json:"result"`
		}
		if err := json.Unmarshal(response, &rpc); err != nil {
			t.Fatalf("Unmarshal service_logs returned error: %v", err)
		}
		if rpc.Result.IsError {
			t.Fatalf("expected service_logs success response: %s", response)
		}
		if strings.Contains(rpc.Result.StructuredContent.Logs.Stdout, wantStdout) && strings.Contains(rpc.Result.StructuredContent.Logs.Stderr, wantStderr) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for service %q logs", serviceName)
}

func writeServiceManifestForMCPTest(t *testing.T, a *app.App, projectPath string, services []app.ServiceSpec) {
	t.Helper()
	wsPath, err := a.EnsureWorkspace("crawlly")
	if err != nil {
		t.Fatalf("EnsureWorkspace returned error: %v", err)
	}
	manifest := app.Manifest{
		SchemaVersion: 1,
		Name:          "crawlly",
		ProjectPath:   projectPath,
		Packages:      []app.PackageSpec{},
		Tasks:         []app.TaskSpec{},
		Services:      services,
		Env:           map[string]string{},
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsPath, "manifest.json"), data, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}

func slicesContainsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func slicesContainsAnyString(items []any, want string) bool {
	for _, item := range items {
		if value, ok := item.(string); ok && value == want {
			return true
		}
	}
	return false
}

func setupMCPIndexedWorkspace(t *testing.T, a *app.App, root string) string {
	t.Helper()

	projectPath := filepath.Join(root, "repos", "crawlly")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := a.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}
	if err := a.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "engine.go"), []byte("package demo\n\n// vault damage\n type Engine struct{}\n\nfunc (e *Engine) DamagePerHeat() {}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile engine.go returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "notes.txt"), []byte("vault context workspace\n"), 0o600); err != nil {
		t.Fatalf("WriteFile notes.txt returned error: %v", err)
	}
	if _, err := a.UpdateIndex("crawlly"); err != nil {
		t.Fatalf("UpdateIndex returned error: %v", err)
	}
	return projectPath
}
