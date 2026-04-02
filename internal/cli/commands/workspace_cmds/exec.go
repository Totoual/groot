package workspacecmds

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/totoual/groot/internal/app"
)

type ExecCmd struct{}

func (e *ExecCmd) Name() string { return "exec" }

func (e *ExecCmd) Help() string { return "Run a command inside a workspace" }

func (e *ExecCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("exec", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot ws exec <name> <cmd> [args...]")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), e.Help())
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() < 2 {
		fs.Usage()
		return fmt.Errorf("workspace name and command required")
	}
	name := fs.Arg(0)
	command := fs.Arg(1)
	commandArgs := fs.Args()[2:]

	err := a.ExecWorkspace(name, command, commandArgs)
	if err != nil {
		return fmt.Errorf("couldn't exec workspace: %w", err)
	}
	return nil
}
