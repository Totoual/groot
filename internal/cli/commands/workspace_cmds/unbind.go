package workspacecmds

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/totoual/groot/internal/app"
)

type UnbindCmd struct{}

func (u *UnbindCmd) Name() string { return "unbind" }

func (u *UnbindCmd) Help() string { return "Clear a workspace project binding" }

func (u *UnbindCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("unbind", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot ws unbind <name>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), u.Help())
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

	if err := a.UnbindWorkspace(fs.Arg(0)); err != nil {
		return fmt.Errorf("couldn't unbind workspace: %w", err)
	}
	return nil
}
