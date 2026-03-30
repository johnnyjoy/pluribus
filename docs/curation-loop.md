# Curation digest loop (structured capture → pending → materialize)

Pluribus can turn post-work signals into **pending candidate rows** with structured JSON, then **materialize** them into durable memory (and optional evidence links), using the same **promotion** gates as other promotion paths (`promotion.*` in config).

**Wire truth:** `control-plane/internal/curation/types.go` (`DigestRequest`, `DigestResult`, …) and handlers in `internal/curation/handlers.go`. **Full route list:** [http-api-index.md](http-api-index.md).

### Distillation (advisory → candidate)

**`POST /v1/episodes/distill`** (see [episodic-similarity.md](episodic-similarity.md)) creates **pending** rows in the same **`candidate_events`** queue using keyword rules over advisory episode text. It is **not** digest and **not** LLM-based; it exists to funnel “possible learning” from episodes into **review**, not into canonical memory. **`proposal_json`** may include **`source_advisory_episode_id`** for traceability. When **`distillation.auto_from_advisory_episodes`** is enabled, **ingesting** an advisory episode can create or merge the **same** kind of pending rows **without** calling distill explicitly — still **candidate-only**, still reviewable, still not canon until materialized. **Materialize** is unchanged: a human/process still promotes to durable memory when appropriate. **REST proof** for distill → review → materialize (and boundaries vs recall/enforcement): [evidence/episodic-proof.md](../evidence/episodic-proof.md), **`make proof-episodic`**.

**Consolidation:** repeated distillations that would produce the same **kind + normalized statement** update the **existing pending row** (higher **`distill_support_count`**, merged **`source_advisory_episode_ids`**, slightly higher salience) instead of flooding the queue. Review stays **one row per distinct lesson**, not five near-duplicates.

---

## Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/v1/curation/digest` | Classify `curation_answers` + `work_summary` into `proposal_json` rows (or `dry_run` only). |
| `POST` | `/v1/episodes/distill` | Optional: advisory episode (or inline `summary`) → pending `candidate_events` via keyword distillation (`distillation.enabled`). |
| `GET` | `/v1/curation/pending` | List **all** pending candidates (**no query parameters**). Entries with `proposal_json` include `structured_kind` and `statement_preview`. |
| `GET` | `/v1/curation/candidates/{id}/review` | Read-only **review assistance** for a pending candidate (explanation, signal, supporting episode summaries, promotion preview). **No writes.** |
| `POST` | `/v1/curation/evaluate` | Salience evaluate a single text (`EvaluateRequest` in curation package — **`text`** field). |
| `POST` | `/v1/curation/candidates/{id}/materialize` | Create memory from `proposal_json`, link evidence IDs, mark candidate `promoted`. |
| `POST` | `/v1/curation/auto-promote` | Optional **batch** auto-materialization when `promotion.auto_promote` is **true** and thresholds pass (see Controlled Promotion). |
| `POST` | `/v1/curation/candidates/{id}/promote` | Alternate promote path (empty body or payload per handler — object-lesson flows). |
| `POST` | `/v1/curation/candidates/{id}/reject` | Reject candidate. |

---

## Controlled Promotion

**Principle:** The system may **accelerate** promotion only under **explicit config**, **deterministic rules**, and **full traceability**. It does **not** invent truth; it applies the same materialize path as manual promotion after guardrails.

### Readiness (derived, not binding)

Pending rows with `proposal_json` include **`promotion_readiness`** and **`readiness_reason`** on **`GET /v1/curation/pending`** and **`GET .../review`**:

| Value | Meaning |
|-------|--------|
| `not_ready` | Insufficient support, salience, or statement strength for confident promotion. |
| `review_recommended` | Worth a human pass before materialize. |
| `high_confidence` | Strong repetition + salience for suitable kinds (e.g. failure/pattern); still not auto unless `auto_promote` is on. |

### Optional auto-promote

Config under **`promotion`** (see `configs/config.example.yaml`):

- **`auto_promote`** — default **false**. When **true**, **`POST /v1/curation/auto-promote`** may materialize **eligible** pending candidates in one batch.
- **`min_support_count`**, **`min_salience`**, **`allowed_kinds`** — conservative defaults (e.g. support ≥ 4, salience ≥ 0.7, kinds `failure` + `pattern` when `allowed_kinds` omitted).

