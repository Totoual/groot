package commands

import (
	"fmt"
	"os"

	"github.com/totoual/groot/internal/app"
)

func resolveProjectWorkspace(a *app.App, projectPath string) (string, error) {
	workspaceName, created, err := a.ResolveOrCreateWorkspaceByProjectPath(projectPath)
	if err != nil {
		return "", fmt.Errorf("couldn't resolve workspace for project path: %w", err)
	}
	if created {
		fmt.Fprintf(os.Stderr, "Created workspace %q for %s\n", workspaceName, projectPath)
	}
	return workspaceName, nil
}
