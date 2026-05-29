package app

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type IndexFileRecord struct {
	ID        int    `json:"id"`
	Path      string `json:"path"`
	Extension string `json:"extension"`
	Language  string `json:"language"`
	SizeBytes int64  `json:"size_bytes"`
}

type IndexSymbolRecord struct {
	ID            int    `json:"id"`
	FileID        int    `json:"file_id"`
	FilePath      string `json:"file_path"`
	Name          string `json:"name"`
	QualifiedName string `json:"qualified_name"`
	Kind          string `json:"kind"`
	PackageName   string `json:"package_name"`
	Receiver      string `json:"receiver,omitempty"`
	LineStart     int    `json:"line_start"`
	LineEnd       int    `json:"line_end"`
}

type IndexTermRecord struct {
	ID   int    `json:"id"`
	Term string `json:"term"`
}

type IndexForwardRecord struct {
	TermID    int   `json:"term_id"`
	FileIDs   []int `json:"file_ids"`
	SymbolIDs []int `json:"symbol_ids"`
}

type IndexReverseRecord struct {
	Kind    string `json:"kind"`
	ID      int    `json:"id"`
	TermIDs []int  `json:"term_ids"`
}

type IndexHashRecord struct {
	FileID int    `json:"file_id"`
	Path   string `json:"path"`
	Hash   string `json:"hash"`
}

type IndexSearchOptions struct {
	Limit int
}

type IndexSearchHit struct {
	File  IndexFileRecord `json:"file"`
	Score int             `json:"score"`
}

type IndexSymbolSearchHit struct {
	Symbol IndexSymbolRecord `json:"symbol"`
	Score  int               `json:"score"`
}

type IndexStats struct {
	FileCount    int            `json:"file_count"`
	SymbolCount  int            `json:"symbol_count"`
	TermCount    int            `json:"term_count"`
	ForwardCount int            `json:"forward_count"`
	ReverseCount int            `json:"reverse_count"`
	HashCount    int            `json:"hash_count"`
	ByLanguage   map[string]int `json:"by_language"`
}

type indexBuilder struct {
	nextFileID    int
	nextSymbolID  int
	nextTermID    int
	files         []IndexFileRecord
	symbols       []IndexSymbolRecord
	hashes        []IndexHashRecord
	reverse       []IndexReverseRecord
	termIDs       map[string]int
	termFileIDs   map[int]map[int]struct{}
	termSymbolIDs map[int]map[int]struct{}
}

func (a *App) InitIndex(workspaceName string) error {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return err
	}

	for _, path := range []string{
		indexFilesPath(wsPath),
		indexSymbolsPath(wsPath),
		indexTermsPath(wsPath),
		indexForwardPath(wsPath),
		indexReversePath(wsPath),
		indexHashesPath(wsPath),
	} {
		if err := ensureJSONLFile(path); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) UpdateIndex(workspaceName string) (IndexStats, error) {
	if err := a.InitIndex(workspaceName); err != nil {
		return IndexStats{}, err
	}

	projectFiles := []ProjectFile{}
	if err := a.WalkWorkspaceProjectFiles(workspaceName, ProjectScanOptions{}, func(file ProjectFile) error {
		projectFiles = append(projectFiles, file)
		return nil
	}); err != nil {
		return IndexStats{}, err
	}
	sort.Slice(projectFiles, func(i, j int) bool {
		return projectFiles[i].RelativePath < projectFiles[j].RelativePath
	})

	builder := indexBuilder{
		nextFileID:    1,
		nextSymbolID:  1,
		nextTermID:    1,
		termIDs:       map[string]int{},
		termFileIDs:   map[int]map[int]struct{}{},
		termSymbolIDs: map[int]map[int]struct{}{},
	}

	for _, file := range projectFiles {
		if err := builder.addFile(file); err != nil {
			return IndexStats{}, err
		}
	}

	terms := builder.terms()
	forward := builder.forward()

	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return IndexStats{}, err
	}
	if err := writeJSONLAtomic(indexFilesPath(wsPath), builder.files); err != nil {
		return IndexStats{}, err
	}
	if err := writeJSONLAtomic(indexSymbolsPath(wsPath), builder.symbols); err != nil {
		return IndexStats{}, err
	}
	if err := writeJSONLAtomic(indexTermsPath(wsPath), terms); err != nil {
		return IndexStats{}, err
	}
	if err := writeJSONLAtomic(indexForwardPath(wsPath), forward); err != nil {
		return IndexStats{}, err
	}
	if err := writeJSONLAtomic(indexReversePath(wsPath), builder.reverse); err != nil {
		return IndexStats{}, err
	}
	if err := writeJSONLAtomic(indexHashesPath(wsPath), builder.hashes); err != nil {
		return IndexStats{}, err
	}
	stats, err := a.IndexStats(workspaceName)
	if err != nil {
		return IndexStats{}, err
	}
	if err := a.writeIndexMetadata(workspaceName, wsPath, time.Now().UTC(), stats); err != nil {
		return IndexStats{}, err
	}
	return stats, nil
}

