package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Manifest struct {
	SchemaVersion int               `json:"schema_version"`
	CreatedAt     time.Time         `json:"created_at"`
	Name          string            `json:"name"`
	Packages      []Component       `json:"packages"`
	Services      []Component       `json:"services"`
	Env           map[string]string `json:"env"`
}

type Component struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func NewManifest(name string) *Manifest {
	return &Manifest{
		SchemaVersion: 1,
		CreatedAt:     time.Now(),
		Name:          name,
		Packages:      make([]Component, 0),
		Services:      make([]Component, 0),
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

func (a *App) createComponents(args []string) []Component {
	components := make([]Component, 0)
	for _, arg := range args {
		argParts := strings.Split(arg, "@")
		comp := Component{
			Name:    argParts[0],
			Version: argParts[1],
		}
		components = append(components, comp)
	}

	return components
}

func (a *App) CreateManifest(name string) error {
	wsPath, err := a.EnsureWorkspace(name)
	if err != nil {
		return err
	}
	manifest := NewManifest(name)

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(wsPath, "manifest.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}

	return nil
}
