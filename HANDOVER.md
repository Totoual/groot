# Handover: Vault Edge Creation Support

## Architecture Understanding

Vault edges already exist in Groot as a stored data type, but edge creation is not a first-class feature yet.

Current state:
- `edges.jsonl` is part of vault initialization.
- `VaultEdge` is a defined app-layer record type.
- `VaultStats` and `VaultMetadata` already count edges.
- There is no app-layer method to create an edge.
- There is no CLI command to create an edge.
- There is no MCP tool to create an edge.

The current vault model is append-only and workspace-scoped:
- `nodes.jsonl` stores `VaultNode`
- `edges.jsonl` stores `VaultEdge`
- `changes.jsonl` stores `VaultChange`

Node creation already follows a consistent pattern:
1. validate input
2. ensure vault files exist
3. append a typed record to JSONL
4. append an audit event to `changes.jsonl`
5. refresh `vault_meta.json`

Edge creation should follow the same app-layer pattern rather than bypassing it.

The likely V1.1/V1.2 direction is:
- add an app-layer `VaultAppendEdge` or similarly named method
- keep CLI and MCP thin wrappers over that method
- keep append-only semantics
- keep deterministic validation

Important constraint:
- today `VaultEdge` stores only `id`, `from_id`, `to_id`, `type`, and `created_at`
- rationale and confidence are not part of the stored edge schema
- if edge creation support is added narrowly, it should probably preserve that schema and place any richer audit detail into `changes.jsonl`

## Relevant Files

- [internal/app/vault.go](/Users/aristotelistriantafyllidis/Documents/groot/internal/app/vault.go:1)
  Core vault model and behavior. Defines `VaultEdge`, `VaultStats`, `InitVault`, `VaultAppend`, `vaultEdges`, and path helpers.

- [internal/app/metadata.go](/Users/aristotelistriantafyllidis/Documents/groot/internal/app/metadata.go:17)
  `VaultMetadata` already includes `edge_count`, and `writeVaultMetadata` already reads `edges.jsonl`.

- [internal/cli/commands/vault.go](/Users/aristotelistriantafyllidis/Documents/groot/internal/cli/commands/vault.go:1)
  Current CLI surface: `init`, `append`, `search`, `recent`, `stats`. No edge command exists.

- [internal/mcp/server.go](/Users/aristotelistriantafyllidis/Documents/groot/internal/mcp/server.go:1358)
  Current MCP vault tool registration. Has `vault_init`, `vault_recent`, `vault_search`, `vault_append`. No edge creation tool exists.

- [internal/app/vault_test.go](/Users/aristotelistriantafyllidis/Documents/groot/internal/app/vault_test.go:1)
  Existing app-layer vault tests. Good place to add edge append coverage.

- [internal/cli/commands/vault_test.go](/Users/aristotelistriantafyllidis/Documents/groot/internal/cli/commands/vault_test.go:1)
  Existing CLI vault command tests. Good place to add a new command test if a CLI edge subcommand is introduced.

- [internal/mcp/server_test.go](/Users/aristotelistriantafyllidis/Documents/groot/internal/mcp/server_test.go:69)
  Existing MCP registration and tool-call tests. Good place to add tool list assertions and structured content tests for any new vault edge MCP tool.

## Relevant Symbols

- [VaultEdge](/Users/aristotelistriantafyllidis/Documents/groot/internal/app/vault.go:45)
  Current stored edge schema.

- [VaultChange](/Users/aristotelistriantafyllidis/Documents/groot/internal/app/vault.go:52)
  Existing audit/change record type. Likely place to capture edge append rationale/confidence if the edge schema stays narrow.

- [VaultStats](/Users/aristotelistriantafyllidis/Documents/groot/internal/app/vault.go:85)
  Already exposes `EdgeCount`.

- [App.InitVault](/Users/aristotelistriantafyllidis/Documents/groot/internal/app/vault.go:93)
  Already ensures `edges.jsonl` exists.

- [App.VaultAppend](/Users/aristotelistriantafyllidis/Documents/groot/internal/app/vault.go:111)
  Canonical example of append-only node creation flow.

- [App.vaultEdges](/Users/aristotelistriantafyllidis/Documents/groot/internal/app/vault.go:294)
  Existing edge read path.

- [vaultEdgesPath](/Users/aristotelistriantafyllidis/Documents/groot/internal/app/vault.go:320)
  Path helper for `edges.jsonl`.

- [App.writeVaultMetadata](/Users/aristotelistriantafyllidis/Documents/groot/internal/app/metadata.go:77)
  Already recomputes edge counts from `edges.jsonl`.

- [VaultCmd.defaultVaultCommands](/Users/aristotelistriantafyllidis/Documents/groot/internal/cli/commands/vault.go:29)
  Where a future `edge` or `link` subcommand would be registered.

- [Server.tools](/Users/aristotelistriantafyllidis/Documents/groot/internal/mcp/server.go:1358)
  MCP tool registry section where a future `vault_edge_append` or similar tool would be declared.

- [Server.vaultAppendTool](/Users/aristotelistriantafyllidis/Documents/groot/internal/mcp/server.go:3011)
  Canonical MCP wrapper pattern for vault writes.

## Implementation Plan

1. Add a dedicated app-layer edge append method.
   Suggested shape:
   - `func (a *App) VaultAppendEdge(workspaceName string, spec VaultEdgeAppendSpec) (VaultEdge, error)`

   Suggested spec fields:
   - `FromID string`
   - `ToID string`
   - `Type string`
   - optional `Rationale string`
   - optional `Confidence float64`

