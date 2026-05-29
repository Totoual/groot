package app

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	contextBroadVaultLimit           = 10
	contextBroadFileLimit            = 10
	contextBroadSymbolLimit          = 10
	contextBroadRecentLimit          = 5
	contextBroadSuggestedReadLimit   = 5
	contextTaskResumeLimit           = 3
	contextNarrowVaultLimit          = 3
	contextNarrowFileLimit           = 3
	contextNarrowSymbolLimit         = 5
	contextNarrowSuggestedReadLimit  = 3
	contextHandoffFileLimit          = 5
	contextHandoffSymbolLimit        = 5
	contextHandoffSuggestedReadLimit = 3
)

type ContextMode string

const (
	ContextModeNarrow  ContextMode = "narrow"
	ContextModeHandoff ContextMode = "handoff"
	ContextModeBroad   ContextMode = "broad"
)

type ContextBuildOptions struct {
	Mode ContextMode
}

type ContextPack struct {
	Mode                 ContextMode          `json:"mode"`
	Task                 string               `json:"task"`
	TaskResume           *VaultTaskResume     `json:"task_resume,omitempty"`
	VaultEntries         []VaultNode          `json:"vault_entries"`
	TaskResumeCandidates []VaultTaskCandidate `json:"task_resume_candidates"`
	Files                []IndexFileRecord    `json:"files"`
	Symbols              []IndexSymbolRecord  `json:"symbols"`
	RecentActivity       []VaultNode          `json:"recent_activity"`
	SuggestedReads       []string             `json:"suggested_reads"`
}

func (a *App) BuildContextPack(workspaceName, task string) (ContextPack, error) {
	return a.BuildContextPackWithOptions(workspaceName, task, ContextBuildOptions{Mode: ContextModeNarrow})
}

func (a *App) BuildContextPackWithOptions(workspaceName, task string, opts ContextBuildOptions) (ContextPack, error) {
	task = strings.TrimSpace(task)
	if task == "" {
		return ContextPack{}, fmt.Errorf("context task required")
	}
	mode, err := normalizeContextMode(string(opts.Mode))
	if err != nil {
		return ContextPack{}, err
	}

	switch mode {
	case ContextModeNarrow:
		return a.buildContextPackNarrow(workspaceName, task)
	case ContextModeHandoff:
		return a.buildContextPackHandoff(workspaceName, task)
	case ContextModeBroad:
		return a.buildContextPackBroad(workspaceName, task)
	default:
		return ContextPack{}, fmt.Errorf("unsupported context mode %q", mode)
	}
}

func (a *App) buildContextPackBroad(workspaceName, task string) (ContextPack, error) {
	fileHits, err := a.IndexSearch(workspaceName, task, IndexSearchOptions{Limit: contextBroadFileLimit})
	if err != nil {
		return ContextPack{}, err
	}
	symbolHits, err := a.IndexSymbols(workspaceName, task, IndexSearchOptions{Limit: contextBroadSymbolLimit})
	if err != nil {
		return ContextPack{}, err
	}
	vaultHits, err := a.VaultSearch(workspaceName, task, VaultSearchOptions{Limit: contextBroadVaultLimit})
	if err != nil {
		return ContextPack{}, err
	}
	recentNodes, err := a.VaultRecent(workspaceName, VaultRecentOptions{Limit: contextBroadVaultLimit * 2})
	if err != nil {
		return ContextPack{}, err
	}
	candidates, err := a.FindVaultTaskCandidates(workspaceName, task, contextTaskResumeLimit)
	if err != nil {
		return ContextPack{}, err
	}

	pack := ContextPack{
		Mode:                 ContextModeBroad,
		Task:                 task,
		VaultEntries:         dedupeVaultSearchHits(vaultHits, contextBroadVaultLimit),
		TaskResumeCandidates: candidates,
		Files:                dedupeIndexSearchHits(fileHits, contextBroadFileLimit),
		Symbols:              dedupeIndexSymbolHits(symbolHits, contextBroadSymbolLimit),
	}
	pack.RecentActivity = filterRecentVaultActivity(recentNodes, pack.VaultEntries, contextBroadRecentLimit)
	pack.SuggestedReads = buildSuggestedReads(pack.Files, pack.Symbols, contextBroadSuggestedReadLimit)
	return pack, nil
}