func (a *App) IndexSearch(workspaceName, query string, opts IndexSearchOptions) ([]IndexSearchHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("index query required")
	}

	files, err := a.indexFiles(workspaceName)
	if err != nil {
		return nil, err
	}
	terms, err := a.indexTerms(workspaceName)
	if err != nil {
		return nil, err
	}
	forward, err := a.indexForward(workspaceName)
	if err != nil {
		return nil, err
	}

	filesByID := make(map[int]IndexFileRecord, len(files))
	for _, file := range files {
		filesByID[file.ID] = file
	}
	termIDsByTerm := make(map[string]int, len(terms))
	for _, term := range terms {
		termIDsByTerm[term.Term] = term.ID
	}
	forwardByTermID := make(map[int]IndexForwardRecord, len(forward))
	for _, record := range forward {
		forwardByTermID[record.TermID] = record
	}

	queryTokens := searchTerms(query)
	scores := map[int]int{}
	for _, token := range queryTokens {
		termID, ok := termIDsByTerm[token]
		if !ok {
			continue
		}
		record, ok := forwardByTermID[termID]
		if !ok {
			continue
		}
		for _, fileID := range record.FileIDs {
			scores[fileID]++
		}
	}

	lowerQuery := strings.ToLower(query)
	hits := make([]IndexSearchHit, 0)
	for fileID, score := range scores {
		file := filesByID[fileID]
		if strings.Contains(strings.ToLower(file.Path), lowerQuery) {
			score++
		}
		hits = append(hits, IndexSearchHit{File: file, Score: score})
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].File.Path < hits[j].File.Path
	})
	if opts.Limit > 0 && len(hits) > opts.Limit {
		hits = hits[:opts.Limit]
	}
	return hits, nil
}

func (a *App) IndexSymbols(workspaceName, query string, opts IndexSearchOptions) ([]IndexSymbolSearchHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("symbol query required")
	}

	symbols, err := a.indexSymbols(workspaceName)
	if err != nil {
		return nil, err
	}

	lowerQuery := strings.ToLower(query)
	queryTokens := searchTerms(query)
	hits := make([]IndexSymbolSearchHit, 0)
	for _, symbol := range symbols {
		score := indexSymbolSearchScore(symbol, lowerQuery, queryTokens)
		if score == 0 {
			continue
		}
		hits = append(hits, IndexSymbolSearchHit{Symbol: symbol, Score: score})
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		if hits[i].Symbol.QualifiedName != hits[j].Symbol.QualifiedName {
			return hits[i].Symbol.QualifiedName < hits[j].Symbol.QualifiedName
		}
		if hits[i].Symbol.FilePath != hits[j].Symbol.FilePath {
			return hits[i].Symbol.FilePath < hits[j].Symbol.FilePath
		}
		return hits[i].Symbol.LineStart < hits[j].Symbol.LineStart
	})
	if opts.Limit > 0 && len(hits) > opts.Limit {
		hits = hits[:opts.Limit]
	}
	return hits, nil
}

