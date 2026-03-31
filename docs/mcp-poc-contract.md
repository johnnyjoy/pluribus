# MCP POC — tool → HTTP contract

**Canonical transport:** **MCP over HTTP** on the control-plane — **`POST /v1/mcp`** with JSON-RPC 2.0 (`initialize`, `tools/list`, `tools/call`, `prompts/*`, `resources/*`). Implementation: [`control-plane/internal/mcp`](../control-plane/internal/mcp). Tool calls loop back into the same router (same **`X-API-Key`** semantics as direct REST).

**Compatibility:** thin **stdio** adapter [`control-plane/cmd/pluribus-mcp`](../control-plane/cmd/pluribus-mcp) proxies to HTTP only — use when your client cannot speak MCP HTTP. See [mcp-service-first.md](mcp-service-first.md) and [mcp-migration-stdio-to-http.md](mcp-migration-stdio-to-http.md).

Neither path adds re-ranking or alternate memory authority.

Operational usage doctrine (timing/lifecycle/canon-vs-candidate): [mcp-discipline-doctrine.md](mcp-discipline-doctrine.md).

---

## Dual-layer MCP (automatic mind + optional skills)

**Product stance:** Pluribus **augments** agent cognition (persistence, continuity beyond context limits, experience-shaped recall). The host still invokes tools; the product minimizes **which** tools matter for routine improvement.

### Layer 1 — Default cognitive loop (primary)

Agents are **not** expected to manage memory as a workflow. The **minimal path** for learning and recall is:

| Capability | Tool(s) | Notes |
|------------|---------|--------|
| **Recall** | **`recall_context`** (preferred) · **`memory_context_resolve`** (compat alias) | Same handler: `task_description` / `task` → deterministic compile + **`mcp_context`** (why) + **`recall_bundle`**. |
| **Experience capture** | **`record_experience`** (preferred) · **`mcp_episode_ingest`** (compat alias), optionally **`memory_log_if_relevant`** | Summary-first ingest; server may **auto-distill** → pending candidates when **`distillation.auto_from_advisory_episodes`** is on. |
| **Pre-change gate** | **`enforcement_evaluate`** | Deterministic enforcement before risky edits (not “inspection”). |

**Success:** behavior improves over time **without** mandatory review, explicit distill, or manual promotion — subject to server **promotion / auto-promote** config. Canonical memory still follows [memory-doctrine.md](memory-doctrine.md); automation does not bypass authority.

### Layer 2 — Supplemental skills (optional, pull-based)

Used for **debugging, inspection, deeper control** — **never required** for Layer 1 to function.

| Category | Examples |
|----------|-----------|
| **Introspection** | `curation_pending`, `curation_review_candidate`, `curation_promotion_suggestions`, `curation_strengthened` |
| **Control** | `curation_materialize`, `curation_promote_candidate`, `curation_reject_candidate`, `curation_auto_promote` |
| **Analysis** | `memory_relationships_get`, `memory_relationships_create`, `memory_detect_contradictions`, `memory_list_contradictions`, `evidence_list`, `episode_search_similar` |
| **Advanced recall** | `recall_compile`, `recall_get`, `memory_recall_advanced` (prefer Layer 1 **`recall_context`** first) |

**Rule:** Layer 2 tools are **pull-based**; the product does not require agents to chain them for normal operation.

---

## MCP capability coverage

**MCP is the primary agent interface.** Tools are thin wrappers over the same HTTP handlers the proof harness exercises (loopback inside `POST /v1/mcp`); they are **not** a parallel ontology or ranking path.

**Primary agent pair (Layer 1):** **`recall_context`** / **`memory_context_resolve`** (same tool; recall + deterministic “why”) and **`record_experience`** / **`mcp_episode_ingest`** / **`memory_log_if_relevant`** (experience capture). **`tools/list`** orders behavior-first names first, then compatibility aliases; descriptions encode **when / why / what** in [`control-plane/internal/mcp/tools.go`](../control-plane/internal/mcp/tools.go) and [`tool_surface.go`](../control-plane/internal/mcp/tool_surface.go).

**What agents can do without raw REST** (when the underlying feature is enabled in config):

