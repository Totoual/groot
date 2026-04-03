package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/totoual/groot/internal/app"
)

func writeFirstOpenRuntimePlan(plan app.FirstOpenRuntimePlan) {
	if len(plan.Detected) == 0 {
		return
	}

	fmt.Fprintf(os.Stderr, "Detected likely runtimes for workspace %q: %s\n", plan.WorkspaceName, formatDetectedToolchains(plan.Detected))

	if !plan.AttachRequested {
		fmt.Fprintln(os.Stderr, "First-open behavior is warn-only for now: Groot did not attach toolchains automatically.")
		return
	}

	if len(plan.Attached) > 0 {
		fmt.Fprintf(os.Stderr, "Auto-attached detected runtimes for workspace %q: %s\n", plan.WorkspaceName, formatDetectedToolchains(plan.Attached))
		fmt.Fprintf(os.Stderr, "Install them with:\n  groot ws install %s\n", plan.WorkspaceName)
	}
	if len(plan.Skipped) > 0 {
		fmt.Fprintf(os.Stderr, "Skipped detected runtimes without a concrete version for workspace %q: %s\n", plan.WorkspaceName, formatDetectedToolchains(plan.Skipped))
		fmt.Fprintln(os.Stderr, "Attach those manually once you choose the desired versions.")
	}
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
	if runtimeStrictModeEnabled() {
		fmt.Fprintln(&b, "Strict runtime mode is enabled via GROOT_STRICT_RUNTIME, so this command will stop here.")
	}
	fmt.Fprint(os.Stderr, b.String())
}
