# Episodic REST proof — evidence and inventory

Automated **REST-only** proof for the advisory episodic lane: ingest → similar (time/entity filters) → distill (**explicit** and/or **automatic** post-ingest when enabled in config — same candidate pipeline) → curation (pending, review, materialize, optional auto-promote) → recall / enforcement. Canonical **`memories`** remain the authority for recall and enforcement; advisory rows and pending candidates do not bind until materialized. **Vocabulary:** ingest channel (**`source`**) vs distill mode (**`pluribus_distill_origin`**) — [memory-doctrine.md](../docs/memory-doctrine.md) (Terminology).

## How to run

**Required:** Postgres DSN and a **clean** `public` schema (harness enforces this — see `internal/migrate/proof_clean.go`).

```bash
export TEST_PG_DSN='postgres://user:pass@host:5432/dbname?sslmode=disable'
make proof-episodic
```

Or:

```bash
cd control-plane && TEST_PG_DSN="$TEST_PG_DSN" make proof-episodic
```

Wrapper (same as `make proof-episodic`):

```bash
./scripts/proof-episodic.sh
```

**Docker (matches CI regression image):**

```bash
docker compose -p recall-regression -f docker-compose.regression.yml run --rm regression-runner \
  go test -tags=integration -count=1 -p 1 ./internal/eval/... \
  -run 'TestProofHarnessREST_Postgres|TestEpisodicProofSprintREST_Postgres'
```

Full `make regression` runs `./...` including these tests against ephemeral Postgres.

## What runs

| Layer | Test entry | Role |
|--------|------------|------|
| JSON scenarios | `TestProofHarnessREST_Postgres` | All embedded `internal/eval/scenarios/proof-*.json` (HTTP-only), **two-pass** pass/fail signature match |
| Go sprint | `TestEpisodicProofSprintREST_Postgres` | Stateful adversarial subtests (same router, shared DB, unique tags per case) |
| **HTTP MCP integration** | `TestIntegration_HTTP_MCP_*` / `TestIntegration_stdio_pluribusMcp_smoke` in **`cmd/controlplane/mcp_memory_formation_integration_test.go`** | **JSON-RPC** on **`POST /v1/mcp`** (same router as production): `mcp_episode_ingest` → advisory + auto-distill; curation tools; **no** canonical memory without materialize; **dedup** + optional **stdio** `pluribus-mcp` subprocess smoke. Requires **`go test -tags=integration`** and **`TEST_PG_DSN`**. |

Logging: episodic JSON scenarios and sprint lines use **`[EPISODIC PROOF]`**; other proof JSON uses **`[PROOF]`**.

## Scenario inventory (JSON, `suite: episodic`)

| ID | Proves |
|----|--------|
| `proof-episodic-advisory-001` | Ingest + time/entities; similar in/out of window; enforcement ignores advisory; recall does not surface episode text |
| `proof-episodic-time-window-bad-001` | Inverted `occurred_after` / `occurred_before` → **400** |
| `proof-episodic-time-boundary-equal-001` | Equal lower/upper time bound (single instant) still matches |
| `proof-episodic-distill-weak-001` | Weak/vague distill → no useful candidates |
| `proof-episodic-repetition-merge-001` | Repeated distill merges support count |
| `proof-episodic-pipeline-materialize-001` | Full chain: similar → distill → **GET review** → materialize → duplicate materialize **400** → recall → enforcement |
| `proof-episodic-backward-compat-001` | Sparse ingest (no `tags` / `occurred_at` / `entities` keys); wire shows empty arrays; similar by unique query |
| `proof-episodic-supersession-search-001` | `supersedes_id` on **POST /v1/memory**; superseded row still searchable; active search excludes superseded statement |

## Sprint subtests (Go, `episodic_proof_sprint_integration_test.go`)

| Subtest | Proves |
|---------|--------|
| `conflicting_evidence_two_distinct_candidates` | Competing episodes → distinct pending rows (no silent merge into one “truth”) |
| `time_skew_occurred_at_not_ingest_time` | Filtering uses `occurred_at`, not ingestion clock |
| `advisory_boundary_recall_ignores_episode_until_canon` | Recall ignores advisory text until materialize |
| `enforcement_stable_on_repeat` | Same enforcement input → same decision |
| `soak_distill_merge_idempotent_support_monotonic` | Repeated merges: support count non-decreasing |
| `inverted_time_window_similar_returns_400` | REST validation for bad time window |
| `weak_inline_distill_yields_no_candidates` | Short/vague inline distill → no candidates |
| `auto_distill_on_ingest_pending_without_explicit_distill` | With **`auto_from_advisory_episodes`**, ingest alone creates pending with **`pluribus_distill_origin":"auto"`**; recall still ignores until materialize |
| `enforcement_ignores_pending_distilled_candidate` | Distilled but not materialized rows do not bind enforcement |
| `recall_compile_identical_requests_match` | Identical compile requests → identical bodies |
| `review_supporting_episodes_cap_three_with_four_merges` | **≤3** `supporting_episodes` in review UI; explanation still cites **4** merged supports |
| `promotion_auto_promote_disabled_403` | Auto-promote off → **403** |
| `promotion_auto_promote_enabled_materializes_eligible` | Separate server with auto-promote on → recall sees canonical failure after batch |
| `entity_overlap_unrelated_summary_not_dominant` | Shared entity + A-tuned query: top hit is A; unrelated B absent from result set |
| `historical_occurred_at_found_despite_recent_ingest` | Old `occurred_at` discoverable in historical window |
| `duplicate_entities_normalized_ingest` | Duplicate entity strings normalize cleanly |
| `soak_recall_stable_three_iterations` | Recall stable over three identical compiles after promotion |

## What is **not** proved

- **Semantic/embeddings** for episodic similarity (lexical + tags + configured signals only).
- **LLM** distillation (keyword / rule distillation only).
- **Every** natural-language enforcement pattern — engine is intentionally bounded; scenarios use wording that hits shipped matchers after canon exists.
- **MCP / LSP** transport (REST is the contract under test here).
- **Fresh DB per scenario** — one reset per test process; isolation is via unique tags / `{{RUN_ID}}` in JSON runs.

## Failure triage

1. Exit code non-zero → search logs for **`[EPISODIC PROOF]`** / **`[PROOF]`** and the failing `scenario=` / `phase=`.
2. Schema error on boot → use a new database or allow `TEST_PG_RESET_SCHEMA` where documented for integration.
3. Determinism failure → compares aggregate pass signatures across two harness passes; inspect which scenario flipped.

## Implementation pointers

- Router: `control-plane/internal/apiserver/router.go`
- JSON loader: `internal/eval/proof_types.go`, runner: `proof_rest_runner.go`
- Sprint: `internal/eval/episodic_proof_sprint_integration_test.go`
- Review cap: `internal/curation/review_build.go` (`maxSupportingEpisodeSummaries`)
