# Groot Task List

This file tracks the next implementation milestones while Groot is still in active design and iteration.

## Product Direction

Groot is not meant to stop at shell wrappers, workspace naming, or multi-language tool installs.

The product target is:

- open a repo
- get the right runtime
- get the right toolchains
- keep project runtime and agent state out of the user's normal machine profile
- keep IDEs usable while doing it

If Groot does not make that experience materially better than normal local development, it is not yet doing its job.

## Next Product Milestone: First Open Owns The Runtime

This is the next milestone that should define whether Groot is becoming a real product.

When a user runs:

```bash
groot open ~/dev/some-project
```

Groot should move toward:

- resolving or creating the workspace automatically
- creating the manifest automatically
- detecting likely runtimes from the repo
- attaching or at least suggesting the right toolchains
- making it obvious when the workspace is still using host toolchains
- making IDE and terminal behavior feel clearly Groot-managed

### Immediate Next Tasks

- [x] Detect likely runtimes on first open, for example Go, Node, Python, Rust, Bun, Deno, PHP, Java.
- [x] Define first-open behavior when no toolchains are attached yet:
  - warn only
  - suggest attach/install
  - or auto-attach common runtimes
- [x] Surface when a workspace is still using host toolchains instead of Groot-managed ones.
- [x] Add a stricter runtime mode or warning mode for undeclared toolchains.
- [x] Decide whether `groot open <path>` should only open, or whether it should also offer/setup runtime ownership on first use.
- [x] Add an opt-in first-open setup path that can auto-attach and install detected runtimes with flags.
- [x] Add a path-based status/inspect view so users can see detected, attached, installed, and host-fallback runtime state.
- [x] Make the first-open experience feel product-shaped, not like a thin alias over lower-level commands.

## Layer 1: Core Runtime

This is the stable control plane Groot should expose for shells, IDE launchers, and future agents.

### Workspace Model

- [x] Make `delete` the canonical workspace removal command.
- [x] Add `project_path` to the workspace manifest.
- [x] Add `groot ws bind <workspace> <path>`.
- [x] Make `groot ws shell <workspace>` use `project_path` as the working directory when bound.
- [x] Extract a reusable workspace environment builder for shell and exec flows.
- [x] Add `ExecWorkspace(name, command, args)` in the app layer.
- [x] Add `groot ws exec <workspace> <cmd> [args...]`.
- [x] Add `groot ws env <workspace>` to expose the runtime environment for IDE launching.

### Manifest And CLI Hardening

- [x] Validate `name@version` parsing in `ws attach`.
- [x] Reject malformed or incomplete tool specs with clear errors.
- [x] Update existing manifest entries instead of duplicating toolchains.
- [x] Write manifests atomically via temp file + rename.
- [x] Keep CLI help and README command examples aligned.

### Installer Vertical Slice

- [x] Keep `groot ws install <workspace>` as the install/apply entrypoint.
- [x] Finalize the installer interface owned by the app runtime.
- [x] Complete the Go installer vertical slice end-to-end.
- [x] Verify downloaded archives before extraction.
- [x] Harden extraction against path traversal and partial installs.
- [x] Reuse installed toolchains in both `ws shell` and `ws exec`.

### Tests And Cleanup

- [x] Add tests for manifest load/save behavior.
- [x] Add tests for bind/unbind behavior.
- [x] Add tests for workspace env generation.
- [x] Add tests for attach dedupe/update logic.
- [x] Add tests for installer path helpers.
- [x] Add tests for checksum verification.
- [x] Add shared toolchain garbage collection after the runtime flow is stable.

## Layer 2: Agent-Driven UX

This layer should make Groot feel simple for normal developers by letting an agent drive the lower-level Groot primitives.

- [ ] Decide whether MCP is the primary agent adapter for Groot.
- [x] Add workspace lookup by `project_path`.
- [ ] Define the agent-to-Groot contract: create, bind, attach, install, exec, inspect.
- [ ] Decide the primary agent entrypoint:
  - `groot agent "<intent>"`
  - `groot "<intent>"`
  - `groot open <path> --agent`
- [ ] Define the first supported agent intents, for example:
  - "start crawlly with go@1.25 and node@25"
  - "open this repo in a clean workspace"
  - "set up this project for me"
- [ ] Add a path-based setup flow so the agent can create or resolve a workspace from a repo path and move toward runtime ownership on first open.
- [x] Add a path-based open/enter flow for non-agent fallback usage.
- [ ] Ensure the agent can auto-create or auto-bind a workspace when a repo is first seen.
- [ ] Document the normal user workflow as agent-first, with `groot ws ...` kept as advanced/runtime commands.

