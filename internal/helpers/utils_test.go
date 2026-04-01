package helpers

import (
	"path/filepath"
	"testing"
)

func TestResolveGrootHomeUsesDefaultHomeDirectory(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("GROOT_HOME", "")

	got, err := ResolveGrootHome()
	if err != nil {
		t.Fatalf("ResolveGrootHome returned error: %v", err)
	}

	want := filepath.Join(homeDir, ".groot")
	if got != want {
		t.Fatalf("ResolveGrootHome = %q, want %q", got, want)
	}
}

func TestResolveGrootHomeExpandsEnvPath(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("GROOT_HOME", "~/custom/groot")

	got, err := ResolveGrootHome()
	if err != nil {
		t.Fatalf("ResolveGrootHome returned error: %v", err)
	}

	want := filepath.Join(homeDir, "custom", "groot")
	if got != want {
		t.Fatalf("ResolveGrootHome = %q, want %q", got, want)
	}
}

func TestResolveGrootHomeReturnsAbsoluteCleanPath(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("GROOT_HOME", "./tmp-root/../tmp-root-a")

	got, err := ResolveGrootHome()
	if err != nil {
		t.Fatalf("ResolveGrootHome returned error: %v", err)
	}

	want, err := filepath.Abs("./tmp-root-a")
	if err != nil {
		t.Fatalf("filepath.Abs returned error: %v", err)
	}

	if got != filepath.Clean(want) {
		t.Fatalf("ResolveGrootHome = %q, want %q", got, filepath.Clean(want))
	}
}
