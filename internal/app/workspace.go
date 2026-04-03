package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

var errWorkspaceNotBoundToProjectPath = errors.New("no workspace bound to project path")

var runtimePassthroughEnvKeys = []string{
	"SHELL",
	"TERM",
	"TERM_PROGRAM",
	"TERM_PROGRAM_VERSION",
	"COLORTERM",
	"LANG",
	"LC_ALL",
	"LC_CTYPE",
	"LC_COLLATE",
	"LC_MESSAGES",
	"LC_MONETARY",
	"LC_NUMERIC",
	"LC_TIME",
	"TMPDIR",
	"USER",
	"LOGNAME",
	"SSH_AUTH_SOCK",
	"DISPLAY",
	"WAYLAND_DISPLAY",
	"XAUTHORITY",
}

type workspaceRuntimeMode struct {
	baseEnv       func() []string
	hostPath      func() []string
	isolateHome   bool
	includePrompt bool
}

var (
	strictWorkspaceRuntimeMode = workspaceRuntimeMode{
		isolateHome:   true,
		includePrompt: true,
	}
	softOpenRuntimeMode = workspaceRuntimeMode{
		isolateHome:   false,
		includePrompt: false,
	}
)

func (a *App) CreateNewWorkspace(name string) error {
	if name == "" || name == "." || name == ".." || strings.Contains(name, "/") {
		return fmt.Errorf("invalid workspace name %q", name)
	}
	if err := a.Init(); err != nil {
		return err
	}
	wsPath := filepath.Join(a.WorkspaceDir(), name)

	if _, err := os.Stat(wsPath); err == nil {
		return fmt.Errorf("workspace %q already exists", name)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("workspace stat %s: %w", wsPath, err)
	}

	for _, d := range []string{
		wsPath,
		filepath.Join(wsPath, "home"),
		filepath.Join(wsPath, "state"),
		filepath.Join(wsPath, "logs"),
	} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}

	err := a.CreateManifest(name)
	if err != nil {
		return err
	}
	return nil
}

func (a *App) DeleteWorkspace(name string) error {
	if name == "" || name == "." || name == ".." || strings.Contains(name, "/") {
		return fmt.Errorf("invalid workspace name %q", name)
	}

	wsPath := filepath.Join(a.WorkspaceDir(), name)

	if _, err := os.Stat(wsPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("workspace %q doesn't exist", name)
		}
		return fmt.Errorf("stat workspace %s: %w", wsPath, err)
	}

	if err := os.RemoveAll(wsPath); err != nil {
		return fmt.Errorf("remove workspace %s: %w", wsPath, err)
	}

	return nil
}

func (a *App) WorkspaceShell(name string) error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	args := []string{}
	base := filepath.Base(shell)
	if base == "bash" || base == "zsh" {
		args = append(args, "-i")
	}

	return a.ExecWorkspace(name, shell, args)
}

func (a *App) ExecWorkspace(name, command string, args []string) error {
	return a.runWorkspaceProcess(a.strictRuntimeMode(), name, command, args)
}

func (a *App) OpenWorkspace(name, program string, args []string) error {
	if program == "" {
		program = defaultIDEProgram()
	}
	workDir, err := a.workspaceWorkDir(name)
	if err != nil {
		return err
	}
	openArgs := append([]string{}, args...)
	if len(openArgs) == 0 {
		openArgs = defaultOpenArgs(program, workDir)
	}

	return a.runWorkspaceProcess(a.softOpenMode(), name, program, openArgs)
}

func (a *App) WorkspaceEnv(name string) (string, error) {
	env, workDir, err := a.workspaceRuntime(name)
	if err != nil {
		return "", err
	}

	envMap := make(map[string]string, len(env)+1)
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			continue
		}
		if key == "PS1" || key == "PROMPT" {
			continue
		}
		envMap[key] = value
	}
	envMap["GROOT_WORKDIR"] = workDir

	keys := make([]string, 0, len(envMap))
	for key := range envMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, key := range keys {
		fmt.Fprintf(&b, "export %s=%s\n", key, shellQuote(envMap[key]))
	}

	return b.String(), nil
}

func (a *App) ShellHook() (string, error) {
	name := strings.TrimSpace(os.Getenv("GROOT_WORKSPACE"))
	if name == "" {
		return "", nil
	}
	return a.WorkspaceEnv(name)
}

func (a *App) workspaceRuntime(name string) ([]string, string, error) {
	return a.workspaceRuntimeForMode(name, a.strictRuntimeMode())
}

func (a *App) workspaceOpenRuntime(name string) ([]string, string, error) {
	return a.workspaceRuntimeForMode(name, a.softOpenMode())
}

func (a *App) strictRuntimeMode() workspaceRuntimeMode {
	mode := strictWorkspaceRuntimeMode
	mode.baseEnv = a.runtimeBaseEnv
	mode.hostPath = a.runtimeHostPathEntries
	return mode
}

