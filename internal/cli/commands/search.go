package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/totoual/groot/internal/app"
	"github.com/totoual/groot/internal/cli/cliutil"
)

type SearchCmd struct{}

func (c *SearchCmd) Name() string { return "search" }

func (c *SearchCmd) Help() string {
	return "Search indexed files and symbols with one human-friendly command"
}

func (c *SearchCmd) Run(a *app.App, args []string) error {
	if cliutil.IsHelpRequest(args) {
		fs := flag.NewFlagSet("search", flag.ContinueOnError)
		fs.SetOutput(os.Stdout)
		fs.Usage = func() {
			fmt.Fprintln(fs.Output(), "usage: groot search <workspace> <query>")
			fmt.Fprintln(fs.Output())
			fmt.Fprintln(fs.Output(), c.Help())
		}
		fs.Usage()
		return nil
	}
	if len(args) == 0 {
		fs := flag.NewFlagSet("search", flag.ContinueOnError)
		fs.SetOutput(os.Stdout)
		fs.Usage = func() {
			fmt.Fprintln(fs.Output(), "usage: groot search <workspace> <query>")
			fmt.Fprintln(fs.Output())
			fmt.Fprintln(fs.Output(), c.Help())
		}
		fs.Usage()
		return fmt.Errorf("workspace name required")
	}

	workspaceName := args[0]
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fileLimit := fs.Int("files", 5, "maximum number of file results")
	symbolLimit := fs.Int("symbols", 5, "maximum number of symbol results")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot search <workspace> <query>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() < 1 {
		fs.Usage()
		return fmt.Errorf("search query required")
	}
	if *fileLimit < 0 || *symbolLimit < 0 {
		return fmt.Errorf("limits must be >= 0")
	}

	workspaceName, err := requireWorkspaceArg(a, workspaceName)
	if err != nil {
		return err
	}
	query := strings.Join(fs.Args(), " ")
	fileHits, err := a.IndexSearch(workspaceName, query, app.IndexSearchOptions{Limit: *fileLimit})
	if err != nil {
		return err
	}
	symbolHits, err := a.IndexSymbols(workspaceName, query, app.IndexSearchOptions{Limit: *symbolLimit})
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, "# Groot Search")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "Query: %s\n\n", query)
	fmt.Fprintln(os.Stdout, "Files:")
	if len(fileHits) == 0 {
		fmt.Fprintln(os.Stdout, "-")
	} else {
		for _, hit := range fileHits {
			fmt.Fprintf(os.Stdout, "- [score=%d] %s\n", hit.Score, hit.File.Path)
		}
	}
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Symbols:")
	if len(symbolHits) == 0 {
		fmt.Fprintln(os.Stdout, "-")
	} else {
		for _, hit := range symbolHits {
			fmt.Fprintf(os.Stdout, "- [score=%d] %s (%s) — %s:%d-%d\n", hit.Score, hit.Symbol.QualifiedName, hit.Symbol.Kind, hit.Symbol.FilePath, hit.Symbol.LineStart, hit.Symbol.LineEnd)
		}
	}
	return nil
}
