package commands

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/totoual/groot/internal/app"
	"github.com/totoual/groot/internal/cli/cliutil"
	"github.com/totoual/groot/internal/cli/interfaces"
)

type VaultCmd struct {
	subcmds map[string]interfaces.Cmd
}

func NewVaultCmd(cmds ...interfaces.Cmd) *VaultCmd {
	if len(cmds) == 0 {
		cmds = defaultVaultCommands()
	}
	return &VaultCmd{subcmds: vaultCommandMap(cmds...)}
}

func vaultCommandMap(cmds ...interfaces.Cmd) map[string]interfaces.Cmd {
	m := make(map[string]interfaces.Cmd, len(cmds))
	for _, c := range cmds {
		m[c.Name()] = c
	}
	return m
}

func defaultVaultCommands() []interfaces.Cmd {
	return []interfaces.Cmd{
		&vaultInitCmd{},
		&vaultAppendCmd{},
		&vaultEdgeCmd{},
		&vaultSearchCmd{},
		&vaultRecentCmd{},
		&vaultResumeCmd{},
		&vaultStatsCmd{},
	}
}

func (c *VaultCmd) commands() map[string]interfaces.Cmd {
	if c.subcmds == nil {
		c.subcmds = vaultCommandMap(defaultVaultCommands()...)
	}
	return c.subcmds
}

func (c *VaultCmd) Name() string { return "vault" }

func (c *VaultCmd) Help() string {
	return "Store and search workspace-scoped vault entries"
}

func (c *VaultCmd) Run(a *app.App, args []string) error {
	if cliutil.IsHelpRequest(args) {
		c.PrintHelp(os.Stdout)
		return nil
	}
	if len(args) == 0 {
		c.PrintHelp(os.Stdout)
		return fmt.Errorf("vault command required")
	}

	subcmd, ok := c.commands()[args[0]]
	if !ok {
		return fmt.Errorf("unknown vault command %q (try: groot vault -h)", args[0])
	}
	return subcmd.Run(a, args[1:])
}

func (c *VaultCmd) PrintHelp(w io.Writer) {
	fmt.Fprintln(w, "usage: groot vault <command> [args]")
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
	fmt.Fprintln(w, "Run 'groot vault <command> -h' for more information on a command.")
}

type vaultInitCmd struct{}

func (c *vaultInitCmd) Name() string { return "init" }
func (c *vaultInitCmd) Help() string { return "Create vault storage files for a workspace" }

