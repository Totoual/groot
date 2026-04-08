package app

import (
	"path/filepath"
	"strings"
)

type WorkspaceInspection struct {
	WorkspaceName string                    `json:"workspace_name"`
	WorkspaceDir  string                    `json:"workspace_dir"`
	ManifestPath  string                    `json:"manifest_path"`
	HomeDir       string                    `json:"home_dir"`
	StateDir      string                    `json:"state_dir"`
	LogsDir       string                    `json:"logs_dir"`
	Manifest      Manifest                  `json:"manifest"`
	Runtime       WorkspaceRuntimeOwnership `json:"runtime"`
}

func (a *App) InspectWorkspace(name string) (WorkspaceInspection, error) {
	wsPath, err := a.EnsureWorkspace(name)
	if err != nil {
		return WorkspaceInspection{}, err
	}

	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return WorkspaceInspection{}, err
	}

	runtime, err := a.InspectWorkspaceRuntimeOwnership(name)
	if err != nil {
		return WorkspaceInspection{}, err
	}

	return WorkspaceInspection{
		WorkspaceName: name,
		WorkspaceDir:  wsPath,
		ManifestPath:  filepath.Join(wsPath, "manifest.json"),
		HomeDir:       filepath.Join(wsPath, "home"),
		StateDir:      filepath.Join(wsPath, "state"),
		LogsDir:       filepath.Join(wsPath, "logs"),
		Manifest:      manifest,
		Runtime:       runtime,
	}, nil
}

func (a *App) WorkspaceEnvMap(name string) (map[string]string, string, error) {
	env, workDir, err := a.workspaceRuntime(name)
	if err != nil {
		return nil, "", err
	}

	return exportedWorkspaceEnvMap(env, workDir), workDir, nil
}

func exportedWorkspaceEnvMap(env []string, workDir string) map[string]string {
	envMap := make(map[string]string, len(env)+1)
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			continue
		}
		if key == "PS1" || key == "PROMPT" {
			continue
		}
		if !exportedWorkspaceEnvKey(key) {
			continue
		}
		envMap[key] = value
	}
	envMap["GROOT_WORKDIR"] = workDir
	return envMap
}

func exportedWorkspaceEnvKey(key string) bool {
	switch key {
	case "HOME", "PATH", "SHELL", "LANG", "TMPDIR":
		return true
	}

	if strings.HasPrefix(key, "GROOT_") || strings.HasPrefix(key, "XDG_") || strings.HasPrefix(key, "LC_") {
		return true
	}

	if strings.HasSuffix(key, "_HOME") || strings.HasSuffix(key, "_ROOT") {
		return true
	}

	return false
}
