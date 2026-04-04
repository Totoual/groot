package agent

import (
	"errors"
	"fmt"
	"strings"
)

var ErrAgentNotImplemented = errors.New("groot agent is not implemented yet")

type Service struct {
	groot GrootContract
}

func NewService(groot GrootContract) *Service {
	return &Service{groot: groot}
}

func (s *Service) SupportedIntents() []IntentSpec {
	return []IntentSpec{
		{
			Kind:        IntentCreateProject,
			Description: "Create a new project and prepare a Groot-managed runtime",
		},
		{
			Kind:        IntentOpenSetup,
			Description: "Open an existing project and move it toward a Groot-managed runtime",
		},
		{
			Kind:        IntentInspectStatus,
			Description: "Inspect runtime ownership and workspace status for a project",
		},
		{
			Kind:        IntentRunCommand,
			Description: "Run a command in the strict Groot-managed runtime for a project",
		},
	}
}

func (s *Service) InspectStatus(projectPath string) (RuntimeStatus, error) {
	projectPath = strings.TrimSpace(projectPath)
	if projectPath == "" {
		return RuntimeStatus{}, fmt.Errorf("project path required")
	}
	return s.groot.Status(projectPath)
}

func (s *Service) OpenAndSetup(projectPath string) (RuntimeStatus, error) {
	projectPath = strings.TrimSpace(projectPath)
	if projectPath == "" {
		return RuntimeStatus{}, fmt.Errorf("project path required")
	}
	if err := s.groot.OpenSetup(projectPath); err != nil {
		return RuntimeStatus{}, err
	}
	return s.groot.Status(projectPath)
}

func (s *Service) HandleIntent(raw string) error {
	_ = raw
	return ErrAgentNotImplemented
}
