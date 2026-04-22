package commands

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/totoual/groot/internal/app"
	"github.com/totoual/groot/internal/cli/cliutil"
	"github.com/totoual/groot/internal/cli/interfaces"
)

type EventCmd struct {
	subcmds map[string]interfaces.Cmd
}

func NewEventCmd(cmds ...interfaces.Cmd) *EventCmd {
	if len(cmds) == 0 {
		cmds = defaultEventCommands()
	}
	return &EventCmd{subcmds: eventCommandMap(cmds...)}
}

func eventCommandMap(cmds ...interfaces.Cmd) map[string]interfaces.Cmd {
	m := make(map[string]interfaces.Cmd, len(cmds))
	for _, c := range cmds {
		m[c.Name()] = c
	}
	return m
}

func defaultEventCommands() []interfaces.Cmd {
	return []interfaces.Cmd{
		&eventListCmd{},
	}
}

func (c *EventCmd) commands() map[string]interfaces.Cmd {
	if c.subcmds == nil {
		c.subcmds = eventCommandMap(defaultEventCommands()...)
	}
	return c.subcmds
}

func (c *EventCmd) Name() string { return "event" }
func (c *EventCmd) Help() string { return "Inspect workspace runtime events" }

func (c *EventCmd) Run(a *app.App, args []string) error {
	if cliutil.IsHelpRequest(args) {
		c.PrintHelp(os.Stdout)
		return nil
	}

	subcmd, ok := c.commands()[args[0]]
	if !ok {
		return fmt.Errorf("unknown event command %q (try: groot event -h)", args[0])
	}
	return subcmd.Run(a, args[1:])
}

func (c *EventCmd) PrintHelp(w io.Writer) {
	fmt.Fprintln(w, "usage: groot event <command> [args]")
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
	fmt.Fprintln(w, "Run 'groot event <command> -h' for more information on a command.")
}

type eventListCmd struct{}

func (c *eventListCmd) Name() string { return "list" }
func (c *eventListCmd) Help() string { return "List runtime events for a project path" }

func (c *eventListCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("event list", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	limit := fs.Int("limit", 0, "maximum number of events to print")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot event list <path> [--limit n]")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("project path required")
	}
	if *limit < 0 {
		return fmt.Errorf("limit must be >= 0")
	}

	resolved, err := resolveProjectWorkspace(a, fs.Arg(0))
	if err != nil {
		return err
	}
	events, err := a.EventList(resolved.Name, app.EventListOptions{Limit: *limit})
	if err != nil {
		return err
	}
	if len(events) == 0 {
		fmt.Fprintln(os.Stdout, "No events.")
		return nil
	}
	for _, event := range events {
		fmt.Fprintf(os.Stdout, "%s\t%s\t%s\t%s\t%s\n", event.ID, event.Timestamp.Format("2006-01-02T15:04:05Z"), event.Kind, event.ResourceID, event.Message)
	}
	return nil
}