- **Recall:** **`recall_context`** / **`memory_context_resolve`** (preferred entry), compile/get/run-multi, shaped compile (`memory_recall_advanced`), preflight (`memory_preflight_check`).
- **Episodic:** similar episodes (`episode_search_similar`), explicit distill (`episode_distill_explicit`), ingest (**`record_experience`** / **`mcp_episode_ingest`**), opportunistic **`memory_log_if_relevant`**.
- **Curation:** digest, pending, suggestions, strengthened, full review (`curation_review_candidate`), materialize (`curation_materialize` / alias `curation_promote_candidate`), reject (`curation_reject_candidate`), auto-promote batch (`curation_auto_promote`).
- **Contradictions:** detect pair (`memory_detect_contradictions`), list (`memory_list_contradictions`).
- **Evidence:** list (`evidence_list`), create+link in one step (`evidence_attach`).
- **Memory graph:** list edges (`memory_relationships_get`), create edge (`memory_relationships_create`).
- **Enforcement:** `enforcement_evaluate`.

**Parity expectation:** new agent-facing capabilities should ship as **named tools** first; REST remains the stable wire for tests and custom clients. Remaining HTTP-only surfaces (drift, compile-multi, batch memories, cognition ingest, …) are listed in [http-api-index.md](http-api-index.md).

---

## Environment

| Variable | Default | Purpose |
|----------|---------|---------|
| `CONTROL_PLANE_URL` | `http://127.0.0.1:8123` | Base URL for all HTTP calls |
| `CONTROL_PLANE_API_KEY` | *(empty)* | If set, sent as **`X-API-Key`** when the server has **`PLURIBUS_API_KEY`** configured |

---

## MCP tools → HTTP mapping

**Terminology:** advisory **`source`** is the **ingest channel**; **`pluribus_distill_origin`** on candidates is the **distill mode**. Wire JSON keys are unchanged — see [memory-doctrine.md](memory-doctrine.md) (Terminology).

Shipped **`tools/list`** is defined in [`control-plane/internal/mcp/tools.go`](../control-plane/internal/mcp/tools.go). **`tools.go` is authoritative** for tool names; [http-api-index.md](http-api-index.md) maps routes to types and notes MCP coverage per route. Anything still **without** a tool is available via **direct HTTP** only. The **RC1 narrative subset** (examples for compile / enforcement / run-multi) is [api-contract.md](api-contract.md). Default agent path is **memory-first** (**tags** + **retrieval_query**) ([pluribus-memory-first-ontology.md](pluribus-memory-first-ontology.md)).

