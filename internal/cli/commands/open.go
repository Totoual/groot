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
	projectPath, ide, attachDetected, openArgs, err := parsePathOpenArgs(args)
	if err != nil {
		c.printUsage()
		if err == errPathOpenHelpRequested {
			return nil
		}
		return err
	}

	resolved, err := resolveProjectWorkspace(a, projectPath)
	if err != nil {
		return err
	}
	detected, err := a.DetectProjectToolchains(projectPath)
	if err != nil {
		return fmt.Errorf("couldn't detect project toolchains: %w", err)
	}
	if len(detected) > 0 && resolved.Created {
		fmt.Fprintf(os.Stderr, "Detected likely runtimes for workspace %q: %s\n", resolved.Name, formatDetectedToolchains(detected))
		if attachDetected {
			attached, skipped, err := a.AttachDetectedToolchains(resolved.Name, detected)
			if err != nil {
				return fmt.Errorf("couldn't attach detected toolchains: %w", err)
			}
			if len(attached) > 0 {
				fmt.Fprintf(os.Stderr, "Auto-attached detected runtimes for workspace %q: %s\n", resolved.Name, formatDetectedToolchains(attached))
				fmt.Fprintf(os.Stderr, "Install them with:\n  groot ws install %s\n", resolved.Name)
			}
			if len(skipped) > 0 {
				fmt.Fprintf(os.Stderr, "Skipped detected runtimes without a concrete version for workspace %q: %s\n", resolved.Name, formatDetectedToolchains(skipped))
				fmt.Fprintln(os.Stderr, "Attach those manually once you choose the desired versions.")
			}
		} else {
			fmt.Fprintln(os.Stderr, "First-open behavior is warn-only for now: Groot did not attach toolchains automatically.")
		}
	}
	if err := enforceWorkspaceOwnership(a, resolved.Name); err != nil {
		return err
	}
	if err := a.OpenWorkspace(resolved.Name, ide, openArgs); err != nil {
		return fmt.Errorf("couldn't open workspace: %w", err)
	}
	return nil
}

var errPathOpenHelpRequested = fmt.Errorf("help requested")

func parsePathOpenArgs(args []string) (string, string, bool, []string, error) {
	projectPath := ""
	ide := ""
	attachDetected := false
	openArgs := make([]string, 0)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help" || arg == "help":
			return "", "", false, nil, errPathOpenHelpRequested
		case arg == "--":
			openArgs = append(openArgs, args[i+1:]...)
			i = len(args)
		case arg == "--attach-detected":
			attachDetected = true
		case arg == "--ide":
			if i+1 >= len(args) {
				return "", "", false, nil, fmt.Errorf("ide value required")
			}
			ide = args[i+1]
			i++
		case strings.HasPrefix(arg, "--ide="):
			ide = strings.TrimPrefix(arg, "--ide=")
		case strings.HasPrefix(arg, "-"):
			return "", "", false, nil, fmt.Errorf("unknown flag %q", arg)
		case projectPath == "":
			projectPath = arg
		default:
			openArgs = append(openArgs, arg)
		}
	}

	if projectPath == "" {
		return "", "", false, nil, fmt.Errorf("project path required")
	}

	return projectPath, ide, attachDetected, openArgs, nil
}

func (c *OpenCmd) printUsage() {
	fmt.Fprintln(os.Stdout, "usage: groot open <path> [--ide code|cursor|zed|...] [--attach-detected] [-- args...]")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, c.Help())
}
