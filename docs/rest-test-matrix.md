# Pluribus — REST boundary test matrix (canonical)

**Purpose:** Map **shipped HTTP routes** to **required behavior**, **forbidden wire shapes**, and **tests that lock the service boundary**. MCP and LSP are adapters; this document is about **REST as product truth**.

**If you only need one command:** *“Does the memory substrate behave as claimed?”* → **`cd control-plane && TEST_PG_DSN='…' make proof-rest`** (Postgres **+ pgvector**, clean DB). That is the **canonical behavioral proof** at the HTTP boundary. This matrix maps routes to tests; full verification story: [evaluation.md](evaluation.md) and [evidence/memory-proof.md](../evidence/memory-proof.md).

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
| Create behavioral row | Valid `kind`, `statement`, `authority`; optional `tags` | Required container IDs | `cmd/controlplane` `TestIntegration_memories_createSearch`; `internal/memory` `TestHandlers_Create_*` |
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

### Curation — `GET /v1/curation/pending`, materialize / promote / reject

| Aspect | Required | Primary tests |
|--------|----------|---------------|
| Pending list | No query params | `docs/curation-loop.md` + handler tests |
| Materialize | Path `{id}` | `cmd/controlplane` `TestIntegration_promoteCandidateToPattern` (service path); extend HTTP materialize when needed |

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
| Evidence, contradictions, ingest, advisory-episodes | Support | Package tests under `internal/evidence`, `contradiction`, `ingest`, `similarity` |
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
