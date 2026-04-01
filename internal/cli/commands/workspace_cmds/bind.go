package workspacecmds

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/totoual/groot/internal/app"
)

type BindCmd struct{}

func (b *BindCmd) Name() string { return "bind" }

func (b *BindCmd) Help() string { return "Bind a workspace to an existing project directory" }

func (b *BindCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("bind", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot ws bind <name> <path>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), b.Help())
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 2 {
		fs.Usage()
		return fmt.Errorf("workspace name and project path required")
	}

	name := fs.Arg(0)
	projectPath := fs.Arg(1)

	fmt.Println("Binding workspace:", name)
	if err := a.BindWorkspace(name, projectPath); err != nil {
		return fmt.Errorf("couldn't bind workspace: %w", err)
	}
	return nil
}
