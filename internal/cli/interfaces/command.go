package interfaces

import "github.com/totoual/groot/internal/app"

type Cmd interface {
	Name() string
	Help() string
	Run(a *app.App, args []string) error
}
