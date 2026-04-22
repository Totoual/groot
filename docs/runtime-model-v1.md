# GROOT_RUNTIME_MODEL_V1

This document defines the next runtime step after workspace ownership.

Groot already owns:

- workspace identity
- workspace runtime setup
- toolchain attachment and installation
- project activation across CLI and MCP

What it does not yet own is execution lifecycle.

This spec defines the first structured runtime resources needed to move from:

- workspace ownership

to:

- task ownership
- service ownership
- event ownership

The goal is to make Groot operable through structured runtime state instead of ad hoc shell glue.

## Purpose

Groot should remain a workspace-native runtime substrate.

It should not become a general-purpose agent, scheduler, or distributed system.

The purpose of this model is narrower:

- a workspace should own what is running inside it
- a workspace should expose inspectable runtime state
- CLI and MCP should map to the same app-layer lifecycle operations
- later planning, approval, and coordinator layers should build on top of this model

## Resources

Runtime Model V1 introduces three first-class resources:

1. `Task`
2. `Service`
3. `Event`

These resources are always owned by a workspace.

## Ownership Model

Ownership rules:

- a workspace owns tasks
- a workspace owns services
- a workspace owns emitted events
- tasks and services never exist outside a workspace
- task and service state must live under the workspace state directory, not in the manifest

This keeps desired configuration separate from live runtime state.

## Manifest Model

The manifest remains the desired configuration layer.

It should eventually support declared tasks and services alongside packages and env.

Illustrative shape:

```json
{
  "schema_version": 1,
  "name": "crawlly",
  "project_path": "/Users/example/Documents/crawlly",
  "packages": [
    { "name": "go", "version": "1.25.4" }
  ],
  "tasks": [
    {
      "name": "test",
      "command": ["go", "test", "./..."],
      "cwd": "."
    }
  ],
  "services": [
    {
      "name": "redis",
      "command": ["redis-server"],
      "restart": "on-failure"
    }
  ]
}
```

V1 constraints:

- declared tasks are optional
- declared services are optional
- ad hoc tasks may exist without manifest entries
- services should be manifest-declared before they are long-lived

## Runtime State Model

Live state belongs under the workspace `state/` directory.

Examples:

- task records
- service records
- event log
- pid / process metadata
- timestamps
- exit codes
- runtime health state

The manifest must not become the mutable execution database.

## Task Resource

A task has two related shapes:

- `TaskSpec`: an optional manifest declaration describing what can be run
- `TaskRun`: one persisted execution record for a specific run

The task lifecycle work in V1 is centered on `TaskRun`.

A task run is a workspace-owned execution record that captures one command's lifecycle, state, logs, and result inside the workspace runtime.

Groot does not replace the OS process table. The OS still owns the process. Groot records, controls, and observes the execution through the workspace runtime.

### TaskSpec

`TaskSpec` lives in the manifest as desired configuration.

Minimum `TaskSpec` fields:

- `name`
- `command`
- `cwd`

### TaskRun

`TaskRun` lives under the workspace `state/` directory.

Minimum `TaskRun` fields:

- `id`
- `name`
- `workspace`
- `command`
- `args`
- `cwd`
- `created_at`
- `started_at`
- `finished_at`
- `exit_code`
- `state`
- `stdout_log`
- `stderr_log`

Task run states:

- `pending`
- `running`
- `succeeded`
- `failed`
- `cancelled`

Possible later states:

- `unknown`
- `orphaned`

Minimum task operations:

- `start`
- `stop`
- `status`
- `list`
- `logs`

V1 task rules:

- tasks execute inside the same Groot-managed workspace runtime as `exec`
- task runs inherit the workspace env, cwd resolution, toolchain `PATH`, and `GROOT_*` vars from the strict runtime builder
- ad hoc tasks may be started without a manifest declaration
- declared tasks should be startable by name
- finished tasks remain inspectable after completion
- logs are per task run, so starting the same task twice creates separate stdout/stderr log files
- tasks remain single execution units; they are not DAGs, workflows, or pipelines

## Service Resource

A service is a named long-running process owned by a workspace.

Services are not just long-running tasks. They also have desired lifecycle intent.

Minimum service fields:

- `name`
- `workspace`
- `command`
- `args`
- `cwd`
- `restart_policy`
- `state`
- `pid`
- `started_at`
- `stopped_at`
- `health`
- `stdout_log`
- `stderr_log`

Service states:

- `stopped`
- `starting`
- `running`
- `unhealthy`
- `failed`

Minimum service operations:

- `start`
- `stop`
- `restart`
- `status`
- `list`
- `logs`

V1 service rules:

- services should be declared in the manifest
- service runtime state belongs in workspace state
- health may be simple at first
- restart policy may be simple at first
- ports and dependencies are optional later extensions, not V1 requirements

## Event Resource

Events are structured runtime facts emitted by Groot as tasks and services change state.

Example event kinds:

- `workspace.opened`
- `workspace.activated`
- `task.started`
- `task.exited`
- `task.failed`
- `service.started`
- `service.stopped`
- `service.unhealthy`

Minimum event fields:

- `id`
- `workspace`
- `kind`
- `timestamp`
- `resource_type`
- `resource_id`
- `message`
- `payload`

Minimum event operations:

- `list`
- later `stream`

V1 event rules:

- event persistence is more important than live streaming
- event streaming can come after event creation and listing are reliable

## App-Layer API Shape

Illustrative app-layer surface:

- `TaskStart(workspace, spec)`
- `TaskStop(workspace, taskID)`
- `TaskStatus(workspace, taskID)`
- `TaskList(workspace)`
- `TaskLogs(workspace, taskID)`
- `ServiceStart(workspace, name)`
- `ServiceStop(workspace, name)`
- `ServiceRestart(workspace, name)`
- `ServiceStatus(workspace, name)`
- `ServiceList(workspace)`
- `ServiceLogs(workspace, name)`
- `EventList(workspace, cursor)`

These are app-layer primitives first.

CLI and MCP should be adapters over the same methods.

## CLI Mapping

Illustrative CLI shape:

```bash
groot task start <path> <name-or-command>
groot task status <path> <task-id>
groot task list <path>
groot task logs <path> <task-id>

groot service start <path> <name>
groot service stop <path> <name>
groot service status <path> <name>
groot service list <path>
groot service logs <path> <name>
```

This is illustrative, not final.

The important point is that CLI maps to structured runtime objects, not shell-only flows.

## MCP Mapping

MCP task tools:

- `task_start`
- `task_stop`
- `task_status`
- `task_list`
- `task_logs`

Illustrative future MCP service/event tools:

- `service_start`
- `service_stop`
- `service_restart`
- `service_status`
- `service_list`
- `service_logs`
- `event_list`

Likely MCP resources:

- task metadata
- service metadata
- event history
- log resources

## Non-Goals

Runtime Model V1 does not include:

- distributed task scheduling
- cross-workspace orchestration
- multi-agent semantics
- service dependency graphs
- advanced health-check DSLs
- machine-wide policy engine
- streaming-first protocols before durable event history exists

## Why This Matters

Without tasks, services, and events, Groot owns places.

With tasks, services, and events, Groot starts owning runtime lifecycle.

That is the bridge from:

- workspace runtime tool

to:

- operating substrate for agent-driven systems

## Immediate Next Step

Before implementation, Groot should use this document to drive one focused design pass:

1. finalize task and service state machines
2. decide manifest additions for declared tasks and services
3. decide state-dir record layout
4. define app-layer interfaces
5. only then map them to CLI and MCP
