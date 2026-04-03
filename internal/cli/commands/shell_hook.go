package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/totoual/groot/internal/app"
)

type ShellHookCmd struct{}

func (c *ShellHookCmd) Name() string { return "shell-hook" }

func (c *ShellHookCmd) Help() string {
	return "Print shell exports for the current Groot workspace context"
}

func (c *ShellHookCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("shell-hook", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot shell-hook")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 0 {
		fs.Usage()
		return fmt.Errorf("shell-hook does not accept arguments")
	}

	output, err := a.ShellHook()
	if err != nil {
		return fmt.Errorf("couldn't build shell hook: %w", err)
	}

	fmt.Fprint(os.Stdout, output)
	return nil
}
