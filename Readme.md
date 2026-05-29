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
groot status crawlly
groot resume crawlly
groot search crawlly "vault edge"
```

Advanced usage stays available through:

- `groot index ...`
- `groot vault ...`
- `groot context ...`
- `groot mcp`

For the full CLI reference, use [docs/reference.md](/Users/aristotelistriantafyllidis/Documents/groot/docs/reference.md).

## MCP

Groot also exposes the same runtime over stdio MCP:

```bash
groot mcp
```

Recommended flow:

- start the MCP server
- activate a project with `workspace_activate`
- use the same runtime operations through structured tools

For MCP tool contracts and schemas, use [docs/agent-contract.md](/Users/aristotelistriantafyllidis/Documents/groot/docs/agent-contract.md).

## Docs

- [docs/runtime-model-v1.md](/Users/aristotelistriantafyllidis/Documents/groot/docs/runtime-model-v1.md)
  Runtime model and product direction.
- [docs/reference.md](/Users/aristotelistriantafyllidis/Documents/groot/docs/reference.md)
  CLI reference, shell hook details, manifests, toolchains, and runtime behavior.
- [docs/agent-contract.md](/Users/aristotelistriantafyllidis/Documents/groot/docs/agent-contract.md)
  MCP contract and structured tool/resource behavior.
- [docs/agent.md](/Users/aristotelistriantafyllidis/Documents/groot/docs/agent.md)
  Agent-facing product direction.
