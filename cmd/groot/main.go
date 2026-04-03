package main

import (
	"log"
	"os"

	"github.com/totoual/groot/internal/app"
	"github.com/totoual/groot/internal/cli/commands"
	workspacecmds "github.com/totoual/groot/internal/cli/commands/workspace_cmds"
	"github.com/totoual/groot/internal/cli/router"
	"github.com/totoual/groot/internal/helpers"
)

func main() {
	root, err := helpers.ResolveGrootHome()
	if err != nil {
		log.Fatalf("Failed to find the groot home!")
	}
	groot_app := app.NewApp(root)

	wscmd := commands.NewWorkspaceCmd(
		&workspacecmds.BindCmd{},
		&workspacecmds.CreateCmd{},
		&workspacecmds.DeleteCmd{},
		&workspacecmds.EnvCmd{},
		&workspacecmds.ExecCmd{},
		&workspacecmds.GCCmd{},
		&workspacecmds.OpenCmd{},
		&workspacecmds.ShellCmd{},
		&workspacecmds.AttachCmd{},
		&workspacecmds.InstallCmd{},
		&workspacecmds.UnbindCmd{},
	)

	groot_router := router.NewRouter(
		&commands.EnterCmd{},
		&commands.ExecCmd{},
		&commands.InitCmd{},
		&commands.OpenCmd{},
		&commands.ShellHookCmd{},
		wscmd,
	)

	err = groot_router.Run(groot_app, os.Args[1:])
	if err != nil {
		log.Fatalf("groot command failed: %v", err)
	}
}
