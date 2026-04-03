package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/totoual/groot/internal/app"
)

type EnterCmd struct{}

func (c *EnterCmd) Name() string { return "enter" }

func (c *EnterCmd) Help() string {
	return "Resolve or create a workspace from a project path and start a shell"
}

func (c *EnterCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("enter", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot enter <path>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("project path required")
	}

	projectPath := fs.Arg(0)
	workspaceName, err := resolveProjectWorkspace(a, projectPath)
	if err != nil {
		return err
	}
	return a.WorkspaceShell(workspaceName)
}
