# Groot Agent Direction

This document defines the near-term agent approach for Groot.

## Goal

Groot should support a user experience like:

```bash
groot "create me a project called X with golang and node"
```

or:

```bash
groot agent "open this repo and set it up"
```

The agent should understand intent and then drive Groot's runtime primitives.

## Decision

The first Groot agent should be:

- separate from the Groot core
- in the same repository
- implemented as a thin local orchestration layer
- built on top of Groot CLI + JSON first

The initial repo shape should move toward:

- `cmd/groot`
- `cmd/groot-agent`
- `internal/...`
- `agent/`

This keeps Groot itself focused on:

- workspace/runtime control
- detection
- attach/install/setup
- exec/open/status
- later MCP and machine-readable surfaces

And it keeps the agent focused on:

- intent parsing
- planning
- sequencing Groot commands
- reporting user-facing progress

The initial skeleton now exists in:

- [main.go](/Users/aristotelistriantafyllidis/Documents/groot/cmd/groot-agent/main.go)
- [internal/agent](/Users/aristotelistriantafyllidis/Documents/groot/internal/agent)

The first thin entrypoint now supports:

- `groot-agent status <path>`
- `groot-agent setup <path>`

## Non-Goals

The first agent should not try to be:

- a general-purpose agent framework
- a distributed queue/RPC system
- tightly coupled to a specific external framework
- a replacement for Groot's core runtime logic

## Near-Term Transport

Before MCP, the first agent should call Groot through CLI + JSON.

The concrete v0 command contract is documented in [docs/agent-contract.md](/Users/aristotelistriantafyllidis/Documents/groot/docs/agent-contract.md).

The first structured command already available is:

```bash
groot status <path> --json
```

The first action commands the agent can rely on are:

```bash
groot open <path> --setup
groot open <path> --attach-detected
groot exec <path> <cmd> [args...]
groot enter <path>
```

## First Supported Intents

The first agent should only support a small set of intents:

1. Create a new project with requested runtimes
   Example:
   `create me a project called the_grime_tcg with golang and node`

2. Open and set up an existing repo
   Example:
   `open this repo and set it up`

3. Inspect runtime/workspace status
   Example:
   `what is the status of this project`

4. Run a command in the managed runtime
   Example:
   `run go test in this project`

## Expected Agent Flow

For an intent like:

```bash
create me a project called X with golang and node
```

the agent should eventually:

1. decide the project path
2. create the folder structure
3. call Groot to create/resolve the workspace
4. attach toolchains
5. install toolchains
6. verify runtime ownership
7. leave the project ready to open

## Why This Approach

This is the smallest path that keeps the product coherent:

- Groot remains a user-facing product
- Groot also remains an agent substrate
- the agent can ship with Groot without owning Groot's core design
- MCP can be added later without rewriting the product split

## Next Technical Step

The next implementation step after this document should be:

- implement the first agent layer on top of the v0 CLI + JSON contract
