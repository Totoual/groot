package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/totoual/groot/internal/app"
)

type OpenCmd struct{}

type pathOpenOptions struct {
	projectPath     string
	ide             string
	attachDetected  bool
	installDetected bool
	openArgs        []string
}

func (c *OpenCmd) Name() string { return "open" }

func (c *OpenCmd) Help() string {
	return "Resolve or create a workspace from a project path and open it in an IDE"
}

func (c *OpenCmd) Run(a *app.App, args []string) error {
	opts, err := parsePathOpenArgs(args)
	if err != nil {
		c.printUsage()
		if err == errPathOpenHelpRequested {
			return nil
		}
		return err
	}

	resolved, err := resolveProjectWorkspace(a, opts.projectPath)
	if err != nil {
		return err
	}
	if resolved.Created {
		plan, err := a.BuildFirstOpenRuntimePlan(resolved.Name, opts.projectPath, opts.attachDetected, opts.installDetected)
		if err != nil {
			return fmt.Errorf("couldn't prepare first-open runtime plan: %w", err)
		}
		writeFirstOpenRuntimePlan(plan)
	}
	if err := enforceWorkspaceOwnership(a, resolved.Name); err != nil {
		return err
	}
	if err := a.OpenWorkspace(resolved.Name, opts.ide, opts.openArgs); err != nil {
		return fmt.Errorf("couldn't open workspace: %w", err)
	}
	return nil
}

var errPathOpenHelpRequested = fmt.Errorf("help requested")

func parsePathOpenArgs(args []string) (pathOpenOptions, error) {
	opts := pathOpenOptions{}
	openArgs := make([]string, 0)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help" || arg == "help":
			return pathOpenOptions{}, errPathOpenHelpRequested
		case arg == "--":
			openArgs = append(openArgs, args[i+1:]...)
			i = len(args)
		case arg == "--attach-detected":
			opts.attachDetected = true
		case arg == "--install-detected":
			opts.attachDetected = true
			opts.installDetected = true
		case arg == "--setup-detected":
			opts.attachDetected = true
			opts.installDetected = true
		case arg == "--ide":
			if i+1 >= len(args) {
				return pathOpenOptions{}, fmt.Errorf("ide value required")
			}
			opts.ide = args[i+1]
			i++
		case strings.HasPrefix(arg, "--ide="):
			opts.ide = strings.TrimPrefix(arg, "--ide=")
		case strings.HasPrefix(arg, "-"):
			return pathOpenOptions{}, fmt.Errorf("unknown flag %q", arg)
		case opts.projectPath == "":
			opts.projectPath = arg
		default:
			openArgs = append(openArgs, arg)
		}
	}

	if opts.projectPath == "" {
		return pathOpenOptions{}, fmt.Errorf("project path required")
	}

	opts.openArgs = openArgs
	return opts, nil
}

func (c *OpenCmd) printUsage() {
	fmt.Fprintln(os.Stdout, "usage: groot open <path> [--ide code|cursor|zed|...] [--attach-detected|--install-detected|--setup-detected] [-- args...]")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, c.Help())
}
