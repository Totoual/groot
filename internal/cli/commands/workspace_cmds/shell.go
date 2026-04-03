package workspacecmds

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/totoual/groot/internal/app"
)

type ShellCmd struct{}

func (s *ShellCmd) Name() string { return "shell" }

func (s *ShellCmd) Help() string { return "Activate a workspace" }

func (s *ShellCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("shell", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot ws shell <name>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), s.Help())
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("workspace name required")
	}

	name := fs.Arg(0)

	err := a.WorkspaceShell(name)
	if err != nil {
		return fmt.Errorf("couldn't activate workspace: %w", err)
	}
	return nil
}
