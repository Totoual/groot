package app

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var ignoredWorkspaceProjectDirs = map[string]struct{}{
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
	MaxDepth int
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

func shouldSkipWorkspaceProjectDir(name string) bool {
	_, ok := ignoredWorkspaceProjectDirs[name]
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
			if shouldSkipWorkspaceProjectDir(d.Name()) {
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
	projectPath, err := a.WorkspaceProjectPath(name)
	if err != nil {
		return err
	}
	return walkProjectFiles(projectPath, opts, fn)
}
