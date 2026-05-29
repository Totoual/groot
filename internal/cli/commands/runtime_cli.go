package commands

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/totoual/groot/internal/app"
)

func requireWorkspaceArg(a *app.App, raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if _, err := a.EnsureWorkspace(name); err != nil {
		return "", err
	}
	return name, nil
}

func resolveWorkspaceOrProjectArg(a *app.App, raw string) (string, bool, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", false, fmt.Errorf("workspace name or project path required")
	}
	if _, err := a.EnsureWorkspace(name); err == nil {
		return name, false, nil
	}

	projectPath, err := app.NormalizeProjectPath(name)
	if err == nil {
		if info, statErr := os.Stat(projectPath); statErr == nil && info.IsDir() {
			resolved, resolveErr := resolveProjectWorkspace(a, projectPath)
			if resolveErr != nil {
				return "", false, resolveErr
			}
			return resolved.Name, true, nil
		}
	}

	return "", false, fmt.Errorf("workspace %q not found", name)
}

func splitFlagArgsAndCommand(args []string) ([]string, []string) {
	for i, arg := range args {
		if arg == "--" {
			return args[:i], args[i+1:]
		}
	}
	return nil, args
}

func parseTaskDeclarationArgs(fs *flag.FlagSet, args []string) (string, string, []string, error) {
	if len(args) < 2 {
		fs.Usage()
		return "", "", nil, fmt.Errorf("workspace name and task name required")
	}

	workspaceName := strings.TrimSpace(args[0])
	taskName := strings.TrimSpace(args[1])
	flagArgs, command := splitFlagArgsAndCommand(args[2:])
	if flagArgs == nil {
		flagArgs = []string{}
	}
	if err := fs.Parse(flagArgs); err != nil {
		return "", "", nil, err
	}
	if len(command) == 0 {
		fs.Usage()
		return "", "", nil, fmt.Errorf("task command required")
	}
	return workspaceName, taskName, command, nil
}

func parseServiceDeclarationArgs(fs *flag.FlagSet, args []string) (string, string, []string, error) {
	if len(args) < 2 {
		fs.Usage()
		return "", "", nil, fmt.Errorf("workspace name and service name required")
	}

	workspaceName := strings.TrimSpace(args[0])
	serviceName := strings.TrimSpace(args[1])
	flagArgs, command := splitFlagArgsAndCommand(args[2:])
	if flagArgs == nil {
		flagArgs = []string{}
	}
	if err := fs.Parse(flagArgs); err != nil {
		return "", "", nil, err
	}
	if len(command) == 0 {
		fs.Usage()
		return "", "", nil, fmt.Errorf("service command required")
	}
	return workspaceName, serviceName, command, nil
}