func (a *App) softOpenMode() workspaceRuntimeMode {
	mode := softOpenRuntimeMode
	mode.baseEnv = os.Environ
	mode.hostPath = a.runtimeOpenHostPathEntries
	return mode
}

func (a *App) runWorkspaceProcess(mode workspaceRuntimeMode, name, command string, args []string) error {
	env, workDir, err := a.workspaceRuntimeForMode(name, mode)
	if err != nil {
		return err
	}

	cmd := exec.Command(command, args...)
	cmd.Dir = workDir
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (a *App) workspaceWorkDir(name string) (string, error) {
	wsPath, err := a.EnsureWorkspace(name)
	if err != nil {
		return "", err
	}
	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return "", err
	}
	if manifest.ProjectPath != "" {
		return manifest.ProjectPath, nil
	}
	return wsPath, nil
}

func (a *App) workspaceRuntimeForMode(name string, mode workspaceRuntimeMode) ([]string, string, error) {
	wsPath, err := a.EnsureWorkspace(name)
	if err != nil {
		return nil, "", err
	}
	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return nil, "", err
	}

	toolchainBinDirs := make([]string, 0, len(manifest.Packages))
	toolchainEnv := make(map[string]string)
	seenBinDirs := make(map[string]struct{}, len(manifest.Packages))
	for _, pkg := range manifest.Packages {
		if err := a.ensureToolchainInstalled(pkg); err != nil {
			return nil, "", err
		}

		binDir, err := a.toolchainBinDir(pkg)
		if err != nil {
			return nil, "", err
		}
		if _, ok := seenBinDirs[binDir]; ok {
			continue
		}
		seenBinDirs[binDir] = struct{}{}
		toolchainBinDirs = append(toolchainBinDirs, binDir)

		extraEnv, err := a.toolchainEnv(pkg)
		if err != nil {
			return nil, "", err
		}
		for key, value := range extraEnv {
			toolchainEnv[key] = value
		}
	}

	wsHome := filepath.Join(wsPath, "home")
	workDir := wsPath
	if manifest.ProjectPath != "" {
		workDir = manifest.ProjectPath
	}

	env := mode.baseEnv()
	env = a.setEnv(env, "GROOT_HOME", a.Root)
	env = a.setEnv(env, "GROOT_WORKSPACE", name)
	env = a.setEnv(env, "GROOT_WORKSPACE_DIR", wsPath)
	env = a.setEnv(env, "GROOT_WORKDIR", workDir)
	if mode.isolateHome {
		env = a.setEnv(env, "HOME", wsHome)
		env = a.setEnv(env, "XDG_CONFIG_HOME", filepath.Join(wsHome, ".config"))
		env = a.setEnv(env, "XDG_CACHE_HOME", filepath.Join(wsHome, ".cache"))
		env = a.setEnv(env, "XDG_DATA_HOME", filepath.Join(wsHome, ".local", "share"))
	}
	for key, value := range toolchainEnv {
		env = a.setEnv(env, key, value)
	}
	pathParts := a.uniquePaths(toolchainBinDirs, mode.hostPath())
	if len(pathParts) > 0 {
		env = a.setEnv(env, "PATH", strings.Join(pathParts, string(os.PathListSeparator)))
	}

	if mode.includePrompt {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
		base := filepath.Base(shell)
		if base == "zsh" {
			p := fmt.Sprintf("(groot:%s) %%n@%%m %%1~ %%# ", name)
			env = a.setEnv(env, "PROMPT", p)
			env = a.setEnv(env, "PS1", p)
		} else {
			env = a.setEnv(env, "PS1", fmt.Sprintf("(groot:%s) ", name)+"$PS1")
		}
	}

	return env, workDir, nil
}

func (a *App) runtimeBaseEnv() []string {
	env := make([]string, 0, len(runtimePassthroughEnvKeys))
	for _, key := range runtimePassthroughEnvKeys {
		value, ok := os.LookupEnv(key)
		if !ok || value == "" {
			continue
		}
		env = append(env, key+"="+value)
	}
	return env
}

func (a *App) runtimeHostPathEntries() []string {
	currentPath := os.Getenv("PATH")
	if currentPath == "" {
		return nil
	}

	homeDir, _ := os.UserHomeDir()
	filtered := make([]string, 0)
	for _, entry := range strings.Split(currentPath, string(os.PathListSeparator)) {
		if entry == "" {
			continue
		}

		clean := filepath.Clean(entry)
		if !filepath.IsAbs(clean) {
			continue
		}

		if homeDir != "" && pathWithin(clean, homeDir) {
			continue
		}

		filtered = append(filtered, clean)
	}

	return filtered
}

