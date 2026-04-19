package workspacecmds

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/totoual/groot/internal/app"
)

type AttachCmd struct{}

func (at *AttachCmd) Name() string { return "attach" }

func (at *AttachCmd) Help() string { return "Attach one or more toolchains in a workspace" }

func (at *AttachCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("attach", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot ws attach <name> <tool@version> [tool@version...]")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), at.Help())
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() < 2 {
		fs.Usage()
		return fmt.Errorf("workspace name required and at least one toolchain")
	}

	name := fs.Arg(0)

	err := a.AttachToWorkspace(name, args[1:])
	if err != nil {
		return fmt.Errorf("couldn't attach toolchains to workspace: %w", err)
	}
	return nil
}
