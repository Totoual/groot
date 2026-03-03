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
		&workspacecmds.CreateCmd{},
		&workspacecmds.DeleteCmd{},
	)

	groot_router := router.NewRouter(&commands.InitCmd{}, wscmd)

	err = groot_router.Run(groot_app, os.Args[1:])
	if err != nil {
		log.Fatalf(err.Error())
	}
}
