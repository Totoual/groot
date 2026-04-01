package workspacecmds

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/totoual/groot/internal/app"
)

type DeleteCmd struct{}

func (d *DeleteCmd) Name() string { return "delete" }

func (d *DeleteCmd) Help() string { return "Delete a workspace" }

func (d *DeleteCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot ws delete <name>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), d.Help())
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

	fmt.Println("Deleting workspace:", name)
	err := a.DeleteWorkspace(name)
	if err != nil {
		return fmt.Errorf("Couldn't delete workspace: %w", err)
	}
	return nil
}
