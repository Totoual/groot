package commands

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/totoual/groot/internal/app"
	"github.com/totoual/groot/internal/cli/cliutil"
	"github.com/totoual/groot/internal/cli/interfaces"
)

type ContextCmd struct {
	subcmds map[string]interfaces.Cmd
}

func NewContextCmd(cmds ...interfaces.Cmd) *ContextCmd {
	if len(cmds) == 0 {
		cmds = defaultContextCommands()
	}
	return &ContextCmd{subcmds: contextCommandMap(cmds...)}
}

func contextCommandMap(cmds ...interfaces.Cmd) map[string]interfaces.Cmd {
	m := make(map[string]interfaces.Cmd, len(cmds))
	for _, c := range cmds {
		m[c.Name()] = c
	}
	return m
}

func defaultContextCommands() []interfaces.Cmd {
	return []interfaces.Cmd{
		&contextBuildCmd{},
	}
}

func (c *ContextCmd) commands() map[string]interfaces.Cmd {
	if c.subcmds == nil {
		c.subcmds = contextCommandMap(defaultContextCommands()...)
	}
	return c.subcmds
}

func (c *ContextCmd) Name() string { return "context" }

func (c *ContextCmd) Help() string {
	return "Build a compact workspace context pack"
}

func (c *ContextCmd) Run(a *app.App, args []string) error {
	if cliutil.IsHelpRequest(args) {
		c.PrintHelp(os.Stdout)
		return nil
	}
	if len(args) == 0 {
		c.PrintHelp(os.Stdout)
		return fmt.Errorf("context command required")
	}

	subcmd, ok := c.commands()[args[0]]
	if !ok {
		return fmt.Errorf("unknown context command %q (try: groot context -h)", args[0])
	}
	return subcmd.Run(a, args[1:])
}

func (c *ContextCmd) PrintHelp(w io.Writer) {
	fmt.Fprintln(w, "usage: groot context <command> [args]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "commands:")
	var names []string
	for name := range c.commands() {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		cmd := c.commands()[name]
		fmt.Fprintf(w, "  %-12s %s\n", cmd.Name(), cmd.Help())
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'groot context <command> -h' for more information on a command.")
}

type contextBuildCmd struct{}

func (c *contextBuildCmd) Name() string { return "build" }
func (c *contextBuildCmd) Help() string {
	return "Build a compact deterministic context pack for a workspace task"
}

func (c *contextBuildCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("context build", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot context build <workspace> <task>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() < 2 {
		fs.Usage()
		return fmt.Errorf("workspace name and task required")
	}

	workspaceName, err := requireWorkspaceArg(a, fs.Arg(0))
	if err != nil {
		return err
	}
	pack, err := a.BuildContextPack(workspaceName, strings.Join(fs.Args()[1:], " "))
	if err != nil {
		return err
	}
	fmt.Fprint(os.Stdout, pack.Markdown())
	return nil
}
