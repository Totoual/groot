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

	resolved, err := resolveProjectWorkspace(a, projectPath)
	if err != nil {
		return err
	}
	detected, err := a.DetectProjectToolchains(projectPath)
	if err != nil {
		return fmt.Errorf("couldn't detect project toolchains: %w", err)
	}
	missing, err := a.MissingWorkspaceToolchains(resolved.Name, detected)
	if err != nil {
		return fmt.Errorf("couldn't compare detected toolchains with workspace manifest: %w", err)
	}
	if len(detected) > 0 && resolved.Created {
		fmt.Fprintf(os.Stderr, "Detected likely runtimes for workspace %q: %s\n", resolved.Name, formatDetectedToolchains(detected))
		fmt.Fprintln(os.Stderr, "First-open behavior is warn-only for now: Groot did not attach toolchains automatically.")
	}
	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "Workspace %q does not declare detected runtimes: %s\n", resolved.Name, formatDetectedToolchains(missing))
		fmt.Fprintln(os.Stderr, "Commands may fall back to host toolchains until these are attached and installed.")
		fmt.Fprintln(os.Stderr, "Attach them with:")
		fmt.Fprintf(os.Stderr, "  groot ws attach %s %s\n", resolved.Name, suggestedAttachArgs(missing))
		fmt.Fprintf(os.Stderr, "  groot ws install %s\n", resolved.Name)
	}
	if err := a.OpenWorkspace(resolved.Name, ide, openArgs); err != nil {
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

func formatDetectedToolchains(detected []app.DetectedToolchain) string {
	parts := make([]string, 0, len(detected))
	for _, tc := range detected {
		if tc.Version != "" {
			parts = append(parts, fmt.Sprintf("%s@%s", tc.Name, tc.Version))
			continue
		}
		parts = append(parts, tc.Name)
	}
	return strings.Join(parts, ", ")
}

func suggestedAttachArgs(detected []app.DetectedToolchain) string {
	parts := make([]string, 0, len(detected))
	for _, tc := range detected {
		version := tc.Version
		if version == "" {
			version = "<version>"
		}
		parts = append(parts, fmt.Sprintf("%s@%s", tc.Name, version))
	}
	return strings.Join(parts, " ")
}
