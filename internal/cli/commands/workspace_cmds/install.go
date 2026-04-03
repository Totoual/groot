package workspacecmds

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/totoual/groot/internal/app"
)

type InstallCmd struct{}

func (i *InstallCmd) Name() string { return "install" }

func (i *InstallCmd) Help() string { return "Download and install the tools specified in the manifest" }

func (i *InstallCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot ws install <name>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), i.Help())
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

	err := a.InstallToWorkspace(name)
	if err != nil {
		return fmt.Errorf("Couldn't install tools for workspace: %w", err)
	}
	return nil
}
