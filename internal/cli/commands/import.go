package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/totoual/groot/internal/app"
)

type ImportCmd struct{}

type importOptions struct {
	exportPath      string
	projectPath     string
	workspaceName   string
	installAttached bool
}

func (c *ImportCmd) Name() string { return "import" }

func (c *ImportCmd) Help() string {
	return "Import a portable workspace contract for an existing project path"
}

func (c *ImportCmd) Run(a *app.App, args []string) error {
	opts, showHelp, err := parseImportArgs(args)
	if showHelp {
		c.printUsage(os.Stdout)
		return nil
	}
	if err != nil {
		if err.Error() == "export file required" || err.Error() == "project path required" {
			c.printUsage(os.Stdout)
		}
		return err
	}

	exported, err := readWorkspaceExport(opts.exportPath, os.Stdin)
	if err != nil {
		return err
	}
	imported, err := a.ImportWorkspaceAs(exported, opts.projectPath, opts.workspaceName, opts.installAttached)
	if err != nil {
		return fmt.Errorf("couldn't import workspace: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Imported workspace %q for %s\n", imported.WorkspaceName, imported.ProjectPath)
	return nil
}

func (c *ImportCmd) printUsage(out io.Writer) {
	fmt.Fprintln(out, "usage: groot import <export.json|-> --project-path <path> [--workspace-name name] [--install-attached]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, c.Help())
}

func parseImportArgs(args []string) (importOptions, bool, error) {
	opts := importOptions{}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return importOptions{}, true, nil
		case arg == "--project-path":
			if i+1 >= len(args) {
				return importOptions{}, false, fmt.Errorf("project path value required")
			}
			opts.projectPath = args[i+1]
			i++
		case strings.HasPrefix(arg, "--project-path="):
			opts.projectPath = strings.TrimPrefix(arg, "--project-path=")
		case arg == "--workspace-name":
			if i+1 >= len(args) {
				return importOptions{}, false, fmt.Errorf("workspace name value required")
			}
			opts.workspaceName = args[i+1]
			i++
		case strings.HasPrefix(arg, "--workspace-name="):
			opts.workspaceName = strings.TrimPrefix(arg, "--workspace-name=")
		case arg == "--install-attached":
			opts.installAttached = true
		case arg == "-" && opts.exportPath == "":
			opts.exportPath = arg
		case strings.HasPrefix(arg, "-"):
			return importOptions{}, false, fmt.Errorf("unknown flag %q", arg)
		case opts.exportPath == "":
			opts.exportPath = arg
		default:
			return importOptions{}, false, fmt.Errorf("unexpected argument %q", arg)
		}
	}

	if opts.exportPath == "" {
		return importOptions{}, false, fmt.Errorf("export file required")
	}
	if strings.TrimSpace(opts.projectPath) == "" {
		return importOptions{}, false, fmt.Errorf("project path required")
	}
	return opts, false, nil
}

func readWorkspaceExport(path string, stdin io.Reader) (app.WorkspaceExport, error) {
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(stdin)
		if err != nil {
			return app.WorkspaceExport{}, fmt.Errorf("read import stdin: %w", err)
		}
	} else {
		data, err = os.ReadFile(path)
		if err != nil {
			return app.WorkspaceExport{}, fmt.Errorf("read import file: %w", err)
		}
	}

	var exported app.WorkspaceExport
	if err := json.Unmarshal(data, &exported); err != nil {
		return app.WorkspaceExport{}, fmt.Errorf("parse import json: %w", err)
	}
	return exported, nil
}