func (a *App) runtimeOpenHostPathEntries() []string {
	currentPath := os.Getenv("PATH")
	if currentPath == "" {
		return nil
	}

	entries := make([]string, 0)
	for _, entry := range strings.Split(currentPath, string(os.PathListSeparator)) {
		if entry == "" {
			continue
		}
		clean := filepath.Clean(entry)
		if !filepath.IsAbs(clean) {
			continue
		}
		entries = append(entries, clean)
	}
	return entries
}

func (a *App) uniquePaths(groups ...[]string) []string {
	seen := make(map[string]struct{})
	paths := make([]string, 0)

	for _, group := range groups {
		for _, entry := range group {
			if entry == "" {
				continue
			}

			clean := filepath.Clean(entry)
			if _, ok := seen[clean]; ok {
				continue
			}
			seen[clean] = struct{}{}
			paths = append(paths, clean)
		}
	}

	return paths
}

func pathWithin(path, root string) bool {
	if path == root {
		return true
	}

	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func defaultOpenArgs(program, workDir string) []string {
	base := filepath.Base(program)
	switch base {
	case "code", "cursor", "code-insiders":
		return []string{"-n", workDir}
	default:
		return []string{workDir}
	}
}

func defaultIDEProgram() string {
	for _, key := range []string{"GROOT_IDE", "VISUAL", "EDITOR"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return "code"
}

func normalizeProjectPath(projectPath string) (string, error) {
	projectPath = strings.TrimSpace(projectPath)
	if projectPath == "" {
		return "", fmt.Errorf("project path required")
	}

	if strings.HasPrefix(projectPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		projectPath = filepath.Join(home, strings.TrimPrefix(projectPath, "~"))
	}

	absPath, err := filepath.Abs(filepath.Clean(projectPath))
	if err != nil {
		return "", fmt.Errorf("resolve project path %q: %w", projectPath, err)
	}

	return absPath, nil
}

func sameProjectPath(left, right string) (bool, error) {
	normalizedLeft, err := normalizeProjectPath(left)
	if err != nil {
		return false, err
	}
	normalizedRight, err := normalizeProjectPath(right)
	if err != nil {
		return false, err
	}
	if normalizedLeft == normalizedRight {
		return true, nil
	}

	leftResolved, err := filepath.EvalSymlinks(normalizedLeft)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("resolve symlinks for %q: %w", normalizedLeft, err)
	}
	rightResolved, err := filepath.EvalSymlinks(normalizedRight)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("resolve symlinks for %q: %w", normalizedRight, err)
	}

	return leftResolved == rightResolved, nil
}

func workspaceNameFromProjectPath(projectPath string) string {
	base := strings.TrimSpace(filepath.Base(projectPath))
	if base == "" || base == "." || base == ".." {
		base = "workspace"
	}

	var b strings.Builder
	lastDash := false
	for _, r := range base {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}

	name := strings.Trim(b.String(), "-.")
	if name == "" || name == "." || name == ".." {
		return "workspace"
	}
	return name
}

func (a *App) nextAvailableWorkspaceName(base string) (string, error) {
	if err := a.Init(); err != nil {
		return "", err
	}

	for i := 0; ; i++ {
		name := base
		if i > 0 {
			name = fmt.Sprintf("%s-%d", base, i+1)
		}

		_, err := os.Stat(filepath.Join(a.WorkspaceDir(), name))
		if os.IsNotExist(err) {
			return name, nil
		}
		if err != nil {
			return "", fmt.Errorf("stat workspace %q: %w", name, err)
		}
	}
}

func (a *App) AttachToWorkspace(name string, args []string) error {
	wsPath, err := a.EnsureWorkspace(name)
	if err != nil {
		return err
	}
	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return err
	}
	components, err := a.parseComponents(args)
	if err != nil {
		return err
	}
	for _, comp := range components {
		updated := false
		for i := range manifest.Packages {
			if manifest.Packages[i].Name == comp.Name {
				manifest.Packages[i].Version = comp.Version
				updated = true
				break
			}
		}
		if !updated {
			manifest.Packages = append(manifest.Packages, comp)
		}
	}

	return a.writeManifest(wsPath, manifest)
}

func (a *App) BindWorkspace(name, projectPath string) error {
	wsPath, err := a.EnsureWorkspace(name)
	if err != nil {
		return err
	}

	absPath, err := normalizeProjectPath(projectPath)
	if err != nil {
		return err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("project path %q does not exist", absPath)
		}
		return fmt.Errorf("stat project path %q: %w", absPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("project path %q is not a directory", absPath)
	}

	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return err
	}
	manifest.ProjectPath = absPath

	return a.writeManifest(wsPath, manifest)
}

func (a *App) UnbindWorkspace(name string) error {
	wsPath, err := a.EnsureWorkspace(name)
	if err != nil {
		return err
	}

	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return err
	}
	manifest.ProjectPath = ""

	return a.writeManifest(wsPath, manifest)
}

