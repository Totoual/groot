package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/totoual/groot/internal/app"
)

type ExecCmd struct{}

func (c *ExecCmd) Name() string { return "exec" }

func (c *ExecCmd) Help() string {
	return "Resolve or create a workspace from a project path and run one command"
}

func (c *ExecCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("exec", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot exec <path> <cmd> [args...]")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() < 2 {
		fs.Usage()
		return fmt.Errorf("project path and command required")
	}

	projectPath := fs.Arg(0)
	command := fs.Arg(1)
	commandArgs := fs.Args()[2:]

	workspaceName, err := resolveProjectWorkspace(a, projectPath)
	if err != nil {
		return err
	}
	return a.ExecWorkspace(workspaceName, command, commandArgs)
}
