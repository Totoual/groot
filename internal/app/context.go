package app

import (
	"fmt"
	"strings"
)

const (
	contextVaultLimit         = 10
	contextFileLimit          = 10
	contextSymbolLimit        = 10
	contextRecentLimit        = 5
	contextSuggestedReadLimit = 5
)

type ContextPack struct {
	Task           string              `json:"task"`
	VaultEntries   []VaultNode         `json:"vault_entries"`
	Files          []IndexFileRecord   `json:"files"`
	Symbols        []IndexSymbolRecord `json:"symbols"`
	RecentActivity []VaultNode         `json:"recent_activity"`
	SuggestedReads []string            `json:"suggested_reads"`
}

func (a *App) BuildContextPack(workspaceName, task string) (ContextPack, error) {
	task = strings.TrimSpace(task)
	if task == "" {
		return ContextPack{}, fmt.Errorf("context task required")
	}

	fileHits, err := a.IndexSearch(workspaceName, task, IndexSearchOptions{Limit: contextFileLimit})
	if err != nil {
		return ContextPack{}, err
	}
	symbolHits, err := a.IndexSymbols(workspaceName, task, IndexSearchOptions{Limit: contextSymbolLimit})
	if err != nil {
		return ContextPack{}, err
	}
	vaultHits, err := a.VaultSearch(workspaceName, task, VaultSearchOptions{Limit: contextVaultLimit})
	if err != nil {
		return ContextPack{}, err
	}
	recentNodes, err := a.VaultRecent(workspaceName, VaultRecentOptions{Limit: contextVaultLimit * 2})
	if err != nil {
		return ContextPack{}, err
	}

	pack := ContextPack{
		Task:         task,
		VaultEntries: dedupeVaultSearchHits(vaultHits, contextVaultLimit),
		Files:        dedupeIndexSearchHits(fileHits, contextFileLimit),
		Symbols:      dedupeIndexSymbolHits(symbolHits, contextSymbolLimit),
	}
	pack.RecentActivity = filterRecentVaultActivity(recentNodes, pack.VaultEntries, contextRecentLimit)
	pack.SuggestedReads = buildSuggestedReads(pack.Files, pack.Symbols, contextSuggestedReadLimit)
	return pack, nil
}

func (p ContextPack) Markdown() string {
	var b strings.Builder
	b.WriteString("# Groot Context Pack\n\n")
	b.WriteString("Task:\n")
	b.WriteString(p.Task)
	b.WriteString("\n")

	writeContextVaultEntries(&b, p.VaultEntries)
	writeContextFiles(&b, p.Files)
	writeContextSymbols(&b, p.Symbols)
	writeContextRecentActivity(&b, p.RecentActivity)
	writeContextSuggestedReads(&b, p.SuggestedReads)
	return b.String()
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

func writeContextSymbols(b *strings.Builder, symbols []IndexSymbolRecord) {
	b.WriteString("\nRelevant Symbols:\n")
	if len(symbols) == 0 {
		b.WriteString("- none\n")
		return
	}
	for _, symbol := range symbols {
		b.WriteString("- ")
		b.WriteString(symbol.QualifiedName)
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
