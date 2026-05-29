package app

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var defaultIgnoredWorkspaceProjectDirs = map[string]struct{}{
	".git":         {},
	".groot":       {},
	".next":        {},
	".venv":        {},
	"bin":          {},
	"build":        {},
	"dist":         {},
	"node_modules": {},
	"target":       {},
	"vendor":       {},
}

type ProjectFile struct {
	AbsolutePath string
	RelativePath string
	Extension    string
}

type ProjectScanOptions struct {
	MaxDepth    int
	IgnoredDirs map[string]struct{}
}

func (a *App) WorkspaceProjectPath(name string) (string, error) {
	wsPath, err := a.EnsureWorkspace(name)
	if err != nil {
		return "", err
	}

	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(manifest.ProjectPath) == "" {
		return "", fmt.Errorf("workspace %q is not bound to a project path", name)
	}

	projectPath, err := normalizeProjectPath(manifest.ProjectPath)
	if err != nil {
		return "", err
	}
	return projectPath, nil
}

func shouldSkipWorkspaceProjectDir(name string, ignoredDirs map[string]struct{}) bool {
	if ignoredDirs == nil {
		ignoredDirs = defaultIgnoredWorkspaceProjectDirs
	}
	_, ok := ignoredDirs[name]
	return ok
}

func walkProjectFiles(projectPath string, opts ProjectScanOptions, fn func(ProjectFile) error) error {
	normalizedPath, err := normalizeProjectPath(projectPath)
	if err != nil {
		return err
	}

	info, err := os.Stat(normalizedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("project path %q does not exist", normalizedPath)
		}
		return fmt.Errorf("stat project path %q: %w", normalizedPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("project path %q is not a directory", normalizedPath)
	}

	return filepath.WalkDir(normalizedPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(normalizedPath, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		depth := strings.Count(rel, string(os.PathSeparator))
		if d.IsDir() {
			if shouldSkipWorkspaceProjectDir(d.Name(), opts.IgnoredDirs) {
				return filepath.SkipDir
			}
			if opts.MaxDepth > 0 && depth >= opts.MaxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if opts.MaxDepth > 0 && depth > opts.MaxDepth {
			return nil
		}

		return fn(ProjectFile{
			AbsolutePath: path,
			RelativePath: filepath.ToSlash(rel),
			Extension:    strings.ToLower(filepath.Ext(path)),
		})
	})
}

func (a *App) WalkWorkspaceProjectFiles(name string, opts ProjectScanOptions, fn func(ProjectFile) error) error {
	projectPath, ignoredDirs, err := a.workspaceProjectPathAndIgnoredDirs(name)
	if err != nil {
		return err
	}
	opts.IgnoredDirs = mergeIgnoredWorkspaceProjectDirs(ignoredDirs, opts.IgnoredDirs)
	return walkProjectFiles(projectPath, opts, fn)
}

func (a *App) workspaceProjectPathAndIgnoredDirs(name string) (string, map[string]struct{}, error) {
	wsPath, err := a.EnsureWorkspace(name)
	if err != nil {
		return "", nil, err
	}

	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return "", nil, err
	}
	if strings.TrimSpace(manifest.ProjectPath) == "" {
		return "", nil, fmt.Errorf("workspace %q is not bound to a project path", name)
	}

	projectPath, err := normalizeProjectPath(manifest.ProjectPath)
	if err != nil {
		return "", nil, err
	}
	return projectPath, ignoredWorkspaceProjectDirs(manifest.Index.Ignore), nil
}

func ignoredWorkspaceProjectDirs(configured []string) map[string]struct{} {
	ignored := make(map[string]struct{}, len(defaultIgnoredWorkspaceProjectDirs)+len(configured))
	for name := range defaultIgnoredWorkspaceProjectDirs {
		ignored[name] = struct{}{}
	}
	for _, name := range configured {
		normalized, ok := normalizeWorkspaceProjectDirName(name)
		if !ok {
			continue
		}
		ignored[normalized] = struct{}{}
	}
	return ignored
}

func mergeIgnoredWorkspaceProjectDirs(base map[string]struct{}, extra map[string]struct{}) map[string]struct{} {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}

	merged := make(map[string]struct{}, len(base)+len(extra))
	for name := range base {
		merged[name] = struct{}{}
	}
	for name := range extra {
		normalized, ok := normalizeWorkspaceProjectDirName(name)
		if !ok {
			continue
		}
		merged[normalized] = struct{}{}
	}
	return merged
}

func normalizeWorkspaceProjectDirName(name string) (string, bool) {
	name = strings.TrimSpace(strings.Trim(name, `/\`))
	if name == "" || name == "." || name == ".." {
		return "", false
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return "", false
	}
	return name, true
}
