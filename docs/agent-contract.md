# Groot MCP Contract v0

This contract corresponds to the current Phase 1.5 surface:

- Groot runtime first
- MCP as a structured control plane over that runtime
- planning and intent compilation coming later as Phase 2

This document defines the first MCP surface for Groot.

It replaces the earlier standalone `groot-agent` direction.

## Transport

Groot currently exposes MCP over stdio:

```bash
groot mcp
```

Recommended normal flow:

```bash
groot mcp
```

Then activate one project for the session through `workspace_activate`.

Optional hard-lock startup scope:

```bash
groot mcp --workspace the_grime_tcg
groot mcp --project /Users/example/Documents/crawlly --project /Users/example/Documents/the_grime_tcg
```

The server follows the MCP stdio transport:

- JSON-RPC messages over stdin/stdout
- newline-delimited messages
- stderr reserved for logs if needed

## Scope

MCP scope is explicit:

- no scope flags: the server starts unscoped
- `workspace_activate`: sets the live MCP session scope to one project path or bound workspace
- in a normal unscoped session, `workspace_activate` can switch the active project later
- `--project <path>`: only that normalized project path is allowed
- `--workspace <name>`: resolves the workspace's bound `project_path` and scopes MCP to it
- repeated `--project` or `--workspace` flags create an explicit multi-project allowlist

When a tool call targets a project path outside the active or configured MCP scope, Groot returns an error instead of running the action.

## Current Scope

The current MCP surface is intentionally small:

- inspect runtime ownership for a project path
- set up a workspace for a project path
- execute one command in the strict Groot runtime
- inspect the concrete workspace, manifest, and runtime state
- return the strict workspace environment as structured data
- attach explicit toolchains to a workspace
- install attached toolchains into Groot's managed store
- export the current workspace contract as portable structured data
- import that exported contract onto an existing local repo path

It does not yet cover:

- workspace creation by explicit name
- logs or other long-lived execution history resources
- manifest planning / preview / approval flows

## Available Tools

### `workspace_activate`

Activate one project path or bound workspace as the live MCP session scope.

Input:

```json
{
  "path": "/Users/example/Documents/the_grime_tcg"
}
```

or:

```json
{
  "workspace": "the_grime_tcg"
}
```

Structured result:

- `active_project`
- `workspace_name` when the activated target maps to a bound workspace

### `workspace_status`

Resolve or create a workspace from a project path and return runtime ownership state.

Input:

```json
{
  "path": "/Users/example/Documents/the_grime_tcg"
}
```

Structured result:

- `created`
- `status.workspace_name`
- `status.project_path`
- `status.status`
- `status.detected`
- `status.attached`
- `status.installed`
- `status.attached_uninstalled`
- `status.missing`

### `workspace_setup`

Resolve or create a workspace from a project path and move it toward runtime ownership.

Input:

```json
{
  "path": "/Users/example/Documents/the_grime_tcg",
  "attach_detected": true,
  "install_detected": true
}
```

Defaults:

- `attach_detected=true`
- `install_detected=true`

Structured result:

- `created`
- `plan`
- `status`

The `plan` includes:

- `detected`
- `attached`
- `installed`
- `skipped`
- `missing`
- `attach_requested`
- `install_requested`

### `workspace_exec`

Resolve or create a workspace from a project path and run one command in the strict Groot runtime.

Input:

```json
{
  "path": "/Users/example/Documents/the_grime_tcg",
  "command": "go",
  "args": ["test", "./..."]
}
```

Structured result:

- `created`
- `workspace`
- `command`
- `args`
- `workdir`
- `stdout`
- `stderr`
- `exit_code`
- `warnings`
- `strict_mode`

### `workspace_inspect`

Resolve or create a workspace from a project path and return the concrete workspace state.

Input:

```json
{
  "path": "/Users/example/Documents/the_grime_tcg"
}
```

Structured result:

- `created`
- `inspect.workspace_name`
- `inspect.workspace_dir`
- `inspect.manifest_path`
- `inspect.home_dir`
- `inspect.state_dir`
- `inspect.logs_dir`
- `inspect.manifest`
- `inspect.runtime`

### `workspace_env`

Resolve or create a workspace from a project path and return the strict runtime environment as key/value pairs.

Input:

```json
{
  "path": "/Users/example/Documents/the_grime_tcg"
}
```

Structured result:

- `created`
- `workspace_name`
- `workdir`
- `env`

The `env` payload is intentionally filtered to the stable runtime keys Groot owns or injects for the workspace, such as `GROOT_*`, `HOME`, `XDG_*`, `PATH`, locale values, and toolchain-specific homes like `JAVA_HOME` or `CARGO_HOME`. It should not be treated as a dump of the full host session environment.

### `workspace_attach`

Resolve or create a workspace from a project path and attach explicit toolchain specs.

Input:

```json
{
  "path": "/Users/example/Documents/the_grime_tcg",
  "toolchains": ["go@1.25.4", "node@25.8.1"]
}
```

Structured result:

- `created`
- `attached`
- `status`

### `workspace_install`

Resolve or create a workspace from a project path and install all attached toolchains.

Input:

```json
{
  "path": "/Users/example/Documents/the_grime_tcg"
}
```

Structured result:

- `created`
- `installed`
- `status`

### `workspace_export`

Export the existing workspace contract for a project path as portable structured data.

Input:

```json
{
  "path": "/Users/example/Documents/the_grime_tcg"
}
```

Structured result:

- `export.schema_version`
- `export.exported_at`
- `export.workspace.name`
- `export.workspace.project_path`
- `export.workspace.manifest`
- `export.workspace.runtime`

### `workspace_import`

Import a portable workspace contract for an existing project path.

Input:

```json
{
  "path": "/Users/example/Documents/the_grime_tcg",
  "export": {
    "schema_version": 1,
    "workspace": {
      "name": "the_grime_tcg"
    }
  },
  "workspace_name": "the_grime_tcg-copy",
  "install_attached": false
}
```

Structured result:

- `created`
- `workspace_name`
- `project_path`
- `status`

If the exported workspace name already exists on the target machine, pass `workspace_name` to import the contract under a different local workspace identity.

Import restores the Groot workspace contract, not the source repository contents. If the target path does not contain the original project files yet, Groot can still report attached and installed runtimes from the imported contract even though runtime detection at that path is still empty.

## Available Resources

When a project is active or startup-scoped, Groot exposes read-only MCP resources for that workspace.

### Manifest Resource

URI shape:

```text
groot://workspace/<workspace-name>/manifest
```

Content:

- the raw `manifest.json` content as JSON

### Metadata Resource

URI shape:

```text
groot://workspace/<workspace-name>/metadata
```

Content:

- workspace name
- bound project path
- workspace dir
- manifest path
- home/state/log dirs
- runtime ownership snapshot

Resources are scoped the same way as tools. In an unscoped MCP session, agents should activate a project first so these resources become available.

## Design Rules

External agents should treat Groot as:

- the runtime authority
- the workspace resolver
- the owner of runtime detection and setup

External agents should not:

- guess workspace names on their own
- edit Groot manifests directly
- assume host fallback is acceptable when Groot reports otherwise

## Near-Term Expansion

The next MCP additions should likely be:

1. logs and other execution-history resources
2. `workspace_bind` / `workspace_create` if agents need lower-level control
3. richer import conflict and recovery flows
