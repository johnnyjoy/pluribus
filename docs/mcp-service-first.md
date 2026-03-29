# Pluribus MCP — service-first

The control-plane is the MCP surface: **JSON-RPC 2.0 over HTTP** at **`POST /v1/mcp`**. The same process that serves `/v1/*` REST implements MCP; tool invocations use an internal HTTP loopback into the already-wrapped router so **API key middleware and routes match real traffic**.

## Endpoints and auth

| Item | Detail |
|------|--------|
| Path | `POST /v1/mcp` |
| Body | JSON-RPC 2.0 object or batch array |
| Auth | Same as REST: **`X-API-Key`** when **`PLURIBUS_API_KEY`** is set; MCP-only **`?token=`** is also accepted on **`/v1/mcp`** |
| Disable | `mcp.disabled: true` in config — wrapper is not applied; `/v1/mcp` is not an MCP endpoint |

## Surface

- **Tools** — same set as the stdio MCP adapter (`health`, `recall_compile`, `recall_get`, `recall_run_multi`, `memory_create`, `memory_promote`, `curation_digest`, `curation_materialize`, `enforcement_evaluate`). Descriptions and validation live in **`control-plane/internal/mcp`**. Routes **without** an MCP tool (drift, preflight, compile-multi, evidence, …) are **HTTP-only** — [http-api-index.md](http-api-index.md). See [mcp-poc-contract.md](mcp-poc-contract.md).
- **Prompts** — `pluribus_memory_grounding`, `pluribus_pre_change_enforcement`, `pluribus_memory_curation`, `pluribus_canon_vs_advisory` (embedded `*.md` in `internal/mcp/`; list descriptions include **`SurfaceVersion`** — see `surface_version.go`). **Breaking note:** former prompt names were renamed for memory-first ontology — see [archive/pluribus-semantic-cutover-report.md](archive/pluribus-semantic-cutover-report.md) (**archived**).
- **Resources** — five markdown URIs in **`resources/list`** (bodies in `resources.go`; descriptions include **`SurfaceVersion`**):
  - `pluribus://discipline/doctrine`
  - `pluribus://discipline/lifecycle`
  - `pluribus://discipline/canon-vs-advisory` (alternate URI: `pluribus://discipline/canon-advisory`)
  - `pluribus://discipline/history-not-memory`
  - `pluribus://architecture/active-context-vs-durable-store` (alternate URI: `pluribus://discipline/architecture-notes`)

**`initialize`** returns `serverInfo.name` **`pluribus`** on HTTP (stdio adapter still reports **`pluribus-mcp`** for clarity).

**Docs:** audit [mcp-prompt-resource-audit.md](mcp-prompt-resource-audit.md), proof map [mcp-prompt-resource-proof.md](mcp-prompt-resource-proof.md), versioning [mcp-prompt-resource-versioning.md](mcp-prompt-resource-versioning.md).

## Why HTTP first

- One deployment artifact: no separate MCP sidecar for the same tool contract.
- Clients that support MCP over HTTP can attach to an existing URL; stdio remains a narrow compatibility path (`cmd/pluribus-mcp`).

## References

- Tool ↔ HTTP mapping: [mcp-poc-contract.md](mcp-poc-contract.md)
- Migrating from stdio: [mcp-migration-stdio-to-http.md](mcp-migration-stdio-to-http.md)
- Discipline / lifecycle copy also in [mcp-discipline-doctrine.md](mcp-discipline-doctrine.md)
