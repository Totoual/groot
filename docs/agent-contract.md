# Groot MCP Contract v0

This document defines the first MCP surface for Groot.

It replaces the earlier standalone `groot-agent` direction.

## Transport

Groot currently exposes MCP over stdio:

```bash
groot mcp
```

The server follows the MCP stdio transport:

- JSON-RPC messages over stdin/stdout
- newline-delimited messages
- stderr reserved for logs if needed

## Current Scope

The current MCP surface is intentionally small:

- inspect runtime ownership for a project path
- set up a workspace for a project path
- execute one command in the strict Groot runtime
- inspect the concrete workspace, manifest, and runtime state
- return the strict workspace environment as structured data
- attach explicit toolchains to a workspace
- install attached toolchains into Groot's managed store

It does not yet cover:

- workspace creation by explicit name
- direct bind/attach/install tools
- machine-readable workspace env output
- workspace resources like logs or manifest

## Available Tools

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
- `workdir`
- `env`

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

1. manifest and log resources
2. `workspace_bind` / `workspace_create` if agents need lower-level control
3. import/export surfaces
