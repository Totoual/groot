## 🪴 Groot

Groot is a workspace-first runtime layer for local development.

It gives each workspace its own home directory and manifest, while keeping shared runtime state under a single `~/.groot` root. The current version is focused on workspace lifecycle, manifest management, project binding, toolchain installation, and shell activation.

## Product Goal

Groot is meant to provide a reproducible project environment that can be created, opened, exported, and recreated on other machines without polluting the user's normal machine profile.

The intended split is:

- Project runtime state should belong to Groot.
- Project-scoped agent state should belong to Groot.
- GUI IDE identity should remain compatible with the user's normal machine profile.

The agent-facing direction is for Groot to expose the same runtime core through a structured interface, likely MCP, instead of making agents depend on ad hoc shell scripting.

## Current Scope

- Initialize a Groot root under `~/.groot`
- Enter a project path by resolving or auto-creating the matching workspace
- Execute one-off commands against a project path by resolving or auto-creating the matching workspace
- Open a project path by resolving or auto-creating the matching workspace
- Create and delete workspaces
- Bind a workspace to an existing project directory
- Clear a workspace project binding
- Attach toolchain requirements to a workspace manifest
- Install attached toolchains into the shared Groot toolchain root
- Garbage collect unreferenced toolchains from the shared store
- Open a workspace shell with workspace-scoped `HOME` and XDG directories
- Run one-off commands inside the workspace runtime
- Open a workspace in an IDE with a softer GUI runtime
- Print shell exports for the resolved workspace runtime
- Provide a stable base for a future agent-facing interface on top of the same runtime core

## Principles

- All Groot state lives under one root directory: `~/.groot`
- Each workspace has its own isolated runtime state
- Source code stays in its normal location outside the Groot runtime root
- Toolchain requirements are declared in `manifest.json`
- Workspaces should be recreatable on other machines
- Workspaces should be exportable without depending on the user's global machine setup
- Toolchain installation is moving toward a shared global store, not per-workspace duplication

## State Model

Groot needs to treat different kinds of state differently.

### 1. Project Runtime State

This should be isolated and managed by Groot.

- toolchains
- workspace env
- project-specific caches where needed
- logs
- services and runtime state

### 2. Agent Workspace State

This should also be isolated and managed by Groot.

- project-specific memory
- conversation history
- indexed project knowledge
- execution history
- generated artifacts and plans

### 3. GUI IDE Identity

This should usually remain global so editors still behave normally.

- editor preferences
- extensions
- keychain/login integration
- GUI app settings and identity

The long-term goal is strict isolation for project runtime and agent state, without breaking normal IDE behavior.

That likely means:

- CLI for humans
- MCP for agents
- one shared Groot runtime underneath both

## Runtime Layout

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

## Commands

```bash
groot enter <path>
groot exec <path> <cmd> [args...]
groot init
groot open <path>
groot shell-hook
groot shell-hook install

groot ws attach <name> <tool@version> [tool@version...]
groot ws bind <name> <path>
groot ws create <name>
groot ws delete <name>
groot ws env <name>
groot ws exec <name> <cmd> [args...]
groot ws gc
groot ws install <name>
groot ws open <name> [--ide code|cursor|zed|...]
groot ws shell <name>
groot ws unbind <name>
```

## Quick Open

```bash
groot init
groot open ~/Documents/crawlly
```

`groot open <path>` resolves the bound workspace for that repo path and, on first open, creates and binds a workspace automatically before launching the IDE.

## Path-Based Shell And Exec

```bash
groot enter ~/Documents/crawlly
groot exec ~/Documents/crawlly git status
```

These commands resolve the workspace by `project_path` first and create/bind one automatically on first use when needed.

## Manual Workspace Flow

```bash
groot init
groot ws create crawlly
groot ws bind crawlly ~/Documents/crawlly
groot ws attach crawlly go@1.25 node@22
groot ws install crawlly
groot ws open crawlly --ide code
groot ws shell crawlly
```

## Shell Hook

To make integrated terminals automatically re-enter the strict Groot runtime after `ws open`, install the shell hook into your shell rc file.

Recommended:

```bash
groot shell-hook install
```

This currently supports `zsh` and `bash`, and installs a managed block into the detected rc file:

```bash
# >>> groot shell hook >>>
eval "$(groot shell-hook)"
# <<< groot shell hook <<<
```

If you prefer to manage your shell config yourself, add the hook line near the end of your shell config manually.

For `zsh`:

```bash
eval "$(groot shell-hook)"
```

For `bash`:

```bash
eval "$(groot shell-hook)"
```

Behavior:

- when `GROOT_WORKSPACE` is not set, the hook prints nothing and does nothing
- when `GROOT_WORKSPACE` is set, the hook reapplies the strict workspace runtime for the shell
- this keeps `ws open` editor-agnostic while letting integrated terminals use Groot-managed toolchain precedence automatically
- `groot shell-hook install` is idempotent and will not add the managed block twice

## Supported Toolchains

Groot currently supports these toolchains:

- `bun`
- `deno`
- `go`
- `php`
- `node`
- `java`
- `python`
- `rust`

Current install behavior:

