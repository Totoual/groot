package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectProjectToolchainsFindsMonorepoRuntimesRecursively(t *testing.T) {
	root := t.TempDir()
	app := NewApp(t.TempDir())

	backendDir := filepath.Join(root, "backend")
	frontendDir := filepath.Join(root, "frontend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(frontendDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	goMod := "module example.com/tcg\n\ngo 1.25.4\n"
	if err := os.WriteFile(filepath.Join(backendDir, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatalf("WriteFile go.mod returned error: %v", err)
	}
	packageJSON := `{"engines":{"node":"25.8.1"}}`
	if err := os.WriteFile(filepath.Join(frontendDir, "package.json"), []byte(packageJSON), 0o600); err != nil {
		t.Fatalf("WriteFile package.json returned error: %v", err)
	}

	detected, err := app.DetectProjectToolchains(root)
	if err != nil {
		t.Fatalf("DetectProjectToolchains returned error: %v", err)
	}
	if len(detected) != 2 {
		t.Fatalf("expected 2 detected toolchains, got %#v", detected)
	}

	if detected[0].Name != "go" || detected[0].Version != "1.25.4" {
		t.Fatalf("unexpected go detection: %#v", detected[0])
	}
	if detected[1].Name != "node" || detected[1].Version != "25.8.1" {
		t.Fatalf("unexpected node detection: %#v", detected[1])
	}
}

func TestDetectProjectToolchainsUsesVersionHintFiles(t *testing.T) {
	root := t.TempDir()
	app := NewApp(t.TempDir())

	if err := os.WriteFile(filepath.Join(root, ".python-version"), []byte("3.12.2\n"), 0o600); err != nil {
		t.Fatalf("WriteFile .python-version returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "rust-toolchain"), []byte("stable\n"), 0o600); err != nil {
		t.Fatalf("WriteFile rust-toolchain returned error: %v", err)
	}

	detected, err := app.DetectProjectToolchains(root)
	if err != nil {
		t.Fatalf("DetectProjectToolchains returned error: %v", err)
	}
	if len(detected) != 2 {
		t.Fatalf("expected 2 detected toolchains, got %#v", detected)
	}

	if detected[0].Name != "python" || detected[0].Version != "3.12.2" {
		t.Fatalf("unexpected python detection: %#v", detected[0])
	}
	if detected[1].Name != "rust" || detected[1].Version != "stable" {
		t.Fatalf("unexpected rust detection: %#v", detected[1])
	}
}