func (a *App) buildContextPackNarrow(workspaceName, task string) (ContextPack, error) {
	fileHits, err := a.IndexSearch(workspaceName, task, IndexSearchOptions{Limit: contextBroadFileLimit})
	if err != nil {
		return ContextPack{}, err
	}
	symbolHits, err := a.IndexSymbols(workspaceName, task, IndexSearchOptions{Limit: contextBroadSymbolLimit})
	if err != nil {
		return ContextPack{}, err
	}
	vaultHits, err := a.VaultSearch(workspaceName, task, VaultSearchOptions{Limit: contextBroadVaultLimit})
	if err != nil {
		return ContextPack{}, err
	}
	candidates, err := a.FindVaultTaskCandidates(workspaceName, task, 1)
	if err != nil {
		return ContextPack{}, err
	}

	files := selectNarrowContextFiles(task, fileHits, contextNarrowFileLimit)
	symbols := dedupeIndexSymbolHits(symbolHits, contextNarrowSymbolLimit)
	vaultEntries := dedupeVaultSearchHits(vaultHits, contextNarrowVaultLimit)
	return ContextPack{
		Mode:                 ContextModeNarrow,
		Task:                 task,
		VaultEntries:         vaultEntries,
		TaskResumeCandidates: candidates,
		Files:                files,
		Symbols:              symbols,
		SuggestedReads:       buildSuggestedReads(files, symbols, contextNarrowSuggestedReadLimit),
	}, nil
}

func (a *App) buildContextPackHandoff(workspaceName, task string) (ContextPack, error) {
	pack := ContextPack{
		Mode: ContextModeHandoff,
		Task: task,
	}

	searchQuery := task
	resume, err := a.ResolveTaskResume(workspaceName, task)
	if err == nil {
		pack.TaskResume = &resume
		pack.VaultEntries = combineTaskResumeNodes(resume)
		searchQuery = resume.Task.Title
	} else {
		candidates, candidateErr := a.FindVaultTaskCandidates(workspaceName, task, contextTaskResumeLimit)
		if candidateErr != nil {
			return ContextPack{}, candidateErr
		}
		pack.TaskResumeCandidates = candidates
	}

	fileHits, err := a.IndexSearch(workspaceName, searchQuery, IndexSearchOptions{Limit: contextHandoffFileLimit})
	if err != nil {
		return ContextPack{}, err
	}
	symbolHits, err := a.IndexSymbols(workspaceName, searchQuery, IndexSearchOptions{Limit: contextHandoffSymbolLimit})
	if err != nil {
		return ContextPack{}, err
	}
	pack.Files = dedupeIndexSearchHits(fileHits, contextHandoffFileLimit)
	pack.Symbols = dedupeIndexSymbolHits(symbolHits, contextHandoffSymbolLimit)
	pack.SuggestedReads = buildSuggestedReads(pack.Files, pack.Symbols, contextHandoffSuggestedReadLimit)
	return pack, nil
}

func (p ContextPack) Markdown() string {
	var b strings.Builder
	b.WriteString("# Groot Context Pack\n\n")
	b.WriteString("Task:\n")
	b.WriteString(p.Task)
	b.WriteString("\n")

	if p.Mode == ContextModeHandoff && p.TaskResume != nil {
		writeContextTaskResume(&b, *p.TaskResume)
	} else {
		writeContextVaultEntries(&b, p.VaultEntries)
		writeContextTaskResumeCandidates(&b, p.TaskResumeCandidates)
	}
	writeContextFiles(&b, p.Files)
	writeContextSymbols(&b, p.Symbols)
	if p.Mode == ContextModeBroad {
		writeContextRecentActivity(&b, p.RecentActivity)
	}
	writeContextSuggestedReads(&b, p.SuggestedReads)
	return b.String()
}

func normalizeContextMode(raw string) (ContextMode, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", string(ContextModeNarrow):
		return ContextModeNarrow, nil
	case string(ContextModeHandoff):
		return ContextModeHandoff, nil
	case string(ContextModeBroad):
		return ContextModeBroad, nil
	default:
		return "", fmt.Errorf("unsupported context mode %q", raw)
	}
}

