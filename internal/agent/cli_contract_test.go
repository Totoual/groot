package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIContractStatusParsesJSON(t *testing.T) {
	root := t.TempDir()
	argsFile := filepath.Join(root, "args.txt")
	scriptPath := filepath.Join(root, "groot-stub.sh")
	script := "#!/bin/sh\nprintf '%s\n' \"$@\" > \"$ARGS_FILE\"\nprintf '%s\n' '{\"workspace_name\":\"the_grime_tcg\",\"project_path\":\"/tmp/the_grime_tcg\",\"status\":\"runtime owned by Groot\",\"detected\":[{\"name\":\"go\",\"version\":\"1.25.4\"}],\"attached\":[{\"name\":\"go\",\"version\":\"1.25.4\"}],\"installed\":[{\"name\":\"go\",\"version\":\"1.25.4\"}],\"attached_uninstalled\":[],\"missing\":[]}'\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	t.Setenv("ARGS_FILE", argsFile)

	contract := NewCLIContract(scriptPath)
	status, err := contract.Status("/tmp/the_grime_tcg")
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.WorkspaceName != "the_grime_tcg" {
		t.Fatalf("WorkspaceName = %q, want %q", status.WorkspaceName, "the_grime_tcg")
	}
	if status.Status != "runtime owned by Groot" {
		t.Fatalf("Status = %q, want %q", status.Status, "runtime owned by Groot")
	}

	gotArgs, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.TrimSpace(string(gotArgs)) != "status\n/tmp/the_grime_tcg\n--json" {
		t.Fatalf("unexpected args:\n%s", string(gotArgs))
	}
}

func TestCLIContractOpenSetupUsesExpectedArgs(t *testing.T) {
	root := t.TempDir()
	argsFile := filepath.Join(root, "args.txt")
	scriptPath := filepath.Join(root, "groot-stub.sh")
	script := "#!/bin/sh\nprintf '%s\n' \"$@\" > \"$ARGS_FILE\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	t.Setenv("ARGS_FILE", argsFile)

	contract := NewCLIContract(scriptPath)
	if err := contract.OpenSetup("/tmp/the_grime_tcg"); err != nil {
		t.Fatalf("OpenSetup returned error: %v", err)
	}

	gotArgs, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.TrimSpace(string(gotArgs)) != "open\n/tmp/the_grime_tcg\n--setup" {
		t.Fatalf("unexpected args:\n%s", string(gotArgs))
	}
}

func TestCLIContractExecUsesExpectedArgs(t *testing.T) {
	root := t.TempDir()
	argsFile := filepath.Join(root, "args.txt")
	scriptPath := filepath.Join(root, "groot-stub.sh")
	script := "#!/bin/sh\nprintf '%s\n' \"$@\" > \"$ARGS_FILE\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	t.Setenv("ARGS_FILE", argsFile)

	contract := NewCLIContract(scriptPath)
	if err := contract.Exec("/tmp/the_grime_tcg", "go", []string{"test", "./..."}); err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}

	gotArgs, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.TrimSpace(string(gotArgs)) != "exec\n/tmp/the_grime_tcg\ngo\ntest\n./..." {
		t.Fatalf("unexpected args:\n%s", string(gotArgs))
	}
}

func TestCLIContractReturnsCommandOutputOnFailure(t *testing.T) {
	root := t.TempDir()
	scriptPath := filepath.Join(root, "groot-stub.sh")
	script := "#!/bin/sh\necho 'boom failure' >&2\nexit 2\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	contract := NewCLIContract(scriptPath)
	err := contract.Open("/tmp/the_grime_tcg")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "boom failure") {
		t.Fatalf("expected stderr in error, got %v", err)
	}
}

func TestCLIContractStatusRejectsInvalidJSON(t *testing.T) {
	root := t.TempDir()
	scriptPath := filepath.Join(root, "groot-stub.sh")
	script := "#!/bin/sh\necho 'not-json'\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	contract := NewCLIContract(scriptPath)
	_, err := contract.Status("/tmp/the_grime_tcg")
	if err == nil {
		t.Fatal("expected json parse error")
	}
	if !strings.Contains(err.Error(), "parse groot status json") {
		t.Fatalf("unexpected error: %v", err)
	}
}
