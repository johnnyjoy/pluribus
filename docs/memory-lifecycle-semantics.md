# Memory lifecycle recall semantics (Phase 8 contract)

This document defines how Pluribus distinguishes **current guidance** from **historical context** in recall. It complements [memory-doctrine.md](memory-doctrine.md) and [memory-utility-reputation.md](memory-utility-reputation.md).

## Core distinction

| Term | Meaning |
|------|---------|
| **Current guidance** | Memory allowed to guide normal agent behavior now. Default recall mode. |
| **Historical context** | Preserved memory that explains prior states, supersession, archival, or demotion — **not** automatic present guidance. |

**Product honesty:**

- Superseded does **not** mean false.
- Archived does **not** mean deleted.
- Outdated does **not** mean wrong.
- Low utility does **not** mean untrue.
- **`pending`** is **recallable memory with slightly lower weight** — labeled `pending_context` in bundles. It is **not** a separate store and **not** excluded from compile or binding. It should be **exceptionally rare** in a healthy hive.

Pluribus does **not** implement persistence classes, half-life decay, or automatic durable/ephemeral classification in Phase 8.

## Recall modes

### `current` (default)

Used when `recall_mode` is omitted or explicitly set to `current`.

**Includes:** `active` and **`pending`** memories (same pool; pending ranked with a small trust dampener).

**Excludes by default:** `superseded`, `archived`, `rejected`, `quarantined`, `deleted`.

**Legacy compatibility:** When the retrieval query explicitly asks about lifecycle change (keywords such as `deprecated`, `superseded`, `obsolete`, `legacy`, or certain `sqlite` + `still`/`use` patterns), compile may merge **superseded** candidates and label them — this preserves recall-benchmark behavior for migration queries without treating all history as current guidance.

Utility ranking and contradiction policy still apply.

### `historical`

Used for history, change, supersession, archival, outdated, or refuted queries.

**Includes:** `active`, `pending`, `superseded`, `archived` (when relevant to query).

**Excludes:** `rejected`, `quarantined`, `deleted`.

**Behavior:**

- Historical rows receive bounded score caps so they do not masquerade as top current guidance within the same bundle.
- Each item exposes `lifecycle_role` and `status` in the recall bundle.
- Historical recall does **not** mutate authority, utility, or status.

## REST

`POST /v1/recall/compile` accepts:

```json
{
  "retrieval_query": "why did Phase 5 add memory formation gates?",
  "recall_mode": "historical",
  "occurred_after": "2023-06-15T00:00:00Z",
  "occurred_before": "2023-06-16T00:00:00Z"
}
```

Date filters use **`occurred_at`** when present; otherwise **`created_at`** (documented fallback). Invalid timestamps return HTTP 400.

Alternative:

```json
{
  "retrieval_query": "...",
  "include_status": ["active", "superseded", "archived"]
}
```

Invalid `recall_mode` returns a validation error (HTTP 400).

## MCP

`recall_context` and `memory_context_resolve` accept the same `recall_mode`, `include_status`, `occurred_after`, and `occurred_before` fields, forwarded to compile.

`memory_create` accepts optional `supersedes_id` (parity with REST `POST /v1/memory`).

## Bundle metadata

Each recalled memory item may include:

| Field | Purpose |
|-------|---------|
| `status` | Row status (`active`, `superseded`, `archived`, …) |
| `lifecycle_role` | Agent-facing role (`current_guidance`, `superseded_context`, `archived_context`, `refuted_context`, `outdated_context`, …) |
| `utility_score` | Bounded aggregate utility when enabled |
| `superseded_by` | Replacing memory id when known |

Top-level bundle field `lifecycle_recall` records the mode applied.

## Utility demotion vs existence

Utility feedback (`wrong`, `outdated`, `refuted`, …) lowers influence via **utility score** and ranking. It does **not** delete memory rows. In `historical` mode, demoted memories can appear when query-relevant and are labeled accordingly.

## Archive and TTL (Phase 9B)

| Concept | Behavior |
|---------|----------|
| **Archive** | Status transition (`active` → `archived`); row preserved |
| **TTL expire** | `POST /v1/memory/expire` archives eligible rows; never deletes |
| **Historical recovery** | `recall_mode: historical` retrieves archived/superseded rows |
| **TTL guard** | Rows with utility history, evidence, relationships, contradictions, durable tags, material `occurred_at`, or supersession payload are **not** TTL-archived |

Archive is a **historical lifecycle state**, not a trash can. TTL is **conservative operational cleanup** for disposable low-authority rows only.

## Advisory garbage pruning (separate from canonical memory)

Rejected **advisory episodes** may be deleted via **`POST /v1/advisory-episodes/prune-rejected`**. This is explicit garbage handling for episodic sludge, **not** canonical memory lifecycle.

## What Phase 8 does not claim

- Semantic / vector recall as lifecycle truth verification
- Persistence class taxonomy or half-life decay engine
- Human review workflow or RBAC
- Automatic promotion of historical memory into current guidance
