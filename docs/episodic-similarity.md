# Advisory episodic similarity (“have we seen something like this?”)

This layer is **subordinate** to canonical recall. It surfaces **advisory episodes** from `advisory_episodes` using **in-process** signals only: token overlap (Jaccard on normalized words), tag overlap, project locality, and recency tie-breaks.

## What this is not

- **Not** canonical memory — episodes do not replace `memory_objects` or recall bundles.
- **Not** a blocking gate — advisory only.

## Hierarchy (do not invert)

1. **Canonical recall** — constraints, decisions, failures, patterns, object lessons.
2. **Supporting evidence** — when enabled, `MemoryItem.supporting_evidence` ([evidence-in-recall.md](evidence-in-recall.md)).
3. **Advisory similar cases** — `POST /v1/advisory-episodes/similar` → `advisory_similar_cases`.

## Storage

- Table: **`advisory_episodes`** — text summary, tags, source, optional `related_memory_id`, timestamps.

## Scoring (shipped)

1. Optional **tag filter** on the request.
2. **Lexical Jaccard** + **tag** blend (`internal/similarity/lexical.go`).
3. **`min_resemblance`** threshold (default **0.08**).
4. **Top-k** + recency tie-break.

Response items may include **`resemblance_signals`** (e.g. `lexical_overlap`, `shared_tags`).

## API

| Method | Path | Purpose |
|--------|------|--------|
| `POST` | `/v1/advisory-episodes` | Store a compact episode. |
| `POST` | `/v1/advisory-episodes/similar` | Rank episodes by resemblance. |

- **Create** (201): requires **`similarity.enabled`**. Otherwise **403**.
- **Similar** (200): requires **`similarity.enabled`**; otherwise empty `advisory_similar_cases`.

## Config (`similarity`)

Default: **off**. See `control-plane/configs/config.example.yaml`.

- `max_summary_bytes`, `max_episodes_scan`, `max_results`, **`min_resemblance`**.

## Drift (separate)

`POST /v1/drift/check` uses **substring**, **fuzzy failure patterns**, **object lessons**, and optional **LSP** reference risk — a **different** code path from advisory episodic similarity.
