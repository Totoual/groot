package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/totoual/groot/internal/app"
)

type SyncCmd struct{}

func (c *SyncCmd) Name() string { return "sync" }

func (c *SyncCmd) Help() string {
	return "Refresh index state when needed and print a workspace sync summary"
}

func (c *SyncCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot sync <workspace>")
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
	report, err := a.SyncWorkspace(workspaceName)
	if err != nil {
		return err
	}
	writeWorkspaceSyncReport(report)
	return nil
}

func writeWorkspaceSyncReport(report app.WorkspaceSyncReport) {
	fmt.Fprintf(os.Stdout, "Workspace: %s\n", report.WorkspaceName)
	if strings.TrimSpace(report.ProjectPath) == "" {
		fmt.Fprintln(os.Stdout, "Project Path: -")
	} else {
		fmt.Fprintf(os.Stdout, "Project Path: %s\n", report.ProjectPath)
	}
	fmt.Fprintf(os.Stdout, "Index Before: %s%s\n", report.Before.Reason, formatIndexedAt(report.Before.IndexedAt))
	if report.UpdatedIndex {
		fmt.Fprintln(os.Stdout, "Index Action: updated")
	} else {
		fmt.Fprintln(os.Stdout, "Index Action: no update needed")
	}
	fmt.Fprintf(os.Stdout, "Index After: %s%s\n", report.After.Reason, formatIndexedAt(report.After.IndexedAt))
	fmt.Fprintf(os.Stdout, "Vault: %d nodes, %d edges, %d changes\n", report.VaultStats.NodeCount, report.VaultStats.EdgeCount, report.VaultStats.ChangeCount)
	writeWorkspaceStatusNodeLine("Latest Task", report.LatestTask)
	writeWorkspaceStatusNodeLine("Latest Progress", report.LatestProgress)
	fmt.Fprintln(os.Stdout, "Counts:")
	fmt.Fprintf(os.Stdout, "  Files: %d -> %d\n", report.Before.FileCount, report.After.FileCount)
	fmt.Fprintf(os.Stdout, "  Symbols: %d -> %d\n", report.Before.SymbolCount, report.After.SymbolCount)
	fmt.Fprintf(os.Stdout, "  Terms: %d -> %d\n", report.Before.TermCount, report.After.TermCount)
}

func formatIndexedAt(indexedAt time.Time) string {
	if indexedAt.IsZero() {
		return ""
	}
	return fmt.Sprintf(" (%s)", indexedAt.Format(time.RFC3339))
}