func (c *vaultInitCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("vault init", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot vault init <workspace>")
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
	if err := a.InitVault(workspaceName); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Initialized vault for workspace %q.\n", workspaceName)
	return nil
}

type vaultAppendCmd struct{}

func (c *vaultAppendCmd) Name() string { return "append" }
func (c *vaultAppendCmd) Help() string { return "Append one vault node to a workspace" }

func (c *vaultAppendCmd) Run(a *app.App, args []string) error {
	if len(args) == 0 {
		fs := flag.NewFlagSet("vault append", flag.ContinueOnError)
		fs.SetOutput(os.Stdout)
		fs.Usage = func() {
			fmt.Fprintln(fs.Output(), "usage: groot vault append <workspace> --type <type> --title <title> --body <body> [--tags tag1,tag2]")
			fmt.Fprintln(fs.Output())
			fmt.Fprintln(fs.Output(), c.Help())
		}
		fs.Usage()
		return fmt.Errorf("workspace name required")
	}
	workspaceName := args[0]
	fs := flag.NewFlagSet("vault append", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	nodeType := fs.String("type", "", "vault node type")
	title := fs.String("title", "", "vault node title")
	body := fs.String("body", "", "vault node body")
	tagsRaw := fs.String("tags", "", "comma-separated vault tags")
	source := fs.String("source", "human", "vault node source")
	confidence := fs.Float64("confidence", 1.0, "vault node confidence")
	status := fs.String("status", "active", "vault node status")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot vault append <workspace> --type <type> --title <title> --body <body> [--tags tag1,tag2]")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return fmt.Errorf("vault append does not accept positional arguments after the workspace name")
	}

	workspaceName, err := requireWorkspaceArg(a, workspaceName)
	if err != nil {
		return err
	}
	node, err := a.VaultAppend(workspaceName, app.VaultAppendSpec{
		Type:       *nodeType,
		Title:      *title,
		Body:       *body,
		Tags:       parseCommaSeparatedValues(*tagsRaw),
		Source:     *source,
		Confidence: *confidence,
		Status:     *status,
	})
	if err != nil {
		return err
	}

	writeVaultNode(node)
	return nil
}

type vaultSearchCmd struct{}

type vaultEdgeCmd struct{}

func (c *vaultEdgeCmd) Name() string { return "edge" }
func (c *vaultEdgeCmd) Help() string { return "Append one vault edge between existing workspace nodes" }

func (c *vaultEdgeCmd) Run(a *app.App, args []string) error {
	if len(args) == 0 {
		fs := flag.NewFlagSet("vault edge", flag.ContinueOnError)
		fs.SetOutput(os.Stdout)
		fs.Usage = func() {
			fmt.Fprintln(fs.Output(), "usage: groot vault edge <workspace> --from <node-id> --to <node-id> --type <type>")
			fmt.Fprintln(fs.Output())
			fmt.Fprintln(fs.Output(), c.Help())
		}
		fs.Usage()
		return fmt.Errorf("workspace name required")
	}
	workspaceName := args[0]
	fs := flag.NewFlagSet("vault edge", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fromID := fs.String("from", "", "source vault node id")
	toID := fs.String("to", "", "destination vault node id")
	edgeType := fs.String("type", "", "vault edge type")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot vault edge <workspace> --from <node-id> --to <node-id> --type <type>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return fmt.Errorf("vault edge does not accept positional arguments after the workspace name")
	}

	workspaceName, err := requireWorkspaceArg(a, workspaceName)
	if err != nil {
		return err
	}
	edge, err := a.VaultAppendEdge(workspaceName, app.VaultEdgeAppendSpec{
		FromID: *fromID,
		ToID:   *toID,
		Type:   *edgeType,
	})
	if err != nil {
		return err
	}

	writeVaultEdge(edge)
	return nil
}

func (c *vaultSearchCmd) Name() string { return "search" }
func (c *vaultSearchCmd) Help() string { return "Search vault nodes for a workspace" }

func (c *vaultSearchCmd) Run(a *app.App, args []string) error {
	if len(args) == 0 {
		fs := flag.NewFlagSet("vault search", flag.ContinueOnError)
		fs.SetOutput(os.Stdout)
		fs.Usage = func() {
			fmt.Fprintln(fs.Output(), "usage: groot vault search <workspace> <query>")
			fmt.Fprintln(fs.Output())
			fmt.Fprintln(fs.Output(), c.Help())
		}
		fs.Usage()
		return fmt.Errorf("workspace name required")
	}
	workspaceName := args[0]
	fs := flag.NewFlagSet("vault search", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	limit := fs.Int("limit", 10, "maximum number of vault results to print")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot vault search <workspace> <query>")
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
		return fmt.Errorf("vault query required")
	}
	if *limit < 0 {
		return fmt.Errorf("limit must be >= 0")
	}

	workspaceName, err := requireWorkspaceArg(a, workspaceName)
	if err != nil {
		return err
	}
	query := strings.Join(fs.Args(), " ")
	hits, err := a.VaultSearch(workspaceName, query, app.VaultSearchOptions{Limit: *limit})
	if err != nil {
		return err
	}
	if len(hits) == 0 {
		fmt.Fprintln(os.Stdout, "No vault entries.")
		return nil
	}
	for _, hit := range hits {
		fmt.Fprintf(os.Stdout, "[score=%d] ", hit.Score)
		writeVaultNode(hit.Node)
	}
	return nil
}

type vaultRecentCmd struct{}

func (c *vaultRecentCmd) Name() string { return "recent" }
func (c *vaultRecentCmd) Help() string { return "Print recent vault nodes for a workspace" }

func (c *vaultRecentCmd) Run(a *app.App, args []string) error {
	if len(args) == 0 {
		fs := flag.NewFlagSet("vault recent", flag.ContinueOnError)
		fs.SetOutput(os.Stdout)
		fs.Usage = func() {
			fmt.Fprintln(fs.Output(), "usage: groot vault recent <workspace>")
			fmt.Fprintln(fs.Output())
			fmt.Fprintln(fs.Output(), c.Help())
		}
		fs.Usage()
		return fmt.Errorf("workspace name required")
	}
	workspaceName := args[0]
	fs := flag.NewFlagSet("vault recent", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	limit := fs.Int("limit", 10, "maximum number of recent vault nodes to print")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot vault recent <workspace>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return fmt.Errorf("vault recent does not accept positional arguments after the workspace name")
	}
	if *limit < 0 {
		return fmt.Errorf("limit must be >= 0")
	}

	workspaceName, err := requireWorkspaceArg(a, workspaceName)
	if err != nil {
		return err
	}
	nodes, err := a.VaultRecent(workspaceName, app.VaultRecentOptions{Limit: *limit})
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		fmt.Fprintln(os.Stdout, "No vault entries.")
		return nil
	}
	for _, node := range nodes {
		writeVaultNode(node)
	}
	return nil
}

