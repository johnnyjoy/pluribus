# Pluribus — HTTP API and MCP surface index (canonical)

**Purpose:** One map from **implemented routes** (`control-plane/internal/apiserver/router.go`) to **Go request types**, **default MCP tools** (`control-plane/internal/mcp/tools.go`), and **capability class**. This is the **authoritative route list** for the shipped binary.

**Not duplicated here:** field-by-field JSON semantics for the **RC1 subset** — see [api-contract.md](api-contract.md) (**subset** contract: health, ready, minimal memory, compile, enforcement, run-multi). For any route, the **source of truth for JSON keys** is the `json` struct tags in the referenced `internal/*/types.go` file (handlers use `DisallowUnknownFields` where noted in handler code).

**Authority order:** [memory-doctrine.md](memory-doctrine.md) (product model) → this index + Go types (wire truth) → [api-contract.md](api-contract.md) (documented subset).

**LSP:** Optional **server-side** enrichment via in-process **gopls client** — not an editor protocol. See [pluribus-lsp-mcp-boundary.md](pluribus-lsp-mcp-boundary.md) and [control-plane/docs/lsp-features.md](../control-plane/docs/lsp-features.md). Recall/drift **do not require** LSP for correctness.

**MCP transport:** **`POST /v1/mcp`** (JSON-RPC). **Stdio** `pluribus-mcp` is a **thin HTTP proxy** to the same tools — same tool names, same JSON bodies as REST.

---

## Legend

| Column | Meaning |
|--------|--------|
| **MCP tool** | Name in **`tools/list`** when exposed; **`—`** = HTTP only (no MCP tool). |
| **Class** | **core** = memory / recall / enforcement / curation path; **support** = drift, evidence, contradictions, ingest, advisory; **meta** = health, MCP. |

---

## Global and MCP

| Method | Path | Request type | MCP tool | Class |
|--------|------|--------------|----------|--------|
| `GET` | `/healthz` | — | `health` | meta |
| `GET` | `/readyz` | — | — | meta |
| `POST` | `/v1/mcp` | JSON-RPC body | — | meta |

---

## Memory

| Method | Path | Request type (`control-plane/…`) | MCP tool | Class |
|--------|------|-----------------------------------|----------|--------|
| `POST` | `/v1/memories` | `internal/memory/types.go` — `MemoriesCreateRequest` | — | core |
| `POST` | `/v1/memories/search` | `MemoriesSearchRequest` | — | core |
| `POST` | `/v1/memory` | `CreateRequest` | `memory_create` | core |
| `POST` | `/v1/memory/promote` | `PromoteRequest` | `memory_promote` | core |
| `POST` | `/v1/memory/search` | `SearchRequest` | — | core |
| `POST` | `/v1/memory/pattern-elevation/run` | (handler-specific body) | — | support |
| `PUT` | `/v1/memory/{id}/attributes` | attributes payload | — | support |
| `POST` | `/v1/memory/{id}/authority/event` | authority event body | — | support |
| `POST` | `/v1/memory/expire` | expire request | — | support |

---

## Recall

| Method | Path | Request type | MCP tool | Class |
|--------|------|--------------|----------|--------|
| `GET` | `/v1/recall/` | Query → `CompileRequest` fields (see `handlers_getbundle.go`) | `recall_get` | core |
| `POST` | `/v1/recall/preflight` | `internal/recall/types.go` — `PreflightRequest` | — | core |
| `POST` | `/v1/recall/compile` | `CompileRequest` | `recall_compile` | core |
| `POST` | `/v1/recall/compile-multi` | `CompileMultiRequest` | — | core |
| `POST` | `/v1/recall/run-multi` | `RunMultiRequest` | `recall_run_multi` | core |

**GET `/v1/recall/`** accepts query keys: `retrieval_query` or `query`, `tags` (repeat or comma-separated), `symbols`, `max_per_kind`, `max_total`, `max_tokens` — see `internal/recall/handlers_getbundle.go`.

---

## Enforcement

| Method | Path | Request type | MCP tool | Class |
|--------|------|--------------|----------|--------|
| `POST` | `/v1/enforcement/evaluate` | `internal/enforcement/types.go` — `EvaluateRequest` | `enforcement_evaluate` | core |

---

## Drift

| Method | Path | Request type | MCP tool | Class |
|--------|------|--------------|----------|--------|
| `POST` | `/v1/drift/check` | `internal/drift/types.go` — `CheckRequest` | — | support |

---

## Curation

| Method | Path | Request type | MCP tool | Class |
|--------|------|--------------|----------|--------|
| `POST` | `/v1/curation/digest` | `internal/curation/types.go` — `DigestRequest` | `curation_digest` | core |
| `POST` | `/v1/curation/evaluate` | `EvaluateRequest` (curation package) | — | support |
| `GET` | `/v1/curation/pending` | **No query parameters** | — | core |
| `GET` | `/v1/curation/candidates/{id}/review` | — (path id); `CandidateReviewResponse` in `internal/curation/types.go` | — | core |
| `POST` | `/v1/curation/auto-promote` | optional `{}` body; `AutoPromoteResponse`; **403** if `promotion.auto_promote` is false | — | core |
| `POST` | `/v1/curation/candidates/{id}/materialize` | — (path id) | `curation_materialize` | core |
| `POST` | `/v1/curation/candidates/{id}/promote` | optional body per handler | — | support |
| `POST` | `/v1/curation/candidates/{id}/reject` | optional body per handler | — | support |