- `bun` downloads the official prebuilt ZIP archive for the current OS and architecture
- `deno` downloads the official prebuilt ZIP archive for the current OS and architecture
- `go` downloads the official prebuilt archive for the current OS and architecture
- `php` downloads the official source tarball and builds it locally
- `node` downloads the official prebuilt archive for the current OS and architecture
- `java` resolves the latest matching Temurin JDK for the requested feature version
- `python` downloads the official source tarball and builds it locally
- `rust` bootstraps through `rustup-init` inside the workspace-managed toolchain root

## Version Semantics

Version values are stored in the manifest and interpreted per toolchain.

- `bun@1.3.10` means an exact Bun release
- `deno@2.7.5` means an exact Deno release
- `go@1.26.1` means an exact Go release
- `php@8.5.4` means an exact PHP source release
- `node@25.8.1` means an exact Node release
- `java@21` means the latest available Temurin JDK for feature version `21`
- `python@3.14` means the latest available Python `3.14.x` source release
- `python@3.14.0` means an exact Python source release
- `rust@stable` means the Rust stable channel via `rustup`

Examples:

```bash
groot ws attach frontend bun@1.3.10 deno@2.7.5
groot ws attach backend go@1.26.1 node@25.8.1
groot ws attach api java@21
groot ws attach legacy php@8.5.4
groot ws attach scripts python@3.14
groot ws attach scripts python@3.14.0
groot ws attach systems rust@stable
```

## Workspace Manifest

Each workspace stores its desired state in `manifest.json`.

Example:

```json
{
  "schema_version": 1,
  "created_at": "2026-03-04T15:43:56.144288Z",
  "name": "crawlly",
  "project_path": "/Users/example/Documents/crawlly",
  "packages": [
    {
      "name": "go",
      "version": "1.25"
    },
    {
      "name": "node",
      "version": "22"
    }
  ],
  "services": [],
  "env": {}
}
```

## Current Behavior Notes

- `ws attach` validates `name@version` specs, rejects unsupported toolchains, and updates existing package entries by name
- `services` exists in the schema but is not actively used yet
- `ws bind` stores the project location in `project_path`
- `ws unbind` clears `project_path` without deleting the workspace runtime
- `open` resolves a workspace from a project path and auto-creates/binds one on first open when needed
- `enter` resolves a workspace from a project path and opens the strict workspace shell
- `exec` resolves a workspace from a project path and runs one strict-runtime command
- `ws install` downloads and installs attached toolchains into the shared Groot toolchain root
- `ws gc` removes unreferenced toolchain versions from the shared Groot toolchain root
- `ws shell` ensures attached toolchains are installed, prepends their `bin` directories to `PATH`, and sets toolchain-specific env vars when needed
- `ws shell` starts in the bound `project_path` when present, otherwise in the workspace root under `~/.groot/workspaces/<name>`
- `ws env` prints shell exports for the resolved workspace runtime and includes `GROOT_WORKDIR` for the chosen working directory
- workspace runtimes export `GROOT_HOME`, so nested `groot` commands keep using the shared Groot root instead of falling back to the workspace `HOME`
- `ws exec` runs a specific command in the same workspace environment and working directory resolution used by `ws shell`; this is the right primitive for automation and future agents
- `ws open` launches an IDE or GUI program in a softer runtime that keeps the project cwd, toolchain `PATH`, and `GROOT_*` vars while preserving the user's normal `HOME`
- `ws open` is for human GUI workflows; it is not the primary execution primitive for future agents
- `shell-hook` turns a shell with `GROOT_WORKSPACE` set back into the strict workspace runtime, which is how integrated terminals can become fully Groot-managed without editor-specific settings
- `ws open` defaults to `GROOT_IDE`, then `VISUAL`, then `EDITOR`, and finally `code` when no IDE is specified
- `ws env` omits interactive shell prompt variables such as `PS1` and `PROMPT`
- host `PATH` is filtered before reuse, so user-home shims and editor-specific entries are dropped while system paths remain available
- `ws open` keeps the host `PATH` and `HOME` so GUI IDEs can behave more like normal desktop apps
- archive extraction rejects path traversal and staged archive installs replace the final toolchain dir only after a successful extract
- GUI IDEs launched with full workspace `HOME` isolation may still have integration issues such as keychain/profile friction
- `php` and `python` installation are slower than the other supported toolchains because they are built from source

## Architecture Overview

```mermaid
flowchart TD
    USER["User"] --> CLI["groot CLI"]
    CLI --> APP["App Runtime Core"]

    APP --> WS["Workspace Folder"]
    APP --> MANIFEST["manifest.json"]
    APP --> SHELL["Shell Process"]
    APP --> STORE["Shared Toolchain Store"]

    WS --> HOME["home/"]
    WS --> STATE["state/"]
    WS --> LOGS["logs/"]
    WS --> PROJECT_PATH["project_path"]

    APP --> ROOT["groot root"]
    ROOT --> TOOLCHAINS["toolchains/"]
    ROOT --> BIN["bin/"]
    ROOT --> CACHE["cache/"]
    ROOT --> STORE

    PROJECT_PATH --> PROJECT["project directory outside root"]
```
