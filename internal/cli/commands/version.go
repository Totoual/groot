package commands

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/totoual/groot/internal/app"
	"github.com/totoual/groot/internal/versioninfo"
)

var currentVersionInfo = versioninfo.Current

type VersionCmd struct{}

func (c *VersionCmd) Name() string { return "version" }

func (c *VersionCmd) Help() string {
	return "Print Groot version information"
}

func (c *VersionCmd) Run(_ *app.App, args []string) error {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	jsonOutput := fs.Bool("json", false, "print version info as JSON")
	verboseOutput := fs.Bool("verbose", false, "print detailed build metadata")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot version [--json|--verbose]")
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
		return fmt.Errorf("version does not accept positional arguments")
	}

	info := currentVersionInfo()
	if *jsonOutput {
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal version json: %w", err)
		}
		fmt.Fprintln(os.Stdout, string(data))
		return nil
	}

	if !*verboseOutput {
		fmt.Fprintln(os.Stdout, info.Version)
		return nil
	}

	fmt.Fprintf(os.Stdout, "Version: %s\n", info.Version)
	if info.ModulePath != "" {
		fmt.Fprintf(os.Stdout, "Module: %s\n", info.ModulePath)
	}
	if info.ModuleVersion != "" {
		fmt.Fprintf(os.Stdout, "Module Version: %s\n", info.ModuleVersion)
	}
	if info.GoVersion != "" {
		fmt.Fprintf(os.Stdout, "Go: %s\n", info.GoVersion)
	}
	if info.VCSRevision != "" {
		fmt.Fprintf(os.Stdout, "Revision: %s\n", info.VCSRevision)
	}
	if info.VCSTime != "" {
		fmt.Fprintf(os.Stdout, "Build Time: %s\n", info.VCSTime)
	}
	if info.VCSModified != "" {
		fmt.Fprintf(os.Stdout, "Modified: %s\n", info.VCSModified)
	}
	if info.BinaryPath != "" {
		fmt.Fprintf(os.Stdout, "Binary: %s\n", info.BinaryPath)
	}
	return nil
}