2. Add deterministic validation.
   Minimum validation:
   - workspace vault exists
   - `from_id` and `to_id` are non-empty
   - `type` is non-empty and belongs to an allowed set
   - referenced node ids exist in `nodes.jsonl`
   - optionally reject self-links if that is considered invalid

3. Decide the V1 stored schema boundary.
   Clean narrow option:
   - keep `VaultEdge` unchanged
   - store only `id`, `from_id`, `to_id`, `type`, `created_at` in `edges.jsonl`
   - put `rationale` and `confidence` into the matching `edge.appended` audit event in `changes.jsonl`

   This matches the current append-only and inspectable design without expanding the edge record yet.

4. Append an audit event for every edge creation.
   Follow the node append pattern:
   - `kind: "edge.appended"`
   - summary with source/target/type
   - payload including edge id, from id, to id, type
   - optionally include rationale/confidence in the payload

5. Refresh vault metadata after edge append.
   Reuse `writeVaultMetadata` so `edge_count` and `change_count` stay correct.

6. Add CLI support.
   Smallest reasonable surface:
   - `groot vault edge <workspace> --from <node-id> --to <node-id> --type <type> [--rationale ...] [--confidence ...]`

   Alternative:
   - `groot vault link ...`

   Keep it thin over the app-layer method.

7. Add MCP support.
   Smallest reasonable surface:
   - tool name like `vault_edge_append`

   Suggested arguments:
   - `path`
   - `from_id`
   - `to_id`
   - `type`
   - optional `rationale`
   - optional `confidence`

   Keep the structured response deterministic:
   - `created`
   - `edge`

8. Keep search/retrieval unchanged for the first pass.
   Do not add edge-aware ranking, graph traversal, or context integration yet.
   V1 edge creation support should be a write-path feature only.

## Risks

- Schema creep risk.
  If rationale/confidence are added directly to `VaultEdge`, this becomes a storage-schema change instead of a narrow write-path addition.

- Validation cost risk.
  Checking node existence requires loading `nodes.jsonl` each time. That is acceptable for V1, but should stay simple.

- Edge type scope risk.
  If the allowed edge type set is not explicit, ad hoc strings will make the graph inconsistent.

- Duplicate edge risk.
  A policy decision is needed:
  - allow duplicate identical edges as append-only history
  - or reject exact duplicates deterministically

  The cleaner V1 path is probably to reject exact `from_id + type + to_id` duplicates while still logging changes only for successful appends.

- Directionality ambiguity risk.
  Some types like `related_to` are symmetric in meaning but directional in storage. The API should document that the caller chooses the direction.

- Surface sprawl risk.
  It is tempting to add edge search, edge listing, edge-aware context, and graph bootstrap together. That should be avoided here.

- Existing worktree risk.
  The repo already has unrelated modified files:
  - `Readme.md`
  - `internal/app/import_test.go`
  - `internal/app/index_test.go`
  - `internal/app/manifest.go`
  - `internal/app/project_detection.go`
  - `internal/app/project_scan.go`
  - `internal/app/project_scan_test.go`
  - `internal/app/workspace_test.go`

  Any implementation should avoid overwriting or reverting those unrelated changes.

## Testing Strategy

### App-Layer Tests

Add tests in [internal/app/vault_test.go](/Users/aristotelistriantafyllidis/Documents/groot/internal/app/vault_test.go:1):

- `TestVaultAppendEdgeAppendsEdgeAndChange`
  - create a workspace
  - append two nodes
  - append one edge
  - verify:
    - one edge exists in `edges.jsonl`
    - one new `edge.appended` change exists
    - vault metadata shows updated `edge_count` and `change_count`

- `TestVaultAppendEdgeRejectsUnknownNodeIDs`
  - verify deterministic failure when either endpoint node does not exist

- `TestVaultAppendEdgeRejectsUnsupportedType`
  - if edge types are restricted, verify invalid types fail cleanly

- `TestVaultAppendEdgeRejectsDuplicateExactEdge`
  - only if duplicate rejection is chosen

### CLI Tests

Add tests in [internal/cli/commands/vault_test.go](/Users/aristotelistriantafyllidis/Documents/groot/internal/cli/commands/vault_test.go:1):

- help output includes the new edge subcommand
- command appends an edge between two existing nodes
- stats output reflects the new edge count
- command fails cleanly for missing `--from`, `--to`, or bad node ids

### MCP Tests

Add tests in [internal/mcp/server_test.go](/Users/aristotelistriantafyllidis/Documents/groot/internal/mcp/server_test.go:69):

- tool list includes the new edge append MCP tool
- tool call returns structured content with the created edge
- tool fails deterministically for invalid node ids or invalid type
- follow the existing `vault_append` / `vault_init` test style

### Non-Goals For This Test Pass

Do not add tests for:
- graph ranking
- context integration with edges
- semantic retrieval
- edge search/list traversal

Those are separate features.

## Telemetry Report

Exact token telemetry was not available from the client/runtime.

- Files opened: `7`
- Lines read estimate: `~1985`
- Searches performed: `3`
- Retrieval/tool calls performed: `0` Groot retrieval calls, `4` local inspection tool invocations
- Files modified: `HANDOVER.md`
- Tests run: `0`

ROUGH ESTIMATE ONLY — not actual billed tokens.

- Estimated context consumption: `~15880` tokens (`1985 × 8`)
