package app

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	VaultNodeTypeRule     = "rule"
	VaultNodeTypeDecision = "decision"
	VaultNodeTypeTask     = "task"
	VaultNodeTypeFailure  = "failure"
	VaultNodeTypePattern  = "pattern"
	VaultNodeTypeNote     = "note"
	VaultNodeTypeFile     = "file"
	VaultNodeTypeSymbol   = "symbol"
)

var supportedVaultNodeTypes = map[string]struct{}{
	VaultNodeTypeRule:     {},
	VaultNodeTypeDecision: {},
	VaultNodeTypeTask:     {},
	VaultNodeTypeFailure:  {},
	VaultNodeTypePattern:  {},
	VaultNodeTypeNote:     {},
	VaultNodeTypeFile:     {},
	VaultNodeTypeSymbol:   {},
}

type VaultNode struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Title      string    `json:"title"`
	Body       string    `json:"body"`
	Tags       []string  `json:"tags"`
	CreatedAt  time.Time `json:"created_at"`
	Source     string    `json:"source"`
	Confidence float64   `json:"confidence"`
	Status     string    `json:"status"`
}

type VaultEdge struct {
	ID        string    `json:"id"`
	FromID    string    `json:"from_id"`
	ToID      string    `json:"to_id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

type VaultChange struct {
	ID        string         `json:"id"`
	Kind      string         `json:"kind"`
	NodeID    string         `json:"node_id,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	Summary   string         `json:"summary"`
	Payload   map[string]any `json:"payload,omitempty"`
}

type VaultAppendSpec struct {
	Type       string
	Title      string
	Body       string
	Tags       []string
	Source     string
	Confidence float64
	Status     string
}

type VaultSearchOptions struct {
	Limit int
}

type VaultSearchHit struct {
	Node  VaultNode `json:"node"`
	Score int       `json:"score"`
}

type VaultRecentOptions struct {
	Limit int
}

type VaultStats struct {
	NodeCount   int            `json:"node_count"`
	EdgeCount   int            `json:"edge_count"`
	ChangeCount int            `json:"change_count"`
	ByType      map[string]int `json:"by_type"`
	ByStatus    map[string]int `json:"by_status"`
}

func (a *App) InitVault(workspaceName string) error {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return err
	}

	for _, path := range []string{
		vaultNodesPath(wsPath),
		vaultEdgesPath(wsPath),
		vaultChangesPath(wsPath),
	} {
		if err := ensureJSONLFile(path); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) VaultAppend(workspaceName string, spec VaultAppendSpec) (VaultNode, error) {
	if err := a.InitVault(workspaceName); err != nil {
		return VaultNode{}, err
	}

	nodeType := strings.ToLower(strings.TrimSpace(spec.Type))
	if _, ok := supportedVaultNodeTypes[nodeType]; !ok {
		return VaultNode{}, fmt.Errorf("unsupported vault node type %q", spec.Type)
	}

	title := strings.TrimSpace(spec.Title)
	if title == "" {
		return VaultNode{}, fmt.Errorf("vault title required")
	}
	body := strings.TrimSpace(spec.Body)
	if body == "" {
		return VaultNode{}, fmt.Errorf("vault body required")
	}

	source := strings.TrimSpace(spec.Source)
	if source == "" {
		source = "human"
	}
	confidence := spec.Confidence
	if confidence == 0 {
		confidence = 1.0
	}
	status := strings.TrimSpace(spec.Status)
	if status == "" {
		status = "active"
	}

	nodeID, err := newVaultRecordID("node")
	if err != nil {
		return VaultNode{}, err
	}
	now := time.Now().UTC()
	node := VaultNode{
		ID:         nodeID,
		Type:       nodeType,
		Title:      title,
		Body:       body,
		Tags:       normalizeVaultTags(spec.Tags),
		CreatedAt:  now,
		Source:     source,
		Confidence: confidence,
		Status:     status,
	}

	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return VaultNode{}, err
	}
	if err := appendJSONLRecord(vaultNodesPath(wsPath), node); err != nil {
		return VaultNode{}, err
	}

	changeID, err := newVaultRecordID("chg")
	if err != nil {
		return VaultNode{}, err
	}
	change := VaultChange{
		ID:        changeID,
		Kind:      "node.appended",
		NodeID:    node.ID,
		Timestamp: now,
		Summary:   fmt.Sprintf("Appended %s %q.", node.Type, node.Title),
		Payload: map[string]any{
			"type":   node.Type,
			"title":  node.Title,
			"status": node.Status,
		},
	}
	if err := appendJSONLRecord(vaultChangesPath(wsPath), change); err != nil {
		return VaultNode{}, err
	}

	return node, nil
}

