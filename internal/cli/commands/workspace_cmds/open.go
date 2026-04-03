package workspacecmds

import (
	"fmt"
	"os"
	"strings"

	"github.com/totoual/groot/internal/app"
)

type OpenCmd struct{}

func (o *OpenCmd) Name() string { return "open" }

func (o *OpenCmd) Help() string { return "Open a workspace in an IDE with a soft GUI runtime" }

func (o *OpenCmd) Run(a *app.App, args []string) error {
	name, ide, openArgs, err := parseOpenArgs(args)
	if err != nil {
		o.printUsage()
		if err == errHelpRequested {
			return nil
		}
		return err
	}

	if err := a.OpenWorkspace(name, ide, openArgs); err != nil {
		return fmt.Errorf("couldn't open workspace: %w", err)
	}
	return nil
}

var errHelpRequested = fmt.Errorf("help requested")

func parseOpenArgs(args []string) (string, string, []string, error) {
	name := ""
	ide := ""
	openArgs := make([]string, 0)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help" || arg == "help":
			return "", "", nil, errHelpRequested
		case arg == "--":
			openArgs = append(openArgs, args[i+1:]...)
			i = len(args)
		case arg == "--ide":
			if i+1 >= len(args) {
				return "", "", nil, fmt.Errorf("ide value required")
			}
			ide = args[i+1]
			i++
		case strings.HasPrefix(arg, "--ide="):
			ide = strings.TrimPrefix(arg, "--ide=")
		case strings.HasPrefix(arg, "-"):
			return "", "", nil, fmt.Errorf("unknown flag %q", arg)
		case name == "":
			name = arg
		default:
			openArgs = append(openArgs, arg)
		}
	}

	if name == "" {
		return "", "", nil, fmt.Errorf("workspace name required")
	}

	return name, ide, openArgs, nil
}

func (o *OpenCmd) printUsage() {
	fmt.Fprintln(os.Stdout, "usage: groot ws open <name> [--ide code|cursor|zed|...] [-- args...]")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, o.Help())
}