func (a *App) IndexStats(workspaceName string) (IndexStats, error) {
	files, err := a.indexFiles(workspaceName)
	if err != nil {
		return IndexStats{}, err
	}
	symbols, err := a.indexSymbols(workspaceName)
	if err != nil {
		return IndexStats{}, err
	}
	terms, err := a.indexTerms(workspaceName)
	if err != nil {
		return IndexStats{}, err
	}
	forward, err := a.indexForward(workspaceName)
	if err != nil {
		return IndexStats{}, err
	}
	reverse, err := a.indexReverse(workspaceName)
	if err != nil {
		return IndexStats{}, err
	}
	hashes, err := a.indexHashes(workspaceName)
	if err != nil {
		return IndexStats{}, err
	}

	stats := IndexStats{
		FileCount:    len(files),
		SymbolCount:  len(symbols),
		TermCount:    len(terms),
		ForwardCount: len(forward),
		ReverseCount: len(reverse),
		HashCount:    len(hashes),
		ByLanguage:   map[string]int{},
	}
	for _, file := range files {
		language := file.Language
		if language == "" {
			language = "unknown"
		}
		stats.ByLanguage[language]++
	}
	return stats, nil
}

func (a *App) indexFiles(workspaceName string) ([]IndexFileRecord, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return nil, err
	}
	if err := a.InitIndex(workspaceName); err != nil {
		return nil, err
	}
	return readJSONLRecords[IndexFileRecord](indexFilesPath(wsPath))
}

func (a *App) indexSymbols(workspaceName string) ([]IndexSymbolRecord, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return nil, err
	}
	if err := a.InitIndex(workspaceName); err != nil {
		return nil, err
	}
	return readJSONLRecords[IndexSymbolRecord](indexSymbolsPath(wsPath))
}

func (a *App) indexTerms(workspaceName string) ([]IndexTermRecord, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return nil, err
	}
	if err := a.InitIndex(workspaceName); err != nil {
		return nil, err
	}
	return readJSONLRecords[IndexTermRecord](indexTermsPath(wsPath))
}

func (a *App) indexForward(workspaceName string) ([]IndexForwardRecord, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return nil, err
	}
	if err := a.InitIndex(workspaceName); err != nil {
		return nil, err
	}
	return readJSONLRecords[IndexForwardRecord](indexForwardPath(wsPath))
}

func (a *App) indexReverse(workspaceName string) ([]IndexReverseRecord, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return nil, err
	}
	if err := a.InitIndex(workspaceName); err != nil {
		return nil, err
	}
	return readJSONLRecords[IndexReverseRecord](indexReversePath(wsPath))
}

func (a *App) indexHashes(workspaceName string) ([]IndexHashRecord, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return nil, err
	}
	if err := a.InitIndex(workspaceName); err != nil {
		return nil, err
	}
	return readJSONLRecords[IndexHashRecord](indexHashesPath(wsPath))
}

func indexRootPath(wsPath string) string {
	return filepath.Join(wsPath, "index")
}

func indexFilesPath(wsPath string) string {
	return filepath.Join(indexRootPath(wsPath), "files.jsonl")
}

func indexSymbolsPath(wsPath string) string {
	return filepath.Join(indexRootPath(wsPath), "symbols.jsonl")
}

func indexTermsPath(wsPath string) string {
	return filepath.Join(indexRootPath(wsPath), "terms.jsonl")
}

func indexForwardPath(wsPath string) string {
	return filepath.Join(indexRootPath(wsPath), "forward.jsonl")
}

