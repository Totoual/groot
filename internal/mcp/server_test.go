package mcp

import (
	"bytes"
	"encoding/json"
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
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &listResponse); err != nil {
		t.Fatalf("Unmarshal tools/list response returned error: %v", err)
	}
	if len(listResponse.Result.Tools) != 16 {
		t.Fatalf("len(tools) = %d, want %d", len(listResponse.Result.Tools), 16)
	}
	names := make([]string, 0, len(listResponse.Result.Tools))
	for _, tool := range listResponse.Result.Tools {
		names = append(names, tool.Name)
	}
	for _, want := range []string{"task_start", "task_status", "task_list", "task_logs", "task_stop", "event_list"} {
		if !slicesContainsString(names, want) {
			t.Fatalf("missing tool %q in %#v", want, names)
		}
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
	in.WriteString(`{"jsonrpc":"2.0","id":2,"method":"ping"}` + "\n")

	var out bytes.Buffer
	if err := server.Serve(&in, &out); err != nil {
		t.Fatalf("Serve returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 response lines, got %d: %q", len(lines), out.String())
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

func slicesContainsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
