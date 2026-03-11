package app

import (
	"path/filepath"
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

func getManifestPath(wsPath string) string {
	return filepath.Join(wsPath, "manifest.json")
}
