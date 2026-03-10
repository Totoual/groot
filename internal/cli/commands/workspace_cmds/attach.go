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

func (at *AttachCmd) Help() string { return "Attach a tool or a service in a workspace" }

func (at *AttachCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("attach", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot ws attach <name> <tool1> <tool2>")
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
		return fmt.Errorf("workspace name required")
	}

	name := fs.Arg(0)

	fmt.Println("Attaching to workspace:", name)
	err := a.AttachToWorkspace(name, args[1:])
	if err != nil {
		return fmt.Errorf("Couldn't attach tools workspace: %w", err)
	}
	return nil
}
