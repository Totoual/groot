package app

type WorkspaceSyncReport struct {
	WorkspaceName  string      `json:"workspace_name"`
	ProjectPath    string      `json:"project_path"`
	Before         IndexStatus `json:"before"`
	After          IndexStatus `json:"after"`
	UpdatedIndex   bool        `json:"updated_index"`
	VaultStats     VaultStats  `json:"vault_stats"`
	LatestTask     *VaultNode  `json:"latest_task,omitempty"`
	LatestProgress *VaultNode  `json:"latest_progress,omitempty"`
}

func (a *App) SyncWorkspace(workspaceName string) (WorkspaceSyncReport, error) {
	inspection, err := a.InspectWorkspace(workspaceName)
	if err != nil {
		return WorkspaceSyncReport{}, err
	}

	before, err := a.IndexStatus(workspaceName)
	if err != nil {
		return WorkspaceSyncReport{}, err
	}
	report := WorkspaceSyncReport{
		WorkspaceName: workspaceName,
		ProjectPath:   inspection.Manifest.ProjectPath,
		Before:        before,
		After:         before,
	}

	if !before.Fresh {
		if _, err := a.UpdateIndex(workspaceName); err != nil {
			return WorkspaceSyncReport{}, err
		}
		report.UpdatedIndex = true
		after, err := a.IndexStatus(workspaceName)
		if err != nil {
			return WorkspaceSyncReport{}, err
		}
		report.After = after
	}

	vaultStats, err := a.VaultStats(workspaceName)
	if err != nil {
		return WorkspaceSyncReport{}, err
	}
	recentNodes, err := a.VaultRecent(workspaceName, VaultRecentOptions{Limit: 12})
	if err != nil {
		return WorkspaceSyncReport{}, err
	}
	report.VaultStats = vaultStats
	report.LatestTask = firstVaultNodeOfType(recentNodes, VaultNodeTypeTask)
	report.LatestProgress = firstVaultNodeOfType(recentNodes, VaultNodeTypeProgress)
	return report, nil
}

func firstVaultNodeOfType(nodes []VaultNode, nodeType string) *VaultNode {
	for _, node := range nodes {
		if node.Type == nodeType {
			copyNode := node
			return &copyNode
		}
	}
	return nil
}
