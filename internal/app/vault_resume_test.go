package app

import (
	"strings"
	"testing"
	"time"
)

func TestBuildTaskResumeByExactTaskIDIncludesLinkedNodes(t *testing.T) {
	app, task, latestProgress, _, decision, rule, pattern, failure := setupTaskResumeFixture(t)

	resume, err := app.BuildTaskResume("crawlly", task.ID)
	if err != nil {
		t.Fatalf("BuildTaskResume returned error: %v", err)
	}

	if resume.Task.ID != task.ID {
		t.Fatalf("resume task = %q, want %q", resume.Task.ID, task.ID)
	}
	if resume.LatestProgress == nil || resume.LatestProgress.ID != latestProgress.ID {
		t.Fatalf("latest progress = %#v, want %q", resume.LatestProgress, latestProgress.ID)
	}
	if len(resume.Decisions) != 1 || resume.Decisions[0].ID != decision.ID {
		t.Fatalf("unexpected decisions: %#v", resume.Decisions)
	}
	if len(resume.Rules) != 1 || resume.Rules[0].ID != rule.ID {
		t.Fatalf("unexpected rules: %#v", resume.Rules)
	}
	if len(resume.Patterns) != 1 || resume.Patterns[0].ID != pattern.ID {
		t.Fatalf("unexpected patterns: %#v", resume.Patterns)
	}
	if len(resume.Failures) != 1 || resume.Failures[0].ID != failure.ID {
		t.Fatalf("unexpected failures: %#v", resume.Failures)
	}
	if len(resume.Edges) != 6 {
		t.Fatalf("expected 6 relevant edges, got %#v", resume.Edges)
	}
	if resume.Freshness.Workspace != "crawlly" || resume.Freshness.VaultUpdatedAt.IsZero() {
		t.Fatalf("unexpected freshness metadata: %#v", resume.Freshness)
	}
}

func TestFindVaultTaskCandidatesByQueryAndResolveTaskResume(t *testing.T) {
	app, task, latestProgress, _, _, _, _, _ := setupTaskResumeFixture(t)

	otherTask, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeTask,
		Title: "Implement unrelated command",
		Body:  "body",
	})
	if err != nil {
		t.Fatalf("VaultAppend other task returned error: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	newestTask, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeTask,
		Title: "Implement vault relationship queries later",
		Body:  "body",
	})
	if err != nil {
		t.Fatalf("VaultAppend newest task returned error: %v", err)
	}

	candidates, err := app.FindVaultTaskCandidates("crawlly", "vault relationship queries", 5)
	if err != nil {
		t.Fatalf("FindVaultTaskCandidates returned error: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %#v", candidates)
	}
	if candidates[0].Task.ID != newestTask.ID || candidates[1].Task.ID != task.ID {
		t.Fatalf("unexpected candidate order: %#v", candidates)
	}

	resume, err := app.ResolveTaskResume("crawlly", task.Title)
	if err != nil {
		t.Fatalf("ResolveTaskResume returned error: %v", err)
	}
	if resume.Task.ID != task.ID {
		t.Fatalf("resolved task = %q, want %q", resume.Task.ID, task.ID)
	}
	if resume.LatestProgress == nil || resume.LatestProgress.ID != latestProgress.ID {
		t.Fatalf("resolved latest progress = %#v, want %q", resume.LatestProgress, latestProgress.ID)
	}

	exactByID, err := app.FindVaultTaskCandidates("crawlly", otherTask.ID, 5)
	if err != nil {
		t.Fatalf("FindVaultTaskCandidates exact id returned error: %v", err)
	}
	if len(exactByID) != 1 || exactByID[0].Task.ID != otherTask.ID {
		t.Fatalf("unexpected exact-id candidates: %#v", exactByID)
	}
}

func TestBuildTaskResumeDeterministicOrderingAndMarkdown(t *testing.T) {
	app, task, _, _, _, _, _, _ := setupTaskResumeFixture(t)

	first, err := app.BuildTaskResume("crawlly", task.ID)
	if err != nil {
		t.Fatalf("BuildTaskResume first returned error: %v", err)
	}
	second, err := app.BuildTaskResume("crawlly", task.ID)
	if err != nil {
		t.Fatalf("BuildTaskResume second returned error: %v", err)
	}
	if first.Markdown() != second.Markdown() {
		t.Fatalf("expected deterministic markdown output\nfirst:\n%s\nsecond:\n%s", first.Markdown(), second.Markdown())
	}
	if !strings.Contains(first.Markdown(), "# Groot Task Resume") ||
		!strings.Contains(first.Markdown(), "Latest Progress:") ||
		!strings.Contains(first.Markdown(), "Rules / Constraints:") {
		t.Fatalf("unexpected resume markdown:\n%s", first.Markdown())
	}
}

