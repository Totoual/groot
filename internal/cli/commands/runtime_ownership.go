package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/totoual/groot/internal/app"
)

func enforceWorkspaceOwnership(a *app.App, workspaceName string) error {
	missing, err := a.WorkspaceUndeclaredToolchains(workspaceName)
	if err != nil {
		return fmt.Errorf("couldn't inspect workspace runtime ownership: %w", err)
	}
	if len(missing) == 0 {
		return nil
	}

	warning := workspaceOwnershipWarning(workspaceName, missing)
	fmt.Fprint(os.Stderr, warning)

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

func workspaceOwnershipWarning(workspaceName string, missing []app.DetectedToolchain) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Workspace %q does not declare detected runtimes: %s\n", workspaceName, formatDetectedToolchains(missing))
	fmt.Fprintln(&b, "Commands may fall back to host toolchains until these are attached and installed.")
	fmt.Fprintln(&b, "Attach them with:")
	fmt.Fprintf(&b, "  groot ws attach %s %s\n", workspaceName, suggestedAttachArgs(missing))
	fmt.Fprintf(&b, "  groot ws install %s\n", workspaceName)
	if runtimeStrictModeEnabled() {
		fmt.Fprintln(&b, "Strict runtime mode is enabled via GROOT_STRICT_RUNTIME, so this command will stop here.")
	}
	return b.String()
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
