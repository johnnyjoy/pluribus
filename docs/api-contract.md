# Pluribus control-plane — RC1 HTTP subset contract

This document is a **frozen integrator narrative** for a **subset** of routes: health, readiness, durable memory create/search (minimal + full), recall **compile**, enforcement **evaluate**, and **run-multi**. It is **not** the full HTTP surface.

**Canonical full route map:** **[http-api-index.md](http-api-index.md)** — every meaningful `router.go` path, MCP tool mapping, and pointer to Go types.

**Global rules** (apply broadly; see handlers for edge cases)

- **Content-Type:** requests with a JSON body use **`application/json`**.
- **JSON decode:** `DisallowUnknownFields()` is used — **unknown top-level JSON keys → `400`** with `invalid JSON: ...` (wrapped decoder message).
- **Errors (typical):** `{"error":"<message>"}` except where noted (e.g. duplicate memory).
- **Authentication:** if **`PLURIBUS_API_KEY`** is set in the server environment (non-empty after trim), all routes except **`GET /healthz`** and **`GET /readyz`** require **`X-API-Key`** matching that secret. **`POST /v1/mcp`** may also pass the same key as **`?token=`** (MCP-only). Missing key → **`401`** with a clear `error` string; wrong key → **`403`** with a clear `error` string; legacy query params **`api_key`**, **`apikey`**, **`auth`**, or **`Authorization`** headers are **rejected** when auth is enabled. If **`PLURIBUS_API_KEY`** is unset, **no** API authentication is required.
- **Auth quick reference:** see [authentication.md](authentication.md).

### Memory-first ontology (integrators)

- **Durable memory** lives in the shared **`memories`** pool (tag-first). Recall and enforcement draw from that pool using **tags**, **retrieval text**, and server ranking — not partition IDs in the public JSON body.
- **Public JSON** for **`POST /v1/recall/compile`**, **`POST /v1/enforcement/evaluate`**, and **`POST /v1/recall/run-multi`** matches the Go structs in `internal/recall/types.go` and `internal/enforcement/types.go`: there is **no** workspace/partition UUID, **`context_id`**, or **`target_id`** on those requests. Unknown top-level keys still produce **`400`** (`DisallowUnknownFields`).
- **Correlation** on the wire is **tags**, **retrieval text** / **`retrieval_query`**, and optional **`agent_id`** where the struct allows it — not workspace rows. **`/v1/hives`** is **not** registered on the current router (see `internal/apiserver/router.go`); older writeups that reference it are historical.

Canonical narrative: [pluribus-memory-first-ontology.md](pluribus-memory-first-ontology.md). Historical cutover notes: [archive/pluribus-semantic-cutover-report.md](archive/pluribus-semantic-cutover-report.md) (**archived**).

### Recall routes not fully specified here

For **`GET /v1/recall/`**, **`POST /v1/recall/preflight`**, **`POST /v1/recall/compile-multi`**, and all non-subset routes (drift, evidence, curation HTTP details, contradictions, ingest, advisory episodes), use **[http-api-index.md](http-api-index.md)** and the linked `internal/*/types.go` files.

---

## `GET /healthz`

**Purpose:** Liveness (process up).

| | |
|--|--|
| **Success** | **`200`**, body plain text `ok` (not JSON) |
| **Auth** | Not required |

---

## `GET /readyz`

**Purpose:** Readiness — Postgres reachable and the **`memories`** table exists (baseline DDL from boot has been applied). There is **no** separate schema-version or “upgrade current” gate in this pre-release tree.

| | |
|--|--|
| **Success** | **`200`**, body plain text `ok` |
| **Failure** | **`503`**, plain text body (not JSON): e.g. database unreachable, readiness check error, or **`database schema incomplete: baseline not applied …`** |
| **Auth** | Not required |

---

## Durable memory (storage)

All creates and searches below persist to **`memories`** and **`memories_tags`** in Postgres. JSON responses use **`MemoryObject`** from `control-plane/internal/memory/types.go`.

**Timing fields:** **`created_at`** / **`updated_at`** are system timestamps (recorded / last updated). Optional **`occurred_at`** is **event time** — when the underlying fact or event took place. Recall ranking uses effective recency **`coalesce(occurred_at, updated_at)`** so a memory ingested today about a long-ago event is not treated as “new” merely because of ingest time. Advisory / episodic similarity remains a separate lane; see [episodic-similarity.md](episodic-similarity.md).

---

## `POST /v1/memories`

**Purpose:** Create a durable memory with a minimal body (memory-first; no project required).

### Request

Required: **`kind`**, **`statement`**. Optional: **`tags`**, **`authority`** (default 5 if omitted/zero), **`payload`** (required shape for `object_lesson`), **`status`**, **`occurred_at`** (RFC3339 event time).

