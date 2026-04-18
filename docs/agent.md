# Groot Agent Direction

Groot no longer ships or grows a standalone `groot-agent` binary.

The product direction is:

- Groot stays the user-facing runtime product.
- External agents connect to Groot through MCP.
- Groot keeps one runtime core for humans, automation, and agents.

In the broader roadmap, this MCP layer is Phase 1.5:

- Phase 1 proves the workspace runtime is worth using daily.
- Phase 1.5 exposes that runtime through MCP.
- Phase 2 will let external agents turn intent into reviewed Groot plans instead of mutating state directly.

## Why

Groot is already useful on its own through:

- `groot open <path>`
- `groot open <path> --setup`
- `groot status <path>`
- `groot exec <path> <cmd> [args...]`
- `groot enter <path>`

Adding a separate agent product surface on top of that would create:

- duplicate UX
- extra maintenance
- another contract to stabilize

So the near-term agent strategy is:

- keep Groot as the main CLI product
- expose a thin MCP server through `groot mcp`
- let existing MCP-capable agents drive Groot
- use MCP as the bridge into future planning flows, not as a reason to build a second agent product

## Current MCP Direction

The first MCP surface is intentionally small and testable.

Current server:

```bash
groot mcp
```

Recommended normal flow:

```bash
groot mcp
```

Then let the agent select the active project for the session through `workspace_activate`.

Optional hard-lock startup scope:

```bash
groot mcp --workspace crawlly
groot mcp --project ~/Documents/crawlly --project ~/Documents/the_grime_tcg
```

Current tools:

- `workspace_activate`
- `workspace_status`
- `workspace_setup`
- `workspace_exec`
- `workspace_inspect`
- `workspace_env`
- `workspace_attach`
- `workspace_install`
- `workspace_export`
- `workspace_import`

Those tools are documented in [docs/agent-contract.md](/Users/aristotelistriantafyllidis/Documents/groot/docs/agent-contract.md).

## Non-Goals

Groot should not currently try to be:

- a standalone agent application
- a general-purpose agent framework
- a queue or RPC runtime for agents
- a second product parallel to the main Groot CLI

## Next Step

Keep expanding MCP only where it directly helps real external agents use Groot's runtime:

- richer inspect/status
- more workspace tools
- import/export surfaces
- resources for manifest and logs

The next bigger step after that is not “more agent framework”.
It is a planning surface where an external agent can ask Groot for:

- manifest proposals
- runtime plans
- diffs / previews
- explicit apply after approval

Security boundary:

- prefer activating one project per MCP session with `workspace_activate`
- in a normal unscoped session, `workspace_activate` can switch the live project later if the user redirects the agent
- only allow multi-project MCP sessions explicitly
- use `--project` / `--workspace` startup flags when you want a hard lock before any tool calls happen
- treat unscoped `groot mcp` as trusted local power-user mode until a project is activated

Do not rebuild a separate `groot-agent` path unless real MCP usage proves that one is necessary later.
