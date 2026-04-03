package commands

import (
	"fmt"
	"os"

	"github.com/totoual/groot/internal/app"
)

type resolvedProjectWorkspace struct {
	Name    string
	Created bool
}

func resolveProjectWorkspace(a *app.App, projectPath string) (resolvedProjectWorkspace, error) {
	workspaceName, created, err := a.ResolveOrCreateWorkspaceByProjectPath(projectPath)
	if err != nil {
		return resolvedProjectWorkspace{}, fmt.Errorf("couldn't resolve workspace for project path: %w", err)
	}
	if created {
		fmt.Fprintf(os.Stderr, "Created workspace %q for %s\n", workspaceName, projectPath)
	}
	return resolvedProjectWorkspace{
		Name:    workspaceName,
		Created: created,
	}, nil
}
