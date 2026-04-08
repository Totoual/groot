package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/totoual/groot/internal/app"
)

const ProtocolVersion = "2025-06-18"

type Server struct {
	app             *app.App
	allowedProjects []string
	activeProjects  []string
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type toolDefinition struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	InputSchema  map[string]any `json:"inputSchema"`
	OutputSchema map[string]any `json:"outputSchema,omitempty"`
}

type toolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type toolResult struct {
	Content           []toolContent `json:"content"`
	StructuredContent any           `json:"structuredContent,omitempty"`
	IsError           bool          `json:"isError,omitempty"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type resourceDefinition struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mimeType,omitempty"`
}

type resourceReadParams struct {
	URI string `json:"uri"`
}

type resourceReadResult struct {
	Contents []resourceContent `json:"contents"`
}

type resourceContent struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}

type workspaceStatusResult struct {
	Created bool                         `json:"created"`
	Status  app.WorkspaceRuntimeSnapshot `json:"status"`
}

type workspaceSetupResult struct {
	Created bool                         `json:"created"`
	Plan    app.FirstOpenRuntimePlan     `json:"plan"`
	Status  app.WorkspaceRuntimeSnapshot `json:"status"`
}

type workspaceExecResult struct {
	Created    bool                         `json:"created"`
	Workspace  app.WorkspaceRuntimeSnapshot `json:"workspace"`
	Command    string                       `json:"command"`
	Args       []string                     `json:"args"`
	WorkDir    string                       `json:"workdir"`
	Stdout     string                       `json:"stdout,omitempty"`
	Stderr     string                       `json:"stderr,omitempty"`
	ExitCode   int                          `json:"exit_code"`
	Warnings   []string                     `json:"warnings,omitempty"`
	StrictMode bool                         `json:"strict_mode"`
}

type workspaceInspectResult struct {
	Created bool                `json:"created"`
	Inspect workspaceInspection `json:"inspect"`
}

type workspaceExportResult struct {
	Export app.WorkspaceExport `json:"export"`
}

type workspaceImportResult struct {
	Created       bool                         `json:"created"`
	WorkspaceName string                       `json:"workspace_name"`
	ProjectPath   string                       `json:"project_path"`
	Status        app.WorkspaceRuntimeSnapshot `json:"status"`
}

type workspaceEnvResult struct {
	Created       bool              `json:"created"`
	WorkspaceName string            `json:"workspace_name"`
	WorkDir       string            `json:"workdir"`
	Env           map[string]string `json:"env"`
}

type workspaceAttachResult struct {
	Created  bool                         `json:"created"`
	Attached []app.Component              `json:"attached"`
	Status   app.WorkspaceRuntimeSnapshot `json:"status"`
}

type workspaceInstallResult struct {
	Created   bool                         `json:"created"`
	Installed []app.Component              `json:"installed"`
	Status    app.WorkspaceRuntimeSnapshot `json:"status"`
}

type workspaceActivateResult struct {
	ActiveProject string `json:"active_project"`
	WorkspaceName string `json:"workspace_name,omitempty"`
}

type workspaceInspection struct {
	WorkspaceName string                       `json:"workspace_name"`
	WorkspaceDir  string                       `json:"workspace_dir"`
	ManifestPath  string                       `json:"manifest_path"`
	HomeDir       string                       `json:"home_dir"`
	StateDir      string                       `json:"state_dir"`
	LogsDir       string                       `json:"logs_dir"`
	Manifest      app.Manifest                 `json:"manifest"`
	Runtime       app.WorkspaceRuntimeSnapshot `json:"runtime"`
}

type workspaceMetadataResource struct {
	WorkspaceName string                       `json:"workspace_name"`
	ProjectPath   string                       `json:"project_path,omitempty"`
	WorkspaceDir  string                       `json:"workspace_dir"`
	ManifestPath  string                       `json:"manifest_path"`
	HomeDir       string                       `json:"home_dir"`
	StateDir      string                       `json:"state_dir"`
	LogsDir       string                       `json:"logs_dir"`
	Runtime       app.WorkspaceRuntimeSnapshot `json:"runtime"`
}

func NewServer(a *app.App) *Server {
	return &Server{app: a}
}

func NewScopedServer(a *app.App, allowedProjects []string) *Server {
	scoped := make([]string, 0, len(allowedProjects))
	for _, projectPath := range allowedProjects {
		normalized, err := app.NormalizeProjectPath(projectPath)
		if err != nil {
			continue
		}
		scoped = append(scoped, normalized)
	}
	return &Server{
		app:             a,
		allowedProjects: scoped,
	}
}

