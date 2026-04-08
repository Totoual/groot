package app

import "testing"

func TestRuntimeOwnershipStatusLabelImportedWorkspaceWithoutDetectedProjectRuntime(t *testing.T) {
	report := WorkspaceRuntimeOwnership{
		WorkspaceName: "crawlly-imported",
		Attached:      []Component{{Name: "go", Version: "1.25.4"}},
		Installed:     []Component{{Name: "go", Version: "1.25.4"}},
	}

	got := RuntimeOwnershipStatusLabel(report)
	want := "workspace runtime available, but no project runtimes detected"
	if got != want {
		t.Fatalf("RuntimeOwnershipStatusLabel() = %q, want %q", got, want)
	}
}
