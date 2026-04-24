package app

import (
	"testing"
)

func TestTaskLifecycleEventsArePersisted(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)

	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	task, err := app.StartTask("crawlly", TaskStartSpec{
		Name:    "echo",
		Command: "/bin/sh",
		Args:    []string{"-c", "printf ok"},
	})
	if err != nil {
		t.Fatalf("StartTask returned error: %v", err)
	}
	waitForTaskState(t, app, "crawlly", task.ID, TaskRunSucceeded)

	events, err := app.EventList("crawlly", EventListOptions{})
	if err != nil {
		t.Fatalf("EventList returned error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d: %#v", len(events), events)
	}
	if events[0].Kind != EventKindTaskExited {
		t.Fatalf("newest event kind = %q, want %q", events[0].Kind, EventKindTaskExited)
	}
	if _, ok := events[0].Payload["finished_at"].(string); !ok {
		t.Fatalf("expected terminal event to include finished_at, got %#v", events[0].Payload)
	}
	if events[1].Kind != EventKindTaskStarted {
		t.Fatalf("oldest event kind = %q, want %q", events[1].Kind, EventKindTaskStarted)
	}
	for _, event := range events {
		if event.ID == "" {
			t.Fatalf("event id was empty: %#v", event)
		}
		if event.Workspace != "crawlly" {
			t.Fatalf("event workspace = %q, want crawlly", event.Workspace)
		}
		if event.ResourceType != "task" || event.ResourceID != task.ID {
			t.Fatalf("unexpected resource identity: %#v", event)
		}
	}
}

func TestTaskTerminalEventIsEmittedOnce(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)

	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	task, err := app.StartTask("crawlly", TaskStartSpec{
		Name:    "fail",
		Command: "/bin/sh",
		Args:    []string{"-c", "exit 7"},
	})
	if err != nil {
		t.Fatalf("StartTask returned error: %v", err)
	}
	waitForTaskState(t, app, "crawlly", task.ID, TaskRunFailed)
	if _, err := app.TaskStatus("crawlly", task.ID); err != nil {
		t.Fatalf("TaskStatus returned error: %v", err)
	}
	if _, err := app.TaskList("crawlly"); err != nil {
		t.Fatalf("TaskList returned error: %v", err)
	}

	events, err := app.EventList("crawlly", EventListOptions{})
	if err != nil {
		t.Fatalf("EventList returned error: %v", err)
	}
	var failed int
	for _, event := range events {
		if event.Kind == EventKindTaskFailed {
			failed++
		}
	}
	if failed != 1 {
		t.Fatalf("expected one failed event, got %d: %#v", failed, events)
	}
}

func TestEventListLimit(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)

	if err := app.CreateNewWorkspace("crawlly"); err != nil {
		t.Fatalf("CreateNewWorkspace returned error: %v", err)
	}

	for _, name := range []string{"first", "second"} {
		task, err := app.StartTask("crawlly", TaskStartSpec{
			Name:    name,
			Command: "/bin/sh",
			Args:    []string{"-c", "printf ok"},
		})
		if err != nil {
			t.Fatalf("StartTask returned error: %v", err)
		}
		waitForTaskState(t, app, "crawlly", task.ID, TaskRunSucceeded)
	}

	events, err := app.EventList("crawlly", EventListOptions{Limit: 1})
	if err != nil {
		t.Fatalf("EventList returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}
