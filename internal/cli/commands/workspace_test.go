package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/totoual/groot/internal/app"
	"github.com/totoual/groot/internal/cli/interfaces"
)

type stubCmd struct {
	name    string
	help    string
	ranArgs []string
}

func (s *stubCmd) Name() string { return s.name }
func (s *stubCmd) Help() string { return s.help }
func (s *stubCmd) Run(_ *app.App, args []string) error {
	s.ranArgs = append([]string{}, args...)
	return nil
}

var _ interfaces.Cmd = (*stubCmd)(nil)

func TestWorkspaceCmdRunDispatchesSubcommand(t *testing.T) {
	cmd := &stubCmd{name: "bind", help: "Bind workspace"}
	ws := NewWorkspaceCmd(cmd)

	if err := ws.Run(nil, []string{"bind", "crawlly", "/tmp/repo"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(cmd.ranArgs) != 2 || cmd.ranArgs[0] != "crawlly" || cmd.ranArgs[1] != "/tmp/repo" {
		t.Fatalf("unexpected ranArgs: %#v", cmd.ranArgs)
	}
}

func TestWorkspaceCmdRunRejectsUnknownSubcommand(t *testing.T) {
	ws := NewWorkspaceCmd(&stubCmd{name: "bind", help: "Bind workspace"})

	err := ws.Run(nil, []string{"nope"})
	if err == nil {
		t.Fatal("expected unknown subcommand error")
	}
	if !strings.Contains(err.Error(), `unknown ws command "nope"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkspaceCmdPrintHelpIncludesSortedCommands(t *testing.T) {
	ws := NewWorkspaceCmd(
		&stubCmd{name: "shell", help: "Activate"},
		&stubCmd{name: "attach", help: "Attach"},
		&stubCmd{name: "bind", help: "Bind"},
	)

	var buf bytes.Buffer
	ws.PrintHelp(&buf)
	output := buf.String()

	if !strings.Contains(output, "usage: groot ws <command> [args]") {
		t.Fatalf("unexpected help output: %q", output)
	}
	attachIdx := strings.Index(output, "attach")
	bindIdx := strings.Index(output, "bind")
	shellIdx := strings.Index(output, "shell")
	if !(attachIdx < bindIdx && bindIdx < shellIdx) {
		t.Fatalf("expected sorted commands, got output %q", output)
	}
}
