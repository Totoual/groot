## 🪴 Groot

Groot is a workspace-first runtime layer for local development.

It treats the workspace as the primary unit of runtime ownership:

- isolated workspace `HOME`
- attached toolchains in a shared Groot-managed store
- path-first project resolution
- tracked task and service execution
- durable runtime events
- one runtime core exposed through CLI and MCP

Groot does not replace the host OS, the shell, or the IDE. It adds discipline around project runtime state.

## Product Direction

Groot is being built in deliberate phases:

- Phase 1: workspace-first runtime for local development
- Phase 1.5: MCP control plane over the same runtime
- Phase 2: planning and intent surfaces on top of external agents
- Phase 3: deeper GOS-style runtime evolution only if the earlier phases prove daily value

The architectural bridge toward that longer-term direction is:

- workspace ownership
- task ownership
- service ownership
- event ownership

That runtime direction is defined in [docs/runtime-model-v1.md](/Users/aristotelistriantafyllidis/Documents/groot/docs/runtime-model-v1.md).

## Current Scope

Today Groot can:

- resolve or auto-create a workspace from a project path
- open, enter, exec, export, import, and inspect project workspaces
- attach and install toolchains in a shared Groot store
- run tracked task executions with persisted logs and status
- run manifest-declared services with current logs and state
- persist runtime lifecycle events for tasks and services
- expose the same runtime core through MCP

## Install

Install the `groot` binary with Go:

```bash
go install ./cmd/groot
```

Make sure your Go binary install directory is on `PATH`.

Then initialize Groot and install the shell hook:

```bash
groot init
groot shell-hook install
```

That gives you:

- the shared Groot root under `~/.groot`
- the `groot` CLI on your shell path
- automatic terminal re-entry into the strict runtime for supported editor terminals

## Quick Start

```bash
groot init
groot open ~/Documents/crawlly --setup
```

`groot open <path>` resolves the workspace for that repo path and, on first open, creates and binds one automatically before launching the IDE.

On first open, Groot also scans the repo for likely runtimes such as Go, Node, Python, Rust, Bun, Deno, PHP, and Java.

Current first-open modes:

- default: warn and suggest attach/install
- `--attach-detected`: attach detected runtimes with concrete versions
- `--install-detected`, `--setup`, or `--setup-detected`: attach and install detected runtimes
- `GROOT_STRICT_RUNTIME=1`: fail instead of warning when detected runtimes are still undeclared

Examples:

```bash
groot open ~/Documents/crawlly
groot open ~/Documents/crawlly --attach-detected
groot open ~/Documents/crawlly --setup
```

## Daily Commands

Path-first workflows:

```bash
groot enter ~/Documents/crawlly
groot exec ~/Documents/crawlly git status
groot status ~/Documents/crawlly
groot status ~/Documents/crawlly --json

groot task start ~/Documents/crawlly --name tests go test ./...
groot task list ~/Documents/crawlly
groot task status ~/Documents/crawlly <task-id>
groot task logs ~/Documents/crawlly <task-id>

groot service start ~/Documents/crawlly api
groot service list ~/Documents/crawlly
groot service status ~/Documents/crawlly api
groot service logs ~/Documents/crawlly api
groot service stop ~/Documents/crawlly api

groot event list ~/Documents/crawlly

groot export ~/Documents/crawlly
groot import crawlly-export.json --project-path ~/Documents/crawlly-copy --workspace-name crawlly-copy
```

What these mean:

- `open` is the main human GUI shortcut
- `enter` and `exec` use the strict workspace runtime
- `task ...` manages tracked execution records and per-run logs
- `service ...` manages manifest-declared long-running workspace-owned services
- `event ...` lists persisted runtime lifecycle events such as `task.started`, `task.exited`, `service.started`, and `service.failed`
- `status` shows detected, attached, installed, and host-fallback runtime state for the project path
- `export` and `import` move the workspace contract, not the repository contents
- `ws ...` remains the lower-level explicit workspace surface

## MCP

Groot exposes a testable MCP server over stdio:

```bash
groot mcp
```

Recommended normal flow:

```bash
groot mcp
```

Then let the agent activate one project for the session with `workspace_activate`.

Optional hard-lock startup scope:

```bash
groot mcp --workspace crawlly
groot mcp --project ~/Documents/crawlly --project ~/Documents/the_grime_tcg
```

Current MCP tools:

- `workspace_activate`
- `workspace_status`
- `workspace_setup`
- `workspace_exec`
- `workspace_inspect`
- `workspace_env`
- `workspace_attach`
- `workspace_install`
- `workspace_export`
- `workspace_import`
- `task_start`
- `task_status`
- `task_list`
- `task_logs`
- `task_stop`
- `service_start`
- `service_status`
- `service_list`
- `service_logs`
- `service_stop`
- `event_list`

Current MCP resources:

- workspace manifest
- workspace metadata and runtime snapshot

Detailed MCP semantics and tool contracts live in [docs/agent-contract.md](/Users/aristotelistriantafyllidis/Documents/groot/docs/agent-contract.md).

## Workspace Commands

Use `groot ws ...` when you want explicit workspace-by-name control:

```bash
groot ws create crawlly
groot ws bind crawlly ~/Documents/crawlly
groot ws attach crawlly go@1.26 node@25.8.1
groot ws install crawlly
groot ws open crawlly --ide code
groot ws shell crawlly
```

Available subcommands:

- `attach`
- `bind`
- `create`
- `delete`
- `env`
- `exec`
- `gc`
- `install`
- `open`
- `shell`
- `unbind`

## Supported Toolchains

Groot currently supports:

- `bun`
- `deno`
- `go`
- `java`
- `node`
- `php`
- `python`
- `rust`

See [docs/reference.md](/Users/aristotelistriantafyllidis/Documents/groot/docs/reference.md) for toolchain install behavior and version semantics.

## Workspace Layout

```bash
~/.groot/
  bin/
  cache/
  store/
  toolchains/
  workspaces/
    crawlly/
      manifest.json
      home/
      state/
      logs/
```

## Docs

- [docs/runtime-model-v1.md](/Users/aristotelistriantafyllidis/Documents/groot/docs/runtime-model-v1.md)
  Runtime ownership model for tasks, services, and events.
- [docs/agent.md](/Users/aristotelistriantafyllidis/Documents/groot/docs/agent.md)
  Product direction for external-agent integration through MCP.
- [docs/agent-contract.md](/Users/aristotelistriantafyllidis/Documents/groot/docs/agent-contract.md)
  Current MCP tool and resource contract.
- [docs/reference.md](/Users/aristotelistriantafyllidis/Documents/groot/docs/reference.md)
  CLI reference, shell hook details, manifest shape, supported toolchains, and runtime behavior notes.