---

## Contradictions

| Method | Path | MCP tool | Class |
|--------|------|----------|--------|
| `POST` | `/v1/contradictions` | — | support |
| `POST` | `/v1/contradictions/detect` | — | support |
| `GET` | `/v1/contradictions` | — | support |
| `GET` | `/v1/contradictions/{id}` | — | support |
| `PATCH` | `/v1/contradictions/{id}/resolution` | — | support |

Request/response types: `internal/contradiction` (see handlers).

---

## Evidence

| Method | Path | Notes | MCP tool | Class |
|--------|------|--------|----------|--------|
| `GET` | `/v1/evidence` | `?memory_id=` (traceability + score) or `?kind=` list | — | support |
| `POST` | `/v1/evidence` | create | — | support |
| `GET` | `/v1/evidence/{id}` | metadata | — | support |
| `POST` | `/v1/evidence/{id}/link` | link to memory | — | support |

Types: `internal/evidence`.

---

## Ingest

| Method | Path | MCP tool | Class |
|--------|------|----------|--------|
| `POST` | `/v1/ingest/cognition` | — | support |
| `POST` | `/v1/ingest/{id}/commit` | — | support |

---

## Advisory episodes (similarity)

| Method | Path | MCP tool | Class |
|--------|------|----------|--------|
| `POST` | `/v1/advisory-episodes` | — | support |
| `POST` | `/v1/advisory-episodes/similar` | — | support |
| `POST` | `/v1/episodes/distill` | — | support |

Advisory only — not canonical recall authority per [episodic-similarity.md](episodic-similarity.md).

**Create:** `summary` (required), optional `source` (default `manual`), `tags`, `occurred_at`, `entities`, `related_memory_id`. **201** response always includes **`tags`** and **`entities`** arrays (possibly empty). When **`distillation.enabled`** and **`distillation.auto_from_advisory_episodes`** are **true**, the server may append **pending** distilled candidates (same rules as **`POST /v1/episodes/distill`**) after the write; failures there do **not** change the **201** status (logged only). See [episodic-similarity.md](episodic-similarity.md).

**Similar:** `query` (required), optional `tags`, `occurred_after` / `occurred_before` (inclusive on **effective** time), `entity` and/or **`entities`** (any overlap). If both time bounds are set, **`occurred_after` ≤ `occurred_before`** or **400**. **200** body: `{ "advisory_similar_cases": [ … ] }`.

**Distill:** `episode_id` (loads `advisory_episodes`) **or** inline **`summary`** (optional `tags` / `entities`). **200** → `{ "candidates": [ … ] }` each with `candidate_id`, `kind`, `distill_support_count`, `merged`, `source_advisory_episode_ids`, traceability. Pending rows **dedupe** by kind + normalized statement; repeats **merge** (see [episodic-similarity.md](episodic-similarity.md)). **403** if `distillation.enabled` is false. Does **not** write `memories`.

---

## Operator / CI (not HTTP routes)

| Entry | Purpose |
|-------|--------|
| `make eval` / `make stress-eval` | Go tests in `internal/eval` — see [evaluation.md](evaluation.md) |
| `make proof-rest` / `make proof-episodic` | Postgres + integration: **`proof-rest`** = all **`proof-*.json`** with two-pass determinism; **`proof-episodic`** = same JSON suite **plus** **`TestEpisodicProofSprintREST_Postgres`** (extended adversarial episodic chain). Wrapper: **`scripts/proof-episodic.sh`**. See [evaluation.md](evaluation.md), [evidence/episodic-proof.md](../evidence/episodic-proof.md). |

---

## `DigestRequest` (wire truth)

Fields on **`POST /v1/curation/digest`**: `work_summary` (required), `signals`, `curation_answers`, `evidence_ids`, `artifact_refs`, `options` — see `internal/curation/types.go`. **There are no `target_id` or `context_id` JSON fields** on this body in the current code.

---

## Database baseline

Embedded SQL: `control-plane/migrations/0001_memory_baseline.sql`, `0002_advisory_episodes_episodic.sql`, `0003_memories_occurred_at.sql` (applied on server boot). Canonical **`memories.occurred_at`** is optional event time; see [api-contract.md](api-contract.md). **`candidate_events`** columns: `id`, `raw_text`, `salience_score`, `promotion_status`, `proposal_json`, `created_at` — no separate migration file per table in-repo.

---

## See also

- [rest-test-matrix.md](rest-test-matrix.md) — REST-first behavioral tests and forbidden wire shapes  
- [mcp-poc-contract.md](mcp-poc-contract.md) — tool → HTTP mapping and agent flows  
- [api-contract.md](api-contract.md) — **RC1 subset** narrative contract  
- [pluribus-lsp-mcp-boundary.md](pluribus-lsp-mcp-boundary.md) — LSP vs MCP  
- [authentication.md](authentication.md) — API keys  