func (s *Server) Serve(in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		response, err := s.HandleMessage(line)
		if err != nil {
			return err
		}
		if len(response) == 0 {
			continue
		}
		if _, err := out.Write(response); err != nil {
			return fmt.Errorf("write mcp response: %w", err)
		}
		if _, err := out.Write([]byte("\n")); err != nil {
			return fmt.Errorf("write mcp delimiter: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read mcp request: %w", err)
	}
	return nil
}

func (s *Server) HandleMessage(message []byte) ([]byte, error) {
	message = bytesTrimSpace(message)
	if len(message) == 0 {
		return nil, nil
	}

	if message[0] == '[' {
		var rawBatch []json.RawMessage
		if err := json.Unmarshal(message, &rawBatch); err != nil {
			return marshalResponse(rpcResponse{
				JSONRPC: "2.0",
				ID:      json.RawMessage("null"),
				Error:   &rpcError{Code: -32700, Message: "parse error"},
			})
		}
		responses := make([]json.RawMessage, 0, len(rawBatch))
		for _, raw := range rawBatch {
			response, err := s.handleSingle(raw)
			if err != nil {
				return nil, err
			}
			if len(response) == 0 {
				continue
			}
			responses = append(responses, response)
		}
		if len(responses) == 0 {
			return nil, nil
		}
		return json.Marshal(responses)
	}

	return s.handleSingle(message)
}

func (s *Server) handleSingle(message []byte) ([]byte, error) {
	var req rpcRequest
	if err := json.Unmarshal(message, &req); err != nil {
		return marshalResponse(rpcResponse{
			JSONRPC: "2.0",
			ID:      json.RawMessage("null"),
			Error:   &rpcError{Code: -32700, Message: "parse error"},
		})
	}
	if req.JSONRPC != "2.0" || req.Method == "" {
		return marshalResponse(rpcResponse{
			JSONRPC: "2.0",
			ID:      responseID(req.ID),
			Error:   &rpcError{Code: -32600, Message: "invalid request"},
		})
	}
	if len(req.ID) == 0 {
		return nil, nil
	}

	switch req.Method {
	case "initialize":
		return marshalResponse(rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": ProtocolVersion,
				"capabilities": map[string]any{
					"tools":     map[string]any{},
					"resources": map[string]any{},
				},
				"serverInfo": map[string]any{
					"name":    "groot",
					"version": "dev",
				},
			},
		})
	case "ping":
		return marshalResponse(rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{},
		})
	case "tools/list":
		return marshalResponse(rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"tools": s.tools(),
			},
		})
	case "resources/list":
		return marshalResponse(rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"resources": s.resources(),
			},
		})
	case "resources/read":
		var params resourceReadParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return marshalResponse(rpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &rpcError{Code: -32602, Message: "invalid resource read params"},
			})
		}
		result, rpcErr := s.readResource(params)
		if rpcErr != nil {
			return marshalResponse(rpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   rpcErr,
			})
		}
		return marshalResponse(rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		})
	case "tools/call":
		var params toolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return marshalResponse(rpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &rpcError{Code: -32602, Message: "invalid tool call params"},
			})
		}
		result := s.callTool(params)
		return marshalResponse(rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		})
	default:
		return marshalResponse(rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32601, Message: "method not found"},
		})
	}
}

