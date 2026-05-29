package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/totoual/groot/internal/app"
)

type ResumeCmd struct{}

func (c *ResumeCmd) Name() string { return "resume" }

func (c *ResumeCmd) Help() string {
	return "Resume interrupted work from the latest task and progress context"
}

func (c *ResumeCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("resume", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot resume <workspace>")
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
	resume, err := latestWorkspaceTaskResume(a, workspaceName)
	if err != nil {
		return err
	}
	pack, err := a.BuildContextPackWithOptions(workspaceName, resume.Task.Title, app.ContextBuildOptions{
		Mode: app.ContextModeHandoff,
	})
	if err != nil {
		return err
	}
	indexStatus, err := a.IndexStatus(workspaceName)
	if err != nil {
		return err
	}
	writeResumeReport(workspaceName, resume, pack, indexStatus)
	return nil
}

func latestWorkspaceTaskResume(a *app.App, workspaceName string) (app.VaultTaskResume, error) {
	recentNodes, err := a.VaultRecent(workspaceName, app.VaultRecentOptions{Limit: 25})
	if err != nil {
		return app.VaultTaskResume{}, err
	}

	for _, node := range recentNodes {
		switch node.Type {
		case app.VaultNodeTypeProgress:
			edges, err := a.VaultQueryEdges(workspaceName, app.VaultEdgeQueryOptions{
				NodeID:    node.ID,
				Direction: "outgoing",
				Type:      app.VaultEdgeTypeForTask,
				Limit:     1,
			})
			if err != nil {
				return app.VaultTaskResume{}, err
			}
			if len(edges) == 0 {
				continue
			}
			return a.BuildTaskResume(workspaceName, edges[0].ToID)
		case app.VaultNodeTypeTask:
			return a.BuildTaskResume(workspaceName, node.ID)
		}
	}

	return app.VaultTaskResume{}, fmt.Errorf("no resumable task found in workspace %q", workspaceName)
}

func writeResumeReport(workspaceName string, resume app.VaultTaskResume, pack app.ContextPack, indexStatus app.IndexStatus) {
	fmt.Fprintln(os.Stdout, "# Groot Resume")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "Workspace: %s\n\n", workspaceName)
	fmt.Fprintln(os.Stdout, "Active Task:")
	fmt.Fprintln(os.Stdout, resume.Task.Title)
	if status := strings.TrimSpace(resume.Task.Status); status != "" {
		fmt.Fprintf(os.Stdout, "Status: %s\n", status)
	}
	if body := strings.TrimSpace(resume.Task.Body); body != "" {
		fmt.Fprintf(os.Stdout, "Goal: %s\n", body)
	}

	writeResumeProgressSections(resume.LatestProgress)
	writeResumeFiles(pack.Files)
	writeResumeSymbols(pack.Symbols)
	writeResumeWarnings(indexStatus, resume)
}

func writeResumeProgressSections(progress *app.VaultNode) {
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Latest Progress:")
	if progress == nil {
		fmt.Fprintln(os.Stdout, "-")
		return
	}
	fmt.Fprintln(os.Stdout, progress.Title)
	sections := parseProgressSections(progress.Body)
	writeResumeSection("Current State", sections["current state"])
	writeResumeSection("Completed Work", sections["completed work"])
	writeResumeSection("Remaining Work", sections["remaining work"])
	writeResumeSection("Next Recommended Step", sections["next recommended step"])
}

func writeResumeFiles(files []app.IndexFileRecord) {
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Relevant Files:")
	if len(files) == 0 {
		fmt.Fprintln(os.Stdout, "-")
		return
	}
	for _, file := range files {
		fmt.Fprintf(os.Stdout, "- %s\n", file.Path)
	}
}

func writeResumeSymbols(symbols []app.IndexSymbolRecord) {
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Relevant Symbols:")
	if len(symbols) == 0 {
		fmt.Fprintln(os.Stdout, "-")
		return
	}
	for _, symbol := range symbols {
		fmt.Fprintf(os.Stdout, "- %s — %s:%d-%d\n", symbol.QualifiedName, symbol.FilePath, symbol.LineStart, symbol.LineEnd)
	}
}

func writeResumeWarnings(indexStatus app.IndexStatus, resume app.VaultTaskResume) {
	warnings := make([]string, 0, 2)
	if !indexStatus.Fresh {
		warnings = append(warnings, fmt.Sprintf("Index is %s.", indexStatus.Reason))
	}
	if resume.LatestProgress == nil {
		warnings = append(warnings, "No linked progress node found for the active task.")
	}

	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Warnings:")
	if len(warnings) == 0 {
		fmt.Fprintln(os.Stdout, "-")
		return
	}
	for _, warning := range warnings {
		fmt.Fprintf(os.Stdout, "- %s\n", warning)
	}
}

func writeResumeSection(label, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	fmt.Fprintf(os.Stdout, "%s: %s\n", label, value)
}

func parseProgressSections(body string) map[string]string {
	sections := map[string]string{}
	body = strings.TrimSpace(body)
	if body == "" {
		return sections
	}

	lines := strings.Split(body, "\n")
	currentKey := "current state"
	buffer := make([]string, 0, len(lines))
	flush := func() {
		text := strings.TrimSpace(strings.Join(buffer, "\n"))
		if text != "" && strings.TrimSpace(sections[currentKey]) == "" {
			sections[currentKey] = text
		}
		buffer = buffer[:0]
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			buffer = append(buffer, "")
			continue
		}
		if key, value, ok := parseProgressLabel(trimmed); ok {
			flush()
			currentKey = key
			if value != "" {
				sections[currentKey] = value
			}
			continue
		}
		buffer = append(buffer, trimmed)
	}
	flush()
	return sections
}

func parseProgressLabel(line string) (string, string, bool) {
	labels := map[string]string{
		"current state":         "current state",
		"completed":             "completed work",
		"completed work":        "completed work",
		"remaining":             "remaining work",
		"remaining work":        "remaining work",
		"next step":             "next recommended step",
		"next recommended step": "next recommended step",
	}
	lower := strings.ToLower(strings.TrimSpace(line))
	for label, key := range labels {
		prefix := label + ":"
		if strings.HasPrefix(lower, prefix) {
			value := strings.TrimSpace(line[len(prefix):])
			return key, value, true
		}
	}
	return "", "", false
}