## Layer 2.5: IDE Strategy

This layer decides how Groot keeps project isolation without breaking normal IDE behavior.

- [ ] Define the boundary between project runtime state, agent workspace state, and GUI IDE identity.
- [x] Prioritize a reliable IDE launch story as the next product milestone after Layer 1.
- [ ] Decide the default IDE launch policy:
  - strict workspace mode
  - soft IDE mode
  - editor-specific behavior
- [x] Implement a soft IDE mode that preserves project cwd, toolchain PATH, and `GROOT_*` vars without forcing full GUI `HOME` isolation.
- [x] Add a dedicated IDE launcher command, for example `groot ws open <name> --ide code`.
- [x] Add a generic shell hook so terminals launched from a soft-opened IDE can automatically re-enter the strict Groot runtime.
- [x] Document shell-hook setup so terminal activation does not depend on IDE-specific settings.
- [ ] Verify first-launch behavior for a brand-new workspace in VS Code before treating IDE support as working.
- [ ] Define which environment variables should be preserved for GUI IDE launches and which should stay isolated.
- [ ] Ensure first-open IDE terminals clearly reflect Groot-managed toolchains when those toolchains are attached and installed.
- [ ] Decide where project-scoped agent memory and conversation state should live inside Groot.
- [ ] Ensure IDEs can open the bound project path without forcing a separate GUI app identity.
- [ ] Document the tradeoff between terminal isolation and GUI IDE compatibility.

## Layer 3: Agent Foundation

This layer makes Groot usable by a top-level agent without inventing a separate runtime path.

- [ ] Keep `ws exec` as the primary agent execution primitive; treat `ws open` as a human GUI action, not an agent-core runtime primitive.
- [ ] Expose the core runtime through MCP tools backed by the same app layer.
- [ ] Define the first MCP tool surface:
  - `workspace_create`
  - `workspace_bind`
  - `workspace_attach`
  - `workspace_install`
  - `workspace_exec`
  - `workspace_env`
  - `workspace_inspect`
- [ ] Add machine-readable command results for agent-driven flows.
- [ ] Add machine-readable workspace inspection, for example `groot ws inspect <name> --json`.
- [ ] Add machine-readable environment output in addition to shell exports.
- [ ] Expose manifest, logs, and workspace metadata as agent-readable resources, likely through MCP resources.
- [ ] Ensure `ws exec` works cleanly for non-interactive commands and long-running processes.
- [ ] Expose logs, state, and workspace metadata in a predictable way.
- [ ] Add deterministic workspace resolution from a repo path for agent use.
- [ ] Define agent-side recovery behavior for partially configured workspaces.
- [ ] Define the agent entry model around Groot primitives instead of direct host access.
- [ ] Keep CLI and MCP surfaces backed by the same runtime core and data model.
- [ ] Define import/export for project runtime state and agent workspace state.

## Layer 4: Human Shortcuts

This layer can add direct human-facing convenience commands after the runtime and agent model are stable.

- [x] Decide whether `groot open <path>` should exist as a first-class non-agent shortcut.
- [x] Decide whether `groot enter <path>` should exist as a shell-first shortcut.
- [ ] Consider `groot init <name> --bind <path>` or `groot init <path>` as an explicit setup shortcut.
- [ ] Keep human shortcuts thin wrappers over the same runtime and agent-capable primitives.

## Recommended Order

1. Finish the core runtime.
2. Make first open own the runtime instead of silently relying on host toolchains.
3. Keep IDE launch reliable for fresh workspaces while preserving runtime ownership.
4. Define the agent-facing contract and agent-driven setup/open flows.
5. Add the machine-readable agent foundation on top of the same runtime.
6. Add optional direct human shortcuts on top of the same runtime.

## Definition Of Success For The Current Phase

The current runtime layer should support:

```bash
groot ws create crawlly
groot ws bind crawlly ~/dev/crawlly
groot ws attach crawlly go@1.25.0
groot ws shell crawlly
go version
git status
```

And this should also work:

```bash
groot ws open crawlly --ide code
```

## Definition Of Success For The Product Direction

Once the agent-driven layer is in place, the workflow should feel closer to:

```bash
groot agent "start crawlly with go@1.25 and node@25"
```

or:

```bash
groot open ~/dev/crawlly
```

with Groot resolving or creating the right workspace, binding the repo, ensuring toolchains, preserving a usable IDE experience, and keeping project runtime and agent state separate from the user's normal machine profile.
