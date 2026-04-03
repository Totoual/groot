package app

import "os"

type WorkspaceRuntimeOwnership struct {
	WorkspaceName string
	ProjectPath   string
	Detected      []DetectedToolchain
	Attached      []Component
	Installed     []Component
	Uninstalled   []Component
	Missing       []DetectedToolchain
}

type FirstOpenRuntimePlan struct {
	WorkspaceName    string
	Detected         []DetectedToolchain
	Attached         []DetectedToolchain
	Installed        []DetectedToolchain
	Skipped          []DetectedToolchain
	Missing          []DetectedToolchain
	AttachRequested  bool
	InstallRequested bool
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
		return WorkspaceRuntimeOwnership{
			WorkspaceName: name,
			Attached:      append([]Component{}, manifest.Packages...),
		}, nil
	}

	detected, err := a.DetectProjectToolchains(manifest.ProjectPath)
	if err != nil {
		return WorkspaceRuntimeOwnership{}, err
	}
	missing, err := a.MissingWorkspaceToolchains(name, detected)
	if err != nil {
		return WorkspaceRuntimeOwnership{}, err
	}
	installed, uninstalled := a.partitionInstalledComponents(manifest.Packages)

	return WorkspaceRuntimeOwnership{
		WorkspaceName: name,
		ProjectPath:   manifest.ProjectPath,
		Detected:      detected,
		Attached:      append([]Component{}, manifest.Packages...),
		Installed:     installed,
		Uninstalled:   uninstalled,
		Missing:       missing,
	}, nil
}

func (a *App) BuildFirstOpenRuntimePlan(name, projectPath string, attachDetected, installDetected bool) (FirstOpenRuntimePlan, error) {
	detected, err := a.DetectProjectToolchains(projectPath)
	if err != nil {
		return FirstOpenRuntimePlan{}, err
	}

	plan := FirstOpenRuntimePlan{
		WorkspaceName:    name,
		Detected:         detected,
		AttachRequested:  attachDetected,
		InstallRequested: installDetected,
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
	if installDetected {
		if err := a.InstallToWorkspace(name); err != nil {
			return FirstOpenRuntimePlan{}, err
		}
		plan.Installed = append([]DetectedToolchain{}, plan.Attached...)
	}

	missing, err := a.MissingWorkspaceToolchains(name, detected)
	if err != nil {
		return FirstOpenRuntimePlan{}, err
	}
	plan.Missing = missing

	return plan, nil
}

func (a *App) partitionInstalledComponents(components []Component) ([]Component, []Component) {
	installed := make([]Component, 0, len(components))
	uninstalled := make([]Component, 0, len(components))

	for _, comp := range components {
		if a.toolchainInstalled(comp) {
			installed = append(installed, comp)
			continue
		}
		uninstalled = append(uninstalled, comp)
	}

	return installed, uninstalled
}

func (a *App) toolchainInstalled(tc Component) bool {
	binDir, err := a.toolchainBinDir(tc)
	if err != nil {
		return false
	}
	info, err := os.Stat(binDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}
