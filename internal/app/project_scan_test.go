package app

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestWalkWorkspaceProjectFilesSkipsIgnoredDirectories(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)

	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	projectPath := filepath.Join(root, "repos", "crawlly")
	for _, dir := range []string{
		projectPath,
		filepath.Join(projectPath, "internal"),
		filepath.Join(projectPath, "node_modules", "react"),
		filepath.Join(projectPath, ".git"),
		filepath.Join(projectPath, "dist"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) returned error: %v", dir, err)
		}
	}
	for path := range map[string]string{
		filepath.Join(projectPath, "go.mod"):                     "module crawlly\n",
		filepath.Join(projectPath, "internal", "main.go"):        "package main\n",
		filepath.Join(projectPath, "node_modules", "react", "x"): "ignored\n",
		filepath.Join(projectPath, ".git", "config"):             "ignored\n",
		filepath.Join(projectPath, "dist", "bundle.js"):          "ignored\n",
	} {
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatalf("WriteFile(%q) returned error: %v", path, err)
		}
	}
	if err := app.BindWorkspace("crawlly", projectPath); err != nil {
		t.Fatalf("BindWorkspace returned error: %v", err)
	}

	var got []string
	if err := app.WalkWorkspaceProjectFiles("crawlly", ProjectScanOptions{}, func(file ProjectFile) error {
		got = append(got, file.RelativePath)
		return nil
	}); err != nil {
		t.Fatalf("WalkWorkspaceProjectFiles returned error: %v", err)
	}

	slices.Sort(got)
	want := []string{"go.mod", "internal/main.go"}
	if !slices.Equal(got, want) {
		t.Fatalf("walked files = %#v, want %#v", got, want)
	}
}

func TestWorkspaceProjectPathRequiresBinding(t *testing.T) {
	app := NewApp(t.TempDir())
	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	_, err := app.WorkspaceProjectPath("crawlly")
	if err == nil {
		t.Fatal("expected WorkspaceProjectPath to fail for an unbound workspace")
	}
}