Same validation and duplicate semantics as **`POST /v1/memory`** (including **`409`** duplicate memory when dedup applies).

### Response

| Status | Body |
|--------|------|
| **`200`** | `MemoryObject` |
| **`400`** | Validation errors |
| **`409`** | Same duplicate shape as **`POST /v1/memory`** |

---

## `POST /v1/memories/search`

**Purpose:** Search memories by optional text match and tag overlap from one shared memory pool (no project-scoped SQL).

### Request

Optional: **`query`**, **`tags`**, **`status`** (default `active`), **`max`** (capped server-side).

### Response

| Status | Body |
|--------|------|
| **`200`** | JSON array of `MemoryObject` |
| **`400`** | Validation / decode errors |

---

## `POST /v1/memory`

**Purpose:** Create a durable memory object.

### Request

Required: `kind`, `authority`, `statement` (and for `object_lesson`, structured `payload` per validation). Optional: `applicability`, `tags`, `supersedes_id`, `ttl_seconds`, `payload`, `status`, `occurred_at` (RFC3339 event time).

See `control-plane/internal/memory/types.go` (`CreateRequest`) and `pkg/api` enums for allowed `kind` / `applicability`.

**Example (minimal decision)**

```json
{
  "kind": "decision",
  "authority": 5,
  "statement": "Use one canonical query path for listings."
}
```

### Response

| Status | Body |
|--------|------|
| **`200`** | `MemoryObject` (see `memory/types.go`) |
| **`400`** | Validation / business rule errors |
| **`409`** | Duplicate canonical memory: `{"error":"duplicate memory","memory_id":"<uuid>"}` |

### Behavioral notes

- **No silent merge** of unknown fields beyond decoder rules.
- **`CreateRequest`** and **`PromoteRequest`** use the fields in `internal/memory/types.go` only (e.g. **`evidence_ids`** on promote) — no partition UUIDs on the wire for these paths.

---

## `POST /v1/recall/compile`

**Purpose:** Compile a **recall bundle** (constraints, decisions, failures, patterns, object lessons, etc.) from memory-first context.

### Request

| Field | Required | Notes |
|-------|----------|--------|
| `retrieval_query` | no | Situation / intent text for retrieval: **semantic** (pgvector), **lexical** overlap, and token-bridge expansion when non-empty (see `recall.semantic_retrieval` in config) |
| `proposal_text` | no | When **`enable_triggered_recall`** is true, used for heuristic trigger detection |
| **`enable_triggered_recall`** | no | Default **false**. When true, server may merge **`retrieval_query`** from **risk** \| **decision** \| **similarity** heuristics (requires **`recall.triggered_recall.enabled`** in YAML) |
| `tags`, `symbols` | no | |
| `agent_id` | no | Opaque client id for salience / reinforcement only (not a recall filter) |
| `repo_root`, `lsp_focus_path`, `lsp_focus_line`, `lsp_focus_column` | no | Optional LSP context when `lsp.enabled` |
| `max_per_kind`, `max_total`, `max_tokens` | no | Defaults from config when 0 |
| `slow_path_*`, `recommended_expansion`, `variant_modifier`, `mode` | no | Advanced |

**Example**

```json
{
  "tags": ["api"],
  "retrieval_query": "Deploy to production after migration.",
  "enable_triggered_recall": true,
  "proposal_text": "Deploy to production after migration."
}
```

### Response

| Status | Body |
|--------|------|
| **`200`** | `RecallBundle` — see `control-plane/internal/recall/types.go`. When **`enable_triggered_recall`** was true, includes optional **`trigger_metadata`**: `{ "triggers": [ { "kind", "reason", "confidence?" } ], "retrieval_query_effective", "skipped_reason" }` (e.g. **`triggered_recall_disabled`** when YAML gate is off). When **`recall.semantic_retrieval`** is enabled and situation text was non-empty, includes **`semantic_retrieval`**: `{ "attempted", "path": "semantic_hybrid" \| "lexical_only", "fallback_reason"? }` — fallback is **never silent** on the wire when semantic was attempted. |
| **`400`** | Compiler/service validation errors |
| **`503`** | `{"error":"recall: compiler not configured"}` if compiler missing (misconfiguration) |

### Behavioral notes

- **Semantic vs lexical:** Retrieval is **hybrid** when embeddings work; **lexical + tags + authority** remain the baseline. Semantic is **additive** and **best-effort** — see **`semantic_retrieval`** on the bundle when applicable.
- **Ranking / RIU:** When configured, selection order is deterministic for a fixed DB state and **`RefTimeForRanking`** (not wall-clock `time.Now()` in ranked paths).
- **Cache:** Optional Redis cache may return a prior bundle; Postgres remains authoritative for truth.

---

