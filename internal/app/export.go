package app

import "time"

type WorkspaceExport struct {
	SchemaVersion int                    `json:"schema_version"`
	ExportedAt    time.Time              `json:"exported_at"`
	Workspace     WorkspaceExportPayload `json:"workspace"`
}

type WorkspaceExportPayload struct {
	Name        string                 `json:"name"`
	ProjectPath string                 `json:"project_path,omitempty"`
	Manifest    Manifest               `json:"manifest"`
	Runtime     WorkspaceExportRuntime `json:"runtime"`
}

type WorkspaceExportRuntime struct {
	Status              string              `json:"status"`
	Detected            []DetectedToolchain `json:"detected"`
	Attached            []Component         `json:"attached"`
	Installed           []Component         `json:"installed"`
	AttachedUninstalled []Component         `json:"attached_uninstalled"`
	Missing             []DetectedToolchain `json:"missing"`
}

func (a *App) ExportWorkspace(name string) (WorkspaceExport, error) {
	wsPath, err := a.EnsureWorkspace(name)
	if err != nil {
		return WorkspaceExport{}, err
	}

	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return WorkspaceExport{}, err
	}
	runtime, err := a.InspectWorkspaceRuntimeOwnership(name)
	if err != nil {
		return WorkspaceExport{}, err
	}

	return WorkspaceExport{
		SchemaVersion: 1,
		ExportedAt:    time.Now().UTC(),
		Workspace: WorkspaceExportPayload{
			Name:        name,
			ProjectPath: manifest.ProjectPath,
			Manifest:    manifest,
			Runtime: WorkspaceExportRuntime{
				Status:              RuntimeOwnershipStatusLabel(runtime),
				Detected:            append([]DetectedToolchain{}, runtime.Detected...),
				Attached:            append([]Component{}, runtime.Attached...),
				Installed:           append([]Component{}, runtime.Installed...),
				AttachedUninstalled: append([]Component{}, runtime.Uninstalled...),
				Missing:             append([]DetectedToolchain{}, runtime.Missing...),
			},
		},
	}, nil
}

func (a *App) ExportWorkspaceByProjectPath(projectPath string) (WorkspaceExport, error) {
	name, err := a.FindWorkspaceByProjectPath(projectPath)
	if err != nil {
		return WorkspaceExport{}, err
	}
	return a.ExportWorkspace(name)
}