func (s *Server) tools() []toolDefinition {
	return []toolDefinition{
		{
			Name:        "workspace_activate",
			Description: "Activate one project path or bound workspace as the MCP session scope. Later tool calls are restricted to that project until another activation or server restart.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute or ~/ project path to activate for this MCP session.",
					},
					"workspace": map[string]any{
						"type":        "string",
						"description": "Bound workspace name to activate for this MCP session.",
					},
				},
				"additionalProperties": false,
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"active_project": map[string]any{"type": "string"},
					"workspace_name": map[string]any{"type": "string"},
				},
				"required": []string{"active_project"},
			},
		},
		{
			Name:        "workspace_status",
			Description: "Resolve or create a workspace from a project path and return runtime ownership status.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute or ~/ project path.",
					},
				},
				"required":             []string{"path"},
				"additionalProperties": false,
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"created": map[string]any{"type": "boolean"},
					"status":  map[string]any{"type": "object"},
				},
				"required": []string{"created", "status"},
			},
		},
		{
			Name:        "workspace_setup",
			Description: "Resolve or create a workspace from a project path and optionally attach/install detected runtimes.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute or ~/ project path.",
					},
					"attach_detected": map[string]any{
						"type":        "boolean",
						"description": "Attach detected runtimes with concrete versions. Defaults to true.",
					},
					"install_detected": map[string]any{
						"type":        "boolean",
						"description": "Install attached detected runtimes. Defaults to true.",
					},
				},
				"required":             []string{"path"},
				"additionalProperties": false,
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"created": map[string]any{"type": "boolean"},
					"plan":    map[string]any{"type": "object"},
					"status":  map[string]any{"type": "object"},
				},
				"required": []string{"created", "plan", "status"},
			},
		},
		{
			Name:        "workspace_exec",
			Description: "Resolve or create a workspace from a project path and run one command in the strict Groot runtime.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute or ~/ project path.",
					},
					"command": map[string]any{
						"type":        "string",
						"description": "Executable to run inside the workspace runtime.",
					},
					"args": map[string]any{
						"type":        "array",
						"description": "Optional command arguments.",
						"items": map[string]any{
							"type": "string",
						},
					},
				},
				"required":             []string{"path", "command"},
				"additionalProperties": false,
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"created":     map[string]any{"type": "boolean"},
					"workspace":   map[string]any{"type": "object"},
					"command":     map[string]any{"type": "string"},
					"args":        map[string]any{"type": "array"},
					"workdir":     map[string]any{"type": "string"},
					"stdout":      map[string]any{"type": "string"},
					"stderr":      map[string]any{"type": "string"},
					"exit_code":   map[string]any{"type": "integer"},
					"warnings":    map[string]any{"type": "array"},
					"strict_mode": map[string]any{"type": "boolean"},
				},
				"required": []string{"created", "workspace", "command", "args", "workdir", "exit_code", "strict_mode"},
			},
		},
		{
			Name:        "workspace_inspect",
			Description: "Resolve or create a workspace from a project path and return the manifest, workspace paths, and runtime ownership state.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute or ~/ project path.",
					},
				},
				"required":             []string{"path"},
				"additionalProperties": false,
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"created": map[string]any{"type": "boolean"},
					"inspect": map[string]any{"type": "object"},
				},
				"required": []string{"created", "inspect"},
			},
		},
		{
			Name:        "workspace_env",
			Description: "Resolve or create a workspace from a project path and return the strict runtime environment as structured key/value pairs.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute or ~/ project path.",
					},
				},
				"required":             []string{"path"},
				"additionalProperties": false,
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"created": map[string]any{"type": "boolean"},
					"workdir": map[string]any{"type": "string"},
					"env":     map[string]any{"type": "object"},
				},
				"required": []string{"created", "workdir", "env"},
			},
		},
		{
			Name:        "workspace_attach",
			Description: "Resolve or create a workspace from a project path and attach explicit toolchain specs like go@1.25.4.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute or ~/ project path.",
					},
					"toolchains": map[string]any{
						"type":        "array",
						"description": "Toolchain specs in name@version format.",
						"items": map[string]any{
							"type": "string",
						},
					},
				},
				"required":             []string{"path", "toolchains"},
				"additionalProperties": false,
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"created":  map[string]any{"type": "boolean"},
					"attached": map[string]any{"type": "array"},
					"status":   map[string]any{"type": "object"},
				},
				"required": []string{"created", "attached", "status"},
			},
		},
		{
			Name:        "workspace_install",
			Description: "Resolve or create a workspace from a project path and install all attached toolchains into Groot's managed store.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute or ~/ project path.",
					},
				},
				"required":             []string{"path"},
				"additionalProperties": false,
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"created":   map[string]any{"type": "boolean"},
					"installed": map[string]any{"type": "array"},
					"status":    map[string]any{"type": "object"},
				},
				"required": []string{"created", "installed", "status"},
			},
		},
		{
			Name:        "workspace_export",
			Description: "Export the existing workspace contract for a project path as portable structured data.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute or ~/ project path bound to an existing workspace.",
					},
				},
				"required":             []string{"path"},
				"additionalProperties": false,
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"export": map[string]any{"type": "object"},
				},
				"required": []string{"export"},
			},
		},
		{
			Name:        "workspace_import",
			Description: "Import a portable workspace contract for an existing project path and optionally install attached toolchains.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute or ~/ existing project path to bind the imported workspace to.",
					},
					"export": map[string]any{
						"type":        "object",
						"description": "Portable workspace export previously returned by workspace_export.",
					},
					"install_attached": map[string]any{
						"type":        "boolean",
						"description": "Install attached toolchains after import. Defaults to false.",
					},
					"workspace_name": map[string]any{
						"type":        "string",
						"description": "Optional workspace name override when the exported workspace name would collide on this machine.",
					},
				},
				"required":             []string{"path", "export"},
				"additionalProperties": false,
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"created":        map[string]any{"type": "boolean"},
					"workspace_name": map[string]any{"type": "string"},
					"project_path":   map[string]any{"type": "string"},
					"status":         map[string]any{"type": "object"},
				},
				"required": []string{"created", "workspace_name", "project_path", "status"},
			},
		},
	}
}

