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

## Phase Model

The roadmap is now intentionally phased:

- Phase 1: Groot Runtime
- Phase 1.5: Groot MCP Control Plane
- Phase 2: Intent Compiler And Planning Surface
- Phase 3: GOS Direction

This keeps the product sequencing honest:

- prove the runtime is useful before broadening scope
- keep MCP as a structured adapter, not the core identity
- add intent/planning without competing with general-purpose agents
- only evolve toward deeper OS-style control if the runtime becomes indispensable

## Phase 1: Groot Runtime

Phase 1 is about proving that workspace-first local runtime is genuinely better than unmanaged machine state.

### What Phase 1 Must Prove

- workspaces reduce global machine pollution
- switching projects feels cleaner than global installs
- deletion and recreation feel safe and powerful
- first-open/setup/import/export feel normal, not clever
- IDE and shell use are good enough for daily work

### Phase 1 Exit Criteria

- Groot is used daily across multiple real repos
- users stop installing at least some toolchains globally
- workspace deletion and recreation are trusted
- `$HOME` pollution is materially reduced
- Groot-managed execution feels predictable enough to prefer it

### Runtime Core

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

### First Open Owns The Runtime

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

### IDE Strategy

- [x] Prioritize a reliable IDE launch story as the next product milestone after the runtime core.
- [x] Implement a soft IDE mode that preserves project cwd, toolchain PATH, and `GROOT_*` vars without forcing full GUI `HOME` isolation.
- [x] Add a dedicated IDE launcher command, for example `groot ws open <name> --ide code`.
- [x] Add a generic shell hook so terminals launched from a soft-opened IDE can automatically re-enter the strict Groot runtime.
- [x] Document shell-hook setup so terminal activation does not depend on IDE-specific settings.
- [ ] Verify first-launch behavior for a brand-new workspace in VS Code before treating IDE support as fully proven.
- [ ] Decide the default IDE launch policy:
  - strict workspace mode
  - soft IDE mode
  - editor-specific behavior
- [ ] Define which environment variables should be preserved for GUI IDE launches and which should stay isolated.
- [ ] Ensure first-open IDE terminals clearly reflect Groot-managed toolchains when those toolchains are attached and installed.
- [ ] Ensure IDEs can open the bound project path without forcing a separate GUI app identity.
- [ ] Document the tradeoff between terminal isolation and GUI IDE compatibility.

### Human Runtime Shortcuts

- [x] Decide whether `groot open <path>` should exist as a first-class non-agent shortcut.
- [x] Decide whether `groot enter <path>` should exist as a shell-first shortcut.
- [ ] Consider `groot init <name> --bind <path>` or `groot init <path>` as an explicit setup shortcut.
- [ ] Keep human shortcuts thin wrappers over the same runtime primitives.

### Runtime Validation And Repair

- [x] Add tests for manifest load/save behavior.
- [x] Add tests for bind/unbind behavior.
- [x] Add tests for workspace env generation.
- [x] Add tests for attach dedupe/update logic.
- [x] Add tests for installer path helpers.
- [x] Add tests for checksum verification.
- [x] Add shared toolchain garbage collection after the runtime flow is stable.
- [ ] Expose logs, state, and workspace metadata in a predictable way for debugging.
- [ ] Define recovery behavior for partially configured or broken workspaces.
- [ ] Keep validating Groot in day-to-day use across multiple real repos.

## Phase 1.5: Groot MCP Control Plane

Phase 1.5 is the structured adapter layer on top of the same runtime core.

The goal is not to make Groot “an agent”.
The goal is to make Groot usable by external agents through a stable, scoped, inspectable control plane.

### What Phase 1.5 Must Prove

- external agents can inspect and operate on one project predictably
- runtime ownership can be verified and enforced through MCP
- mutating actions and read-only context are clearly separated
- MCP feels like a real control plane, not a shell wrapper in disguise

### Completed MCP Surface

- [x] Decide that MCP is the primary agent adapter for Groot.
- [x] Provide a CLI + JSON bridge that external tools can use today via `groot status <path> --json`.
- [x] Expose the core runtime through an initial MCP server backed by the same app layer.
- [x] Define and implement the first MCP tool surface:
  - `workspace_status`
  - `workspace_setup`
  - `workspace_exec`
- [x] Add a machine-readable workspace inspection surface through MCP via `workspace_inspect`.
- [x] Add machine-readable environment output through MCP via `workspace_env`.
- [x] Add explicit MCP control-plane tools for `workspace_attach` and `workspace_install`.
- [x] Add path-based export through the CLI and MCP as portable workspace-contract data.
- [x] Add path-based import through the CLI and MCP so exported workspace contracts can be recreated around an existing local repo path.
- [x] Support explicit workspace renaming on import so contracts can be restored on machines where the original workspace name already exists.
- [x] Add scoped MCP mode so agents can be limited to one project or an explicit multi-project allowlist.
- [x] Add session-level MCP workspace activation so agents can select one project without requiring MCP server reconfiguration.
- [x] Expose manifest and workspace metadata as agent-readable resources through MCP resources.

### Remaining MCP Foundation Work

- [ ] Keep `ws exec` as the primary agent execution primitive; treat `ws open` as a human GUI action, not an agent-core runtime primitive.
- [ ] Add machine-readable command results for agent-driven flows where plain stdout/stderr is not enough.
- [ ] Add machine-readable workspace inspection, for example `groot ws inspect <name> --json`.
- [ ] Add machine-readable environment output in addition to shell exports where that still matters.
- [ ] Expose logs and longer-lived execution history as agent-readable MCP resources.
- [ ] Ensure `ws exec` works cleanly for non-interactive commands and long-running processes.
- [ ] Add deterministic workspace resolution from a repo path for agent use.
- [ ] Keep CLI and MCP surfaces backed by the same runtime core and data model.

