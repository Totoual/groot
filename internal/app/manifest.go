package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/totoual/groot/internal/itoolchain"
)

type Manifest struct {
	SchemaVersion int               `json:"schema_version"`
	CreatedAt     time.Time         `json:"created_at"`
	Name          string            `json:"name"`
	ProjectPath   string            `json:"project_path"`
	Packages      []PackageSpec     `json:"packages"`
	Tasks         []TaskSpec        `json:"tasks"`
	Services      []ServiceSpec     `json:"services"`
	Env           map[string]string `json:"env"`
}

type PackageSpec struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type TaskSpec struct {
	Name    string   `json:"name"`
	Command []string `json:"command"`
	Cwd     string   `json:"cwd,omitempty"`
}

type ServiceSpec struct {
	Name    string   `json:"name"`
	Command []string `json:"command,omitempty"`
	Cwd     string   `json:"cwd,omitempty"`
	Restart string   `json:"restart,omitempty"`
	Version string   `json:"version,omitempty"`
}

type Component = PackageSpec

func NewManifest(name string) *Manifest {
	return &Manifest{
		SchemaVersion: 1,
		CreatedAt:     time.Now(),
		Name:          name,
		ProjectPath:   "",
		Packages:      make([]PackageSpec, 0),
		Tasks:         make([]TaskSpec, 0),
		Services:      make([]ServiceSpec, 0),
		Env:           make(map[string]string),
	}
}

func (a *App) getManifest(wsPath string) (Manifest, error) {
	path := getManifestPath(wsPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	err = json.Unmarshal(data, &manifest)
	if err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}

func getManifestPath(wsPath string) string {
	return filepath.Join(wsPath, "manifest.json")
}

func (a *App) writeManifest(wsPath string, manifest Manifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	path := getManifestPath(wsPath)
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	return nil
}

func (a *App) parsePackageSpecs(args []string) ([]PackageSpec, error) {
	packages := make([]PackageSpec, 0)
	for _, arg := range args {
		name, version, ok := strings.Cut(strings.TrimSpace(arg), "@")
		if !ok {
			return nil, fmt.Errorf("invalid tool spec %q: expected name@version", arg)
		}
		name = strings.ToLower(strings.TrimSpace(name))
		version = strings.TrimSpace(version)
		if name == "" {
			return nil, fmt.Errorf("invalid tool spec %q: tool name required", arg)
		}
		if version == "" {
			return nil, fmt.Errorf("invalid tool spec %q: tool version required", arg)
		}
		installer, ok := a.toolchains[name]
		if !ok {
			return nil, fmt.Errorf("unsupported toolchain %q", name)
		}
		if resolver, ok := installer.(itoolchain.VersionResolver); ok {
			resolvedVersion, err := resolver.ResolveVersion(a.installContext(), version)
			if err != nil {
				return nil, fmt.Errorf("resolve %s version %q: %w", name, version, err)
			}
			version = resolvedVersion
		}

		pkg := PackageSpec{
			Name:    name,
			Version: version,
		}
		packages = append(packages, pkg)
	}

	return packages, nil
}

func (a *App) CreateManifest(name string) error {
	wsPath, err := a.EnsureWorkspace(name)
	if err != nil {
		return err
	}
	manifest := NewManifest(name)

	return a.writeManifest(wsPath, *manifest)
}
