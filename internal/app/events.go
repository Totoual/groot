package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	EventKindTaskStarted   = "task.started"
	EventKindTaskExited    = "task.exited"
	EventKindTaskFailed    = "task.failed"
	EventKindTaskCancelled = "task.cancelled"
)

type RuntimeEvent struct {
	ID           string         `json:"id"`
	Workspace    string         `json:"workspace"`
	Kind         string         `json:"kind"`
	Timestamp    time.Time      `json:"timestamp"`
	ResourceType string         `json:"resource_type"`
	ResourceID   string         `json:"resource_id"`
	Message      string         `json:"message"`
	Payload      map[string]any `json:"payload,omitempty"`
}

type EventListOptions struct {
	Limit int
}

func (a *App) EventList(workspaceName string, opts EventListOptions) ([]RuntimeEvent, error) {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return nil, err
	}

	path := workspaceEventLogPath(wsPath)
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []RuntimeEvent{}, nil
		}
		return nil, fmt.Errorf("open event log: %w", err)
	}
	defer file.Close()

	events := []RuntimeEvent{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event RuntimeEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("decode event log: %w", err)
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read event log: %w", err)
	}

	slices.SortFunc(events, func(a, b RuntimeEvent) int {
		return b.Timestamp.Compare(a.Timestamp)
	})
	if opts.Limit > 0 && len(events) > opts.Limit {
		events = events[:opts.Limit]
	}
	return events, nil
}

func (a *App) emitEvent(workspaceName string, event RuntimeEvent) error {
	wsPath, err := a.EnsureWorkspace(workspaceName)
	if err != nil {
		return err
	}
	if strings.TrimSpace(event.ID) == "" {
		id, err := newRuntimeEventID()
		if err != nil {
			return err
		}
		event.ID = id
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	event.Workspace = workspaceName

	path := workspaceEventLogPath(wsPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("mkdir event state dir: %w", err)
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open event log: %w", err)
	}
	defer file.Close()
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write event log: %w", err)
	}
	return nil
}

func workspaceEventLogPath(wsPath string) string {
	return filepath.Join(wsPath, "state", "events", "events.jsonl")
}

func newRuntimeEventID() (string, error) {
	suffix, err := randomHex(6)
	if err != nil {
		return "", fmt.Errorf("generate event id: %w", err)
	}
	return fmt.Sprintf("%d-%s", time.Now().UTC().UnixMilli(), suffix), nil
}