func indexReversePath(wsPath string) string {
	return filepath.Join(indexRootPath(wsPath), "reverse.jsonl")
}

func indexHashesPath(wsPath string) string {
	return filepath.Join(indexRootPath(wsPath), "hashes.jsonl")
}

func (b *indexBuilder) addFile(file ProjectFile) error {
	data, err := os.ReadFile(file.AbsolutePath)
	if err != nil {
		return fmt.Errorf("read %s: %w", file.RelativePath, err)
	}

	fileID := b.nextFileID
	b.nextFileID++

	fileRecord := IndexFileRecord{
		ID:        fileID,
		Path:      file.RelativePath,
		Extension: file.Extension,
		Language:  detectIndexLanguage(file.Extension),
		SizeBytes: int64(len(data)),
	}
	b.files = append(b.files, fileRecord)
	b.hashes = append(b.hashes, IndexHashRecord{
		FileID: fileID,
		Path:   file.RelativePath,
		Hash:   hashBytes(data),
	})

	fileTerms := searchTerms(file.RelativePath + "\n" + string(data))
	b.addReverseRecord("file", fileID, fileTerms)

	if fileRecord.Language == "go" {
		symbols := extractGoSymbols(file.RelativePath, data)
		for _, symbol := range symbols {
			symbol.ID = b.nextSymbolID
			symbol.FileID = fileID
			b.nextSymbolID++
			b.symbols = append(b.symbols, symbol)
			symbolTerms := searchTerms(symbol.Name + "\n" + symbol.QualifiedName + "\n" + symbol.Kind + "\n" + symbol.PackageName + "\n" + symbol.Receiver)
			b.addReverseRecord("symbol", symbol.ID, symbolTerms)
		}
	}

	return nil
}

func (b *indexBuilder) addReverseRecord(kind string, id int, terms []string) {
	termIDs := make([]int, 0, len(terms))
	for _, term := range terms {
		termID := b.ensureTermID(term)
		termIDs = append(termIDs, termID)
		if kind == "file" {
			if _, ok := b.termFileIDs[termID]; !ok {
				b.termFileIDs[termID] = map[int]struct{}{}
			}
			b.termFileIDs[termID][id] = struct{}{}
			continue
		}
		if _, ok := b.termSymbolIDs[termID]; !ok {
			b.termSymbolIDs[termID] = map[int]struct{}{}
		}
		b.termSymbolIDs[termID][id] = struct{}{}
	}

	sort.Ints(termIDs)
	b.reverse = append(b.reverse, IndexReverseRecord{
		Kind:    kind,
		ID:      id,
		TermIDs: termIDs,
	})
}

func (b *indexBuilder) ensureTermID(term string) int {
	if termID, ok := b.termIDs[term]; ok {
		return termID
	}
	termID := b.nextTermID
	b.nextTermID++
	b.termIDs[term] = termID
	return termID
}

func (b *indexBuilder) terms() []IndexTermRecord {
	terms := make([]IndexTermRecord, 0, len(b.termIDs))
	for term, id := range b.termIDs {
		terms = append(terms, IndexTermRecord{ID: id, Term: term})
	}
	sort.Slice(terms, func(i, j int) bool {
		return terms[i].ID < terms[j].ID
	})
	return terms
}

func (b *indexBuilder) forward() []IndexForwardRecord {
	records := make([]IndexForwardRecord, 0, len(b.termIDs))
	for _, term := range b.terms() {
		records = append(records, IndexForwardRecord{
			TermID:    term.ID,
			FileIDs:   sortedIDSet(b.termFileIDs[term.ID]),
			SymbolIDs: sortedIDSet(b.termSymbolIDs[term.ID]),
		})
	}
	return records
}