## Phase 2: Intent Compiler And Planning Surface

Phase 2 is not “Groot becomes the agent”.

Phase 2 means:

- the user expresses intent to an external agent
- the external agent calls Groot through MCP
- Groot turns intent into a deterministic manifest/runtime plan
- Groot shows a preview or diff
- execution only happens after explicit approval

The agent handles:

- language understanding
- conversation
- orchestration

Groot handles:

- workspace planning
- manifest compilation
- runtime diffing
- approval boundary
- deterministic execution

### What Phase 2 Must Prove

- intent can be compiled into a safe, reviewable runtime plan
- users can approve changes before mutation
- Groot remains predictable even when the entrypoint is higher-level intent
- Groot does not need to compete with general-purpose agents

### Phase 2 Entry Criteria

- Phase 1 is genuinely useful in daily work
- Phase 1.5 MCP is trusted enough for external agents to use it repeatedly
- runtime ownership and import/export are stable enough that planning on top of them is not shaky

### Phase 2 Planning Surface

- [ ] Define the plan object Groot should return for intent-driven changes.
- [ ] Define the manifest proposal shape separately from the apply step.
- [ ] Define what counts as preview-only versus approval-required mutation.
- [ ] Add MCP planning tools, likely along the lines of:
  - `workspace_plan`
  - `workspace_diff`
  - `workspace_apply_plan`
- [ ] Define how toolchain attach/install, bind, import, and setup steps appear inside a plan.
- [ ] Define what happens when Groot cannot confidently infer versions or services.
- [ ] Decide whether Groot needs any direct CLI planning entrypoint at all, or whether Phase 2 stays MCP-first.
- [ ] Define how project-scoped agent memory, conversation state, and generated plans should live inside Groot.

## Phase 3: GOS Direction

Phase 3 is only justified if Phase 1 and Phase 2 become sticky enough in real use.

GOS is not:

- a kernel rewrite
- a desktop OS replacement
- an excuse to broaden scope prematurely

GOS is:

- a workspace-native operating environment built on top of existing kernels, where isolation, capabilities, and structured execution become first-class

### Immediate Bridge To Phase 3

The next architectural bridge is not more workspace naming or shell sugar.

It is:

- from workspace ownership
- to task ownership
- to service ownership
- to event ownership

That bridge is defined in [docs/runtime-model-v1.md](/Users/aristotelistriantafyllidis/Documents/groot/docs/runtime-model-v1.md).

### What Phase 3 Must Prove

- the workspace/runtime abstraction survives backend changes
- stronger isolation can be introduced without breaking the user model
- multi-workspace orchestration and capability grants can be added cleanly above Groot

### Phase 3 Architectural Direction

- [x] Write `GROOT_RUNTIME_MODEL_V1` to define task, service, and event ownership.
- [x] Split the manifest schema so packages, tasks, and services have distinct types.
- [ ] Keep desired runtime config in the manifest and live execution state in the workspace `state/` dir.
- [x] Add first-class task resources in the app layer with:
  - start
  - stop
  - status
  - list
  - logs
- [x] Expose task lifecycle through the human CLI with:
  - `groot task start`
  - `groot task status`
  - `groot task list`
  - `groot task logs`
  - `groot task stop`
- [ ] Add first-class service resources in the app layer with:
  - start
  - stop
  - restart
  - status
  - list
  - logs
- [ ] Add persisted event records for task/service lifecycle changes before adding streaming.
- [x] Expose task lifecycle through MCP from the same app-layer primitives.
- [ ] Expose service and event lifecycle through CLI and MCP from the same app-layer primitives.
- [ ] Keep workspace lifecycle separate from execution.
- [ ] Keep execution separate from installation.
- [ ] Keep installation separate from storage.
- [ ] Keep UI separate from runtime.
- [ ] Keep agents calling structured APIs only.
- [ ] Define a coordinator layer above Groot, not inside its core runtime.
- [ ] Define capability-based access for:
  - project workspaces
  - additional folders
  - apps and services
  - multi-project sessions
- [ ] Explore stronger execution backends only after the abstraction is stable:
  - OCI backend
  - namespace isolation
  - VM backend for macOS
  - supervised service orchestration

## Recommended Order

1. Finish proving daily value for the Phase 1 runtime.
2. Keep Phase 1.5 MCP clean, scoped, and trusted for external agents.
3. Define and then implement `GROOT_RUNTIME_MODEL_V1` so Groot can own execution lifecycle, not just workspace activation.
4. Build the Phase 2 planning surface on top of MCP instead of inventing a separate agent stack.
5. Only explore Phase 3 / GOS evolution if the earlier phases are clearly sticky in real use.

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

Phase 1 success should feel like:

```bash
groot open ~/dev/crawlly --setup
groot enter ~/dev/crawlly
groot exec ~/dev/crawlly go test ./...
```

Phase 1.5 success should feel like:

```bash
external-agent -> groot mcp -> workspace_activate / workspace_status / workspace_setup / workspace_exec
```

Phase 2 success should feel like:

```text
user intent -> external agent -> groot MCP planning surface -> explicit approval -> Groot apply
```

Phase 3 only matters if all of the above already feel worth keeping in daily life.
