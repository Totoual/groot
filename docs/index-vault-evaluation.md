# Groot Index/Vault Evaluation

## Goal

Evaluate whether Groot Index, Vault, and Context reduce repo rediscovery and improve handoff quality for agent-assisted work.

The purpose of this document is to define practical, repeatable checks for whether the current Groot substrate improves workflow outcomes in a measurable way.

## What We Are Not Measuring

- Subjective impressions such as "it feels smarter".
- One lucky anecdote.
- Tiny tasks where direct `rg` or a single file read is already sufficient.
- Empty-vault scenarios with no meaningful prior project cognition.

## Metrics

Use the following metrics for each test run:

- Files opened before first correct target file.
- Number of broad searches.
- Lines read as an approximate token proxy.
- Time to first correct edit.
- Whether the right file or symbol was found on first pass.
- Whether recent vault decisions were respected.
- Whether a second agent can resume from vault and context with minimal rediscovery.

Notes:

- "Broad searches" means generic repo-wide scans that are not already narrowed by Groot retrieval.
- "Lines read" is only a rough proxy for context cost. It is useful for relative comparisons even when exact token accounting is unavailable.
- "First correct edit" means the first edit that is materially aligned with the task, not merely the first file touched.

## Test Modes

### A. Baseline

The agent must not use Groot retrieval first.

Allowed pattern:

- normal file search
- normal file inspection
- direct reasoning from source files only

This mode is the comparison point for repo rediscovery cost.

### B. Groot-Assisted

The agent must use the following before broad file inspection:

- `index_stats`
- `vault_recent`
- `index_search` or `index_symbols` or `vault_search`
- `context_build`

Only after those steps may the agent begin broader file inspection.

This mode tests whether Groot provides a smaller and more useful initial grounding set.

## Task Classes

Use at least one task from each class:

### Implementation Task

Example:

- add a command
- extend an MCP tool
- add a small app-layer feature

This tests whether Groot helps the agent find the right file and symbol quickly.

### Bug-Fix Task

Example:

- fix a flaky test
- fix a cleanup race
- fix an incorrect CLI or MCP response

This tests whether Groot helps narrow the failure surface and identify the right local subsystem.

### Handoff Task

Example:

- stop one agent after partial progress
- start a second agent with access to vault and context only
- measure whether the second agent resumes correctly

This tests whether workspace cognition is durable enough to support disposable agents and cross-session continuity.

## Pass/Fail Signals

### Meaningful Positive Signals

- Fewer irrelevant files opened than baseline.
- Fewer broad searches than baseline.
- Fewer lines read than baseline.
- Faster correct target identification.
- Better respect for recent project decisions stored in the vault.
- Cleaner handoff from one agent to another.

### Negative Signals

- Groot-assisted mode reads more than baseline.
- `context_build` returns noisy or stale context.
- The agent still scans broadly after receiving context.
- Vault entries are too vague to guide decisions.

## Reporting Template

Use the following template for each evaluation run:

```md
## Evaluation Run

Task:

Mode:
- Baseline
- Groot-assisted

Files Opened:

Broad Searches Run:

Lines Read Estimate:

First Correct Target File:

Tests Run:

Outcome:
- Improved
- Neutral
- Worse

Notes:
```

## Current Hypothesis

Groot is expected to help more on larger cross-system tasks and handoffs than on tiny localized tasks.

The current expectation is not that Groot makes an agent inherently better at reasoning. The expected gain is narrower initial retrieval, less repeated repo rediscovery, better visibility into recent project decisions, and more reliable cross-agent continuity.
