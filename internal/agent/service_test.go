package agent

import (
	"errors"
	"testing"
)

type mockContract struct {
	statusCalls []string
	openCalls   []string
	status      RuntimeStatus
	statusErr   error
	openErr     error
}

func (m *mockContract) Status(projectPath string) (RuntimeStatus, error) {
	m.statusCalls = append(m.statusCalls, projectPath)
	if m.statusErr != nil {
		return RuntimeStatus{}, m.statusErr
	}
	return m.status, nil
}

func (m *mockContract) Open(projectPath string) error {
	return nil
}

func (m *mockContract) OpenAttachDetected(projectPath string) error {
	return nil
}

func (m *mockContract) OpenSetup(projectPath string) error {
	m.openCalls = append(m.openCalls, projectPath)
	return m.openErr
}

func (m *mockContract) Exec(projectPath string, command string, args []string) error {
	return nil
}

func (m *mockContract) Enter(projectPath string) error {
	return nil
}

func TestServiceInspectStatus(t *testing.T) {
	mock := &mockContract{
		status: RuntimeStatus{
			WorkspaceName: "the_grime_tcg",
			Status:        "runtime owned by Groot",
		},
	}
	svc := NewService(mock)

	status, err := svc.InspectStatus("/tmp/the_grime_tcg")
	if err != nil {
		t.Fatalf("InspectStatus returned error: %v", err)
	}
	if status.WorkspaceName != "the_grime_tcg" {
		t.Fatalf("WorkspaceName = %q, want %q", status.WorkspaceName, "the_grime_tcg")
	}
	if len(mock.statusCalls) != 1 || mock.statusCalls[0] != "/tmp/the_grime_tcg" {
		t.Fatalf("unexpected status calls: %#v", mock.statusCalls)
	}
}

func TestServiceOpenAndSetup(t *testing.T) {
	mock := &mockContract{
		status: RuntimeStatus{
			WorkspaceName: "the_grime_tcg",
			Status:        "runtime owned by Groot",
		},
	}
	svc := NewService(mock)

	status, err := svc.OpenAndSetup("/tmp/the_grime_tcg")
	if err != nil {
		t.Fatalf("OpenAndSetup returned error: %v", err)
	}
	if status.Status != "runtime owned by Groot" {
		t.Fatalf("Status = %q, want %q", status.Status, "runtime owned by Groot")
	}
	if len(mock.openCalls) != 1 || mock.openCalls[0] != "/tmp/the_grime_tcg" {
		t.Fatalf("unexpected open calls: %#v", mock.openCalls)
	}
	if len(mock.statusCalls) != 1 || mock.statusCalls[0] != "/tmp/the_grime_tcg" {
		t.Fatalf("unexpected status calls: %#v", mock.statusCalls)
	}
}

func TestServiceOpenAndSetupReturnsSetupError(t *testing.T) {
	mock := &mockContract{
		openErr: errors.New("setup failed"),
	}
	svc := NewService(mock)

	_, err := svc.OpenAndSetup("/tmp/the_grime_tcg")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "setup failed" {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.statusCalls) != 0 {
		t.Fatalf("did not expect status call on setup failure, got %#v", mock.statusCalls)
	}
}
