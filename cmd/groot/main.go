package main

import (
	"fmt"
	"log"
	"os"

	"github.com/totoual/groot/internal/app"
	"github.com/totoual/groot/internal/cli/commands"
	"github.com/totoual/groot/internal/cli/router"
	"github.com/totoual/groot/internal/helpers"
)

func main() {
	root, err := helpers.ResolveGrootHome()
	if err != nil {
		log.Fatalf("Failed to find the groot home!")
	}
	groot_app := app.NewApp(root)
	fmt.Println(groot_app)
	groot_router := router.NewRouter(commands.InitCmd{})

	groot_router.Run(groot_app, os.Args[1:])
}