func (s *Server) resources() []resourceDefinition {
	scopeProjects := s.currentScopeProjects()
	if len(scopeProjects) == 0 {
		return []resourceDefinition{}
	}

	resources := make([]resourceDefinition, 0, len(scopeProjects)*2)
	for _, projectPath := range scopeProjects {
		workspaceName, _, err := s.app.ResolveOrCreateWorkspaceByProjectPath(projectPath)
		if err != nil {
			continue
		}
		resources = append(resources,
			resourceDefinition{
				URI:         workspaceResourceURI(workspaceName, "manifest"),
				Name:        fmt.Sprintf("%s manifest", workspaceName),
				Description: fmt.Sprintf("Workspace manifest for %s.", workspaceName),
				MIMEType:    "application/json",
			},
			resourceDefinition{
				URI:         workspaceResourceURI(workspaceName, "metadata"),
				Name:        fmt.Sprintf("%s metadata", workspaceName),
				Description: fmt.Sprintf("Workspace metadata and runtime snapshot for %s.", workspaceName),
				MIMEType:    "application/json",
			},
		)
	}
	return resources
}

func (s *Server) callTool(params toolCallParams) toolResult {
	switch params.Name {
	case "workspace_activate":
		return s.workspaceActivateTool(params.Arguments)
	case "workspace_status":
		return s.workspaceStatusTool(params.Arguments)
	case "workspace_setup":
		return s.workspaceSetupTool(params.Arguments)
	case "workspace_exec":
		return s.workspaceExecTool(params.Arguments)
	case "workspace_inspect":
		return s.workspaceInspectTool(params.Arguments)
	case "workspace_env":
		return s.workspaceEnvTool(params.Arguments)
	case "workspace_attach":
		return s.workspaceAttachTool(params.Arguments)
	case "workspace_install":
		return s.workspaceInstallTool(params.Arguments)
	case "workspace_export":
		return s.workspaceExportTool(params.Arguments)
	case "workspace_import":
		return s.workspaceImportTool(params.Arguments)
	default:
		return errorToolResult(fmt.Sprintf("unknown tool %q", params.Name), nil)
	}
}

func (s *Server) readResource(params resourceReadParams) (resourceReadResult, *rpcError) {
	if strings.TrimSpace(params.URI) == "" {
		return resourceReadResult{}, &rpcError{Code: -32602, Message: `resource read requires "uri"`}
	}

	workspaceName, kind, err := parseWorkspaceResourceURI(params.URI)
	if err != nil {
		return resourceReadResult{}, &rpcError{Code: -32602, Message: err.Error()}
	}

	inspect, err := s.app.InspectWorkspace(workspaceName)
	if err != nil {
		return resourceReadResult{}, &rpcError{Code: -32000, Message: err.Error()}
	}
	if strings.TrimSpace(inspect.Manifest.ProjectPath) == "" {
		return resourceReadResult{}, &rpcError{Code: -32000, Message: fmt.Sprintf("workspace %q is not bound to a project path", workspaceName)}
	}
	if _, err := s.scopedProjectPath(inspect.Manifest.ProjectPath); err != nil {
		return resourceReadResult{}, &rpcError{Code: -32000, Message: err.Error()}
	}

	var payload any
	switch kind {
	case "manifest":
		payload = inspect.Manifest
	case "metadata":
		payload = workspaceMetadataResource{
			WorkspaceName: inspect.WorkspaceName,
			ProjectPath:   inspect.Manifest.ProjectPath,
			WorkspaceDir:  inspect.WorkspaceDir,
			ManifestPath:  inspect.ManifestPath,
			HomeDir:       inspect.HomeDir,
			StateDir:      inspect.StateDir,
			LogsDir:       inspect.LogsDir,
			Runtime:       app.WorkspaceRuntimeSnapshotFor(inspect.Runtime),
		}
	default:
		return resourceReadResult{}, &rpcError{Code: -32602, Message: fmt.Sprintf("unsupported resource kind %q", kind)}
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return resourceReadResult{}, &rpcError{Code: -32000, Message: fmt.Sprintf("marshal resource %q: %v", params.URI, err)}
	}

	return resourceReadResult{
		Contents: []resourceContent{{
			URI:      params.URI,
			MIMEType: "application/json",
			Text:     string(data) + "\n",
		}},
	}, nil
}

func (s *Server) currentScopeProjects() []string {
	if len(s.activeProjects) > 0 {
		return s.activeProjects
	}
	return s.allowedProjects
}

func scopedProjectPathWithin(scopeProjects []string, projectPath string) (string, error) {
	normalizedPath, err := app.NormalizeProjectPath(projectPath)
	if err != nil {
		return "", err
	}
	if len(scopeProjects) == 0 {
		return normalizedPath, nil
	}

	for _, allowed := range scopeProjects {
		match, err := app.ProjectPathsMatch(allowed, normalizedPath)
		if err != nil {
			return "", err
		}
		if match {
			return normalizedPath, nil
		}
	}

	return "", fmt.Errorf("project path %q is outside the MCP scope", normalizedPath)
}

