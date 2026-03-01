package commands

import "github.com/totoual/groot/internal/app"

type InitCmd struct{}

func (c InitCmd) Name() string { return "init" }
func (c InitCmd) Help() string { return "Initialize Groot directories" }

func (c InitCmd) Run(a *app.App, args []string) error {
	return a.Init()
}
