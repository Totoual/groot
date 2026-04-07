package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/totoual/groot/internal/app"
	"github.com/totoual/groot/internal/mcp"
)

type MCPCmd struct{}

func (c *MCPCmd) Name() string { return "mcp" }

func (c *MCPCmd) Help() string {
	return "Run the Groot MCP server over stdio"
}

func (c *MCPCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	var projectScopes stringListFlag
	var workspaceScopes stringListFlag
	fs.Var(&projectScopes, "project", "limit MCP access to a project path; may be provided multiple times")
	fs.Var(&workspaceScopes, "workspace", "limit MCP access to a bound workspace; may be provided multiple times")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot mcp [--project path ...] [--workspace name ...]")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return fmt.Errorf("mcp does not accept positional arguments")
	}

	allowedProjects, err := mcpAllowedProjects(a, projectScopes, workspaceScopes)
	if err != nil {
		return err
	}

	server := mcp.NewServer(a)
	if len(allowedProjects) > 0 {
		server = mcp.NewScopedServer(a, allowedProjects)
	}
	return server.Serve(os.Stdin, os.Stdout)
}

type stringListFlag []string

func (s *stringListFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringListFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("value required")
	}
	*s = append(*s, value)
	return nil
}

func mcpAllowedProjects(a *app.App, projectScopes, workspaceScopes []string) ([]string, error) {
	allowedProjects := make([]string, 0, len(projectScopes)+len(workspaceScopes))

	for _, projectPath := range projectScopes {
		normalized, err := app.NormalizeProjectPath(projectPath)
		if err != nil {
			return nil, err
		}
		allowedProjects = appendUniqueProjectPath(allowedProjects, normalized)
	}

	for _, workspaceName := range workspaceScopes {
		inspect, err := a.InspectWorkspace(workspaceName)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(inspect.Manifest.ProjectPath) == "" {
			return nil, fmt.Errorf("workspace %q is not bound to a project path", workspaceName)
		}
		allowedProjects = appendUniqueProjectPath(allowedProjects, inspect.Manifest.ProjectPath)
	}

	return allowedProjects, nil
}

func appendUniqueProjectPath(paths []string, candidate string) []string {
	for _, existing := range paths {
		if existing == candidate {
			return paths
		}
	}
	return append(paths, candidate)
}
