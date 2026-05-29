package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type IndexMetadata struct {
	Indexed     bool      `json:"indexed"`
	IndexedAt   time.Time `json:"indexed_at"`
	Workspace   string    `json:"workspace"`
	ProjectPath string    `json:"project_path"`
	FileCount   int       `json:"file_count"`
	SymbolCount int       `json:"symbol_count"`
	TermCount   int       `json:"term_count"`
}

const DefaultIndexFreshnessMaxAge = 24 * time.Hour

type IndexStatus struct {
	Indexed       bool      `json:"indexed"`
	Fresh         bool      `json:"fresh"`
	Stale         bool      `json:"stale"`
	Reason        string    `json:"reason"`
	IndexedAt     time.Time `json:"indexed_at"`
	Workspace     string    `json:"workspace"`
	ProjectPath   string    `json:"project_path"`
	FileCount     int       `json:"file_count"`
	SymbolCount   int       `json:"symbol_count"`
	TermCount     int       `json:"term_count"`
	AgeSeconds    int64     `json:"age_seconds"`
	MaxAgeSeconds int64     `json:"max_age_seconds"`
}

type VaultMetadata struct {
	Workspace      string    `json:"workspace"`
	VaultUpdatedAt time.Time `json:"vault_updated_at"`
	NodeCount      int       `json:"node_count"`
	EdgeCount      int       `json:"edge_count"`
	ChangeCount    int       `json:"change_count"`
}

func indexMetaPath(wsPath string) string {
	return filepath.Join(indexRootPath(wsPath), "index_meta.json")
}

func vaultMetaPath(wsPath string) string {
	return filepath.Join(wsPath, "vault", "vault_meta.json")
}

func (a *App) writeIndexMetadata(workspaceName, wsPath string, indexedAt time.Time, stats IndexStats) error {
	projectPath, err := a.workspaceProjectPathOrEmpty(workspaceName, wsPath)
	if err != nil {
		return err
	}
	meta := IndexMetadata{
		Indexed:     true,
		IndexedAt:   indexedAt,
		Workspace:   workspaceName,
		ProjectPath: projectPath,
		FileCount:   stats.FileCount,
		SymbolCount: stats.SymbolCount,
		TermCount:   stats.TermCount,
	}
	return writeJSONAtomic(indexMetaPath(wsPath), meta)
}

func (a *App) IndexMetadata(workspaceName string) (IndexMetadata, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return IndexMetadata{}, err
	}
	projectPath, err := a.workspaceProjectPathOrEmpty(workspaceName, wsPath)
	if err != nil {
		return IndexMetadata{}, err
	}
	stats, err := a.IndexStats(workspaceName)
	if err != nil {
		return IndexMetadata{}, err
	}

	meta, err := readJSONFile[IndexMetadata](indexMetaPath(wsPath))
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return IndexMetadata{}, err
		}
		meta = IndexMetadata{}
		meta.Indexed = false
	}
	meta.Workspace = workspaceName
	meta.ProjectPath = projectPath
	meta.FileCount = stats.FileCount
	meta.SymbolCount = stats.SymbolCount
	meta.TermCount = stats.TermCount
	if !meta.IndexedAt.IsZero() {
		meta.Indexed = true
	}
	return meta, nil
}

func (a *App) IndexStatus(workspaceName string) (IndexStatus, error) {
	return a.indexStatusAt(workspaceName, time.Now().UTC(), DefaultIndexFreshnessMaxAge)
}

func (a *App) indexStatusAt(workspaceName string, now time.Time, maxAge time.Duration) (IndexStatus, error) {
	meta, err := a.IndexMetadata(workspaceName)
	if err != nil {
		return IndexStatus{}, err
	}

	status := IndexStatus{
		Indexed:       meta.Indexed,
		Fresh:         false,
		Stale:         true,
		Reason:        "missing_metadata",
		IndexedAt:     meta.IndexedAt,
		Workspace:     meta.Workspace,
		ProjectPath:   meta.ProjectPath,
		FileCount:     meta.FileCount,
		SymbolCount:   meta.SymbolCount,
		TermCount:     meta.TermCount,
		MaxAgeSeconds: int64(maxAge / time.Second),
	}
	if !meta.Indexed {
		return status, nil
	}
	if meta.IndexedAt.IsZero() {
		status.Reason = "missing_indexed_at"
		return status, nil
	}

	age := now.Sub(meta.IndexedAt)
	if age < 0 {
		age = 0
	}
	status.AgeSeconds = int64(age / time.Second)
	if maxAge > 0 && age > maxAge {
		status.Reason = "stale"
		return status, nil
	}

	status.Fresh = true
	status.Stale = false
	status.Reason = "fresh"
	return status, nil
}

func (a *App) writeVaultMetadata(workspaceName, wsPath string, updatedAt time.Time) error {
	nodes, err := readJSONLRecords[VaultNode](vaultNodesPath(wsPath))
	if err != nil {
		return err
	}
	edges, err := readJSONLRecords[VaultEdge](vaultEdgesPath(wsPath))
	if err != nil {
		return err
	}
	changes, err := readJSONLRecords[VaultChange](vaultChangesPath(wsPath))
	if err != nil {
		return err
	}

	meta := VaultMetadata{
		Workspace:      workspaceName,
		VaultUpdatedAt: updatedAt,
		NodeCount:      len(nodes),
		EdgeCount:      len(edges),
		ChangeCount:    len(changes),
	}
	return writeJSONAtomic(vaultMetaPath(wsPath), meta)
}

func (a *App) workspaceProjectPathOrEmpty(workspaceName, wsPath string) (string, error) {
	manifest, err := a.getManifest(wsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if strings.TrimSpace(manifest.ProjectPath) == "" {
		return "", nil
	}
	return normalizeProjectPath(manifest.ProjectPath)
}