func (a *App) VaultSearch(workspaceName, query string, opts VaultSearchOptions) ([]VaultSearchHit, error) {
	nodes, err := a.vaultNodes(workspaceName)
	if err != nil {
		return nil, err
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("vault query required")
	}
	queryTokens := searchTerms(query)

	hits := make([]VaultSearchHit, 0)
	for _, node := range nodes {
		score := vaultNodeSearchScore(node, query, queryTokens)
		if score == 0 {
			continue
		}
		hits = append(hits, VaultSearchHit{Node: node, Score: score})
	}

	slices.SortFunc(hits, func(a, b VaultSearchHit) int {
		if a.Score != b.Score {
			if a.Score > b.Score {
				return -1
			}
			return 1
		}
		if cmp := b.Node.CreatedAt.Compare(a.Node.CreatedAt); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.Node.Title, b.Node.Title); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.Node.ID, b.Node.ID)
	})

	if opts.Limit > 0 && len(hits) > opts.Limit {
		hits = hits[:opts.Limit]
	}
	return hits, nil
}

func (a *App) VaultRecent(workspaceName string, opts VaultRecentOptions) ([]VaultNode, error) {
	nodes, err := a.vaultNodes(workspaceName)
	if err != nil {
		return nil, err
	}

	slices.SortFunc(nodes, func(a, b VaultNode) int {
		if cmp := b.CreatedAt.Compare(a.CreatedAt); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.ID, b.ID)
	})
	if opts.Limit > 0 && len(nodes) > opts.Limit {
		nodes = nodes[:opts.Limit]
	}
	return nodes, nil
}

func (a *App) VaultStats(workspaceName string) (VaultStats, error) {
	nodes, err := a.vaultNodes(workspaceName)
	if err != nil {
		return VaultStats{}, err
	}
	edges, err := a.vaultEdges(workspaceName)
	if err != nil {
		return VaultStats{}, err
	}
	changes, err := a.vaultChanges(workspaceName)
	if err != nil {
		return VaultStats{}, err
	}

	stats := VaultStats{
		NodeCount:   len(nodes),
		EdgeCount:   len(edges),
		ChangeCount: len(changes),
		ByType:      map[string]int{},
		ByStatus:    map[string]int{},
	}
	for _, node := range nodes {
		stats.ByType[node.Type]++
		stats.ByStatus[node.Status]++
	}
	return stats, nil
}

func (a *App) vaultNodes(workspaceName string) ([]VaultNode, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return nil, err
	}
	if err := a.InitVault(workspaceName); err != nil {
		return nil, err
	}
	return readJSONLRecords[VaultNode](vaultNodesPath(wsPath))
}

func (a *App) vaultEdges(workspaceName string) ([]VaultEdge, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return nil, err
	}
	if err := a.InitVault(workspaceName); err != nil {
		return nil, err
	}
	return readJSONLRecords[VaultEdge](vaultEdgesPath(wsPath))
}

func (a *App) vaultChanges(workspaceName string) ([]VaultChange, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return nil, err
	}
	if err := a.InitVault(workspaceName); err != nil {
		return nil, err
	}
	return readJSONLRecords[VaultChange](vaultChangesPath(wsPath))
}

func vaultNodesPath(wsPath string) string {
	return filepath.Join(wsPath, "vault", "nodes.jsonl")
}

func vaultEdgesPath(wsPath string) string {
	return filepath.Join(wsPath, "vault", "edges.jsonl")
}

func vaultChangesPath(wsPath string) string {
	return filepath.Join(wsPath, "vault", "changes.jsonl")
}

func normalizeVaultTags(tags []string) []string {
	normalized := make([]string, 0, len(tags))
	seen := map[string]struct{}{}
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		normalized = append(normalized, tag)
	}
	slices.Sort(normalized)
	return normalized
}

func newVaultRecordID(prefix string) (string, error) {
	suffix, err := randomHex(6)
	if err != nil {
		return "", fmt.Errorf("generate vault id: %w", err)
	}
	return fmt.Sprintf("%s-%d-%s", prefix, time.Now().UTC().UnixMilli(), suffix), nil
}

func searchTerms(raw string) []string {
	parts := strings.FieldsFunc(strings.ToLower(raw), func(r rune) bool {
		switch {
		case r >= 'a' && r <= 'z':
			return false
		case r >= '0' && r <= '9':
			return false
		case r == '_':
			return false
		default:
			return true
		}
	})

	terms := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		terms = append(terms, part)
	}
	return terms
}

func vaultNodeSearchScore(node VaultNode, query string, queryTokens []string) int {
	title := strings.ToLower(node.Title)
	body := strings.ToLower(node.Body)
	status := strings.ToLower(node.Status)
	query = strings.ToLower(strings.TrimSpace(query))

	score := 0
	if strings.Contains(title, query) {
		score += 8
	}
	if strings.Contains(body, query) {
		score += 3
	}
	for _, token := range queryTokens {
		if strings.Contains(title, token) {
			score += 4
		}
		if strings.Contains(body, token) {
			score += 1
		}
		if strings.Contains(status, token) {
			score += 2
		}
		for _, tag := range node.Tags {
			if strings.EqualFold(tag, token) {
				score += 5
			}
		}
		if strings.EqualFold(node.Type, token) {
			score += 5
		}
	}
	return score
}