func dedupeVaultSearchHits(hits []VaultSearchHit, limit int) []VaultNode {
	results := make([]VaultNode, 0, min(limit, len(hits)))
	seen := map[string]struct{}{}
	for _, hit := range hits {
		if _, ok := seen[hit.Node.ID]; ok {
			continue
		}
		seen[hit.Node.ID] = struct{}{}
		results = append(results, hit.Node)
		if limit > 0 && len(results) >= limit {
			break
		}
	}
	return results
}

func dedupeIndexSearchHits(hits []IndexSearchHit, limit int) []IndexFileRecord {
	results := make([]IndexFileRecord, 0, min(limit, len(hits)))
	seen := map[string]struct{}{}
	for _, hit := range hits {
		if _, ok := seen[hit.File.Path]; ok {
			continue
		}
		seen[hit.File.Path] = struct{}{}
		results = append(results, hit.File)
		if limit > 0 && len(results) >= limit {
			break
		}
	}
	return results
}

func dedupeIndexSymbolHits(hits []IndexSymbolSearchHit, limit int) []IndexSymbolRecord {
	results := make([]IndexSymbolRecord, 0, min(limit, len(hits)))
	seen := map[string]struct{}{}
	for _, hit := range hits {
		key := hit.Symbol.QualifiedName + "\n" + hit.Symbol.FilePath
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		results = append(results, hit.Symbol)
		if limit > 0 && len(results) >= limit {
			break
		}
	}
	return results
}

func filterRecentVaultActivity(recent []VaultNode, relevant []VaultNode, limit int) []VaultNode {
	results := make([]VaultNode, 0, min(limit, len(recent)))
	seen := map[string]struct{}{}
	for _, node := range relevant {
		seen[node.ID] = struct{}{}
	}

	for _, node := range recent {
		if _, ok := seen[node.ID]; ok {
			continue
		}
		seen[node.ID] = struct{}{}
		results = append(results, node)
		if limit > 0 && len(results) >= limit {
			break
		}
	}
	return results
}

func buildSuggestedReads(files []IndexFileRecord, symbols []IndexSymbolRecord, limit int) []string {
	results := make([]string, 0, limit)
	seen := map[string]struct{}{}
	for _, file := range files {
		seen[file.Path] = struct{}{}
	}

	for _, symbol := range symbols {
		if symbol.FilePath == "" {
			continue
		}
		if _, ok := seen[symbol.FilePath]; ok {
			continue
		}
		seen[symbol.FilePath] = struct{}{}
		results = append(results, symbol.FilePath)
		if len(results) >= limit {
			return results
		}
	}
	return results
}

