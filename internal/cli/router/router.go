package router

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/totoual/groot/internal/app"
	"github.com/totoual/groot/internal/cli/cliutil"
	"github.com/totoual/groot/internal/cli/interfaces"
)

type Router struct {
	cmds map[string]interfaces.Cmd
}

func NewRouter(cmds ...interfaces.Cmd) *Router {
	m := make(map[string]interfaces.Cmd, len(cmds))
	for _, c := range cmds {
		m[c.Name()] = c
	}
	return &Router{
		cmds: m,
	}
}

func (r *Router) Run(a *app.App, args []string) error {
	if cliutil.IsHelpRequest(args) {
		r.PrintHelp(os.Stdout)
		return nil
	}

	c, ok := r.cmds[args[0]]
	if !ok {
		return fmt.Errorf("unknown command %q (try: groot help)", args[0])
	}
	return c.Run(a, args[1:])
}

func (r *Router) PrintHelp(w io.Writer) {
	fmt.Fprintln(w, "usage: groot <command> [args]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "commands:")
	var names []string
	for name := range r.cmds {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		cmd := r.cmds[name]
		fmt.Fprintf(w, "  %-12s %s\n", cmd.Name(), cmd.Help())
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'groot <command> -h' for more information on a command.")
}
