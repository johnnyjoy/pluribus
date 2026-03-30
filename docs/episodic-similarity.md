# Advisory episodic similarity (“have we seen something like this?”)

This layer is **subordinate** to canonical recall. It surfaces **advisory episodes** from `advisory_episodes` using **in-process** signals: token overlap (Jaccard on normalized words), tag overlap, optional **event time** filters, optional **entity** overlap, a small **time-proximity** boost inside a bounded window, and tie-breaks. It answers **“what happened / when / involving whom or what”** in a **weak, non-binding** lane—not **“what must we obey”** (that remains canonical memory).

## What this is not

- **Not** canonical memory — episodes do not replace durable rows in `memories` or recall bundles as authority.
- **Not** a blocking gate — advisory only; **pre-change enforcement** does **not** read `advisory_episodes`.
- **Not** container partitioning — `occurred_after` / `occurred_before` / `entity` / `entities` are **filters** on the global advisory store, not project/workspace/task scopes.
- **Not** in **`POST /v1/recall/compile`** output — recall bundles are built from **`memories`** only; advisory text never appears in the compiled bundle from that path.

## Hierarchy (do not invert)

1. **Canonical memory** — constraints, decisions, failures, patterns, state (and distilled experiences **when promoted** to `memories`).
2. **Supporting evidence** — when enabled, `MemoryItem.supporting_evidence` ([evidence-in-recall.md](evidence-in-recall.md)).
3. **Advisory similar cases** — `POST /v1/advisory-episodes/similar` → `advisory_similar_cases`.

## Storage

- Table: **`advisory_episodes`** — `summary_text`, `source`, `tags`, optional `related_memory_id`, `created_at`.
- **Episodic fields (advisory only):**
  - **`occurred_at`** — when the episode *occurred* (optional). If omitted, effective time for filtering and tie-breaks is **`COALESCE(occurred_at, created_at)`**.
  - **`entities`** — JSON array of normalized strings (e.g. people, systems, topics) for **overlap** with request filters—not a graph and not a partition ID.

## JSON contract (REST)

### `POST /v1/advisory-episodes` (201)

| Field | Required | Notes |
|--------|----------|--------|
| `summary` | yes | Non-empty after trim; stored as `summary_text`. |
| `source` | no | Defaults to **`manual`**. One of: `manual`, `digest`, `ingestion_summary`. |
| `tags` | no | Soft filters for similar search; normalized lower-case overlap. |
| `occurred_at` | no | RFC3339; omitted rows use `created_at` as effective time. |
| `entities` | no | String list; normalized (trim, lower-case, dedupe, max length/count). |
| `related_memory_id` | no | Optional UUID link to a canonical row (not promotion). |

**Response (201):** `id`, `summary_text`, `source`, **`tags`** (always an array, possibly empty), **`entities`** (always an array, possibly empty), `created_at`, `occurred_at` when set, optional `related_memory_id`, `advisory: true`, `non_canonical: true`.

### `POST /v1/advisory-episodes/similar` (200)

| Field | Required | Notes |
|--------|----------|--------|
| `query` | yes | Drives lexical resemblance. |
| `tags` | no | If provided, episode must share **at least one** tag with the filter. |
| `occurred_after` | no | Inclusive lower bound on **effective** time `COALESCE(occurred_at, created_at)`. |
| `occurred_before` | no | Inclusive upper bound on the same effective time. |
| `entity` | no | Single entity string; merged with `entities`. |
| `entities` | no | Multiple entities; **any** normalized overlap with the episode’s `entities` is required when filters are non-empty. |
| `max_results` | no | Caps results (server defaults apply). |

**Validation:** If both `occurred_after` and `occurred_before` are set, **`occurred_after` must be ≤ `occurred_before`**. Otherwise the server returns **400** with `occurred_after must be on or before occurred_before`. Omitting one or both bounds means “no bound” on that side (not an error).

**Empty / open time range:** With no time bounds, all scanned candidates pass the time filter. With both bounds equal, the window is a single instant; episodes whose effective time matches fall inside.

**Response (200):** `{ "advisory_similar_cases": [ … ] }` — each item includes `summary`, `resemblance_score`, `resemblance_signals`, `advisory: true`, `created_at`, optional `occurred_at`, `entities`, etc.

When **`similarity.enabled`** is false, **similar** returns **200** with `advisory_similar_cases: []` (create returns **403**).

## Scoring (shipped)

1. Optional **tag filter** on the request (episode must share at least one tag with the filter when tags are provided).
2. **Lexical Jaccard** + **tag** blend (`internal/similarity/lexical.go`).
3. **Entity overlap**: optional request `entity` / `entities` require at least one normalized match against the episode’s `entities`; **entity Jaccard** contributes a small additive boost to the resemblance score (capped).
4. Optional **time window** on **`POST /v1/advisory-episodes/similar`**: `occurred_after`, `occurred_before` filter rows by **effective time** `COALESCE(occurred_at, created_at)` (not a silo).
5. When **both** `occurred_after` and `occurred_before` are set, a small **time proximity** boost favors episodes near the **midpoint** of the window (signal `time_proximity`); episodes still must pass the lexical **`min_resemblance`** floor.
6. **`min_resemblance`** threshold (default **0.08** in config defaults).
7. **Top-k** ordering: higher **resemblance_score** first; on ties, **newer effective time** wins.

