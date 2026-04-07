package mcp

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
			ServerInfo      struct {
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
	if len(listResponse.Result.Tools) != 3 {
		t.Fatalf("len(tools) = %d, want %d", len(listResponse.Result.Tools), 3)
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