func writeContextVaultEntries(b *strings.Builder, nodes []VaultNode) {
	b.WriteString("\nRelevant Vault Entries:\n")
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

func writeContextTaskResume(b *strings.Builder, resume VaultTaskResume) {
	b.WriteString("\nTask Resume:\n")
	b.WriteString(resume.Task.Title)
	b.WriteString("\n")
	if resume.LatestProgress != nil {
		b.WriteString("Latest Progress: ")
		b.WriteString(contextSummary(resume.LatestProgress.Title, resume.LatestProgress.Body))
		b.WriteString("\n")
	}
	for _, section := range []struct {
		label string
		nodes []VaultNode
	}{
		{label: "Decisions", nodes: resume.Decisions},
		{label: "Rules", nodes: resume.Rules},
		{label: "Patterns", nodes: resume.Patterns},
		{label: "Failures", nodes: resume.Failures},
	} {
		if len(section.nodes) == 0 {
			continue
		}
		b.WriteString(section.label)
		b.WriteString(":\n")
		for _, node := range section.nodes {
			b.WriteString("- ")
			b.WriteString(contextSummary(node.Title, node.Body))
			b.WriteString("\n")
		}
	}
}

func writeContextFiles(b *strings.Builder, files []IndexFileRecord) {
	b.WriteString("\nRelevant Files:\n")
	if len(files) == 0 {
		b.WriteString("- none\n")
		return
	}
	for _, file := range files {
		b.WriteString("- ")
		b.WriteString(file.Path)
		b.WriteString("\n")
	}
}

func writeContextTaskResumeCandidates(b *strings.Builder, candidates []VaultTaskCandidate) {
	if len(candidates) == 0 {
		return
	}
	b.WriteString("\nTask Resume Candidates:\n")
	for _, candidate := range candidates {
		b.WriteString("- ")
		b.WriteString(candidate.Task.Title)
		if candidate.LatestProgress != nil {
			b.WriteString(" - ")
			b.WriteString(contextSummary(candidate.LatestProgress.Title, ""))
		}
		b.WriteString("\n")
	}
}

func writeContextSymbols(b *strings.Builder, symbols []IndexSymbolRecord) {
	b.WriteString("\nRelevant Symbols:\n")
	if len(symbols) == 0 {
		b.WriteString("- none\n")
		return
	}
	for _, symbol := range symbols {
		b.WriteString("- ")
		b.WriteString(symbol.QualifiedName)
		if location := contextSymbolLocation(symbol); location != "" {
			b.WriteString(" (")
			b.WriteString(location)
			b.WriteString(")")
		}
		b.WriteString("\n")
	}
}

func writeContextRecentActivity(b *strings.Builder, nodes []VaultNode) {
	b.WriteString("\nRecent Vault Activity:\n")
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

func writeContextSuggestedReads(b *strings.Builder, reads []string) {
	if len(reads) == 0 {
		return
	}
	b.WriteString("\nSuggested Next Reads:\n")
	for _, read := range reads {
		b.WriteString("- ")
		b.WriteString(read)
		b.WriteString("\n")
	}
}

func contextSummary(title, body string) string {
	title = strings.TrimSpace(strings.ReplaceAll(title, "\n", " "))
	body = strings.TrimSpace(strings.ReplaceAll(body, "\n", " "))
	switch {
	case title != "" && body != "" && !strings.EqualFold(title, body):
		return title + ". " + body
	case title != "":
		return title
	default:
		return body
	}
}

func contextSymbolLocation(symbol IndexSymbolRecord) string {
	if symbol.FilePath == "" {
		return ""
	}
	if symbol.LineStart > 0 && symbol.LineEnd > 0 {
		return fmt.Sprintf("%s:%d-%d", symbol.FilePath, symbol.LineStart, symbol.LineEnd)
	}
	return symbol.FilePath
}

func selectNarrowContextFiles(query string, hits []IndexSearchHit, limit int) []IndexFileRecord {
	query = strings.ToLower(strings.TrimSpace(query))
	allowDocs := narrowContextAllowsDocs(query)
	results := make([]IndexFileRecord, 0, min(limit, len(hits)))
	seen := map[string]struct{}{}
	for _, hit := range hits {
		if _, ok := seen[hit.File.Path]; ok {
			continue
		}
		if isContextDocLikePath(hit.File.Path) && !allowDocs {
			continue
		}
		seen[hit.File.Path] = struct{}{}
		results = append(results, hit.File)
		if limit > 0 && len(results) >= limit {
			break
		}
	}
	return results
}

func narrowContextAllowsDocs(query string) bool {
	for _, token := range []string{"doc", "docs", "readme", "markdown"} {
		if strings.Contains(query, token) {
			return true
		}
	}
	return false
}

func isContextDocLikePath(path string) bool {
	lowerPath := strings.ToLower(path)
	base := strings.ToLower(filepath.Base(path))
	return strings.HasSuffix(lowerPath, ".md") || strings.Contains(lowerPath, "/docs/") || strings.HasPrefix(base, "readme")
}

func combineTaskResumeNodes(resume VaultTaskResume) []VaultNode {
	nodes := make([]VaultNode, 0, len(resume.Decisions)+len(resume.Rules)+len(resume.Patterns)+len(resume.Failures))
	nodes = append(nodes, resume.Decisions...)
	nodes = append(nodes, resume.Rules...)
	nodes = append(nodes, resume.Patterns...)
	nodes = append(nodes, resume.Failures...)
	return nodes
}
