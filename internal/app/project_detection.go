package app

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type DetectedToolchain struct {
	Name    string
	Version string
	Source  string
}

func (a *App) MissingWorkspaceToolchains(name string, detected []DetectedToolchain) ([]DetectedToolchain, error) {
	wsPath, err := a.EnsureWorkspace(name)
	if err != nil {
		return nil, err
	}

	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return nil, err
	}

	attached := make(map[string]struct{}, len(manifest.Packages))
	for _, pkg := range manifest.Packages {
		attached[pkg.Name] = struct{}{}
	}

	missing := make([]DetectedToolchain, 0, len(detected))
	for _, tc := range detected {
		if _, ok := attached[tc.Name]; ok {
			continue
		}
		missing = append(missing, tc)
	}

	return missing, nil
}

func (a *App) WorkspaceUndeclaredToolchains(name string) ([]DetectedToolchain, error) {
	wsPath, err := a.EnsureWorkspace(name)
	if err != nil {
		return nil, err
	}

	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return nil, err
	}
	if manifest.ProjectPath == "" {
		return nil, nil
	}

	detected, err := a.DetectProjectToolchains(manifest.ProjectPath)
	if err != nil {
		return nil, err
	}

	return a.MissingWorkspaceToolchains(name, detected)
}

func (a *App) AttachDetectedToolchains(name string, detected []DetectedToolchain) ([]DetectedToolchain, []DetectedToolchain, error) {
	attachable := make([]string, 0, len(detected))
	attached := make([]DetectedToolchain, 0, len(detected))
	skipped := make([]DetectedToolchain, 0, len(detected))

	for _, tc := range detected {
		if tc.Version == "" {
			skipped = append(skipped, tc)
			continue
		}
		attachable = append(attachable, fmt.Sprintf("%s@%s", tc.Name, tc.Version))
		attached = append(attached, tc)
	}

	if len(attachable) > 0 {
		if err := a.AttachToWorkspace(name, attachable); err != nil {
			return nil, nil, err
		}
	}

	return attached, skipped, nil
}

var ignoredProjectScanDirs = map[string]struct{}{
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

func (a *App) DetectProjectToolchains(projectPath string) ([]DetectedToolchain, error) {
	normalizedPath, err := normalizeProjectPath(projectPath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(normalizedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("project path %q does not exist", normalizedPath)
		}
		return nil, fmt.Errorf("stat project path %q: %w", normalizedPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project path %q is not a directory", normalizedPath)
	}

	detected := make(map[string]DetectedToolchain)
	if err := filepath.WalkDir(normalizedPath, func(path string, d fs.DirEntry, walkErr error) error {
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
			if _, ignored := ignoredProjectScanDirs[d.Name()]; ignored {
				return filepath.SkipDir
			}
			if depth >= 3 {
				return filepath.SkipDir
			}
			return nil
		}
		if depth > 3 {
			return nil
		}

		toolchains, err := detectToolchainsFromFile(path, rel)
		if err != nil {
			return err
		}
		for _, tc := range toolchains {
			current, exists := detected[tc.Name]
			if !exists || (current.Version == "" && tc.Version != "") {
				detected[tc.Name] = tc
			}
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("scan project path %q: %w", normalizedPath, err)
	}

	results := make([]DetectedToolchain, 0, len(detected))
	for _, tc := range detected {
		results = append(results, tc)
	}
	slices.SortFunc(results, func(left, right DetectedToolchain) int {
		return strings.Compare(left.Name, right.Name)
	})

	return results, nil
}

func detectToolchainsFromFile(path, rel string) ([]DetectedToolchain, error) {
	switch filepath.Base(path) {
	case "go.mod":
		version, err := detectGoVersion(path)
		if err != nil {
			return nil, err
		}
		return []DetectedToolchain{{Name: "go", Version: version, Source: rel}}, nil
	case "package.json":
		return detectNodeAndBun(path, rel)
	case ".nvmrc", ".node-version":
		version, err := readVersionHint(path)
		if err != nil {
			return nil, err
		}
		return []DetectedToolchain{{Name: "node", Version: version, Source: rel}}, nil
	case "deno.json", "deno.jsonc", "deno.lock":
		return []DetectedToolchain{{Name: "deno", Source: rel}}, nil
	case "bun.lock", "bun.lockb":
		return []DetectedToolchain{{Name: "bun", Source: rel}}, nil
	case ".python-version":
		version, err := readVersionHint(path)
		if err != nil {
			return nil, err
		}
		return []DetectedToolchain{{Name: "python", Version: version, Source: rel}}, nil
	case "pyproject.toml", "requirements.txt", "requirements-dev.txt":
		return []DetectedToolchain{{Name: "python", Source: rel}}, nil
	case "rust-toolchain", "rust-toolchain.toml":
		version, err := detectRustVersion(path)
		if err != nil {
			return nil, err
		}
		return []DetectedToolchain{{Name: "rust", Version: version, Source: rel}}, nil
	case "Cargo.toml":
		return []DetectedToolchain{{Name: "rust", Source: rel}}, nil
	case ".php-version":
		version, err := readVersionHint(path)
		if err != nil {
			return nil, err
		}
		return []DetectedToolchain{{Name: "php", Version: version, Source: rel}}, nil
	case "composer.json":
		return []DetectedToolchain{{Name: "php", Source: rel}}, nil
	case ".java-version":
		version, err := readVersionHint(path)
		if err != nil {
			return nil, err
		}
		return []DetectedToolchain{{Name: "java", Version: version, Source: rel}}, nil
	case "pom.xml", "build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts":
		return []DetectedToolchain{{Name: "java", Source: rel}}, nil
	default:
		return nil, nil
	}
}

func detectGoVersion(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "toolchain ") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "toolchain "))
			value = strings.TrimPrefix(value, "go")
			if version := normalizeVersionHint(value); version != "" {
				return version, nil
			}
		}
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "go ") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "go "))
			if version := normalizeVersionHint(value); version != "" {
				return version, nil
			}
		}
	}

	return "", nil
}

func detectNodeAndBun(path, rel string) ([]DetectedToolchain, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pkg struct {
		Engines struct {
			Node string `json:"node"`
		} `json:"engines"`
		PackageManager string `json:"packageManager"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return []DetectedToolchain{{Name: "node", Source: rel}}, nil
	}

	results := []DetectedToolchain{
		{Name: "node", Version: normalizeVersionHint(pkg.Engines.Node), Source: rel},
	}
	if manager, version, ok := strings.Cut(pkg.PackageManager, "@"); ok && strings.TrimSpace(manager) == "bun" {
		results = append(results, DetectedToolchain{
			Name:    "bun",
			Version: normalizeVersionHint(version),
			Source:  rel,
		})
	}

	return results, nil
}

func detectRustVersion(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	content := strings.TrimSpace(string(data))
	if strings.HasSuffix(filepath.Base(path), ".toml") {
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "channel") {
				continue
			}
			_, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			return normalizeVersionHint(value), nil
		}
		return "", nil
	}

	return normalizeVersionHint(content), nil
}

func readVersionHint(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return normalizeVersionHint(string(data)), nil
}

func normalizeVersionHint(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	value = strings.SplitN(value, "\n", 2)[0]
	value = strings.Trim(value, `"'`)
	value = strings.TrimPrefix(value, "v")
	if value == "" {
		return ""
	}

	for _, disallowed := range []string{" ", ">", "<", "=", "~", "^", "*", "|", ","} {
		if strings.Contains(value, disallowed) {
			return ""
		}
	}
	return value
}
