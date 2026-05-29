package commands

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/totoual/groot/internal/app"
)

type StatusCmd struct{}

func (c *StatusCmd) Name() string { return "status" }

func (c *StatusCmd) Help() string {
	return "Print a quick workspace overview for humans"
}

func (c *StatusCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	jsonOutput := fs.Bool("json", false, "print status as JSON")

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot status <workspace-or-path> [--json]")
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
		return fmt.Errorf("workspace name or project path required")
	}

	workspaceName, fromProjectPath, err := resolveWorkspaceOrProjectArg(a, fs.Arg(0))
	if err != nil {
		return err
	}
	if fromProjectPath {
		report, err := a.InspectWorkspaceRuntimeOwnership(workspaceName)
		if err != nil {
			return fmt.Errorf("couldn't inspect workspace runtime ownership: %w", err)
		}
		if *jsonOutput {
			return writeWorkspaceRuntimeStatusJSON(report)
		}
		writeWorkspaceRuntimeStatus(report)
		return nil
	}

	report, err := buildWorkspaceStatusReport(a, workspaceName)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeWorkspaceStatusJSON(report)
	}
	writeWorkspaceStatus(report)
	return nil
}

func writeWorkspaceRuntimeStatusJSON(report app.WorkspaceRuntimeOwnership) error {
	data, err := json.MarshalIndent(app.WorkspaceRuntimeSnapshotFor(report), "", "  ")
	if err != nil {
		return fmt.Errorf("marshal status json: %w", err)
	}
	fmt.Fprintln(os.Stdout, string(data))
	return nil
}

type workspaceStatusReport struct {
	WorkspaceName  string                        `json:"workspace_name"`
	ProjectPath    string                        `json:"project_path"`
	RuntimeStatus  string                        `json:"runtime_status"`
	IndexStatus    app.IndexStatus               `json:"index_status"`
	VaultStats     app.VaultStats                `json:"vault_stats"`
	LatestTask     *app.VaultNode                `json:"latest_task,omitempty"`
	LatestProgress *app.VaultNode                `json:"latest_progress,omitempty"`
	RecentNodes    []app.VaultNode               `json:"recent_nodes,omitempty"`
	Runtime        app.WorkspaceRuntimeOwnership `json:"runtime"`
}

func buildWorkspaceStatusReport(a *app.App, workspaceName string) (workspaceStatusReport, error) {
	inspection, err := a.InspectWorkspace(workspaceName)
	if err != nil {
		return workspaceStatusReport{}, err
	}
	indexStatus, err := a.IndexStatus(workspaceName)
	if err != nil {
		return workspaceStatusReport{}, err
	}
	vaultStats, err := a.VaultStats(workspaceName)
	if err != nil {
		return workspaceStatusReport{}, err
	}
	recentNodes, err := a.VaultRecent(workspaceName, app.VaultRecentOptions{Limit: 12})
	if err != nil {
		return workspaceStatusReport{}, err
	}

	report := workspaceStatusReport{
		WorkspaceName: workspaceName,
		ProjectPath:   inspection.Manifest.ProjectPath,
		RuntimeStatus: app.RuntimeOwnershipStatusLabel(inspection.Runtime),
		IndexStatus:   indexStatus,
		VaultStats:    vaultStats,
		RecentNodes:   recentNodes,
		Runtime:       inspection.Runtime,
	}
	report.LatestTask = firstVaultNodeByType(recentNodes, app.VaultNodeTypeTask)
	report.LatestProgress = firstVaultNodeByType(recentNodes, app.VaultNodeTypeProgress)
	return report, nil
}

func writeWorkspaceStatusJSON(report workspaceStatusReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal status json: %w", err)
	}
	fmt.Fprintln(os.Stdout, string(data))
	return nil
}

func writeWorkspaceStatus(report workspaceStatusReport) {
	fmt.Fprintf(os.Stdout, "Workspace: %s\n", report.WorkspaceName)
	if strings.TrimSpace(report.ProjectPath) == "" {
		fmt.Fprintln(os.Stdout, "Project Path: -")
	} else {
		fmt.Fprintf(os.Stdout, "Project Path: %s\n", report.ProjectPath)
	}
	fmt.Fprintf(os.Stdout, "Runtime: %s\n", report.RuntimeStatus)

	indexLine := report.IndexStatus.Reason
	if report.IndexStatus.IndexedAt.IsZero() {
		fmt.Fprintf(os.Stdout, "Index: %s\n", indexLine)
	} else {
		fmt.Fprintf(os.Stdout, "Index: %s (%s)\n", indexLine, report.IndexStatus.IndexedAt.Format(time.RFC3339))
	}

	fmt.Fprintf(os.Stdout, "Vault: %d nodes, %d edges, %d changes\n", report.VaultStats.NodeCount, report.VaultStats.EdgeCount, report.VaultStats.ChangeCount)
	writeWorkspaceStatusNodeLine("Latest Task", report.LatestTask)
	writeWorkspaceStatusNodeLine("Latest Progress", report.LatestProgress)

	fmt.Fprintln(os.Stdout, "Counts:")
	fmt.Fprintf(os.Stdout, "  Files: %d\n", report.IndexStatus.FileCount)
	fmt.Fprintf(os.Stdout, "  Symbols: %d\n", report.IndexStatus.SymbolCount)
	fmt.Fprintf(os.Stdout, "  Terms: %d\n", report.IndexStatus.TermCount)
}

func writeWorkspaceStatusNodeLine(label string, node *app.VaultNode) {
	if node == nil {
		fmt.Fprintf(os.Stdout, "%s: -\n", label)
		return
	}
	fmt.Fprintf(os.Stdout, "%s: %s", label, node.Title)
	if status := strings.TrimSpace(node.Status); status != "" {
		fmt.Fprintf(os.Stdout, " [%s]", status)
	}
	fmt.Fprintln(os.Stdout)
}

func firstVaultNodeByType(nodes []app.VaultNode, nodeType string) *app.VaultNode {
	for _, node := range nodes {
		if node.Type == nodeType {
			copyNode := node
			return &copyNode
		}
	}
	return nil
}
