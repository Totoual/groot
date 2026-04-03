package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/totoual/groot/internal/app"
)

type StatusCmd struct{}

func (c *StatusCmd) Name() string { return "status" }

func (c *StatusCmd) Help() string {
	return "Resolve or create a workspace from a project path and print runtime ownership status"
}

func (c *StatusCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot status <path>")
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

	resolved, err := resolveProjectWorkspace(a, fs.Arg(0))
	if err != nil {
		return err
	}
	report, err := a.InspectWorkspaceRuntimeOwnership(resolved.Name)
	if err != nil {
		return fmt.Errorf("couldn't inspect workspace runtime ownership: %w", err)
	}
	writeWorkspaceRuntimeStatus(report)
	return nil
}
