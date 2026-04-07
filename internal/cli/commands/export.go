package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/totoual/groot/internal/app"
)

type ExportCmd struct{}

func (c *ExportCmd) Name() string { return "export" }

func (c *ExportCmd) Help() string {
	return "Export an existing workspace contract from a project path as JSON"
}

func (c *ExportCmd) Run(a *app.App, args []string) error {
	projectPath, outputPath, showHelp, err := parseExportArgs(args)
	if showHelp {
		c.printUsage(os.Stdout)
		return nil
	}
	if err != nil {
		if err.Error() == "project path required" {
			c.printUsage(os.Stdout)
		}
		return err
	}

	exported, err := a.ExportWorkspaceByProjectPath(projectPath)
	if err != nil {
		return fmt.Errorf("couldn't export workspace: %w", err)
	}

	data, err := json.MarshalIndent(exported, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal export json: %w", err)
	}
	data = append(data, '\n')

	if outputPath == "" || outputPath == "-" {
		_, err = os.Stdout.Write(data)
		return err
	}

	if err := os.WriteFile(outputPath, data, 0o600); err != nil {
		return fmt.Errorf("write export file: %w", err)
	}
	return nil
}

func (c *ExportCmd) printUsage(out io.Writer) {
	fmt.Fprintln(out, "usage: groot export <path> [--output file]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, c.Help())
}

func parseExportArgs(args []string) (projectPath, outputPath string, showHelp bool, err error) {
	positionals := make([]string, 0, 1)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return "", "", true, nil
		case arg == "-output" || arg == "--output":
			if i+1 >= len(args) {
				return "", "", false, fmt.Errorf("flag needs an argument: %s", arg)
			}
			outputPath = args[i+1]
			i++
		case strings.HasPrefix(arg, "-output="):
			outputPath = strings.TrimPrefix(arg, "-output=")
		case strings.HasPrefix(arg, "--output="):
			outputPath = strings.TrimPrefix(arg, "--output=")
		case strings.HasPrefix(arg, "-"):
			return "", "", false, fmt.Errorf("flag provided but not defined: %s", arg)
		default:
			positionals = append(positionals, arg)
		}
	}

	if len(positionals) != 1 {
		return "", "", false, fmt.Errorf("project path required")
	}
	return positionals[0], outputPath, false, nil
}
