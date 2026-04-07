package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/totoual/groot/internal/app"
	"github.com/totoual/groot/internal/mcp"
)

type MCPCmd struct{}

func (c *MCPCmd) Name() string { return "mcp" }

func (c *MCPCmd) Help() string {
	return "Run the Groot MCP server over stdio"
}

func (c *MCPCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot mcp")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return fmt.Errorf("mcp does not accept positional arguments")
	}

	return mcp.NewServer(a).Serve(os.Stdin, os.Stdout)
}
