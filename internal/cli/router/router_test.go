package router

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

func TestRouterRunDispatchesCommand(t *testing.T) {
	cmd := &stubCmd{name: "ws", help: "Manage workspaces"}
	r := NewRouter(cmd)

	if err := r.Run(nil, []string{"ws", "create", "crawlly"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(cmd.ranArgs) != 2 || cmd.ranArgs[0] != "create" || cmd.ranArgs[1] != "crawlly" {
		t.Fatalf("unexpected ranArgs: %#v", cmd.ranArgs)
	}
}

func TestRouterRunRejectsUnknownCommand(t *testing.T) {
	r := NewRouter(&stubCmd{name: "ws", help: "Manage workspaces"})

	err := r.Run(nil, []string{"nope"})
	if err == nil {
		t.Fatal("expected unknown command error")
	}
	if !strings.Contains(err.Error(), `unknown command "nope"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRouterPrintHelpIncludesSortedCommands(t *testing.T) {
	r := NewRouter(
		&stubCmd{name: "shell-hook", help: "Install shell hook"},
		&stubCmd{name: "open", help: "Open path"},
		&stubCmd{name: "exec", help: "Exec path"},
		&stubCmd{name: "enter", help: "Enter path"},
		&stubCmd{name: "ws", help: "Manage workspaces"},
		&stubCmd{name: "init", help: "Initialize root"},
	)

	var buf bytes.Buffer
	r.PrintHelp(&buf)
	output := buf.String()

	if !strings.Contains(output, "usage: groot <command> [args]") {
		t.Fatalf("unexpected help output: %q", output)
	}
	enterIdx := strings.Index(output, "enter")
	execIdx := strings.Index(output, "exec")
	initIdx := strings.Index(output, "init")
	openIdx := strings.Index(output, "open")
	shellHookIdx := strings.Index(output, "shell-hook")
	wsIdx := strings.Index(output, "ws")
	if !(enterIdx < execIdx && execIdx < initIdx && initIdx < openIdx && openIdx < shellHookIdx && shellHookIdx < wsIdx) {
		t.Fatalf("expected sorted commands, got output %q", output)
	}
}
