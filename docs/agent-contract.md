# Groot Agent Contract v0

This document defines the first concrete contract between a thin external agent
and the Groot CLI.

This is a CLI + JSON contract, not an MCP contract.

## Scope

The v0 contract covers:

- project runtime inspection
- path-based setup/open
- strict runtime command execution
- shell entry for a human handoff

The v0 contract does not yet cover:

- structured `open` results
- structured `exec` results
- `ws inspect --json`
- project scaffolding for new repositories
- MCP tools/resources

## Current Thin Agent Entry Point

The first agent entrypoint now exists as:

```bash
groot-agent status <path>
groot-agent setup <path>
```

Current behavior:

- `status` calls the Groot CLI + JSON contract and prints the runtime status JSON
- `setup` calls `groot open <path> --setup` and then prints the resulting runtime status JSON

This is intentionally narrow. It proves the contract and orchestration path before
natural-language intent parsing is added.

## Principles

The agent should treat Groot as:

- the runtime authority
- the owner of workspace resolution and runtime state
- the system that decides whether a repo is already managed

The agent should not:

- infer workspace names from repo names on its own
- bypass Groot and manipulate manifests directly
- assume host toolchains are acceptable when Groot reports fallback risk

## Stable Commands

### 1. Inspect Runtime Ownership

```bash
groot status <path> --json
```

This is the primary read operation for the first agent.

Behavior:

- resolves an existing workspace by `project_path`
- auto-creates and binds a workspace on first use if needed
- prints one JSON object to stdout
- prints no human summary to stdout when `--json` is present

Current JSON shape:

```json
{
  "workspace_name": "the_grime_tcg",
  "project_path": "/Users/example/Documents/the_grime_tcg",
  "status": "runtime owned by Groot",
  "detected": [
    { "name": "go", "version": "1.25.4", "source": "backend/go.mod" },
    { "name": "node", "version": "", "source": "frontend/package.json" }
  ],
  "attached": [
    { "name": "go", "version": "1.25.4" },
    { "name": "node", "version": "25.8.1" }
  ],
  "installed": [
    { "name": "go", "version": "1.25.4" },
    { "name": "node", "version": "25.8.1" }
  ],
  "attached_uninstalled": [],
  "missing": []
}
```

Current `status` values:

- `no runtimes detected`
- `partial runtime ownership`
- `runtime declared but install pending`
- `runtime owned by Groot`

Agent expectations:

- if `missing` is non-empty, the runtime is not fully owned yet
- if `attached_uninstalled` is non-empty, install is still pending
- if `status` is `runtime owned by Groot`, the agent can safely move to `exec`

### 2. Open And Set Up A Project

```bash
groot open <path>
groot open <path> --attach-detected
groot open <path> --setup
```

Behavior:

- resolves or creates/binds the workspace
- opens the IDE in soft GUI mode
- may attach/install detected runtimes depending on flags

Current policy:

- default: warn only
- `--attach-detected`: attach only detected runtimes with concrete versions
- `--setup`: attach and install detected runtimes with concrete versions

Agent expectations:

- `open` is a side-effecting action, not the primary inspection primitive
- the agent should usually call `status --json` before and/or after `open`
- the agent should not parse human stderr from `open` as its main state source

### 3. Execute In Managed Runtime

```bash
groot exec <path> <cmd> [args...]
```

Behavior:

- resolves or creates/binds the workspace
- enforces runtime ownership warnings or strict-mode failure
- runs one command in the strict runtime

Agent expectations:

- use this as the main action primitive for non-GUI work
- prefer this over `ws exec` in the first agent because path-based UX is the product path
- do not use `open` when the goal is execution rather than IDE launch

### 4. Enter Strict Runtime Shell

```bash
groot enter <path>
```

Behavior:

- resolves or creates/binds the workspace
- opens an interactive strict runtime shell

Agent expectations:

- use this only for explicit human handoff
- this is not the normal machine-driven execution path

## Current Agent Flow

For an existing repo:

1. call `groot status <path> --json`
2. inspect:
   - `status`
   - `missing`
   - `attached_uninstalled`
3. if runtime is not owned:
   - call `groot open <path> --setup`
4. call `groot status <path> --json` again
5. if runtime is now owned:
   - call `groot exec <path> ...`

For a human IDE workflow:

1. call `groot open <path> --setup`
2. optionally call `groot status <path> --json` to verify ownership

## Error Handling

The agent should treat non-zero exit codes as command failure.

Recommended handling:

- `status --json` fails:
  - stop and surface the error
- `open --setup` fails:
  - stop and surface the error
  - do not assume partial setup succeeded cleanly
- `exec` fails:
  - surface both the command and the exit failure
  - do not reinterpret failure as a workspace-resolution issue unless Groot says so

## Current Gaps

The following should be added next before the contract expands much further:

1. `groot ws inspect <name> --json`
2. machine-readable `open` result output
3. machine-readable `exec` result output
4. project scaffolding/create flow for brand-new projects
5. MCP surface over the same runtime contract

## Design Rule

If a future agent feature needs state that is not available via:

- `status --json`
- path-based `open`
- path-based `exec`
- path-based `enter`

then Groot should expose a clearer machine-readable primitive rather than forcing the agent to parse human output.
