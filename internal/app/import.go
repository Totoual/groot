package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type WorkspaceImportResult struct {
	Created       bool
	WorkspaceName string
	ProjectPath   string
	Status        WorkspaceRuntimeOwnership
}

func (a *App) ImportWorkspace(exported WorkspaceExport, projectPath string, installAttached bool) (WorkspaceImportResult, error) {
	if exported.SchemaVersion != 1 {
		return WorkspaceImportResult{}, fmt.Errorf("unsupported export schema version %d", exported.SchemaVersion)
	}

	return a.ImportWorkspaceAs(exported, projectPath, "", installAttached)
}

func (a *App) ImportWorkspaceAs(exported WorkspaceExport, projectPath, workspaceNameOverride string, installAttached bool) (WorkspaceImportResult, error) {
	if exported.SchemaVersion != 1 {
		return WorkspaceImportResult{}, fmt.Errorf("unsupported export schema version %d", exported.SchemaVersion)
	}

	workspaceName := strings.TrimSpace(exported.Workspace.Name)
	if strings.TrimSpace(workspaceNameOverride) != "" {
		workspaceName = strings.TrimSpace(workspaceNameOverride)
	}
	if workspaceName == "" {
		return WorkspaceImportResult{}, fmt.Errorf("workspace export name required")
	}

	normalizedPath, err := NormalizeProjectPath(projectPath)
	if err != nil {
		return WorkspaceImportResult{}, err
	}
	info, err := os.Stat(normalizedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return WorkspaceImportResult{}, fmt.Errorf("project path %q does not exist", normalizedPath)
		}
		return WorkspaceImportResult{}, fmt.Errorf("stat project path %q: %w", normalizedPath, err)
	}
	if !info.IsDir() {
		return WorkspaceImportResult{}, fmt.Errorf("project path %q is not a directory", normalizedPath)
	}

	if boundWorkspace, err := a.FindWorkspaceByProjectPath(normalizedPath); err == nil {
		if boundWorkspace != workspaceName {
			return WorkspaceImportResult{}, fmt.Errorf("project path %q is already bound to workspace %q", normalizedPath, boundWorkspace)
		}
	} else if !errors.Is(err, errWorkspaceNotBoundToProjectPath) {
		return WorkspaceImportResult{}, err
	}

	wsPath, created, err := a.ensureWorkspaceForImport(workspaceName, normalizedPath)
	if err != nil {
		return WorkspaceImportResult{}, err
	}

	manifest := exported.Workspace.Manifest
	manifest.Name = workspaceName
	manifest.ProjectPath = normalizedPath
	if manifest.Packages == nil {
		manifest.Packages = []Component{}
	}
	if manifest.Services == nil {
		manifest.Services = []Component{}
	}
	if manifest.Env == nil {
		manifest.Env = map[string]string{}
	}

	if err := a.writeManifest(wsPath, manifest); err != nil {
		return WorkspaceImportResult{}, err
	}
	if installAttached {
		if err := a.InstallToWorkspace(workspaceName); err != nil {
			return WorkspaceImportResult{}, err
		}
	}

	status, err := a.InspectWorkspaceRuntimeOwnership(workspaceName)
	if err != nil {
		return WorkspaceImportResult{}, err
	}

	return WorkspaceImportResult{
		Created:       created,
		WorkspaceName: workspaceName,
		ProjectPath:   normalizedPath,
		Status:        status,
	}, nil
}

func (a *App) ensureWorkspaceForImport(name, projectPath string) (string, bool, error) {
	if err := a.Init(); err != nil {
		return "", false, err
	}

	wsPath := filepath.Join(a.WorkspaceDir(), name)
	info, err := os.Stat(wsPath)
	switch {
	case err == nil:
		if !info.IsDir() {
			return "", false, fmt.Errorf("workspace path %q is not a directory", wsPath)
		}
		manifest, err := a.getManifest(wsPath)
		if err != nil {
			return "", false, err
		}
		if strings.TrimSpace(manifest.ProjectPath) != "" {
			match, err := ProjectPathsMatch(manifest.ProjectPath, projectPath)
			if err != nil {
				return "", false, err
			}
			if !match {
				return "", false, fmt.Errorf("workspace %q is already bound to project path %q", name, manifest.ProjectPath)
			}
		}
		return wsPath, false, nil
	case os.IsNotExist(err):
		if err := a.CreateNewWorkspace(name); err != nil {
			return "", false, err
		}
		wsPath, err := a.EnsureWorkspace(name)
		if err != nil {
			return "", false, err
		}
		return wsPath, true, nil
	default:
		return "", false, fmt.Errorf("stat workspace %q: %w", name, err)
	}
}