func sortedIDSet(values map[int]struct{}) []int {
	if len(values) == 0 {
		return []int{}
	}
	result := make([]int, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Ints(result)
	return result
}

func detectIndexLanguage(extension string) string {
	switch extension {
	case ".go":
		return "go"
	case ".md":
		return "markdown"
	case ".txt":
		return "text"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	default:
		if extension == "" {
			return ""
		}
		return strings.TrimPrefix(extension, ".")
	}
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func extractGoSymbols(relativePath string, data []byte) []IndexSymbolRecord {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, relativePath, data, parser.SkipObjectResolution)
	if err != nil {
		return nil
	}

	symbols := []IndexSymbolRecord{
		{
			Name:          file.Name.Name,
			QualifiedName: file.Name.Name,
			Kind:          "package",
			PackageName:   file.Name.Name,
			LineStart:     fset.Position(file.Package).Line,
			LineEnd:       fset.Position(file.Name.End()).Line,
			FilePath:      relativePath,
		},
	}

	for _, spec := range file.Imports {
		pathValue, _ := strconv.Unquote(spec.Path.Value)
		lineStart := fset.Position(spec.Pos()).Line
		lineEnd := fset.Position(spec.End()).Line
		symbols = append(symbols, IndexSymbolRecord{
			Name:          pathValue,
			QualifiedName: pathValue,
			Kind:          "import",
			PackageName:   file.Name.Name,
			LineStart:     lineStart,
			LineEnd:       lineEnd,
			FilePath:      relativePath,
		})
	}

	for _, decl := range file.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range typed.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				symbols = append(symbols, IndexSymbolRecord{
					Name:          typeSpec.Name.Name,
					QualifiedName: typeSpec.Name.Name,
					Kind:          "type",
					PackageName:   file.Name.Name,
					LineStart:     fset.Position(typeSpec.Pos()).Line,
					LineEnd:       fset.Position(typeSpec.End()).Line,
					FilePath:      relativePath,
				})
			}
		case *ast.FuncDecl:
			receiver := ""
			qualifiedName := typed.Name.Name
			kind := "func"
			if typed.Recv != nil && len(typed.Recv.List) > 0 {
				kind = "method"
				receiver = goReceiverName(typed.Recv.List[0].Type)
				if receiver != "" {
					qualifiedName = receiver + "." + typed.Name.Name
				}
			}
			symbols = append(symbols, IndexSymbolRecord{
				Name:          typed.Name.Name,
				QualifiedName: qualifiedName,
				Kind:          kind,
				PackageName:   file.Name.Name,
				Receiver:      receiver,
				LineStart:     fset.Position(typed.Pos()).Line,
				LineEnd:       fset.Position(typed.End()).Line,
				FilePath:      relativePath,
			})
		}
	}

	return symbols
}

func goReceiverName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		return goReceiverName(typed.X)
	case *ast.IndexExpr:
		return goReceiverName(typed.X)
	case *ast.IndexListExpr:
		return goReceiverName(typed.X)
	case *ast.SelectorExpr:
		left := goReceiverName(typed.X)
		if left == "" {
			return typed.Sel.Name
		}
		return left + "." + typed.Sel.Name
	default:
		return ""
	}
}

func indexSymbolSearchScore(symbol IndexSymbolRecord, lowerQuery string, queryTokens []string) int {
	name := strings.ToLower(symbol.Name)
	qualified := strings.ToLower(symbol.QualifiedName)
	score := 0
	switch {
	case qualified == lowerQuery || name == lowerQuery:
		score += 100
	case strings.HasPrefix(qualified, lowerQuery) || strings.HasPrefix(name, lowerQuery):
		score += 50
	case strings.Contains(qualified, lowerQuery) || strings.Contains(name, lowerQuery):
		score += 25
	}

	for _, token := range queryTokens {
		if strings.Contains(qualified, token) {
			score += 5
		}
		if strings.Contains(strings.ToLower(symbol.Kind), token) {
			score += 2
		}
		if strings.Contains(strings.ToLower(symbol.FilePath), token) {
			score++
		}
	}
	return score
}
