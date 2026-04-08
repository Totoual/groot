package app

type WorkspaceRuntimeSnapshot struct {
	WorkspaceName       string              `json:"workspace_name"`
	ProjectPath         string              `json:"project_path,omitempty"`
	Status              string              `json:"status"`
	Detected            []DetectedToolchain `json:"detected"`
	Attached            []Component         `json:"attached"`
	Installed           []Component         `json:"installed"`
	AttachedUninstalled []Component         `json:"attached_uninstalled"`
	Missing             []DetectedToolchain `json:"missing"`
}

func WorkspaceRuntimeSnapshotFor(report WorkspaceRuntimeOwnership) WorkspaceRuntimeSnapshot {
	return WorkspaceRuntimeSnapshot{
		WorkspaceName:       report.WorkspaceName,
		ProjectPath:         report.ProjectPath,
		Status:              RuntimeOwnershipStatusLabel(report),
		Detected:            append([]DetectedToolchain{}, report.Detected...),
		Attached:            append([]Component{}, report.Attached...),
		Installed:           append([]Component{}, report.Installed...),
		AttachedUninstalled: append([]Component{}, report.Uninstalled...),
		Missing:             append([]DetectedToolchain{}, report.Missing...),
	}
}