func (s *Server) scopedProjectPath(projectPath string) (string, error) {
	return scopedProjectPathWithin(s.currentScopeProjects(), projectPath)
}

func (s *Server) setActiveProjects(projects []string) error {
	normalized := make([]string, 0, len(projects))
	for _, projectPath := range projects {
		scopedPath, err := scopedProjectPathWithin(s.allowedProjects, projectPath)
		if err != nil {
			return err
		}
		normalized = append(normalized, scopedPath)
	}
	s.activeProjects = normalized
	return nil
}

func (s *Server) workspaceActivateTool(args map[string]any) toolResult {
	path, hasPath := stringArg(args, "path")
	workspaceName, hasWorkspace := stringArg(args, "workspace")
	if hasPath == hasWorkspace {
		return errorToolResult(`tool "workspace_activate" requires exactly one of "path" or "workspace"`, nil)
	}

	activeProject := ""
	result := workspaceActivateResult{}
	if hasPath {
		normalizedPath, err := app.NormalizeProjectPath(path)
		if err != nil {
			return errorToolResult(err.Error(), nil)
		}
		info, err := os.Stat(normalizedPath)
		if err != nil {
			return errorToolResult(err.Error(), nil)
		}
		if !info.IsDir() {
			return errorToolResult(fmt.Sprintf("project path %q is not a directory", normalizedPath), nil)
		}
		if err := s.setActiveProjects([]string{normalizedPath}); err != nil {
			return errorToolResult(err.Error(), nil)
		}
		activeProject = normalizedPath
		if boundWorkspace, err := s.app.FindWorkspaceByProjectPath(normalizedPath); err == nil {
			result.WorkspaceName = boundWorkspace
		}
	} else {
		inspect, err := s.app.InspectWorkspace(workspaceName)
		if err != nil {
			return errorToolResult(err.Error(), nil)
		}
		if strings.TrimSpace(inspect.Manifest.ProjectPath) == "" {
			return errorToolResult(fmt.Sprintf("workspace %q is not bound to a project path", workspaceName), nil)
		}
		if err := s.setActiveProjects([]string{inspect.Manifest.ProjectPath}); err != nil {
			return errorToolResult(err.Error(), nil)
		}
		activeProject = inspect.Manifest.ProjectPath
		result.WorkspaceName = workspaceName
	}

	result.ActiveProject = activeProject
	return successToolResult(
		fmt.Sprintf("Activated MCP scope for project %q.", activeProject),
		result,
	)
}

func (s *Server) workspaceStatusTool(args map[string]any) toolResult {
	projectPath, ok := stringArg(args, "path")
	if !ok {
		return errorToolResult(`tool "workspace_status" requires string argument "path"`, nil)
	}
	projectPath, err := s.scopedProjectPath(projectPath)
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}

	workspaceName, created, err := s.app.ResolveOrCreateWorkspaceByProjectPath(projectPath)
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}
	report, err := s.app.InspectWorkspaceRuntimeOwnership(workspaceName)
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}

	result := workspaceStatusResult{
		Created: created,
		Status:  app.WorkspaceRuntimeSnapshotFor(report),
	}
	return successToolResult(
		fmt.Sprintf("Workspace %q is %s.", report.WorkspaceName, app.RuntimeOwnershipStatusLabel(report)),
		result,
	)
}

func (s *Server) workspaceSetupTool(args map[string]any) toolResult {
	projectPath, ok := stringArg(args, "path")
	if !ok {
		return errorToolResult(`tool "workspace_setup" requires string argument "path"`, nil)
	}
	projectPath, err := s.scopedProjectPath(projectPath)
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}
	attachDetected := boolArgOrDefault(args, "attach_detected", true)
	installDetected := boolArgOrDefault(args, "install_detected", true)
	if installDetected {
		attachDetected = true
	}

	workspaceName, created, err := s.app.ResolveOrCreateWorkspaceByProjectPath(projectPath)
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}
	plan, err := s.app.BuildFirstOpenRuntimePlan(workspaceName, projectPath, attachDetected, installDetected)
	if err != nil {
		return errorToolResult(err.Error(), map[string]any{
			"workspace_name": workspaceName,
			"created":        created,
		})
	}
	report, err := s.app.InspectWorkspaceRuntimeOwnership(workspaceName)
	if err != nil {
		return errorToolResult(err.Error(), map[string]any{
			"workspace_name": workspaceName,
			"created":        created,
		})
	}

	result := workspaceSetupResult{
		Created: created,
		Plan:    plan,
		Status:  app.WorkspaceRuntimeSnapshotFor(report),
	}
	return successToolResult(
		fmt.Sprintf("Workspace %q setup completed with status %q.", report.WorkspaceName, app.RuntimeOwnershipStatusLabel(report)),
		result,
	)
}

