package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/totoual/groot/internal/app"
)

type OpenCmd struct{}

func (c *OpenCmd) Name() string { return "open" }

func (c *OpenCmd) Help() string {
	return "Resolve or create a workspace from a project path and open it in an IDE"
}

func (c *OpenCmd) Run(a *app.App, args []string) error {
	projectPath, ide, openArgs, err := parsePathOpenArgs(args)
	if err != nil {
		c.printUsage()
		if err == errPathOpenHelpRequested {
			return nil
		}
		return err
	}

	workspaceName, err := resolveProjectWorkspace(a, projectPath)
	if err != nil {
		return err
	}
	if err := a.OpenWorkspace(workspaceName, ide, openArgs); err != nil {
		return fmt.Errorf("couldn't open workspace: %w", err)
	}
	return nil
}

var errPathOpenHelpRequested = fmt.Errorf("help requested")

func parsePathOpenArgs(args []string) (string, string, []string, error) {
	projectPath := ""
	ide := ""
	openArgs := make([]string, 0)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help" || arg == "help":
			return "", "", nil, errPathOpenHelpRequested
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
		case projectPath == "":
			projectPath = arg
		default:
			openArgs = append(openArgs, arg)
		}
	}

	if projectPath == "" {
		return "", "", nil, fmt.Errorf("project path required")
	}

	return projectPath, ide, openArgs, nil
}

func (c *OpenCmd) printUsage() {
	fmt.Fprintln(os.Stdout, "usage: groot open <path> [--ide code|cursor|zed|...] [-- args...]")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, c.Help())
}
