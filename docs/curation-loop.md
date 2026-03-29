# Curation digest loop (structured capture → pending → materialize)

Pluribus can turn post-work signals into **pending candidate rows** with structured JSON, then **materialize** them into durable memory (and optional evidence links), using the same **promotion** gates as other promotion paths (`promotion.*` in config).

**Wire truth:** `control-plane/internal/curation/types.go` (`DigestRequest`, `DigestResult`, …) and handlers in `internal/curation/handlers.go`. **Full route list:** [http-api-index.md](http-api-index.md).

---

## Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/v1/curation/digest` | Classify `curation_answers` + `work_summary` into `proposal_json` rows (or `dry_run` only). |
| `GET` | `/v1/curation/pending` | List **all** pending candidates (**no query parameters**). Entries with `proposal_json` include `structured_kind` and `statement_preview`. |
| `POST` | `/v1/curation/evaluate` | Salience evaluate a single text (`EvaluateRequest` in curation package — **`text`** field). |
| `POST` | `/v1/curation/candidates/{id}/materialize` | Create memory from `proposal_json`, link evidence IDs, mark candidate `promoted`. |
| `POST` | `/v1/curation/candidates/{id}/promote` | Alternate promote path (empty body or payload per handler — object-lesson flows). |
| `POST` | `/v1/curation/candidates/{id}/reject` | Reject candidate. |

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
- **`promotion`** config (`require_evidence`, `min_evidence_links`, `require_review`) applies the same way as run-multi promotion: e.g. materialize can require evidence IDs or create **`pending`** memory when `require_review` is true.
- **`object_lesson`** proposals are not materialized here — use **`POST .../promote`** with a full object-lesson payload when the handler allows.

---

## Config (`curation` + `promotion`)

See `control-plane/configs/config.example.yaml`:

- **`curation.digest_*`** — bounds for digest (max proposals, max bytes for work summary / statements / reasons).
- **`promotion.require_evidence`**, **`min_evidence_links`**, **`require_review`** — shared gates for materialize.

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
