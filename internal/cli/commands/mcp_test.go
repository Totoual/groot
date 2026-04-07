package commands

import (
	"strings"
	"testing"

	"github.com/totoual/groot/internal/app"
)

func TestMCPCmdHelp(t *testing.T) {
	a := app.NewApp(t.TempDir())

	stdout, stderr, err := captureCommandOutput(func() error {
		return (&MCPCmd{}).Run(a, []string{"-h"})
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected stderr to stay quiet, got %q", stderr)
	}
	if !strings.Contains(stdout, "usage: groot mcp") {
		t.Fatalf("expected usage in stdout, got %q", stdout)
	}
}