func (s *Server) workspaceExecTool(args map[string]any) toolResult {
	projectPath, ok := stringArg(args, "path")
	if !ok {
		return errorToolResult(`tool "workspace_exec" requires string argument "path"`, nil)
	}
	projectPath, err := s.scopedProjectPath(projectPath)
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}
	command, ok := stringArg(args, "command")
	if !ok {
		return errorToolResult(`tool "workspace_exec" requires string argument "command"`, nil)
	}
	commandArgs, err := stringSliceArg(args, "args")
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}

	workspaceName, created, err := s.app.ResolveOrCreateWorkspaceByProjectPath(projectPath)
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}
	report, err := s.app.InspectWorkspaceRuntimeOwnership(workspaceName)
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}

	warnings := []string{}
	if len(report.Missing) > 0 {
		warnings = append(warnings, fmt.Sprintf("host fallback risk: %s", formatDetectedToolchains(report.Missing)))
		if app.RuntimeStrictModeEnabled() {
			return errorToolResult(
				fmt.Sprintf("strict runtime mode rejected undeclared detected runtimes for workspace %q", workspaceName),
				workspaceExecResult{
					Created:    created,
					Workspace:  app.WorkspaceRuntimeSnapshotFor(report),
					Command:    command,
					Args:       commandArgs,
					Warnings:   warnings,
					StrictMode: true,
				},
			)
		}
	}

	execResult, err := s.app.ExecWorkspaceCapture(workspaceName, command, commandArgs)
	if err != nil {
		return errorToolResult(err.Error(), workspaceExecResult{
			Created:    created,
			Workspace:  app.WorkspaceRuntimeSnapshotFor(report),
			Command:    command,
			Args:       commandArgs,
			Warnings:   warnings,
			StrictMode: app.RuntimeStrictModeEnabled(),
		})
	}

	result := workspaceExecResult{
		Created:    created,
		Workspace:  app.WorkspaceRuntimeSnapshotFor(report),
		Command:    command,
		Args:       commandArgs,
		WorkDir:    execResult.WorkDir,
		Stdout:     execResult.Stdout,
		Stderr:     execResult.Stderr,
		ExitCode:   execResult.ExitCode,
		Warnings:   warnings,
		StrictMode: app.RuntimeStrictModeEnabled(),
	}
	text := fmt.Sprintf("Command %q finished with exit code %d.", command, execResult.ExitCode)
	if execResult.ExitCode != 0 {
		return errorToolResult(text, result)
	}
	return successToolResult(text, result)
}

func (s *Server) workspaceInspectTool(args map[string]any) toolResult {
	projectPath, ok := stringArg(args, "path")
	if !ok {
		return errorToolResult(`tool "workspace_inspect" requires string argument "path"`, nil)
	}
	projectPath, err := s.scopedProjectPath(projectPath)
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}

	workspaceName, created, err := s.app.ResolveOrCreateWorkspaceByProjectPath(projectPath)
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}
	inspect, err := s.app.InspectWorkspace(workspaceName)
	if err != nil {
		return errorToolResult(err.Error(), map[string]any{
			"workspace_name": workspaceName,
			"created":        created,
		})
	}

	result := workspaceInspectResult{
		Created: created,
		Inspect: makeWorkspaceInspection(inspect),
	}
	return successToolResult(
		fmt.Sprintf("Workspace %q inspection loaded.", inspect.WorkspaceName),
		result,
	)
}

func (s *Server) workspaceEnvTool(args map[string]any) toolResult {
	projectPath, ok := stringArg(args, "path")
	if !ok {
		return errorToolResult(`tool "workspace_env" requires string argument "path"`, nil)
	}
	projectPath, err := s.scopedProjectPath(projectPath)
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}

	workspaceName, created, err := s.app.ResolveOrCreateWorkspaceByProjectPath(projectPath)
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}
	envMap, workDir, err := s.app.WorkspaceEnvMap(workspaceName)
	if err != nil {
		return errorToolResult(err.Error(), map[string]any{
			"workspace_name": workspaceName,
			"created":        created,
		})
	}

	result := workspaceEnvResult{
		Created:       created,
		WorkspaceName: workspaceName,
		WorkDir:       workDir,
		Env:           envMap,
	}
	return successToolResult(
		fmt.Sprintf("Workspace %q environment loaded.", workspaceName),
		result,
	)
}

