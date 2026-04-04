package commands

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/totoual/groot/internal/app"
	"github.com/totoual/groot/internal/cli/interfaces"
)

type WorkspaceCmd struct {
	wscmds map[string]interfaces.Cmd
}

func NewWorkspaceCmd(cmds ...interfaces.Cmd) *WorkspaceCmd {
	m := make(map[string]interfaces.Cmd, len(cmds))
	for _, c := range cmds {
		m[c.Name()] = c
	}
	return &WorkspaceCmd{
		wscmds: m,
	}
}

func (ws *WorkspaceCmd) Name() string { return "ws" }
func (ws *WorkspaceCmd) Help() string { return "Advanced workspace controls" }

func (ws *WorkspaceCmd) Run(a *app.App, args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" || args[0] == "-help" {
		ws.PrintHelp(os.Stdout)
		return nil
	}

	c, ok := ws.wscmds[args[0]]
	if !ok {
		return fmt.Errorf("unknown ws command %q (try: groot ws -h)", args[0])
	}
	return c.Run(a, args[1:])
}

func (ws *WorkspaceCmd) PrintHelp(w io.Writer) {
	fmt.Fprintln(w, "usage: groot ws <command> [args]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "commands:")
	var names []string
	for name := range ws.wscmds {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		cmd := ws.wscmds[name]
		fmt.Fprintf(w, "  %-12s %s\n", cmd.Name(), cmd.Help())
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'groot ws <command> -h' for more information on a command.")
}
