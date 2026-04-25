package app

import (
	"fmt"
	"strings"
)

func (a *App) DeclareTask(workspaceName string, spec TaskSpec) error {
	if err := validateTaskSpec(spec); err != nil {
		return err
	}

	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return err
	}
	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return err
	}

	updated := false
	for i := range manifest.Tasks {
		if manifest.Tasks[i].Name == spec.Name {
			manifest.Tasks[i] = cloneTaskSpec(spec)
			updated = true
			break
		}
	}
	if !updated {
		manifest.Tasks = append(manifest.Tasks, cloneTaskSpec(spec))
	}

	return a.writeManifest(wsPath, manifest)
}

func (a *App) DeleteTask(workspaceName, taskName string) error {
	if err := validateTaskName(taskName); err != nil {
		return err
	}

	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return err
	}
	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return err
	}

	for i := range manifest.Tasks {
		if manifest.Tasks[i].Name != taskName {
			continue
		}
		manifest.Tasks = append(manifest.Tasks[:i], manifest.Tasks[i+1:]...)
		return a.writeManifest(wsPath, manifest)
	}

	return fmt.Errorf("declared task %q not found", taskName)
}

func (a *App) DeclaredTasks(workspaceName string) ([]TaskSpec, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return nil, err
	}
	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return nil, err
	}

	tasks := make([]TaskSpec, 0, len(manifest.Tasks))
	for _, spec := range manifest.Tasks {
		tasks = append(tasks, cloneTaskSpec(spec))
	}
	return tasks, nil
}

func (a *App) DeclareService(workspaceName string, spec ServiceSpec) error {
	if err := validateServiceSpec(spec); err != nil {
		return err
	}

	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return err
	}
	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return err
	}

	updated := false
	for i := range manifest.Services {
		if manifest.Services[i].Name == spec.Name {
			manifest.Services[i] = cloneServiceSpec(spec)
			updated = true
			break
		}
	}
	if !updated {
		manifest.Services = append(manifest.Services, cloneServiceSpec(spec))
	}

	return a.writeManifest(wsPath, manifest)
}

func (a *App) DeleteService(workspaceName, serviceName string) error {
	if err := validateServiceName(serviceName); err != nil {
		return err
	}

	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return err
	}
	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return err
	}

	for i := range manifest.Services {
		if manifest.Services[i].Name != serviceName {
			continue
		}
		manifest.Services = append(manifest.Services[:i], manifest.Services[i+1:]...)
		return a.writeManifest(wsPath, manifest)
	}

	return fmt.Errorf("declared service %q not found", serviceName)
}

func (a *App) DeclaredServices(workspaceName string) ([]ServiceSpec, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return nil, err
	}
	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return nil, err
	}

	services := make([]ServiceSpec, 0, len(manifest.Services))
	for _, spec := range manifest.Services {
		services = append(services, cloneServiceSpec(spec))
	}
	return services, nil
}

func validateTaskSpec(spec TaskSpec) error {
	if err := validateTaskName(spec.Name); err != nil {
		return err
	}
	if len(spec.Command) == 0 {
		return fmt.Errorf("task %q requires a command", spec.Name)
	}
	if strings.TrimSpace(spec.Command[0]) == "" {
		return fmt.Errorf("task %q requires a command", spec.Name)
	}
	return nil
}

func validateTaskName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." || strings.Contains(name, "/") {
		return fmt.Errorf("invalid task name %q", name)
	}
	return nil
}

func validateServiceSpec(spec ServiceSpec) error {
	if err := validateServiceName(spec.Name); err != nil {
		return err
	}
	if len(spec.Command) == 0 {
		return fmt.Errorf("service %q requires a command", spec.Name)
	}
	if strings.TrimSpace(spec.Command[0]) == "" {
		return fmt.Errorf("service %q requires a command", spec.Name)
	}
	return nil
}

func cloneTaskSpec(spec TaskSpec) TaskSpec {
	return TaskSpec{
		Name:    spec.Name,
		Command: append([]string{}, spec.Command...),
		Cwd:     spec.Cwd,
	}
}

func cloneServiceSpec(spec ServiceSpec) ServiceSpec {
	return ServiceSpec{
		Name:    spec.Name,
		Command: append([]string{}, spec.Command...),
		Cwd:     spec.Cwd,
		Restart: spec.Restart,
		Version: spec.Version,
	}
}
