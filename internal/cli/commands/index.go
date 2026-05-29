package commands

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/totoual/groot/internal/app"
	"github.com/totoual/groot/internal/cli/cliutil"
	"github.com/totoual/groot/internal/cli/interfaces"
)

type IndexCmd struct {
	subcmds map[string]interfaces.Cmd
}

func NewIndexCmd(cmds ...interfaces.Cmd) *IndexCmd {
	if len(cmds) == 0 {
		cmds = defaultIndexCommands()
	}
	return &IndexCmd{subcmds: indexCommandMap(cmds...)}
}

func indexCommandMap(cmds ...interfaces.Cmd) map[string]interfaces.Cmd {
	m := make(map[string]interfaces.Cmd, len(cmds))
	for _, c := range cmds {
		m[c.Name()] = c
	}
	return m
}

func defaultIndexCommands() []interfaces.Cmd {
	return []interfaces.Cmd{
		&indexInitCmd{},
		&indexUpdateCmd{},
		&indexSearchCmd{},
		&indexSymbolsCmd{},
		&indexStatsCmd{},
	}
}

func (c *IndexCmd) commands() map[string]interfaces.Cmd {
	if c.subcmds == nil {
		c.subcmds = indexCommandMap(defaultIndexCommands()...)
	}
	return c.subcmds
}

func (c *IndexCmd) Name() string { return "index" }

func (c *IndexCmd) Help() string {
	return "Build and query a workspace-scoped file and symbol index"
}

func (c *IndexCmd) Run(a *app.App, args []string) error {
	if cliutil.IsHelpRequest(args) {
		c.PrintHelp(os.Stdout)
		return nil
	}
	if len(args) == 0 {
		c.PrintHelp(os.Stdout)
		return fmt.Errorf("index command required")
	}

	subcmd, ok := c.commands()[args[0]]
	if !ok {
		return fmt.Errorf("unknown index command %q (try: groot index -h)", args[0])
	}
	return subcmd.Run(a, args[1:])
}