| MCP tool | HTTP | Notes |
|----------|------|--------|
| `health` | `GET /healthz` | No arguments |
| `recall_context` | `POST /v1/recall/compile` | Same as **`memory_context_resolve`** (including **`mcp_context`** hints below). |
| `memory_context_resolve` | `POST /v1/recall/compile` | **`arguments`:** **`task_description`** or **`task`** (required); optional **`mode`**, **`tags`**, **`entities`**. Response JSON: **`mcp_context`** + **`recall_bundle`**. **`mcp_context`** may include low-noise behavioral hints: **`decision_hint`** (always on success), **`relevance_hint`** (only when the recall bundle has at least one memory item in a counted bucket), **`after_work_hint`** (wording depends on whether the pool matched). |
| `memory_log_if_relevant` | `POST /v1/advisory-episodes` *(conditional)* | **`text_block`**; if deterministic signals match, same as **`record_experience`** / **`mcp_episode_ingest`**; else JSON **`skipped`: true**. |
| `recall_compile` | `POST /v1/recall/compile` | Body = **`arguments`** (`CompileRequest`). Fields: `internal/recall/types.go`; narrative examples in [api-contract.md](api-contract.md) (subset). |
| `recall_get` | `GET /v1/recall/` | Query from **`arguments`** — see [http-api-index.md](http-api-index.md) and `handlers_getbundle.go`. |
| `recall_run_multi` | `POST /v1/recall/run-multi` | Body = **`arguments`** (`RunMultiRequest`) |
| `memory_create` | `POST /v1/memory` | Body = **`arguments`** (`CreateRequest` — see `internal/memory/types.go`) |
| `memory_promote` | `POST /v1/memory/promote` | Body = **`arguments`** (`PromoteRequest`) |
| `curation_digest` | `POST /v1/curation/digest` | Body = **`arguments`** = **`DigestRequest`** (`internal/curation/types.go`). **Adapter pre-checks:** non-empty **`work_summary`**. |
| `curation_pending` | `GET /v1/curation/pending` | No body; **`arguments`** may be `{}`. |
| `curation_promotion_suggestions` | `GET /v1/curation/promotion-suggestions` | Pending candidates with **`review_recommended`** or **`high_confidence`** readiness (suggestions only). |
| `curation_strengthened` | `GET /v1/curation/strengthened` | Query **`min_support`** from **`arguments`** (default **2**). |
| `curation_materialize` | `POST /v1/curation/candidates/{id}/materialize` | **`arguments`:** `{ "candidate_id": "<uuid>" }` (`id` alias OK). **Adapter pre-checks:** UUID. |
| `record_experience` | `POST /v1/advisory-episodes` | Same row as **`mcp_episode_ingest`** — preferred name. Successful JSON responses may include **`mcp_affordance`** (advisory-only reminder). |
| `mcp_episode_ingest` | `POST /v1/advisory-episodes` | Body: **`summary`** (required), **`source`** ingest channel is always **`mcp`** on the wire. Optional **`tags`**, **`correlation_id`**, **`event_kind`** (adds tag `mcp:event:…`), **`entities`**. **Adapter pre-checks:** server **`mcp.memory_formation`** policy (min length, signal keywords). **201** may include **`deduplicated": true`** when server reuses a row inside the MCP dedup window. |
| `enforcement_evaluate` | `POST /v1/enforcement/evaluate` | Body = **`EvaluateRequest`**: **`proposal_text`** required; optional fields in `internal/enforcement/types.go` and [api-contract.md](api-contract.md). **Adapter pre-checks:** non-empty **`proposal_text`**, byte cap 32768. |
| `curation_promote_candidate` | `POST /v1/curation/candidates/{id}/materialize` | Same as **`curation_materialize`** (alias for agents). |
| `curation_review_candidate` | `GET /v1/curation/candidates/{id}/review` | **`arguments`:** `candidate_id` or `id` (UUID). |
| `curation_reject_candidate` | `POST /v1/curation/candidates/{id}/reject` | **`arguments`:** `candidate_id` or `id` (UUID). |
| `curation_auto_promote` | `POST /v1/curation/auto-promote` | Optional **`arguments`** body (often `{}`). **403** if auto-promote disabled in config. |
| `episode_search_similar` | `POST /v1/advisory-episodes/similar` | **`query`** or **`summary_text`**; optional time/entity filters per **`SimilarRequest`** (`internal/similarity/types.go`). |
| `episode_distill_explicit` | `POST /v1/episodes/distill` | **`episode_id`** or **`summary`**; **`DistillRequest`** (`internal/distillation/types.go`). **403** if distillation disabled. |
| `memory_recall_advanced` | `POST /v1/recall/compile` | Builds **`CompileRequest`**: **`query`** (→ `retrieval_query`), **`mode`** one of **continuity**, **constraint**, **pattern**, **episodic** (shapes `mode` / `variant_modifier`). |
| `memory_preflight_check` | `POST /v1/recall/preflight` | **`PreflightRequest`**: e.g. **`changed_files_count`**, **`tags`**. |
| `memory_detect_contradictions` | `POST /v1/contradictions/detect` | **`memory_id`**, **`conflict_with_id`** (UUIDs). |
| `memory_list_contradictions` | `GET /v1/contradictions` | Optional **`resolution_state`**, **`memory_id`**, **`limit`** in **`arguments`**. |
| `evidence_attach` | `POST /v1/evidence` + `POST /v1/evidence/{id}/link` | **`memory_id`**, **`evidence_text`** (plaintext; adapter base64-encodes); optional **`kind`**. |
| `evidence_list` | `GET /v1/evidence` | **`memory_id`** *or* **`kind`**. |
| `memory_relationships_get` | `GET /v1/memory/{id}/relationships` | **`memory_id`** (UUID). |
| `memory_relationships_create` | `POST /v1/memory/relationships` | Body = **`arguments`** (`CreateRelationshipRequest` — `internal/memory/relationship_handlers.go`). |

**HTTP-only routes** (still no dedicated tool): drift, **`POST /v1/recall/compile-multi`**, batch **`POST /v1/memories`**, cognition **`/v1/ingest/*`**, pattern elevation, attribute edits, expiry, … — [http-api-index.md](http-api-index.md). There is **no** separate “row resolution” HTTP surface for workspace UUIDs on the shipped router.

**Inspect / debug:** `recall_compile` / `recall_get` — responses include server `debug` / RIU when enabled.

---

## Lifecycle quick map (operator mental model)

1. **Ground / recall (Layer 1 default):** **`recall_context`** (or **`memory_context_resolve`**) with **`task_description`** / **`task`**. For raw control, **`recall_get`** / **`recall_compile`** remain available ([http-api-index.md](http-api-index.md)).
2. **Capture experience (Layer 1 default):** **`record_experience`** (or **`mcp_episode_ingest`**) or **`memory_log_if_relevant`** — auto-distill when configured; no explicit distill required for the default loop.
3. **Before risky proposal:** **`enforcement_evaluate`** (deterministic gate).
4. **Optional deeper promotion (Layer 2):** **`curation_digest`** → inspect **`curation_pending`** / **`curation_review_candidate`** → **`curation_materialize`** when governance requires explicit promotion.

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
