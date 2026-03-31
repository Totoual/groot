# Groot Task List

This file tracks the next implementation milestones while Groot is still in active design and iteration.

## Current Priorities

- [x] Make `delete` the canonical workspace removal command.
- [ ] Add `project_path` to the workspace manifest.
- [ ] Add `groot ws bind <workspace> <path>`.
- [ ] Make `groot ws shell <workspace>` use `project_path` as the working directory when bound.
- [ ] Extract a reusable workspace environment builder for shell and exec flows.
- [ ] Add `ExecInWorkspace(name, command, args)` in the app layer.
- [ ] Add `groot ws exec <workspace> <cmd> [args...]`.
- [ ] Add `groot ws env <workspace>` to expose the runtime environment for IDE launching.

## Manifest And CLI Hardening

- [ ] Validate `name@version` parsing in `ws attach`.
- [ ] Reject malformed or incomplete tool specs with clear errors.
- [ ] Update existing manifest entries instead of duplicating toolchains.
- [ ] Write manifests atomically via temp file + rename.
- [ ] Keep CLI help and README command examples aligned.

## Installer Vertical Slice

- [ ] Keep `groot ws install <workspace>` as the install/apply entrypoint.
- [ ] Finalize the installer interface owned by the app runtime.
- [ ] Complete the Go installer vertical slice end-to-end.
- [ ] Verify downloaded archives before extraction.
- [ ] Harden extraction against path traversal and partial installs.
- [ ] Reuse installed toolchains in both `ws shell` and `ws exec`.

## Tests And Cleanup

- [ ] Add tests for manifest load/save behavior.
- [ ] Add tests for bind/unbind behavior.
- [ ] Add tests for workspace env generation.
- [ ] Add tests for attach dedupe/update logic.
- [ ] Add tests for installer path helpers and checksum verification.
- [ ] Add shared toolchain garbage collection after the runtime flow is stable.

## Definition Of Success For This Phase

The following flow should work cleanly:

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
