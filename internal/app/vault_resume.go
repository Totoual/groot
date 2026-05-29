package app

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"
)

type VaultTaskCandidate struct {
	Task           VaultNode  `json:"task"`
	LatestProgress *VaultNode `json:"latest_progress,omitempty"`
}

type VaultTaskResume struct {
	Task           VaultNode     `json:"task"`
	LatestProgress *VaultNode    `json:"latest_progress,omitempty"`
	Decisions      []VaultNode   `json:"decisions"`
	Rules          []VaultNode   `json:"rules"`
	Patterns       []VaultNode   `json:"patterns"`
	Failures       []VaultNode   `json:"failures"`
	Edges          []VaultEdge   `json:"edges"`
	Freshness      VaultMetadata `json:"freshness"`
}

func (a *App) BuildTaskResume(workspaceName, taskID string) (VaultTaskResume, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return VaultTaskResume{}, fmt.Errorf("vault task resume task id required")
	}

	nodes, nodeByID, edges, freshness, err := a.loadVaultResumeData(workspaceName)
	if err != nil {
		return VaultTaskResume{}, err
	}

	task, ok := nodeByID[taskID]
	if !ok {
		return VaultTaskResume{}, fmt.Errorf("vault task resume task %q not found", taskID)
	}
	if task.Type != VaultNodeTypeTask {
		return VaultTaskResume{}, fmt.Errorf("vault task resume node %q is not a task", taskID)
	}

	return buildVaultTaskResume(task, nodes, nodeByID, edges, freshness), nil
}

func (a *App) FindVaultTaskCandidates(workspaceName, lookup string, limit int) ([]VaultTaskCandidate, error) {
	lookup = strings.TrimSpace(lookup)
	if lookup == "" {
		return nil, fmt.Errorf("vault task lookup required")
	}

	nodes, nodeByID, edges, _, err := a.loadVaultResumeData(workspaceName)
	if err != nil {
		return nil, err
	}

	for _, node := range nodes {
		if node.Type == VaultNodeTypeTask && node.ID == lookup {
			return []VaultTaskCandidate{{
				Task:           node,
				LatestProgress: latestProgressForTask(node.ID, nodeByID, edges),
			}}, nil
		}
	}

	query := strings.ToLower(lookup)
	type scoredTask struct {
		Task  VaultNode
		Score int
	}
	scored := make([]scoredTask, 0)
	for _, node := range nodes {
		if node.Type != VaultNodeTypeTask {
			continue
		}
		title := strings.ToLower(node.Title)
		score := 0
		switch {
		case strings.EqualFold(node.Title, lookup):
			score = 2
		case strings.Contains(title, query):
			score = 1
		}
		if score == 0 {
			continue
		}
		scored = append(scored, scoredTask{Task: node, Score: score})
	}

	slices.SortFunc(scored, func(a, b scoredTask) int {
		if a.Score != b.Score {
			if a.Score > b.Score {
				return -1
			}
			return 1
		}
		if cmp := b.Task.CreatedAt.Compare(a.Task.CreatedAt); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.Task.Title, b.Task.Title); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.Task.ID, b.Task.ID)
	})

	if limit > 0 && len(scored) > limit {
		scored = scored[:limit]
	}
	candidates := make([]VaultTaskCandidate, 0, len(scored))
	for _, hit := range scored {
		candidates = append(candidates, VaultTaskCandidate{
			Task:           hit.Task,
			LatestProgress: latestProgressForTask(hit.Task.ID, nodeByID, edges),
		})
	}
	return candidates, nil
}

func (a *App) ResolveTaskResume(workspaceName, taskOrQuery string) (VaultTaskResume, error) {
	taskOrQuery = strings.TrimSpace(taskOrQuery)
	if taskOrQuery == "" {
		return VaultTaskResume{}, fmt.Errorf("vault task resume task or query required")
	}
	if resume, err := a.BuildTaskResume(workspaceName, taskOrQuery); err == nil {
		return resume, nil
	}

	candidates, err := a.FindVaultTaskCandidates(workspaceName, taskOrQuery, 1)
	if err != nil {
		return VaultTaskResume{}, err
	}
	if len(candidates) == 0 {
		return VaultTaskResume{}, fmt.Errorf("vault task resume found no task matching %q", taskOrQuery)
	}
	return a.BuildTaskResume(workspaceName, candidates[0].Task.ID)
}