type vaultStatsCmd struct{}

type vaultResumeCmd struct{}

func (c *vaultResumeCmd) Name() string { return "resume" }
func (c *vaultResumeCmd) Help() string {
	return "Build a compact task resume view for a workspace task"
}

func (c *vaultResumeCmd) Run(a *app.App, args []string) error {
	if len(args) == 0 {
		fs := flag.NewFlagSet("vault resume", flag.ContinueOnError)
		fs.SetOutput(os.Stdout)
		fs.Usage = func() {
			fmt.Fprintln(fs.Output(), "usage: groot vault resume <workspace> <task-or-query>")
			fmt.Fprintln(fs.Output())
			fmt.Fprintln(fs.Output(), c.Help())
		}
		fs.Usage()
		return fmt.Errorf("workspace name required")
	}
	workspaceName := args[0]
	fs := flag.NewFlagSet("vault resume", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot vault resume <workspace> <task-or-query>")
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
		return fmt.Errorf("task id or query required")
	}

	workspaceName, err := requireWorkspaceArg(a, workspaceName)
	if err != nil {
		return err
	}
	resume, err := a.ResolveTaskResume(workspaceName, strings.Join(fs.Args(), " "))
	if err != nil {
		return err
	}
	fmt.Fprint(os.Stdout, resume.Markdown())
	return nil
}

func (c *vaultStatsCmd) Name() string { return "stats" }
func (c *vaultStatsCmd) Help() string { return "Print vault counts for a workspace" }

func (c *vaultStatsCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("vault stats", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot vault stats <workspace>")
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
	stats, err := a.VaultStats(workspaceName)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Nodes: %d\n", stats.NodeCount)
	fmt.Fprintf(os.Stdout, "Edges: %d\n", stats.EdgeCount)
	fmt.Fprintf(os.Stdout, "Changes: %d\n", stats.ChangeCount)
	writeVaultCountMap("By Type", stats.ByType)
	writeVaultCountMap("By Status", stats.ByStatus)
	return nil
}

func parseCommaSeparatedValues(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		values = append(values, part)
	}
	return values
}

func writeVaultNode(node app.VaultNode) {
	body := strings.ReplaceAll(node.Body, "\n", " ")
	fmt.Fprintf(os.Stdout, "%s\t%s\t%s\t%s\t%s\n", node.ID, node.Type, node.Title, strings.Join(node.Tags, ","), body)
}

func writeVaultEdge(edge app.VaultEdge) {
	fmt.Fprintf(os.Stdout, "%s\t%s\t%s\t%s\n", edge.ID, edge.Type, edge.FromID, edge.ToID)
}

func writeVaultCountMap(label string, counts map[string]int) {
	fmt.Fprintf(os.Stdout, "%s:\n", label)
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(os.Stdout, "  %s: %d\n", key, counts[key])
	}
}