func (c *IndexCmd) PrintHelp(w io.Writer) {
	fmt.Fprintln(w, "usage: groot index <command> [args]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "commands:")
	var names []string
	for name := range c.commands() {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		cmd := c.commands()[name]
		fmt.Fprintf(w, "  %-12s %s\n", cmd.Name(), cmd.Help())
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'groot index <command> -h' for more information on a command.")
}

type indexInitCmd struct{}

func (c *indexInitCmd) Name() string { return "init" }
func (c *indexInitCmd) Help() string { return "Create index storage files for a workspace" }

func (c *indexInitCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("index init", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot index init <workspace>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("workspace name required")
	}

	workspaceName, err := requireWorkspaceArg(a, fs.Arg(0))
	if err != nil {
		return err
	}
	if err := a.InitIndex(workspaceName); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Initialized index for workspace %q.\n", workspaceName)
	return nil
}

type indexUpdateCmd struct{}

func (c *indexUpdateCmd) Name() string { return "update" }
func (c *indexUpdateCmd) Help() string {
	return "Rebuild the workspace index from the bound project path"
}

func (c *indexUpdateCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("index update", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot index update <workspace>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("workspace name required")
	}

	workspaceName, err := requireWorkspaceArg(a, fs.Arg(0))
	if err != nil {
		return err
	}
	stats, err := a.UpdateIndex(workspaceName)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Indexed %d files, %d symbols, and %d terms for workspace %q.\n", stats.FileCount, stats.SymbolCount, stats.TermCount, workspaceName)
	return nil
}

type indexSearchCmd struct{}

func (c *indexSearchCmd) Name() string { return "search" }
func (c *indexSearchCmd) Help() string { return "Search indexed files for a workspace" }

func (c *indexSearchCmd) Run(a *app.App, args []string) error {
	if len(args) == 0 {
		fs := flag.NewFlagSet("index search", flag.ContinueOnError)
		fs.SetOutput(os.Stdout)
		fs.Usage = func() {
			fmt.Fprintln(fs.Output(), "usage: groot index search <workspace> <query>")
			fmt.Fprintln(fs.Output())
			fmt.Fprintln(fs.Output(), c.Help())
		}
		fs.Usage()
		return fmt.Errorf("workspace name required")
	}
	workspaceName := args[0]
	fs := flag.NewFlagSet("index search", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	limit := fs.Int("limit", 10, "maximum number of search results to print")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot index search <workspace> <query>")
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
		return fmt.Errorf("index query required")
	}
	if *limit < 0 {
		return fmt.Errorf("limit must be >= 0")
	}

	workspaceName, err := requireWorkspaceArg(a, workspaceName)
	if err != nil {
		return err
	}
	hits, err := a.IndexSearch(workspaceName, strings.Join(fs.Args(), " "), app.IndexSearchOptions{Limit: *limit})
	if err != nil {
		return err
	}
	if len(hits) == 0 {
		fmt.Fprintln(os.Stdout, "No indexed files.")
		return nil
	}
	for _, hit := range hits {
		fmt.Fprintf(os.Stdout, "[score=%d] %s\n", hit.Score, hit.File.Path)
	}
	return nil
}

type indexSymbolsCmd struct{}

func (c *indexSymbolsCmd) Name() string { return "symbols" }
func (c *indexSymbolsCmd) Help() string { return "Search indexed symbols for a workspace" }

func (c *indexSymbolsCmd) Run(a *app.App, args []string) error {
	if len(args) == 0 {
		fs := flag.NewFlagSet("index symbols", flag.ContinueOnError)
		fs.SetOutput(os.Stdout)
		fs.Usage = func() {
			fmt.Fprintln(fs.Output(), "usage: groot index symbols <workspace> <query>")
			fmt.Fprintln(fs.Output())
			fmt.Fprintln(fs.Output(), c.Help())
		}
		fs.Usage()
		return fmt.Errorf("workspace name required")
	}
	workspaceName := args[0]
	fs := flag.NewFlagSet("index symbols", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	limit := fs.Int("limit", 10, "maximum number of symbol results to print")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot index symbols <workspace> <query>")
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
		return fmt.Errorf("symbol query required")
	}
	if *limit < 0 {
		return fmt.Errorf("limit must be >= 0")
	}

	workspaceName, err := requireWorkspaceArg(a, workspaceName)
	if err != nil {
		return err
	}
	hits, err := a.IndexSymbols(workspaceName, strings.Join(fs.Args(), " "), app.IndexSearchOptions{Limit: *limit})
	if err != nil {
		return err
	}
	if len(hits) == 0 {
		fmt.Fprintln(os.Stdout, "No indexed symbols.")
		return nil
	}
	for _, hit := range hits {
		fmt.Fprintf(os.Stdout, "[score=%d] %s\t%s\t%s:%d-%d\n", hit.Score, hit.Symbol.QualifiedName, hit.Symbol.Kind, hit.Symbol.FilePath, hit.Symbol.LineStart, hit.Symbol.LineEnd)
	}
	return nil
}

type indexStatsCmd struct{}

func (c *indexStatsCmd) Name() string { return "stats" }
func (c *indexStatsCmd) Help() string { return "Print index counts for a workspace" }

func (c *indexStatsCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("index stats", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot index stats <workspace>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("workspace name required")
	}

	workspaceName, err := requireWorkspaceArg(a, fs.Arg(0))
	if err != nil {
		return err
	}
	stats, err := a.IndexStats(workspaceName)
	if err != nil {
		return err
	}
	meta, err := a.IndexMetadata(workspaceName)
	if err != nil {
		return err
	}
	status, err := a.IndexStatus(workspaceName)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Indexed: %t\n", meta.Indexed)
	if meta.IndexedAt.IsZero() {
		fmt.Fprintln(os.Stdout, "Indexed At: -")
	} else {
		fmt.Fprintf(os.Stdout, "Indexed At: %s\n", meta.IndexedAt.Format(time.RFC3339))
	}
	fmt.Fprintf(os.Stdout, "Fresh: %t\n", status.Fresh)
	fmt.Fprintf(os.Stdout, "Status: %s\n", status.Reason)
	fmt.Fprintf(os.Stdout, "Workspace: %s\n", meta.Workspace)
	if meta.ProjectPath == "" {
		fmt.Fprintln(os.Stdout, "Project Path: -")
	} else {
		fmt.Fprintf(os.Stdout, "Project Path: %s\n", meta.ProjectPath)
	}
	fmt.Fprintf(os.Stdout, "Files: %d\n", stats.FileCount)
	fmt.Fprintf(os.Stdout, "Symbols: %d\n", stats.SymbolCount)
	fmt.Fprintf(os.Stdout, "Terms: %d\n", stats.TermCount)
	fmt.Fprintf(os.Stdout, "Forward Entries: %d\n", stats.ForwardCount)
	fmt.Fprintf(os.Stdout, "Reverse Entries: %d\n", stats.ReverseCount)
	fmt.Fprintf(os.Stdout, "Hashes: %d\n", stats.HashCount)
	writeVaultCountMap("By Language", stats.ByLanguage)
	return nil
}
