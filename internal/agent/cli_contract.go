package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type CLIContract struct {
	Binary string
}

func NewCLIContract(binary string) *CLIContract {
	if binary == "" {
		binary = "groot"
	}
	return &CLIContract{Binary: binary}
}

func (c *CLIContract) Status(projectPath string) (RuntimeStatus, error) {
	output, err := c.runCapture([]string{"status", projectPath, "--json"})
	if err != nil {
		return RuntimeStatus{}, err
	}

	var status RuntimeStatus
	if err := json.Unmarshal(output, &status); err != nil {
		return RuntimeStatus{}, fmt.Errorf("parse groot status json: %w", err)
	}
	return status, nil
}

func (c *CLIContract) Open(projectPath string) error {
	return c.run([]string{"open", projectPath})
}

func (c *CLIContract) OpenAttachDetected(projectPath string) error {
	return c.run([]string{"open", projectPath, "--attach-detected"})
}

func (c *CLIContract) OpenSetup(projectPath string) error {
	return c.run([]string{"open", projectPath, "--setup"})
}

func (c *CLIContract) Exec(projectPath string, command string, args []string) error {
	cmdArgs := make([]string, 0, len(args)+3)
	cmdArgs = append(cmdArgs, "exec", projectPath, command)
	cmdArgs = append(cmdArgs, args...)
	return c.run(cmdArgs)
}

func (c *CLIContract) Enter(projectPath string) error {
	return c.run([]string{"enter", projectPath})
}

func (c *CLIContract) run(args []string) error {
	_, err := c.runCapture(args)
	return err
}

func (c *CLIContract) runCapture(args []string) ([]byte, error) {
	cmd := exec.Command(c.Binary, args...)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return nil, fmt.Errorf("run %q: %w", strings.Join(append([]string{c.Binary}, args...), " "), err)
		}
		return nil, fmt.Errorf("run %q: %w: %s", strings.Join(append([]string{c.Binary}, args...), " "), err, trimmed)
	}
	return output, nil
}
