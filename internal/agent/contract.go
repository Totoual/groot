package agent

type GrootContract interface {
	Status(projectPath string) (RuntimeStatus, error)
	Open(projectPath string) error
	OpenAttachDetected(projectPath string) error
	OpenSetup(projectPath string) error
	Exec(projectPath string, command string, args []string) error
	Enter(projectPath string) error
}
