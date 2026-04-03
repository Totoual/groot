package app

type WorkspaceRuntimeOwnership struct {
	WorkspaceName string
	Detected      []DetectedToolchain
	Missing       []DetectedToolchain
}

type FirstOpenRuntimePlan struct {
	WorkspaceName   string
	Detected        []DetectedToolchain
	Attached        []DetectedToolchain
	Skipped         []DetectedToolchain
	Missing         []DetectedToolchain
	AttachRequested bool
}

func (a *App) InspectWorkspaceRuntimeOwnership(name string) (WorkspaceRuntimeOwnership, error) {
	wsPath, err := a.EnsureWorkspace(name)
	if err != nil {
		return WorkspaceRuntimeOwnership{}, err
	}

	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return WorkspaceRuntimeOwnership{}, err
	}
	if manifest.ProjectPath == "" {
		return WorkspaceRuntimeOwnership{WorkspaceName: name}, nil
	}

	detected, err := a.DetectProjectToolchains(manifest.ProjectPath)
	if err != nil {
		return WorkspaceRuntimeOwnership{}, err
	}
	missing, err := a.MissingWorkspaceToolchains(name, detected)
	if err != nil {
		return WorkspaceRuntimeOwnership{}, err
	}

	return WorkspaceRuntimeOwnership{
		WorkspaceName: name,
		Detected:      detected,
		Missing:       missing,
	}, nil
}

func (a *App) BuildFirstOpenRuntimePlan(name, projectPath string, attachDetected bool) (FirstOpenRuntimePlan, error) {
	detected, err := a.DetectProjectToolchains(projectPath)
	if err != nil {
		return FirstOpenRuntimePlan{}, err
	}

	plan := FirstOpenRuntimePlan{
		WorkspaceName:   name,
		Detected:        detected,
		AttachRequested: attachDetected,
	}
	if len(detected) == 0 {
		return plan, nil
	}

	if attachDetected {
		attached, skipped, err := a.AttachDetectedToolchains(name, detected)
		if err != nil {
			return FirstOpenRuntimePlan{}, err
		}
		plan.Attached = attached
		plan.Skipped = skipped
	}

	missing, err := a.MissingWorkspaceToolchains(name, detected)
	if err != nil {
		return FirstOpenRuntimePlan{}, err
	}
	plan.Missing = missing

	return plan, nil
}
