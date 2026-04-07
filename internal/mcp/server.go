package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"github.com/totoual/groot/internal/app"
)

const ProtocolVersion = "2025-06-18"

type Server struct {
	app *app.App
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

type workspaceStatus struct {
	WorkspaceName       string                  `json:"workspace_name"`
	ProjectPath         string                  `json:"project_path,omitempty"`
	Status              string                  `json:"status"`
	Detected            []app.DetectedToolchain `json:"detected"`
	Attached            []app.Component         `json:"attached"`
	Installed           []app.Component         `json:"installed"`
	AttachedUninstalled []app.Component         `json:"attached_uninstalled"`
	Missing             []app.DetectedToolchain `json:"missing"`
}

type workspaceStatusResult struct {
	Created bool            `json:"created"`
	Status  workspaceStatus `json:"status"`
}

type workspaceSetupResult struct {
	Created bool                     `json:"created"`
	Plan    app.FirstOpenRuntimePlan `json:"plan"`
	Status  workspaceStatus          `json:"status"`
}

type workspaceExecResult struct {
	Created    bool            `json:"created"`
	Workspace  workspaceStatus `json:"workspace"`
	Command    string          `json:"command"`
	Args       []string        `json:"args"`
	WorkDir    string          `json:"workdir"`
	Stdout     string          `json:"stdout,omitempty"`
	Stderr     string          `json:"stderr,omitempty"`
	ExitCode   int             `json:"exit_code"`
	Warnings   []string        `json:"warnings,omitempty"`
	StrictMode bool            `json:"strict_mode"`
}

func NewServer(a *app.App) *Server {
	return &Server{app: a}
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
					"tools": map[string]any{},
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
	}
}

func (s *Server) callTool(params toolCallParams) toolResult {
	switch params.Name {
	case "workspace_status":
		return s.workspaceStatusTool(params.Arguments)
	case "workspace_setup":
		return s.workspaceSetupTool(params.Arguments)
	case "workspace_exec":
		return s.workspaceExecTool(params.Arguments)
	default:
		return errorToolResult(fmt.Sprintf("unknown tool %q", params.Name), nil)
	}
}

func (s *Server) workspaceStatusTool(args map[string]any) toolResult {
	projectPath, ok := stringArg(args, "path")
	if !ok {
		return errorToolResult(`tool "workspace_status" requires string argument "path"`, nil)
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
		Status:  makeWorkspaceStatus(report),
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
		Status:  makeWorkspaceStatus(report),
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
					Workspace:  makeWorkspaceStatus(report),
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
			Workspace:  makeWorkspaceStatus(report),
			Command:    command,
			Args:       commandArgs,
			Warnings:   warnings,
			StrictMode: app.RuntimeStrictModeEnabled(),
		})
	}

	result := workspaceExecResult{
		Created:    created,
		Workspace:  makeWorkspaceStatus(report),
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

func makeWorkspaceStatus(report app.WorkspaceRuntimeOwnership) workspaceStatus {
	return workspaceStatus{
		WorkspaceName:       report.WorkspaceName,
		ProjectPath:         report.ProjectPath,
		Status:              app.RuntimeOwnershipStatusLabel(report),
		Detected:            append([]app.DetectedToolchain{}, report.Detected...),
		Attached:            append([]app.Component{}, report.Attached...),
		Installed:           append([]app.Component{}, report.Installed...),
		AttachedUninstalled: append([]app.Component{}, report.Uninstalled...),
		Missing:             append([]app.DetectedToolchain{}, report.Missing...),
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
		return nil, fmt.Errorf(`tool "workspace_exec" requires "args" to be an array of strings`)
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf(`tool "workspace_exec" requires "args" to be an array of strings`)
		}
		result = append(result, s)
	}
	return result, nil
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

func marshalResponse(response rpcResponse) ([]byte, error) {
	data, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("marshal mcp response: %w", err)
	}
	return data, nil
}
