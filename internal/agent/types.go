package agent

type IntentKind string

const (
	IntentCreateProject IntentKind = "create_project"
	IntentOpenSetup     IntentKind = "open_setup"
	IntentInspectStatus IntentKind = "inspect_status"
	IntentRunCommand    IntentKind = "run_command"
)

type IntentSpec struct {
	Kind        IntentKind
	Description string
}

type ToolchainStatus struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Source  string `json:"source,omitempty"`
}

type RuntimeStatus struct {
	WorkspaceName       string            `json:"workspace_name"`
	ProjectPath         string            `json:"project_path,omitempty"`
	Status              string            `json:"status"`
	Detected            []ToolchainStatus `json:"detected"`
	Attached            []ToolchainStatus `json:"attached"`
	Installed           []ToolchainStatus `json:"installed"`
	AttachedUninstalled []ToolchainStatus `json:"attached_uninstalled"`
	Missing             []ToolchainStatus `json:"missing"`
}
