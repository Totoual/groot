package workspacecmds

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/totoual/groot/internal/app"
)

type GCCmd struct{}

func (g *GCCmd) Name() string { return "gc" }

func (g *GCCmd) Help() string { return "Remove unreferenced toolchains from the shared store" }

func (g *GCCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("gc", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot ws gc")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), g.Help())
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 0 {
		fs.Usage()
		return fmt.Errorf("gc does not accept arguments")
	}

	if err := a.GarbageCollectToolchains(); err != nil {
		return fmt.Errorf("couldn't garbage collect toolchains: %w", err)
	}
	return nil
}