Each successful auto run logs **`[AUTO PROMOTE]`** with `candidate_id`, `memory_id`, readiness, and reason. **Disable instantly** by setting **`auto_promote: false`** and restarting (or redeploying config).

### Guardrails (manual and auto)

**`ValidatePromotionCandidate`** runs before any materialize: minimum statement length, evidence gates, duplicate **active/pending memory** (same statement key), and inconsistent salience vs merged support. Failures return **`promotion validation: …`**.

### Traceability into canonical memory

Materialized rows get **`payload.pluribus_promotion`**: `candidate_id`, `supporting_episode_ids`, `distill_support_count_at_promotion` (v1). This answers “why does this memory exist?” without guessing.

### Memory evolution (non-destructive)

**No destructive retire.** There is no API to archive or hide memory to “undo” truth; use the mechanisms below. Do not treat deletion or hidden rows as the correction path.

| Mechanism | Purpose |
|-----------|--------|
| **New memory + higher authority / recency** | Outrank weaker rows in compile ranking. |
| **`supersedes_id`** on **`POST /v1/memory`** | New row supersedes an active prior row (prior becomes **`superseded`** — still in DB, not binding). |
| **`supersedes_memory_id`** on candidate **`proposal_json`** | Same as above when materializing a duplicate statement key: must equal the existing memory id from validation. |
| **`payload.pluribus_evolution`** | Optional on materialize: **`superseded_by`**, **`contradicts`**, **`invalidated_by`** (UUID strings) — additive, auditable links. |
| **Contradiction / enforcement** | Existing contradiction and binding logic continues to resolve conflicts without deleting rows. |

Recall applies a **score penalty** when **`pluribus_evolution.invalidated_by`** is set, so influence drops without hiding the row.

**Legacy `status = archived`:** rows may still exist from older flows; they remain in the database. Default recall/search still prefers **`active`**; archived is low / no influence, not “gone.”

---

## Candidate Review Experience

**Purpose:** Help a human reviewer **understand** a candidate in seconds—**not** to auto-accept or auto-promote. The server does **not** decide truth; it surfaces **deterministic** copy and **bounded** evidence so review is high-signal and low-friction.

**`GET /v1/curation/candidates/{id}/review`** returns **no side effects** (no memory writes, no promotion, no recall or enforcement changes).

| Field | Meaning |
|-------|--------|
| `explanation` | Human-readable sentence(s) built from **kind**, **statement**, **support count**, **`entity:*` tags**, other tags, and **salience** (no LLM). |
| `supporting_episodes` | Up to **three** short **`summary_text`** clips from **`advisory_episodes`**, in trace order from **`source_advisory_episode_ids`** / **`source_advisory_episode_id`**. |
| `signal_strength` | **`low`** / **`moderate`** / **`strong`** — explicit thresholds from **support count** and **salience** (see `internal/curation/review_build.go`). Not opaque ML scoring. |
| `signal_detail` | Plain-language line repeating the inputs (episode count, salience, kind). |
| `tags_grouped` | **`entities`** (from `entity:*` tags) vs **`domain`** (other tags), deduped. |
| `entities_display` | Same entity names as `tags_grouped.entities` for quick scanning. |
| `promotion_preview` | Read-only projection of what **`POST .../materialize`** would apply (kind, statement, tags, authority, applicability, pending vs active note). **Does not** create memory. |
| `promotion_readiness` / `readiness_reason` | Same derived classification as **`GET /v1/curation/pending`** (see Controlled Promotion). |

**Recall / enforcement:** Candidates remain **out of** recall bundles and **out of** enforcement bindings until **materialized** into durable memory; this endpoint does not change that.

---

## `DigestRequest` (POST `/v1/curation/digest`)

Only the following JSON fields exist on the wire (unknown keys → **400**):

| Field | Required | Notes |
|-------|----------|--------|
| `work_summary` | **yes** | Bounded narrative. |
| `signals` | no | String array. |
| `curation_answers` | no | `DigestCurationAnswers`: `what_changed`, `what_learned`, `decision`, `constraint`, `failure`, `pattern`, `never_again`. |
| `evidence_ids` | no | UUIDs validated when present. |
| `artifact_refs` | no | `kind` + `ref` opaque refs. |
| `options` | no | `dry_run`, `max_proposals`. |