func (r VaultTaskResume) Markdown() string {
	var b strings.Builder
	b.WriteString("# Groot Task Resume\n\n")
	b.WriteString("Task:\n")
	b.WriteString(r.Task.Title)
	b.WriteString("\n")
	if status := strings.TrimSpace(r.Task.Status); status != "" {
		b.WriteString("Status: ")
		b.WriteString(status)
		b.WriteString("\n")
	}
	if body := strings.TrimSpace(r.Task.Body); body != "" {
		b.WriteString("\nGoal:\n")
		b.WriteString(body)
		b.WriteString("\n")
	}

	writeTaskResumeLatestProgress(&b, r.LatestProgress)
	writeTaskResumeNodeSection(&b, "Decisions", r.Decisions)
	writeTaskResumeNodeSection(&b, "Rules / Constraints", r.Rules)
	writeTaskResumeNodeSection(&b, "Patterns", r.Patterns)
	writeTaskResumeNodeSection(&b, "Failures / Risks", r.Failures)
	writeTaskResumeEdges(&b, r.Edges)
	writeTaskResumeFreshness(&b, r.Freshness)
	return b.String()
}

func buildVaultTaskResume(task VaultNode, nodes []VaultNode, nodeByID map[string]VaultNode, edges []VaultEdge, freshness VaultMetadata) VaultTaskResume {
	relevantEdges := make([]VaultEdge, 0)
	decisionMap := map[string]VaultNode{}
	ruleMap := map[string]VaultNode{}
	patternMap := map[string]VaultNode{}
	failureMap := map[string]VaultNode{}

	for _, edge := range edges {
		if edge.FromID != task.ID && edge.ToID != task.ID {
			continue
		}
		relevantEdges = append(relevantEdges, edge)
		otherID := edge.FromID
		if otherID == task.ID {
			otherID = edge.ToID
		}
		node, ok := nodeByID[otherID]
		if !ok {
			continue
		}
		switch node.Type {
		case VaultNodeTypeDecision:
			decisionMap[node.ID] = node
		case VaultNodeTypeRule:
			ruleMap[node.ID] = node
		case VaultNodeTypePattern:
			patternMap[node.ID] = node
		case VaultNodeTypeFailure:
			failureMap[node.ID] = node
		}
	}

	sortVaultEdges(relevantEdges)
	return VaultTaskResume{
		Task:           task,
		LatestProgress: latestProgressForTask(task.ID, nodeByID, edges),
		Decisions:      sortedVaultNodeMapValues(decisionMap),
		Rules:          sortedVaultNodeMapValues(ruleMap),
		Patterns:       sortedVaultNodeMapValues(patternMap),
		Failures:       sortedVaultNodeMapValues(failureMap),
		Edges:          relevantEdges,
		Freshness:      freshness,
	}
}

func (a *App) loadVaultResumeData(workspaceName string) ([]VaultNode, map[string]VaultNode, []VaultEdge, VaultMetadata, error) {
	nodes, err := a.vaultNodes(workspaceName)
	if err != nil {
		return nil, nil, nil, VaultMetadata{}, err
	}
	edges, err := a.vaultEdges(workspaceName)
	if err != nil {
		return nil, nil, nil, VaultMetadata{}, err
	}

	nodeByID := make(map[string]VaultNode, len(nodes))
	for _, node := range nodes {
		nodeByID[node.ID] = node
	}

	freshness, err := a.vaultMetadata(workspaceName)
	if err != nil {
		return nil, nil, nil, VaultMetadata{}, err
	}
	return nodes, nodeByID, edges, freshness, nil
}

func (a *App) vaultMetadata(workspaceName string) (VaultMetadata, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return VaultMetadata{}, err
	}
	if err := a.InitVault(workspaceName); err != nil {
		return VaultMetadata{}, err
	}
	meta, err := readJSONFile[VaultMetadata](vaultMetaPath(wsPath))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return VaultMetadata{Workspace: workspaceName}, nil
		}
		return VaultMetadata{}, err
	}
	if meta.Workspace == "" {
		meta.Workspace = workspaceName
	}
	return meta, nil
}