func TestBuildTaskResumeMissingTaskBehavior(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	if _, err := app.BuildTaskResume("crawlly", ""); err == nil || !strings.Contains(err.Error(), "task id required") {
		t.Fatalf("expected empty task id error, got %v", err)
	}
	if _, err := app.BuildTaskResume("crawlly", "node-missing"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected missing task error, got %v", err)
	}
	candidates, err := app.FindVaultTaskCandidates("crawlly", "missing", 5)
	if err != nil {
		t.Fatalf("FindVaultTaskCandidates missing query returned error: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected no candidates, got %#v", candidates)
	}
	if _, err := app.ResolveTaskResume("crawlly", "missing"); err == nil || !strings.Contains(err.Error(), "found no task") {
		t.Fatalf("expected missing query error, got %v", err)
	}
}

func setupTaskResumeFixture(t *testing.T) (*App, VaultNode, VaultNode, VaultNode, VaultNode, VaultNode, VaultNode, VaultNode) {
	t.Helper()

	root := t.TempDir()
	app := NewApp(root)
	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	task, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:   VaultNodeTypeTask,
		Title:  "Implement vault relationship queries",
		Body:   "Add deterministic relationship queries over workspace vault edges.",
		Status: "active",
	})
	if err != nil {
		t.Fatalf("VaultAppend task returned error: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	olderProgress, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeProgress,
		Title: "Stopped after app read support",
		Body:  "Remaining work: MCP query tool and tests.",
	})
	if err != nil {
		t.Fatalf("VaultAppend older progress returned error: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	latestProgress, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeProgress,
		Title: "Stopped after app and MCP read support",
		Body:  "Completed work: app-layer VaultQueryEdges and MCP vault_edge_query are implemented and tested.",
	})
	if err != nil {
		t.Fatalf("VaultAppend latest progress returned error: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	decision, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeDecision,
		Title: "Keep relationship queries node-centric",
		Body:  "Do not expand into graph traversal in the first slice.",
	})
	if err != nil {
		t.Fatalf("VaultAppend decision returned error: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	rule, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeRule,
		Title: "Keep append-only vault semantics",
		Body:  "Do not mutate existing nodes or edges.",
	})
	if err != nil {
		t.Fatalf("VaultAppend rule returned error: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	pattern, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypePattern,
		Title: "App query first, then thin surfaces",
		Body:  "Build deterministic app logic before CLI and MCP wrappers.",
	})
	if err != nil {
		t.Fatalf("VaultAppend pattern returned error: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	failure, err := app.VaultAppend("crawlly", VaultAppendSpec{
		Type:  VaultNodeTypeFailure,
		Title: "Graph semantics can sprawl too early",
		Body:  "Keep the first slice narrow and inspectable.",
	})
	if err != nil {
		t.Fatalf("VaultAppend failure returned error: %v", err)
	}

	for _, spec := range []VaultEdgeAppendSpec{
		{FromID: olderProgress.ID, ToID: task.ID, Type: VaultEdgeTypeForTask},
		{FromID: latestProgress.ID, ToID: task.ID, Type: VaultEdgeTypeForTask},
		{FromID: decision.ID, ToID: task.ID, Type: VaultEdgeTypeSupports},
		{FromID: rule.ID, ToID: task.ID, Type: VaultEdgeTypeSupports},
		{FromID: pattern.ID, ToID: task.ID, Type: VaultEdgeTypeImplements},
		{FromID: failure.ID, ToID: task.ID, Type: VaultEdgeTypeContradicts},
	} {
		time.Sleep(2 * time.Millisecond)
		if _, err := app.VaultAppendEdge("crawlly", spec); err != nil {
			t.Fatalf("VaultAppendEdge %#v returned error: %v", spec, err)
		}
	}

	return app, task, latestProgress, olderProgress, decision, rule, pattern, failure
}
