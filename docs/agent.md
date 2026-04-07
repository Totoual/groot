# Groot Agent Direction

Groot no longer ships or grows a standalone `groot-agent` binary.

The product direction is:

- Groot stays the user-facing runtime product.
- External agents connect to Groot through MCP.
- Groot keeps one runtime core for humans, automation, and agents.

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

## Current MCP Direction

The first MCP surface is intentionally small and testable.

Current server:

```bash
groot mcp
```

Current tools:

- `workspace_status`
- `workspace_setup`
- `workspace_exec`

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
- resources for manifest and logs

Do not rebuild a separate `groot-agent` path unless real MCP usage proves that one is necessary later.