func latestProgressForTask(taskID string, nodeByID map[string]VaultNode, edges []VaultEdge) *VaultNode {
	var latest *VaultNode
	var latestEdgeTime time.Time
	for _, edge := range edges {
		if edge.Type != VaultEdgeTypeForTask || edge.ToID != taskID {
			continue
		}
		node, ok := nodeByID[edge.FromID]
		if !ok || node.Type != VaultNodeTypeProgress {
			continue
		}
		if latest == nil || node.CreatedAt.After(latest.CreatedAt) ||
			(node.CreatedAt.Equal(latest.CreatedAt) && edge.CreatedAt.After(latestEdgeTime)) ||
			(node.CreatedAt.Equal(latest.CreatedAt) && edge.CreatedAt.Equal(latestEdgeTime) && node.ID < latest.ID) {
			nodeCopy := node
			latest = &nodeCopy
			latestEdgeTime = edge.CreatedAt
		}
	}
	return latest
}

func sortedVaultNodeMapValues(nodes map[string]VaultNode) []VaultNode {
	values := make([]VaultNode, 0, len(nodes))
	for _, node := range nodes {
		values = append(values, node)
	}
	sortVaultNodes(values)
	return values
}

func sortVaultNodes(nodes []VaultNode) {
	slices.SortFunc(nodes, func(a, b VaultNode) int {
		if cmp := b.CreatedAt.Compare(a.CreatedAt); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.Title, b.Title); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.ID, b.ID)
	})
}

func sortVaultEdges(edges []VaultEdge) {
	slices.SortFunc(edges, func(a, b VaultEdge) int {
		if cmp := b.CreatedAt.Compare(a.CreatedAt); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.Type, b.Type); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.FromID, b.FromID); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.ToID, b.ToID); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.ID, b.ID)
	})
}

func writeTaskResumeLatestProgress(b *strings.Builder, progress *VaultNode) {
	b.WriteString("\nLatest Progress:\n")
	if progress == nil {
		b.WriteString("- none\n")
		return
	}
	b.WriteString(progress.Title)
	b.WriteString("\n")
	if body := strings.TrimSpace(progress.Body); body != "" {
		b.WriteString(body)
		b.WriteString("\n")
	}
}

func writeTaskResumeNodeSection(b *strings.Builder, title string, nodes []VaultNode) {
	b.WriteString("\n")
	b.WriteString(title)
	b.WriteString(":\n")
	if len(nodes) == 0 {
		b.WriteString("- none\n")
		return
	}
	for _, node := range nodes {
		b.WriteString("- ")
		b.WriteString(node.Type)
		b.WriteString(": ")
		b.WriteString(contextSummary(node.Title, node.Body))
		b.WriteString("\n")
	}
}

func writeTaskResumeEdges(b *strings.Builder, edges []VaultEdge) {
	b.WriteString("\nRelevant Edges:\n")
	if len(edges) == 0 {
		b.WriteString("- none\n")
		return
	}
	for _, edge := range edges {
		b.WriteString("- ")
		b.WriteString(edge.Type)
		b.WriteString(": ")
		b.WriteString(edge.FromID)
		b.WriteString(" -> ")
		b.WriteString(edge.ToID)
		b.WriteString("\n")
	}
}

func writeTaskResumeFreshness(b *strings.Builder, freshness VaultMetadata) {
	if freshness.Workspace == "" && freshness.VaultUpdatedAt.IsZero() {
		return
	}
	b.WriteString("\nFreshness:\n")
	if freshness.Workspace != "" {
		b.WriteString("- workspace: ")
		b.WriteString(freshness.Workspace)
		b.WriteString("\n")
	}
	if !freshness.VaultUpdatedAt.IsZero() {
		b.WriteString("- vault updated at: ")
		b.WriteString(freshness.VaultUpdatedAt.Format(time.RFC3339))
		b.WriteString("\n")
	}
}