func (s *Server) workspaceAttachTool(args map[string]any) toolResult {
	projectPath, ok := stringArg(args, "path")
	if !ok {
		return errorToolResult(`tool "workspace_attach" requires string argument "path"`, nil)
	}
	projectPath, err := s.scopedProjectPath(projectPath)
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}
	toolchains, err := stringSliceArg(args, "toolchains")
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}
	if len(toolchains) == 0 {
		return errorToolResult(`tool "workspace_attach" requires non-empty "toolchains"`, nil)
	}

	workspaceName, created, err := s.app.ResolveOrCreateWorkspaceByProjectPath(projectPath)
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}
	if err := s.app.AttachToWorkspace(workspaceName, toolchains); err != nil {
		return errorToolResult(err.Error(), map[string]any{
			"workspace_name": workspaceName,
			"created":        created,
		})
	}
	report, err := s.app.InspectWorkspaceRuntimeOwnership(workspaceName)
	if err != nil {
		return errorToolResult(err.Error(), map[string]any{
			"workspace_name": workspaceName,
			"created":        created,
		})
	}

	attached := make([]app.Component, 0, len(toolchains))
	for _, spec := range toolchains {
		name, version, ok := stringsCutSpec(spec)
		if ok {
			attached = append(attached, app.Component{Name: name, Version: version})
		}
	}

	result := workspaceAttachResult{
		Created:  created,
		Attached: attached,
		Status:   app.WorkspaceRuntimeSnapshotFor(report),
	}
	return successToolResult(
		fmt.Sprintf("Attached %d toolchains to workspace %q.", len(attached), workspaceName),
		result,
	)
}

func (s *Server) workspaceInstallTool(args map[string]any) toolResult {
	projectPath, ok := stringArg(args, "path")
	if !ok {
		return errorToolResult(`tool "workspace_install" requires string argument "path"`, nil)
	}
	projectPath, err := s.scopedProjectPath(projectPath)
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}

	workspaceName, created, err := s.app.ResolveOrCreateWorkspaceByProjectPath(projectPath)
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}
	if err := s.app.InstallToWorkspace(workspaceName); err != nil {
		return errorToolResult(err.Error(), map[string]any{
			"workspace_name": workspaceName,
			"created":        created,
		})
	}
	report, err := s.app.InspectWorkspaceRuntimeOwnership(workspaceName)
	if err != nil {
		return errorToolResult(err.Error(), map[string]any{
			"workspace_name": workspaceName,
			"created":        created,
		})
	}

	result := workspaceInstallResult{
		Created:   created,
		Installed: append([]app.Component{}, report.Installed...),
		Status:    app.WorkspaceRuntimeSnapshotFor(report),
	}
	return successToolResult(
		fmt.Sprintf("Installed attached toolchains for workspace %q.", workspaceName),
		result,
	)
}

func (s *Server) workspaceExportTool(args map[string]any) toolResult {
	projectPath, ok := stringArg(args, "path")
	if !ok {
		return errorToolResult(`tool "workspace_export" requires string argument "path"`, nil)
	}
	projectPath, err := s.scopedProjectPath(projectPath)
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}

	exported, err := s.app.ExportWorkspaceByProjectPath(projectPath)
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}

	result := workspaceExportResult{
		Export: exported,
	}
	return successToolResult(
		fmt.Sprintf("Workspace %q exported.", exported.Workspace.Name),
		result,
	)
}

func (s *Server) workspaceImportTool(args map[string]any) toolResult {
	projectPath, ok := stringArg(args, "path")
	if !ok {
		return errorToolResult(`tool "workspace_import" requires string argument "path"`, nil)
	}
	projectPath, err := s.scopedProjectPath(projectPath)
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}

	exportValue, ok := args["export"]
	if !ok {
		return errorToolResult(`tool "workspace_import" requires object argument "export"`, nil)
	}
	exported, err := workspaceExportFromArg(exportValue)
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}

	imported, err := s.app.ImportWorkspaceAs(
		exported,
		projectPath,
		stringArgOrDefault(args, "workspace_name", ""),
		boolArgOrDefault(args, "install_attached", false),
	)
	if err != nil {
		return errorToolResult(err.Error(), nil)
	}

	result := workspaceImportResult{
		Created:       imported.Created,
		WorkspaceName: imported.WorkspaceName,
		ProjectPath:   imported.ProjectPath,
		Status:        app.WorkspaceRuntimeSnapshotFor(imported.Status),
	}
	return successToolResult(
		fmt.Sprintf("Workspace %q imported for %s.", imported.WorkspaceName, imported.ProjectPath),
		result,
	)
}

func makeWorkspaceInspection(inspect app.WorkspaceInspection) workspaceInspection {
	return workspaceInspection{
		WorkspaceName: inspect.WorkspaceName,
		WorkspaceDir:  inspect.WorkspaceDir,
		ManifestPath:  inspect.ManifestPath,
		HomeDir:       inspect.HomeDir,
		StateDir:      inspect.StateDir,
		LogsDir:       inspect.LogsDir,
		Manifest:      inspect.Manifest,
		Runtime:       app.WorkspaceRuntimeSnapshotFor(inspect.Runtime),
	}
}

func successToolResult(text string, structured any) toolResult {
	return toolResult{
		Content:           []toolContent{{Type: "text", Text: text}},
		StructuredContent: structured,
	}
}

