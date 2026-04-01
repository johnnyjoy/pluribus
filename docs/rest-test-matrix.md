# Pluribus — REST boundary test matrix (canonical)

**Purpose:** Map **shipped HTTP routes** to **required behavior**, **forbidden wire shapes**, and **tests that lock the service boundary**. MCP and LSP are adapters; this document is about **REST as product truth**.

**If you only need one command:** *“Does the memory substrate behave as claimed?”* → **`cd control-plane && TEST_PG_DSN='…' make proof-rest`** (Postgres **+ pgvector**, clean DB). That is the **canonical behavioral proof** at the HTTP boundary. For the **advisory episodic** lane under stress, add **`make proof-episodic`** — [evidence/episodic-proof.md](../evidence/episodic-proof.md). This matrix maps routes to tests; full verification story: [evaluation.md](evaluation.md) and [evidence/memory-proof.md](../evidence/memory-proof.md).

**Authority:** [memory-doctrine.md](memory-doctrine.md) (model) → [http-api-index.md](http-api-index.md) (route/type map) → Go `json` tags + handlers → this matrix (what CI must prove).

**How tests run**

- **Default `go test ./...`:** package unit tests + **guardrails** (docs/matrix presence, ontology bans).
- **Integration (Postgres):** `cd control-plane && go test -tags=integration ./cmd/controlplane/...` with **`TEST_PG_DSN`** set. These hit **`apiserver.NewRouter`** end-to-end.
- **Proof harness (Postgres + pgvector):** embedded `internal/eval/scenarios/proof-*.json`, REST-only steps, `[PROOF]` logs, two-pass determinism — `make proof-rest` (see [evidence/memory-proof.md](../evidence/memory-proof.md)).

---

## Global rules (every REST test suite)

| Rule | Meaning |
|------|--------|
| **No container ontology on the wire** | Successful create/compile/search paths must **not** require `project_id`, `task_id`, `hive_id`, `workspace_id`, or `scope_*` partition IDs. Unknown JSON keys → **400** (`httpx.DecodeJSON` uses `DisallowUnknownFields`). |
| **Tags are soft context** | `tags` optional; retrieval uses **global pool** + tags / query, not hard partitions. |
| **Memory kinds are behavioral** | `state`, `decision`, `failure`, `constraint`, `pattern` (per `pkg/api`). |

---

## Priority 1 — core product

### Memory — `POST /v1/memory`, `POST /v1/memories`

| Aspect | Required | Forbidden | Primary tests |
|--------|----------|-----------|---------------|
| Create behavioral row | Valid `kind`, `statement`, `authority`; optional `tags`, optional `occurred_at` (RFC3339 event time) | Required container IDs | `cmd/controlplane` `TestIntegration_memories_createSearch`; `internal/memory` `TestHandlers_Create_*` |
| Typed relationships | `POST /v1/memory/relationships` + `GET /v1/memory/{id}/relationships`; `supersedes_id` on create records **`supersedes`** edge | Relationship-as-container / hidden ontology | `cmd/controlplane` `memory_relationships_integration_test.go` (integration) |
| Event time vs ingest | Optional `occurred_at`; recall recency uses event time when set | — | `cmd/controlplane` `TestREST_memories_occurredAt_createAndRecallRank` (integration); `internal/recall` `TestScore_recency_prefersOccurredAtOverUpdatedAt` |
| Unknown JSON keys | **400** | Silently ignored fields | `cmd/controlplane` `TestREST_memoryCreate_rejectsContainerOntologyJSON` |
| Batch create API | `POST /v1/memories` same semantics | — | `TestIntegration_memories_createSearch` |
| Search | `POST /v1/memories/search` by `query` + optional `tags` | Container-scoped search | `TestIntegration_memories_createSearch` |

### Memory — `POST /v1/memory/promote`

| Aspect | Required | Primary tests |
|--------|----------|---------------|
| Valid promote payload | Per `PromoteRequest` / handler validation | `internal/memory/handlers_promote_test.go` |

### Recall — `GET /v1/recall/`, `POST /v1/recall/compile`

