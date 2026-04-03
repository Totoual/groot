package workspacecmds

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/totoual/groot/internal/app"
)

type EnvCmd struct{}

func (e *EnvCmd) Name() string { return "env" }

func (e *EnvCmd) Help() string { return "Print workspace environment exports" }

func (e *EnvCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("env", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot ws env <name>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), e.Help())
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

	output, err := a.WorkspaceEnv(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("couldn't print workspace env: %w", err)
	}

	fmt.Fprint(os.Stdout, output)
	return nil
}
