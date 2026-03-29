# MCP POC — tool → HTTP contract

**Canonical transport:** **MCP over HTTP** on the control-plane — **`POST /v1/mcp`** with JSON-RPC 2.0 (`initialize`, `tools/list`, `tools/call`, `prompts/*`, `resources/*`). Implementation: [`control-plane/internal/mcp`](../control-plane/internal/mcp). Tool calls loop back into the same router (same **`X-API-Key`** semantics as direct REST).

**Compatibility:** thin **stdio** adapter [`control-plane/cmd/pluribus-mcp`](../control-plane/cmd/pluribus-mcp) proxies to HTTP only — use when your client cannot speak MCP HTTP. See [mcp-service-first.md](mcp-service-first.md) and [mcp-migration-stdio-to-http.md](mcp-migration-stdio-to-http.md).

Neither path adds re-ranking or alternate memory authority.

Operational usage doctrine (timing/lifecycle/canon-vs-candidate): [mcp-discipline-doctrine.md](mcp-discipline-doctrine.md).

---

## Environment

| Variable | Default | Purpose |
|----------|---------|---------|
| `CONTROL_PLANE_URL` | `http://127.0.0.1:8123` | Base URL for all HTTP calls |
| `CONTROL_PLANE_API_KEY` | *(empty)* | If set, sent as **`X-API-Key`** when the server has **`PLURIBUS_API_KEY`** configured |

---

## MCP tools → HTTP mapping

Shipped **`tools/list`** is defined in [`control-plane/internal/mcp/tools.go`](../control-plane/internal/mcp/tools.go). Operations **not** listed as MCP tools are still available via **direct HTTP** — see **[http-api-index.md](http-api-index.md)** for every route, request type, and whether an MCP tool exists. The **RC1 narrative subset** (examples for compile / enforcement / run-multi) is [api-contract.md](api-contract.md). Default agent path is **memory-first** (**tags** + **retrieval_query**) ([pluribus-memory-first-ontology.md](pluribus-memory-first-ontology.md)).

| MCP tool | HTTP | Notes |
|----------|------|--------|
| `health` | `GET /healthz` | No arguments |
| `recall_compile` | `POST /v1/recall/compile` | Body = **`arguments`** (`CompileRequest`). Fields: `internal/recall/types.go`; narrative examples in [api-contract.md](api-contract.md) (subset). |
| `recall_get` | `GET /v1/recall/` | Query from **`arguments`** — see [http-api-index.md](http-api-index.md) and `handlers_getbundle.go`. |
| `recall_run_multi` | `POST /v1/recall/run-multi` | Body = **`arguments`** (`RunMultiRequest`) |
| `memory_create` | `POST /v1/memory` | Body = **`arguments`** (`CreateRequest` — see `internal/memory/types.go`) |
| `memory_promote` | `POST /v1/memory/promote` | Body = **`arguments`** (`PromoteRequest`) |
| `curation_digest` | `POST /v1/curation/digest` | Body = **`arguments`** = **`DigestRequest`** (`internal/curation/types.go`). **Adapter pre-checks:** non-empty **`work_summary`**. |
| `curation_materialize` | `POST /v1/curation/candidates/{id}/materialize` | **`arguments`:** `{ "candidate_id": "<uuid>" }` (`id` alias OK). **Adapter pre-checks:** UUID. |
| `enforcement_evaluate` | `POST /v1/enforcement/evaluate` | Body = **`EvaluateRequest`**: **`proposal_text`** required; optional fields in `internal/enforcement/types.go` and [api-contract.md](api-contract.md). **Adapter pre-checks:** non-empty **`proposal_text`**, byte cap 32768. |

**HTTP-only routes** (drift, preflight, compile-multi, contradictions, evidence, ingest, advisory episodes, …): [http-api-index.md](http-api-index.md). There is **no** separate “row resolution” HTTP surface for workspace UUIDs on the shipped router.

**Inspect / debug:** `recall_compile` / `recall_get` — responses include server `debug` / RIU when enabled.

---

## Lifecycle quick map (operator mental model)

1. **Memory grounding / recall:** **`recall_get`** or **`recall_compile`** — prefer **`tags`** + **`retrieval_query`**. JSON fields must match **`CompileRequest`** / GET query mapping ([http-api-index.md](http-api-index.md)); do not send unknown keys.
2. **Before risky proposal:** **`enforcement_evaluate`**.
3. **After meaningful work:** **`curation_digest`** with **`work_summary`**.
4. **Durable learning:** **`curation_materialize`**.

**Ontology:** [pluribus-memory-first-ontology.md](pluribus-memory-first-ontology.md).

For examples and canon/advisory guidance, see [mcp-discipline-doctrine.md](mcp-discipline-doctrine.md).

---

## `tools/call` arguments

Pass a JSON object:

```json
{
  "name": "recall_compile",
  "arguments": {
    "tags": ["api"],
    "retrieval_query": "current situation in one line",
    "max_per_kind": 5
  }
}
```

Field names match control-plane request types — e.g. **`CompileRequest`**, **`RunMultiRequest`**, **`PromoteRequest`**, **`DigestRequest`**, **`EvaluateRequest`** (enforcement: `internal/enforcement/types.go`); see also `internal/recall/types.go`, `internal/memory/types.go`, `internal/curation/types.go`.

---

## Curation loop (MCP) — intended agent flow