| Aspect | Required | Forbidden | Primary tests |
|--------|----------|-----------|---------------|
| Situational recall | `retrieval_query` or `query`, optional `tags`, limits | Mandatory container scoping | `cmd/controlplane` `TestIntegration_restHealthReadyAndRecall`; `internal/recall/handlers_getbundle_test.go` |
| Compiled bundle shape | JSON includes continuity/constraints/experience groupings per `RecallBundle` | Opaque dump without structure | `cmd/controlplane` `TestREST_recallCompile_returnsShapedBundle` |
| POST compile | Same inputs as `CompileRequest` | — | `TestREST_recallCompile_returnsShapedBundle` |

### Recall — `POST /v1/recall/preflight`

| Aspect | Required | Primary tests |
|--------|----------|---------------|
| Valid body | JSON object (e.g. `{}` or `changed_files_count` / `tags`) | `cmd/controlplane` `TestREST_recallPreflight_returnsRiskShape` |

### Enforcement — `POST /v1/enforcement/evaluate`

| Aspect | Required | Primary tests |
|--------|----------|---------------|
| Binding memory affects proposal | `proposal_text` required; decisions reflect constraints/failures/decisions | `cmd/controlplane` `TestIntegration_enforcementEvaluate_postgresVsSqlite`; `cmd/controlplane` `TestREST_enforcementEvaluate_fullRouter` |
| JSON | `EvaluateRequest` only | `internal/enforcement/handlers_test.go` |

### Curation — `POST /v1/curation/digest`

| Aspect | Required | Forbidden | Primary tests |
|--------|----------|-----------|---------------|
| Digest | `work_summary`; optional `curation_answers`, `options.dry_run` | `target_id`, `context_id` on wire | `cmd/controlplane` `TestREST_curationDigest_dryRun`; `internal/curation/service_digest_test.go` |

### Curation — `GET /v1/curation/pending`, `GET /v1/curation/candidates/{id}/review`, materialize / promote / reject

| Aspect | Required | Primary tests |
|--------|----------|---------------|
| Pending list | No query params | `docs/curation-loop.md` + handler tests |
| Candidate review | **Read-only** JSON: `explanation`, `signal_strength` + `signal_detail`, `supporting_episodes` (≤3 summaries), `promotion_preview`; **no** memory writes | `cmd/controlplane` `TestREST_candidateReview_fields`; `internal/curation` `review_build_test.go` |
| Review isolation | Recall compile and enforcement unchanged by review; **no** new `memories` rows from GET review | `cmd/controlplane` `TestREST_candidateReview_isolation_noMemoryWrite` |
| Materialize | Path `{id}`; **`MaterializeOutcome`** (`created` / `strengthened` / `consolidated_into_memory_id`); validation + **`payload.pluribus_promotion`** / **`pluribus_consolidation`** trace | `internal/curation` `service_digest_test.go`; `internal/curation/consolidation_integration_test.go` (integration) |
| Materialize | Path `{id}` (legacy flows) | `cmd/controlplane` `TestIntegration_promoteCandidateToPattern` (service path) |

---

## Priority 2 — authority, patterns, run-multi

| Endpoint group | Purpose | Primary tests |
|----------------|---------|---------------|
| `PUT /v1/memory/{id}/attributes` | Constraint attributes | `internal/memory` lifecycle / handler coverage |
| `POST /v1/memory/{id}/authority/event` | Authority lifecycle | `internal/memory/lifecycle_test.go` |
| `POST /v1/memory/expire` | TTL + low-authority archive batch | `internal/memory/lifecycle_expiration_test.go` |
| `POST /v1/memory/pattern-elevation/run` | Pattern elevation job | `internal/memory/pattern_elevation_test.go` |
| `POST /v1/recall/compile-multi` | Multi-variant compile | `internal/recall` compiler tests; `cmd/controlplane` `TestREST_recallCompileMulti_minimal` |
| `POST /v1/recall/run-multi` | Orchestration | `internal/recall/handlers_runmulti_test.go`, `internal/runmulti` |

---

## Priority 3 — supporting / operator