Response items may include **`resemblance_signals`** (e.g. `lexical_overlap`, `shared_tags`, `entity_overlap`, `time_window_filter`, `time_proximity`).

## API

| Method | Path | Purpose |
|--------|------|--------|
| `POST` | `/v1/advisory-episodes` | Store a compact episode. Body may include **`occurred_at`** (RFC3339), **`entities`** (string array). |
| `POST` | `/v1/advisory-episodes/similar` | Rank episodes. Body may include **`occurred_after`**, **`occurred_before`**, **`entity`**, **`entities`** (filters / overlap only). |

- **Create** (201): requires **`similarity.enabled`**. Otherwise **403**.
- **Similar** (200): requires **`similarity.enabled`** for non-empty results; otherwise empty `advisory_similar_cases`.

## Config (`similarity`)

Default: **off**. See `control-plane/configs/config.example.yaml`. The **`make proof-rest`** and **`make proof-episodic`** harnesses enable **`similarity.enabled`** (and **`distillation.enabled`** for the full pipeline) in-process so episodic JSON scenarios run without editing YAML.

- `max_summary_bytes`, `max_episodes_scan`, `max_results`, **`min_resemblance`**.

## Distillation (advisory → candidate)

**`POST /v1/episodes/distill`** (requires **`distillation.enabled`**) turns advisory episode text into **`candidate_events`** rows with structured `proposal_json` — **not** into `memories`. It uses **deterministic keyword** rules (failure, decision, pattern, constraint signals) and sets **`source_advisory_episode_id`** in the proposal when distilling by **`episode_id`**.

**Automatic distillation (optional, default off):** when **`distillation.enabled`** is **true** and **`distillation.auto_from_advisory_episodes`** is **true**, the server runs the **same** extraction and consolidation logic **after a successful** **`POST /v1/advisory-episodes`** — no second code path. Pending proposals record **`pluribus_distill_origin`** **`auto`** (explicit HTTP distill uses **`manual`**; merges can show **`mixed`**). Candidates remain **non-authoritative**; review and materialization rules are unchanged. If background distillation fails, the episode response is still **201**; the failure is **logged** only and does not corrupt candidates.

### Consolidation (pending dedup)

Pending distilled candidates with the same **`kind`** and **`distill_statement_key`** (SHA-256 of `memorynorm` canonical text) **merge into one row** instead of duplicating the queue. Each merge increments **`distill_support_count`**, appends advisory episode UUIDs to **`source_advisory_episode_ids`**, and **raises salience** modestly (capped). The pending list shows **`statement_preview`** with **(×N)** when support &gt; 1. **Traceability** is preserved via the episode ID list; nothing is auto-promoted.

### Suppression

Very short statements (below a minimum character count, default **20** after trim) **do not** produce candidates — conservative noise reduction. Configure with **`distillation.min_statement_chars`** (optional).

- **Not guaranteed** to extract useful learning; output is **unverified** until human review.
- **Does not** promote, score authority, or auto-apply anything — same **materialize** path as digest when you choose to promote.
- **Recall** and **enforcement** use **`memories` only**; pending candidates do not appear in recall bundles or enforcement until materialized. See [curation-loop.md](curation-loop.md).

## Drift (separate)

`POST /v1/drift/check` uses **substring**, **fuzzy failure patterns**, **object lessons**, and optional **LSP** reference risk — a **different** code path from advisory episodic similarity.

## Proof and tests

- **REST integration (focused modules):** `cmd/controlplane/advisory_episodes_episodic_integration_test.go`, `episode_distill_integration_test.go`, `advisory_auto_distill_integration_test.go` (auto vs explicit distill, merge, recall boundary); `internal/similarity/handlers_create_test.go` (episode **201** if auto-distill errors).
- **Canonical episodic proof (REST-only, rerun frequently):**
  - **JSON:** `internal/eval/scenarios/proof-episodic-*.json` — wired into the global **`proof-*.json`** harness (two-pass determinism in **`TestProofHarnessREST_Postgres`**).
  - **Go sprint:** `internal/eval/episodic_proof_sprint_integration_test.go` — **`TestEpisodicProofSprintREST_Postgres`** — adversarial stateful cases (conflict, time skew, boundaries, enforcement stability, promotion pressure, entity noise, supersession-friendly memory behavior, soak loops).
  - **Run:** **`make proof-episodic`** or **`./scripts/proof-episodic.sh`** (requires **`TEST_PG_DSN`**). Inventory, limits, and triage: [evidence/episodic-proof.md](../evidence/episodic-proof.md).
- **`make proof-rest`** runs every embedded **`proof-*.json`** (including episodic JSON) but does **not** run the Go sprint test alone; use **`make proof-episodic`** for the full episodic gate.
- **Logging:** episodic JSON + sprint lines use **`[EPISODIC PROOF]`**; other proof JSON uses **`[PROOF]`**.