This is **not** automation beyond the server: digest creates **pending** candidates; materialize applies **existing** materialization rules (`promotion.*`, evidence gates, etc.).

1. **`curation_digest`** — Send bounded **`work_summary`** (required), optional **`curation_answers`**, **`evidence_ids`**, **`artifact_refs`**, **`signals`**, **`options.dry_run`** / **`max_proposals`** per **`DigestRequest`** (`internal/curation/types.go`). Response is **raw JSON** (`DigestResult`: `proposals`, `rejected`, `truncated`). Inspect **`candidate_id`** (and/or `proposal_id`) in **`proposals`** when not `dry_run`.
2. Review / select — Agent or operator decides which candidate to materialize (server may return **`rejected`** or empty proposals; HTTP **400** surfaces as MCP **`isError: true`** with status + body).
3. **`curation_materialize`** — Pass **`candidate_id`** from step 1. Success → **201** + **`MemoryObject`** JSON; blocked → **400** with server message (e.g. evidence required), or **404** if candidate missing — MCP preserves **`isError`** and full text for recovery.

**Authority:** The adapter does **not** raise memory authority or auto-promote. **`curation_digest`** does not create durable memory until **`curation_materialize`** (or other server paths) succeed.

**Payload discipline:** Prefer short **`work_summary`** and structured **`curation_answers`** over huge pastes; server **`curation.digest_*`** caps still apply.

---

## Pre-change enforcement (MCP)

Use **`enforcement_evaluate`** when you need a **structured gate decision** (`allow`, `require_review`, `block`, …) against **trusted binding memory** before acting on a proposal. Enable **`enforcement.enabled`** on the control-plane first.

- **vs `curation_digest`:** digest scores/creates **candidates** from post-work text; it does **not** evaluate an arbitrary change proposal against binding constraints.
- **vs drift:** **`/v1/drift/check`** returns violations/warnings and optional **`block_execution`** signals; enforcement returns a single **`decision`** for workflow scripting. See [pre-change-enforcement.md](pre-change-enforcement.md).

---

## Response shape (MCP)

Each tool returns **text content** whose `text` is the **raw HTTP response body** (JSON or plain `ok` for health). On HTTP **4xx/5xx**, MCP sets **`isError`: true** and the text includes **status line + body** (truncated if very large).

---

## Error handling (honest failures)

- Network errors → MCP tool result `isError: true`, message describes dial/timeout.
- HTTP errors → status code and body snippet in the text block.
- **Adapter validation** (digest / materialize) → JSON-RPC **`error`** on the `tools/call` request (e.g. missing **`candidate_id`**) — no HTTP round-trip. **Digest** / **materialize** field errors from the **server** still return HTTP **4xx** with **`isError: true`** on the tool result, same as other tools.
- **Run-multi** requires **`synthesis.enabled: true`** with a valid provider for server-side variant generation; default config has **`synthesis.enabled: false`**, so **`POST /v1/recall/run-multi`** reports that server-side run-multi is not configured until operators opt in (see [../control-plane/docs/backend-synthesis.md](../control-plane/docs/backend-synthesis.md)). That is **intentional** — prefer client-side synthesis unless backend synthesis is explicitly enabled.

---

## Deployment

See [deployment-poc.md](deployment-poc.md) for compose, DB initialization, and ports.

---

## HTTP MCP (preferred)

With **controlplane** listening (e.g. `:8123`), send JSON-RPC to **`POST /v1/mcp`**, header **`Content-Type: application/json`**, optional **`X-API-Key`**. Single request or batch array. Example **`initialize`**:

```bash
curl -sS -X POST http://127.0.0.1:8123/v1/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | jq .
```

Prompt and resource URIs are listed under `internal/mcp/prompts.go` and `internal/mcp/resources.go`.

## Build / run (stdio MCP — compatibility)

From `control-plane/`:

```bash
go build -o pluribus-mcp ./cmd/pluribus-mcp
./pluribus-mcp
```

Configure the client to run this binary with **`CONTROL_PLANE_URL`** pointing at your server (e.g. `http://192.168.1.10:8123`) **only if** you cannot use HTTP MCP on the service.

### If a tool is “not registered” in Cursor

- Point MCP config at the **built binary** (e.g. `.../control-plane/pluribus-mcp`), **not** the `cmd/pluribus-mcp` source directory.
- After changing code, **`go build -o pluribus-mcp ./cmd/pluribus-mcp`** again.
- **Restart Cursor** or reload MCP servers so `tools/list` picks up tool changes (`memory_create`, `recall_get`, etc.).
- Until then, calling the **same HTTP routes** with `curl` is the same API the adapter uses—not a different protocol.

### Cursor IDE: `STATUS.md`, missing `tools/*.json`, “Not connected”

**`pluribus-mcp` does not ship** `tools/*.json` **in the repo**—tools are advertised over **stdio** via **`tools/list`**. Under `~/.cursor/projects/.../mcps/user-pluribus/` you may only see **`SERVER_METADATA.json`** and **`STATUS.md`**; that is **normal** and differs from servers that have static JSON mirrors. If **`STATUS.md`** says the server errored, or logs show **`Client closed`** / **`Not connected`**, reconnect the MCP server (or restart Cursor) before expecting tool calls to work.

See **[mcp-usage.md](mcp-usage.md)** for Cursor setup and recall-driven ordering.
