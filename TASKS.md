# Groot Task List

This file tracks the next implementation milestones while Groot is still in active design and iteration.

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
- [ ] Add `groot ws env <workspace>` to expose the runtime environment for IDE launching.

### Manifest And CLI Hardening

- [ ] Validate `name@version` parsing in `ws attach`.
- [ ] Reject malformed or incomplete tool specs with clear errors.
- [ ] Update existing manifest entries instead of duplicating toolchains.
- [ ] Write manifests atomically via temp file + rename.
- [ ] Keep CLI help and README command examples aligned.

### Installer Vertical Slice

- [ ] Keep `groot ws install <workspace>` as the install/apply entrypoint.
- [ ] Finalize the installer interface owned by the app runtime.
- [ ] Complete the Go installer vertical slice end-to-end.
- [ ] Verify downloaded archives before extraction.
- [ ] Harden extraction against path traversal and partial installs.
- [x] Reuse installed toolchains in both `ws shell` and `ws exec`.

### Tests And Cleanup

- [x] Add tests for manifest load/save behavior.
- [ ] Add tests for bind/unbind behavior.
- [x] Add tests for workspace env generation.
- [ ] Add tests for attach dedupe/update logic.
- [x] Add tests for installer path helpers.
- [ ] Add tests for checksum verification.
- [ ] Add shared toolchain garbage collection after the runtime flow is stable.

## Layer 2: Agent-Driven UX

This layer should make Groot feel simple for normal developers by letting an agent drive the lower-level Groot primitives.

- [ ] Add workspace lookup by `project_path`.
- [ ] Define the agent-to-Groot contract: create, bind, attach, install, exec, inspect.
- [ ] Decide the primary agent entrypoint:
  - `groot agent "<intent>"`
  - `groot "<intent>"`
  - `groot open <path> --agent`
- [ ] Define the first supported agent intents, for example:
  - "start crawlly with go@1.25 and node@25"
  - "open this repo in a clean workspace"
  - "set up this project for me"
- [ ] Add a path-based setup flow so the agent can create or resolve a workspace from a repo path.
- [ ] Add a path-based open/enter flow for non-agent fallback usage.
- [ ] Ensure the agent can auto-create or auto-bind a workspace when a repo is first seen.
- [ ] Document the normal user workflow as agent-first, with `groot ws ...` kept as advanced/runtime commands.

## Layer 3: Agent Foundation

This layer makes Groot usable by a top-level agent without inventing a separate runtime path.

- [ ] Add machine-readable command results for agent-driven flows.
- [ ] Add machine-readable workspace inspection, for example `groot ws inspect <name> --json`.
- [ ] Add machine-readable environment output in addition to shell exports.
- [ ] Ensure `ws exec` works cleanly for non-interactive commands and long-running processes.
- [ ] Expose logs, state, and workspace metadata in a predictable way.
- [ ] Add deterministic workspace resolution from a repo path for agent use.
- [ ] Define agent-side recovery behavior for partially configured workspaces.
- [ ] Define the agent entry model around Groot primitives instead of direct host access.

## Layer 4: Human Shortcuts

This layer can add direct human-facing convenience commands after the runtime and agent model are stable.

- [ ] Decide whether `groot open <path>` should exist as a first-class non-agent shortcut.
- [ ] Decide whether `groot enter <path>` should exist as a shell-first shortcut.
- [ ] Consider `groot init <name> --bind <path>` or `groot init <path>` as an explicit setup shortcut.
- [ ] Keep human shortcuts thin wrappers over the same runtime and agent-capable primitives.

## Recommended Order

1. Finish the core runtime.
2. Add the agent-facing contract and agent-driven setup/open flows.
3. Add the machine-readable agent foundation on top of the same runtime.
4. Add optional direct human shortcuts on top of the same runtime.

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
groot ws exec crawlly code .
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

with Groot resolving or creating the right workspace, binding the repo, ensuring toolchains, and entering the right environment behind the scenes.