| Endpoint group | Class | Primary tests |
|----------------|--------|---------------|
| Evidence, contradictions, ingest | Support | Package tests under `internal/evidence`, `contradiction`, `ingest` |
| Advisory episodes (`/v1/advisory-episodes`, `/similar`) | Ingest-time **`memory_formation_status`** (linked vs rejected); **`/similar`** ranks **non-reject** rows only; reject-bucket coverage in **`proof-ingest-rejected-experience-001`**; enforcement / recall integration tests use **isolated tags** or **reject-bucket** text where “no memory” is required; inverted time window **400** | `cmd/controlplane` `advisory_episodes_episodic_integration_test.go` (integration); `internal/similarity` unit tests; proofs **`proof-episodic-advisory-001`**, **`proof-ingest-accepted-memory-001`**, **`proof-ingest-rejected-experience-001`**, **`proof-episodic-time-window-bad-001`** in `make proof-rest` / `make proof-episodic` |
| Automatic distillation (post-advisory ingest) | Same keyword path as **`POST /v1/episodes/distill`** when **`distillation.auto_from_advisory_episodes`**; **`pluribus_distill_origin`** auto/manual/mixed; merge/suppression unchanged; ingest **201** if distill fails | `cmd/controlplane` `advisory_auto_distill_integration_test.go`; `internal/similarity/handlers_create_test.go` (`TestCreate_Advisory201WhenAutoDistillFails`); sprint subtest **`auto_distill_on_ingest_pending_without_explicit_distill`** in `internal/eval/episodic_proof_sprint_integration_test.go` |
| Episode distillation (`POST /v1/episodes/distill`) | Keyword distillation → `candidate_events`; pending **merge** on kind + statement key; repetition strengthens salience/support count; weak text → no candidates; not recall / not enforcement until materialized | `cmd/controlplane` `episode_distill_integration_test.go` (`TestREST_episodeDistill_consolidatesDuplicates`, …); `internal/distillation` unit tests; proofs **`proof-episodic-distill-weak-001`**, **`proof-episodic-repetition-merge-001`** |
| Episodic → canon chain (materialize + recall + enforcement) | End-to-end REST: distill → `GET …/review` → materialize → second materialize **converges** (canonical consolidation when enabled) or exact-dedup **reinforces** → recall surfaces statement → enforcement **`normative_conflict`** on modeled proposal; plus backward-compat ingest, equal time bound, supersession search, sprint cases — see [evidence/episodic-proof.md](../evidence/episodic-proof.md) | **`proof-episodic-*.json`**; **`TestEpisodicProofSprintREST_Postgres`** in `internal/eval`; `make proof-episodic` |
| Candidate review (`GET /v1/curation/candidates/{id}/review`) | Assistance only; deterministic explanation + bounded episode summaries + interpretable signal + promotion preview | `cmd/controlplane` `candidate_review_integration_test.go`; `internal/curation` unit tests |
| Controlled promotion (`POST /v1/curation/auto-promote`, `promotion_readiness` on pending) | Auto-promote **403** when disabled; when enabled, batch promotes only threshold-eligible rows; `[AUTO PROMOTE]` logs | `cmd/controlplane` `controlled_promotion_integration_test.go`; `internal/curation` `promotion_*_test.go` |
| `POST /v1/drift/check` | Support | `internal/drift` |
| `GET /healthz`, `GET /readyz` | Meta | `cmd/controlplane` `TestIntegration_restHealthReadyAndRecall`, `internal/httpx/ready_test.go` |

---

## Deferred (adapter-only truth)

| Layer | Note |
|-------|------|
| **MCP** (`POST /v1/mcp`) | Test as **thin adapter** once REST matrix is green — same JSON bodies as REST. |
| **LSP** | Optional enrichment; not the memory contract. See [pluribus-lsp-mcp-boundary.md](pluribus-lsp-mcp-boundary.md). |

---

## Related

- [http-api-index.md](http-api-index.md) — full route ↔ type ↔ MCP tool map  
- [api-contract.md](api-contract.md) — RC1 **subset** narrative only  
- [anti-regression.md](anti-regression.md) — CI guardrails  
