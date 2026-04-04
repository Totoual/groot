package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/totoual/groot/internal/app"
)

func enforceWorkspaceOwnership(a *app.App, workspaceName string) error {
	report, err := a.InspectWorkspaceRuntimeOwnership(workspaceName)
	if err != nil {
		return fmt.Errorf("couldn't inspect workspace runtime ownership: %w", err)
	}
	if len(report.Missing) == 0 {
		return nil
	}

	writeWorkspaceOwnershipWarning(report)

	if runtimeStrictModeEnabled() {
		return fmt.Errorf("strict runtime mode rejected undeclared detected runtimes for workspace %q", workspaceName)
	}
	return nil
}

func runtimeStrictModeEnabled() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("GROOT_STRICT_RUNTIME")))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func formatDetectedToolchains(detected []app.DetectedToolchain) string {
	parts := make([]string, 0, len(detected))
	for _, tc := range detected {
		if tc.Version != "" {
			parts = append(parts, fmt.Sprintf("%s@%s", tc.Name, tc.Version))
			continue
		}
		parts = append(parts, tc.Name)
	}
	return strings.Join(parts, ", ")
}

func suggestedAttachArgs(detected []app.DetectedToolchain) string {
	parts := make([]string, 0, len(detected))
	for _, tc := range detected {
		version := tc.Version
		if version == "" {
			version = "<version>"
		}
		parts = append(parts, fmt.Sprintf("%s@%s", tc.Name, version))
	}
	return strings.Join(parts, " ")
}
