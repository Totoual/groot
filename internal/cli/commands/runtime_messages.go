package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/totoual/groot/internal/app"
)

func writeFirstOpenRuntimePlan(plan app.FirstOpenRuntimePlan) {
	if len(plan.Detected) == 0 {
		fmt.Fprintf(os.Stderr, "No likely runtimes detected for workspace %q.\n", plan.WorkspaceName)
		return
	}

	fmt.Fprintf(os.Stderr, "Detected likely runtimes for workspace %q: %s\n", plan.WorkspaceName, formatDetectedToolchains(plan.Detected))

	if !plan.AttachRequested {
		fmt.Fprintln(os.Stderr, "First-open behavior is warn-only for now: Groot did not attach toolchains automatically.")
		return
	}

	if len(plan.Attached) > 0 {
		fmt.Fprintf(os.Stderr, "Auto-attached detected runtimes for workspace %q: %s\n", plan.WorkspaceName, formatDetectedToolchains(plan.Attached))
		if !plan.InstallRequested {
			fmt.Fprintf(os.Stderr, "Install them with:\n  groot ws install %s\n", plan.WorkspaceName)
		}
	}
	if len(plan.Installed) > 0 {
		fmt.Fprintf(os.Stderr, "Installed detected runtimes for workspace %q: %s\n", plan.WorkspaceName, formatDetectedToolchains(plan.Installed))
	}
	if len(plan.Skipped) > 0 {
		fmt.Fprintf(os.Stderr, "Skipped detected runtimes without a concrete version for workspace %q: %s\n", plan.WorkspaceName, formatDetectedToolchains(plan.Skipped))
		fmt.Fprintln(os.Stderr, "Attach those manually once you choose the desired versions.")
	}
	writeFirstOpenSummary(plan)
}

func writeWorkspaceOwnershipWarning(report app.WorkspaceRuntimeOwnership) {
	if len(report.Missing) == 0 {
		return
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Workspace %q does not declare detected runtimes: %s\n", report.WorkspaceName, formatDetectedToolchains(report.Missing))
	fmt.Fprintln(&b, "Commands may fall back to host toolchains until these are attached and installed.")
	fmt.Fprintln(&b, "Attach them with:")
	fmt.Fprintf(&b, "  groot ws attach %s %s\n", report.WorkspaceName, suggestedAttachArgs(report.Missing))
	fmt.Fprintf(&b, "  groot ws install %s\n", report.WorkspaceName)
	if app.RuntimeStrictModeEnabled() {
		fmt.Fprintln(&b, "Strict runtime mode is enabled via GROOT_STRICT_RUNTIME, so this command will stop here.")
	}
	fmt.Fprint(os.Stderr, b.String())
}

func writeWorkspaceRuntimeStatus(report app.WorkspaceRuntimeOwnership) {
	fmt.Fprintf(os.Stdout, "Workspace: %s\n", report.WorkspaceName)
	if report.ProjectPath != "" {
		fmt.Fprintf(os.Stdout, "Project Path: %s\n", report.ProjectPath)
	}
	if len(report.Detected) > 0 {
		fmt.Fprintf(os.Stdout, "Detected: %s\n", formatDetectedToolchains(report.Detected))
	} else {
		fmt.Fprintln(os.Stdout, "Detected: none")
	}
	if len(report.Attached) > 0 {
		fmt.Fprintf(os.Stdout, "Attached: %s\n", formatComponents(report.Attached))
	} else {
		fmt.Fprintln(os.Stdout, "Attached: none")
	}
	if len(report.Installed) > 0 {
		fmt.Fprintf(os.Stdout, "Groot-Managed: %s\n", formatComponents(report.Installed))
	} else {
		fmt.Fprintln(os.Stdout, "Groot-Managed: none")
	}
	if len(report.Uninstalled) > 0 {
		fmt.Fprintf(os.Stdout, "Attached But Not Installed: %s\n", formatComponents(report.Uninstalled))
	}
	if len(report.Missing) > 0 {
		fmt.Fprintf(os.Stdout, "Host Fallback Risk: %s\n", formatDetectedToolchains(report.Missing))
		fmt.Fprintf(os.Stdout, "Status: %s\n", app.RuntimeOwnershipStatusLabel(report))
		return
	}
	fmt.Fprintln(os.Stdout, "Host Fallback Risk: none")
	fmt.Fprintf(os.Stdout, "Status: %s\n", app.RuntimeOwnershipStatusLabel(report))
}

func writeFirstOpenSummary(plan app.FirstOpenRuntimePlan) {
	missingCount := len(plan.Missing)
	switch {
	case len(plan.Detected) == 0:
		fmt.Fprintln(os.Stderr, "First-open summary: no runtimes were detected, so Groot opened the project without runtime changes.")
	case missingCount == 0 && len(plan.Installed) > 0:
		fmt.Fprintln(os.Stderr, "First-open summary: Groot attached and installed the detected runtimes, and the workspace is ready to use.")
	case missingCount == 0 && len(plan.Attached) > 0:
		fmt.Fprintln(os.Stderr, "First-open summary: Groot attached the detected runtimes. Install them to finish claiming the runtime.")
	default:
		fmt.Fprintln(os.Stderr, "First-open summary: Groot detected the project runtime, but host fallback is still possible until the missing toolchains are attached and installed.")
	}
}

func formatComponents(components []app.Component) string {
	parts := make([]string, 0, len(components))
	for _, comp := range components {
		parts = append(parts, fmt.Sprintf("%s@%s", comp.Name, comp.Version))
	}
	return strings.Join(parts, ", ")
}
