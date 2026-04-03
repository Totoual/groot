package workspacecmds

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/totoual/groot/internal/app"
)

type CreateCmd struct{}

func (c *CreateCmd) Name() string { return "create" }

func (c *CreateCmd) Help() string { return "Create a new workspace" }

func (c *CreateCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot ws create <name>")
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
		return fmt.Errorf("workspace name required")
	}

	name := fs.Arg(0)

	err := a.CreateNewWorkspace(name)
	if err != nil {
		return fmt.Errorf("Couldn't create workspace: %w", err)
	}
	return nil
}