func (a *App) FindWorkspaceByProjectPath(projectPath string) (string, error) {
	normalizedPath, err := normalizeProjectPath(projectPath)
	if err != nil {
		return "", err
	}

	if err := a.Init(); err != nil {
		return "", err
	}

	entries, err := os.ReadDir(a.WorkspaceDir())
	if err != nil {
		return "", fmt.Errorf("read workspaces dir %q: %w", a.WorkspaceDir(), err)
	}

	matches := make([]string, 0, 1)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		wsPath := filepath.Join(a.WorkspaceDir(), entry.Name())
		manifest, err := a.getManifest(wsPath)
		if err != nil {
			return "", err
		}
		if manifest.ProjectPath == "" {
			continue
		}

		match, err := sameProjectPath(manifest.ProjectPath, normalizedPath)
		if err != nil {
			return "", err
		}
		if match {
			matches = append(matches, manifest.Name)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("%w %q", errWorkspaceNotBoundToProjectPath, normalizedPath)
	case 1:
		return matches[0], nil
	default:
		sort.Strings(matches)
		return "", fmt.Errorf("multiple workspaces bound to project path %q: %s", normalizedPath, strings.Join(matches, ", "))
	}
}

func (a *App) ResolveOrCreateWorkspaceByProjectPath(projectPath string) (string, bool, error) {
	normalizedPath, err := normalizeProjectPath(projectPath)
	if err != nil {
		return "", false, err
	}

	info, err := os.Stat(normalizedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, fmt.Errorf("project path %q does not exist", normalizedPath)
		}
		return "", false, fmt.Errorf("stat project path %q: %w", normalizedPath, err)
	}
	if !info.IsDir() {
		return "", false, fmt.Errorf("project path %q is not a directory", normalizedPath)
	}

	name, err := a.FindWorkspaceByProjectPath(normalizedPath)
	if err == nil {
		return name, false, nil
	}
	if !errors.Is(err, errWorkspaceNotBoundToProjectPath) {
		return "", false, err
	}

	name, err = a.nextAvailableWorkspaceName(workspaceNameFromProjectPath(normalizedPath))
	if err != nil {
		return "", false, err
	}
	if err := a.CreateNewWorkspace(name); err != nil {
		return "", false, err
	}
	if err := a.BindWorkspace(name, normalizedPath); err != nil {
		return "", false, err
	}

	return name, true, nil
}

func (a *App) InstallToWorkspace(name string) error {
	wsPath, err := a.EnsureWorkspace(name)
	if err != nil {
		return err
	}
	manifest, err := a.getManifest(wsPath)
	if err != nil {
		return err
	}
	if len(manifest.Packages) == 0 {
		return nil
	}
	for _, pkg := range manifest.Packages {
		if err := a.ensureToolchainInstalled(pkg); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) GarbageCollectToolchains() error {
	if err := a.Init(); err != nil {
		return err
	}

	referenced := make(map[string]map[string]struct{})
	workspaceEntries, err := os.ReadDir(a.WorkspaceDir())
	if err != nil {
		return fmt.Errorf("read workspaces: %w", err)
	}

	for _, entry := range workspaceEntries {
		if !entry.IsDir() {
			continue
		}

		manifest, err := a.getManifest(filepath.Join(a.WorkspaceDir(), entry.Name()))
		if err != nil {
			return err
		}

		for _, pkg := range manifest.Packages {
			if _, ok := referenced[pkg.Name]; !ok {
				referenced[pkg.Name] = make(map[string]struct{})
			}
			referenced[pkg.Name][pkg.Version] = struct{}{}
		}
	}

	for name := range a.toolchains {
		toolchainRoot := filepath.Join(a.ToolchainDir(), name)
		entries, err := os.ReadDir(toolchainRoot)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("read toolchain root %s: %w", toolchainRoot, err)
		}

		keep := referenced[name]
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			if _, ok := keep[entry.Name()]; ok {
				continue
			}

			if err := os.RemoveAll(filepath.Join(toolchainRoot, entry.Name())); err != nil {
				return fmt.Errorf("remove toolchain %s/%s: %w", name, entry.Name(), err)
			}
		}
	}

	return nil
}

func (a *App) EnsureWorkspace(name string) (string, error) {
	if name == "" || name == "." || name == ".." || strings.Contains(name, "/") {
		return "", fmt.Errorf("invalid workspace name %q", name)
	}
	if err := a.Init(); err != nil {
		return "", err
	}

	wsPath := filepath.Join(a.WorkspaceDir(), name)

	if _, err := os.Stat(wsPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("workspace %q doesn't exist (run: groot ws create %s)", name, name)
		}
		return "", fmt.Errorf("stat workspace %s: %w", wsPath, err)
	}

	return wsPath, nil
}