## `POST /v1/enforcement/evaluate`

**Purpose:** Evaluate **`proposal_text`** against **binding** trusted memory (memory-first).

### Request

| Field | Required | Notes |
|-------|----------|--------|
| `proposal_text` | **yes** | Non-empty bounded proposal |
| `intent`, `tags`, `rationale`, `goal` | no | `goal` is recommended to enable explicit movement-toward-goal validation |
| `agent_id` | no | Opaque client id for salience / reinforcement only |

**Example**

```json
{
  "proposal_text": "Switch production database to SQLite.",
  "intent": "datastore"
}
```

### Response

| Status | Body |
|--------|------|
| **`200`** | `EvaluateResponse`: `decision`, `explanation`, `triggered_memories`, `validation`, **`evaluation_engine`**, **`evaluation_note`**, optional `remediation_hints`, `override` |
| **`400`** | Validation |
| **`403`** | `{"error":"…"}` — enforcement **disabled** in config (`enforcement.enabled: false`) |
| **`503`** | Service not configured (should not occur in normal boot) |

### Decision values (current evaluator)

`allow` \| `require_review` \| `block` \| `block_overrideable` — see `control-plane/internal/enforcement/types.go`.

`validation.next_action` is always one of: `proceed` \| `revise` \| `reject`.

### Behavioral notes

- **Rule-based only:** Matching uses shipped **reason codes** on **`triggered_memories`** (e.g. **`normative_conflict`**, **`anti_pattern_overlap`**, **`negative_pattern`**). **`evaluation_engine`** is always **`rule_based_heuristic_v1`**; **`evaluation_note`** states that arbitrary natural-language constraints are **not** interpreted beyond those rules — **`allow`** with empty **`triggered_memories`** does **not** certify real-world safety.
- **Default:** When `enforcement.enabled` is **omitted** in YAML, enforcement behaves as **enabled** (RC1). Set **`enabled: false`** to disable and receive **`403`**.
- There is **no** `allow_with_warning` outcome in the current evaluator.

---

## `POST /v1/recall/run-multi`

**Purpose:** Server-side multi-variant recall orchestration (compile-multi → optional backend synthesis per variant → drift scoring → selection). Requires **`synthesis.enabled: true`** and valid **`synthesis`** provider config at process startup, or the runner is not wired.

### Request

| Field | Required | Notes |
|-------|----------|--------|
| `query` | **yes** | Non-empty |
| `merge`, `promote`, `variants`, `tags`, `symbols`, `agent_id`, … | no | See `RunMultiRequest` in `recall/types.go` |
| **`enable_triggered_recall`** | no | Default **false**. When true, compile-multi **`retrieval_query`** may be enriched (requires **`recall.triggered_recall.enabled`**) |
| **`retrieval_query`** | no | Optional situation text forwarded to compile-multi |
| **`promote: true`** | requires **`merge: true`** | Validation error otherwise |

**Example**

```json
{
  "query": "Implement feature X safely",
  "tags": ["feature-x"],
  "merge": true,
  "variants": 3
}
```

### Response

| Status | Body |
|--------|------|
| **`200`** | `RunMultiResponse` — `scores`, `selected`, `merged`, `promoted`, `confidence`, **`debug`** (always present), etc. See `recall/types.go` |
| **`400`** | Validation / merge or promotion failures |
| **`503`** | `{"error":"run-multi is unavailable: the server-side runner is not configured (enable synthesis in config or use client-side run-multi)"}` — synthesis is disabled or startup prevented wiring the runner |

### Behavioral notes

- **Synthesis disabled (default):** Runner is **not** configured → **`503`** with the message above. This is **not** an empty success.
- **Synthesis enabled:** Backend calls the configured provider (Ollama / OpenAI / Anthropic) from the **same process** — no separate reasoner service, no `/v1/reasoning` route.

---

## Revision

| Version | Date | Notes |
|---------|------|--------|
| RC1 | 2026-03-25 | Initial frozen contract for listed endpoints |
| RC1.1 | 2026-03-26 | Triggered recall: `compile` / `run-multi` opt-in fields; `RecallBundle.trigger_metadata` |
| RC1.2 | 2026-03-27 | Contract aligned to wire types: no partition UUID / `context_id` / `target_id` on compile, enforcement, or run-multi bodies; removed non-shipped workspace HTTP sections |
| RC1.3 | 2026-03-27 | Reframed as **subset** contract; full route map → [http-api-index.md](http-api-index.md) |
| RC1.4 | 2026-03-28 | Enforcement: **`evaluation_engine`** / **`evaluation_note`**; compile: **`semantic_retrieval`** debug for fallback visibility |
| RC1.5 | 2026-03-28 | **`/readyz`**: align with implementation (core table check only); pre-release — no DB upgrade semantics |
