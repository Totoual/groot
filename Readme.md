# Groot

Groot is a workspace-first runtime for local development.

It gives one project a stable local runtime surface:

- a workspace-scoped home and state directory
- attached toolchains managed by Groot
- tracked tasks and services
- deterministic local index, vault, and context data
- the same runtime exposed through CLI and MCP

Groot is not an agent, a planner, or a hidden memory system. It is the local substrate that people and MCP clients can share.

## Goal

Groot tries to make project runtime state:

- local-first
- inspectable
- reproducible
- workspace-scoped

The core idea is simple: the project workspace is the unit of runtime ownership.

## What It Does Today

Groot currently supports:

- workspace resolution and binding from a project path
- toolchain attach/install flows
- tracked task execution with logs and status
- managed workspace services
- persisted runtime events
- deterministic workspace indexing
- deterministic workspace vault and task-progress handoff data
- compact context packs
- the same runtime surface over MCP

## Install

```bash
go install ./cmd/groot
groot init
groot shell-hook install
```

That sets up Groot under `~/.groot` and installs the shell integration.

## Quick Start

```bash
groot open ~/Documents/crawlly --setup
```

Daily usage:

```bash
groot sync crawlly
groot status crawlly
groot resume crawlly
groot search crawlly "vault edge"
```

Advanced usage stays available through:

- `groot index ...`
- `groot vault ...`
- `groot context ...`
- `groot mcp`

For the full CLI reference, use [docs/reference.md](docs/reference.md).

## MCP

Groot exposes the same runtime over MCP. Docker packaging is not included in
this repository; run the MCP server directly from the Groot binary.

```bash
groot mcp
```

For VS Code over HTTP, start the MCP server with the HTTP transport:

```bash
groot mcp --http --listen 127.0.0.1:8080 --endpoint /mcp
```

Then connect VS Code to:

```text
http://127.0.0.1:8080/mcp
```

To configure Groot globally in VS Code:

1. Open the Command Palette.
2. Run `MCP: Open User Configuration`.
3. Add Groot to the global `mcp.json`:

```json
{
  "servers": {
    "groot": {
      "type": "http",
      "url": "http://127.0.0.1:8080/mcp"
    }
  }
}
```

This makes Groot available across VS Code workspaces. Keep the Groot MCP
process running while VS Code is using the MCP server.

Recommended flow:

- start the MCP server
- activate a project with `workspace_activate`
- use the same runtime operations through structured tools

For MCP tool contracts and schemas, use [docs/agent-contract.md](docs/agent-contract.md).

## Docs

- [docs/runtime-model-v1.md](docs/runtime-model-v1.md)
  Runtime model and product direction.
- [docs/reference.md](docs/reference.md)
  CLI reference, shell hook details, manifests, toolchains, and runtime behavior.
- [docs/agent-contract.md](docs/agent-contract.md)
  MCP contract and structured tool/resource behavior.
- [docs/agent.md](docs/agent.md)
  Agent-facing product direction.
