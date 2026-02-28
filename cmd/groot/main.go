package main

import (
	"fmt"
	"log"

	"github.com/totoual/groot/internal/app"
	"github.com/totoual/groot/internal/helpers"
)

func main() {
	root, err := helpers.ResolveGrootHome()
	if err != nil {
		log.Fatalf("Failed to find the groot home!")
	}
	groot_app := app.NewApp(root)
	groot_app.Init()
	fmt.Println(groot_app)
}