func errorToolResult(text string, structured any) toolResult {
	return toolResult{
		Content:           []toolContent{{Type: "text", Text: text}},
		StructuredContent: structured,
		IsError:           true,
	}
}

func stringArg(args map[string]any, key string) (string, bool) {
	if args == nil {
		return "", false
	}
	raw, ok := args[key]
	if !ok {
		return "", false
	}
	value, ok := raw.(string)
	if !ok || value == "" {
		return "", false
	}
	return value, true
}

func boolArgOrDefault(args map[string]any, key string, fallback bool) bool {
	if args == nil {
		return fallback
	}
	raw, ok := args[key]
	if !ok {
		return fallback
	}
	value, ok := raw.(bool)
	if !ok {
		return fallback
	}
	return value
}

func stringArgOrDefault(args map[string]any, key string, fallback string) string {
	if args == nil {
		return fallback
	}
	raw, ok := args[key]
	if !ok {
		return fallback
	}
	value, ok := raw.(string)
	if !ok {
		return fallback
	}
	return value
}

func stringSliceArg(args map[string]any, key string) ([]string, error) {
	if args == nil {
		return nil, nil
	}
	raw, ok := args[key]
	if !ok {
		return nil, nil
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf(`tool input requires "%s" to be an array of strings`, key)
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf(`tool input requires "%s" to be an array of strings`, key)
		}
		result = append(result, s)
	}
	return result, nil
}

func workspaceExportFromArg(raw any) (app.WorkspaceExport, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return app.WorkspaceExport{}, fmt.Errorf("marshal workspace export input: %w", err)
	}
	var exported app.WorkspaceExport
	if err := json.Unmarshal(data, &exported); err != nil {
		return app.WorkspaceExport{}, fmt.Errorf("parse workspace export input: %w", err)
	}
	if exported.SchemaVersion != 0 || exported.Workspace.Name != "" {
		return exported, nil
	}

	var payload app.WorkspaceExportPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return app.WorkspaceExport{}, fmt.Errorf("parse workspace export input: %w", err)
	}
	if payload.Name == "" {
		return app.WorkspaceExport{}, fmt.Errorf("workspace export name required")
	}
	return app.WorkspaceExport{
		SchemaVersion: 1,
		Workspace:     payload,
	}, nil
}

func stringsCutSpec(spec string) (string, string, bool) {
	name, version, ok := strings.Cut(spec, "@")
	if !ok || name == "" || version == "" {
		return "", "", false
	}
	return name, version, true
}

func formatDetectedToolchains(detected []app.DetectedToolchain) string {
	parts := make([]string, 0, len(detected))
	for _, tc := range detected {
		if tc.Version != "" {
			parts = append(parts, fmt.Sprintf("%s@%s", tc.Name, tc.Version))
			continue
		}
		parts = append(parts, tc.Name)
	}
	return joinComma(parts)
}

func joinComma(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	default:
		result := parts[0]
		for _, part := range parts[1:] {
			result += ", " + part
		}
		return result
	}
}

func bytesTrimSpace(value []byte) []byte {
	for len(value) > 0 && isSpace(value[0]) {
		value = value[1:]
	}
	for len(value) > 0 && isSpace(value[len(value)-1]) {
		value = value[:len(value)-1]
	}
	return value
}

func isSpace(b byte) bool {
	switch b {
	case ' ', '\n', '\r', '\t':
		return true
	default:
		return false
	}
}

func responseID(id json.RawMessage) json.RawMessage {
	if len(id) == 0 {
		return json.RawMessage("null")
	}
	return id
}

func workspaceResourceURI(workspaceName, kind string) string {
	return fmt.Sprintf("groot://workspace/%s/%s", url.PathEscape(workspaceName), kind)
}

func parseWorkspaceResourceURI(raw string) (workspaceName, kind string, err error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", fmt.Errorf("invalid resource uri %q", raw)
	}
	if u.Scheme != "groot" || u.Host != "workspace" {
		return "", "", fmt.Errorf("unsupported resource uri %q", raw)
	}

	parts := strings.Split(strings.TrimPrefix(u.EscapedPath(), "/"), "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unsupported resource uri %q", raw)
	}

	workspaceName, err = url.PathUnescape(parts[0])
	if err != nil || strings.TrimSpace(workspaceName) == "" {
		return "", "", fmt.Errorf("invalid resource uri %q", raw)
	}
	kind, err = url.PathUnescape(parts[1])
	if err != nil || strings.TrimSpace(kind) == "" {
		return "", "", fmt.Errorf("invalid resource uri %q", raw)
	}

	return workspaceName, kind, nil
}

func marshalResponse(response rpcResponse) ([]byte, error) {
	data, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("marshal mcp response: %w", err)
	}
	return data, nil
}
