package workspacecmds

import (
	"github.com/totoual/groot/internal/app"
)

type AttachCmd struct{}

func (at *AttachCmd) Name() string { return "attach" }

func (at *AttachCmd) Help() string { return "Attach a tool or a service in a workspace" }

func (at *AttachCmd) Run(a *app.App, args []string) error {

	return nil
}