**There is no** legacy target or context identifier field, and no extra partition field beyond what **`DigestRequest`** already carries, in the current code.

---

## Materialize rules

- Candidate must be **`pending`** and have **`proposal_json`**.
- **`ValidatePromotionCandidate`** must pass (statement length, evidence gates, no duplicate statement key, consistent signals).
- **`promotion`** config (`require_evidence`, `min_evidence_links`, `require_review`) applies the same way as run-multi promotion: e.g. materialize can require evidence IDs or create **`pending`** memory when `require_review` is true.
- Created memory includes **`payload.pluribus_promotion`** trace (candidate id, episode ids, support count at promotion) and optional **`payload.pluribus_evolution`** (`superseded_by`, `contradicts`, `invalidated_by`).
- Optional **`supersedes_memory_id`** on **`proposal_json`** sets **`CreateRequest.supersedes_id`** when replacing the duplicate statement key (must match validation).
- **`object_lesson`** proposals are not materialized here — use **`POST .../promote`** with a full object-lesson payload when the handler allows.

---

## Config (`curation` + `promotion`)

See `control-plane/configs/config.example.yaml`:

- **`curation.digest_*`** — bounds for digest (max proposals, max bytes for work summary / statements / reasons).
- **`promotion.require_evidence`**, **`min_evidence_links`**, **`require_review`** — shared gates for materialize.
- **`promotion.auto_promote`**, **`min_support_count`**, **`min_salience`**, **`allowed_kinds`** — optional controlled auto-promotion (default **off**).

---

## Schema

**Baseline:** `control-plane/migrations/0001_memory_baseline.sql`.

**`candidate_events`:** `id`, `raw_text`, `salience_score`, `promotion_status`, `proposal_json`, `created_at`. The shipped baseline has no separate target- or context-id columns on this table.

---

## MCP — agent path

Thin proxy: **`curation_digest`** → **`POST /v1/curation/digest`**; **`curation_materialize`** → **`POST /v1/curation/candidates/{id}/materialize`**. Same semantics as HTTP; see [mcp-poc-contract.md](mcp-poc-contract.md).

| Tool | Arguments (MCP) | HTTP |
|------|-----------------|------|
| **`curation_digest`** | **`DigestRequest`** as **`arguments`**. Adapter requires non-empty **`work_summary`**. | `POST /v1/curation/digest` |
| **`curation_materialize`** | `{ "candidate_id": "uuid" }` (or **`id`**) | `POST` with candidate id in path |
| **`enforcement_evaluate`** | **`EvaluateRequest`** — see [api-contract.md](api-contract.md) subset + `internal/enforcement/types.go` | `POST /v1/enforcement/evaluate` |

For phase-by-phase timing, see [mcp-discipline-doctrine.md](mcp-discipline-doctrine.md).

**Flow:** digest → inspect `proposals` / `rejected` → **`curation_materialize`** with a **`candidate_id`** from digest output when ready.

**Pre-change proposals:** use **`enforcement_evaluate`** or HTTP **`/v1/enforcement/evaluate`** — orthogonal to digest/materialize.

---

## Example curl

```bash
# Dry-run (no rows)
curl -s -X POST http://127.0.0.1:8123/v1/curation/digest \
  -H 'Content-Type: application/json' \
  -d '{
    "work_summary":"Implemented digest endpoint and fixed tests.",
    "curation_answers":{"decision":"Use POST /v1/curation/digest for structured capture"},
    "options":{"dry_run":true}
  }' | jq .

# Persist then materialize (replace CANDIDATE_UUID)
curl -s -X POST http://127.0.0.1:8123/v1/curation/candidates/CANDIDATE_UUID/materialize | jq .
```

---

## Benefit tiers (optional)

See [pluribus-benefit-eval.md](pluribus-benefit-eval.md). Recorded MCP E2E proof (historical): [archive/memory-bank/plans/pluribus-curation-mcp-proof-results-20260323.md](../archive/memory-bank/plans/pluribus-curation-mcp-proof-results-20260323.md).
