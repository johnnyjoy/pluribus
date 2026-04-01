# Control-plane

Go-based **authoritative memory layer** for agents: durable typed memory, **situational recall**, enforcement, curation, evidence, and drift. The product center is **global memory + recall + learning** — see [../docs/memory-doctrine.md](../docs/memory-doctrine.md). Secondary HTTP surfaces may exist for legacy correlation keys; they do **not** define the mental model. See also [../docs/control-plane-design-and-starter.md](../docs/control-plane-design-and-starter.md).

**Public quickstart (Pluribus):** [../docs/pluribus-quickstart.md](../docs/pluribus-quickstart.md) · **Architecture:** [../docs/architecture.md](../docs/architecture.md) · [../docs/pluribus-public-architecture.md](../docs/pluribus-public-architecture.md)

## First-run

On a clean machine (or CI):

**Option A — Docker (postgres + redis + control-plane API)**  
From the **repository root**: `docker compose up -d` starts Postgres, Redis, and **controlplane** on **8123** (uses `configs/config.yaml`; first run builds the image). Container startup uses `scripts/entrypoint.sh`; on boot, **control-plane** waits for DB, runs embedded baseline SQL (`migrations/*.sql`), and checks that core tables (e.g. `memories`) exist. **No host `psql`** required. Intended for a **fresh** Postgres database only.

Verify:

```bash
curl -sS http://127.0.0.1:8123/healthz   # liveness
curl -sS http://127.0.0.1:8123/readyz    # readiness (DB reachable + core schema present)
```

- **Databases only** (run API on host): `docker compose up -d postgres redis` — then start control-plane locally; **`Boot`** applies embedded SQL before serving.
- Compose service name is **`controlplane`** — no trailing `.` on CLI args (`controlplane.` → `no such service`).

**Option B — Local Postgres**

1. **Create the database**
   ```bash
   createdb controlplane
   ```

2. **Build binaries**
   ```bash
   make build
   ```
   Produces: `controlplane`, **`pluribus-mcp`** (optional stdio MCP → HTTP compat; canonical MCP is **`POST /v1/mcp`** on the API — see [../docs/mcp-service-first.md](../docs/mcp-service-first.md), [cmd/pluribus-mcp/README.md](cmd/pluribus-mcp/README.md), [../docs/mcp-poc-contract.md](../docs/mcp-poc-contract.md)).

3. **Start server**
   ```bash
   ./controlplane          # :8123
   ```

4. **Health checks**
   ```bash
   curl http://localhost:8123/healthz
   ```
## Config

Copy `configs/config.example.yaml` to `configs/config.local.yaml` (gitignored) and set `postgres.dsn`, `evidence.root_path`, and optionally `PLURIBUS_API_KEY` (HTTP API authentication) as needed; run with `CONFIG=configs/config.local.yaml ./controlplane`. The tracked `configs/config.yaml` is the Compose/Docker default (service hostnames `postgres` / `redis`). For LSP-based recall and drift (symbol overlap, reference-count risk), see [docs/lsp-features.md](docs/lsp-features.md). **Not** editor LSP: [../docs/pluribus-lsp-mcp-boundary.md](../docs/pluribus-lsp-mcp-boundary.md).

Startup knobs:

- `startup.db_wait_timeout_seconds` (default 60)
- `startup.db_wait_interval_millis` (default 1000)

## Canonical proof — memory substrate (REST)

The **REST API** is the **canonical system boundary** for proving core behavior. **`make proof-rest`** runs embedded adversarial scenarios (`internal/eval/scenarios/proof-*.json`) **only over HTTP**, with **two in-process full passes** and a matching pass/fail signature (determinism). **MCP** and **LSP** are **not** the primary proof surfaces. **`POST /v1/enforcement/evaluate`** responses include **`evaluation_engine`** and **`evaluation_note`** (rule-based matcher, not general natural-language reasoning).

```bash
# From this directory (control-plane/). Postgres must have pgvector; database must be empty (no public.memories).
TEST_PG_DSN='postgres://USER:PASS@HOST:PORT/DB?sslmode=disable' make proof-rest
```

The harness **errors early** if `public.memories` already exists. Local reset: [../scripts/proof-fresh-db.sh](../scripts/proof-fresh-db.sh).

Receipts and limitations: [../evidence/memory-proof.md](../evidence/memory-proof.md). Operator narrative: [../docs/evaluation.md](../docs/evaluation.md).

**Episodic stress proof:** `make proof-episodic` — same DSN and clean-DB rules as **`proof-rest`**, plus **`TestEpisodicProofSprintREST_Postgres`**. Scenario inventory: [../evidence/episodic-proof.md](../evidence/episodic-proof.md).

**Semantic retrieval:** Best-effort **hybrid** when embeddings work; **lexical + tags + authority** stay the baseline. When semantic is enabled but the query embed path is skipped or fails softly, the server logs **`[SEMANTIC FALLBACK] reason=…`** and sets **`semantic_retrieval`** on **`POST /v1/recall/compile`** (`path`, `fallback_reason`). **`[SEMANTIC ERROR]`** covers hard failures on the embed/vector path (still visible, then lexical). DB column is **`vector(1536)`** — align `embedding_dimensions` with your model.

## REST regression checks (CI + optional host DB)

- **CI batch gate (repo root):** `make regression` — the **`recall-regression`** Compose stack spins up Postgres **without host ports**, runs `go test -tags=integration -count=1 ./...` inside a builder image, then `down -v` **only** that stack (does not tear down your dev `docker compose up` stack). GitHub Actions also runs **`go test ./...`** in **`control-plane/`** — see [`.github/workflows/ci.yml`](../.github/workflows/ci.yml).
- **Package + handler tests:** `go test ./...`
- **REST integration tests on a host-managed DB (optional):** `TEST_PG_DSN='postgres://.../controlplane?sslmode=disable' go test -tags=integration -v ./cmd/controlplane -run TestIntegration_rest` — config path defaults to `configs/config.example.yaml` from the module root (or set `CONFIG=/abs/path/to/your.yaml`). Integration tests skip if `TEST_PG_DSN` is unset.
- **YAML proof scenarios (continuity / benefit receipts):** definitions in [`proof-scenarios/`](proof-scenarios/); suite `TestIntegration_proofScenarioSuite` runs inside **`make regression`**. Authoring: [`docs/proof-scenarios.md`](../docs/proof-scenarios.md). These complement **`make proof-rest`**; they do **not** replace it as the **canonical REST invariant harness** for the memory substrate.

## Example workflow (memory-first)

After first-run, **write memory**, **recall**, **enforce**, then optional curation. Use **tags** and **retrieval_query** to shape the situation; see [../docs/memory-doctrine.md](../docs/memory-doctrine.md) and [../docs/pluribus-memory-first-ontology.md](../docs/pluribus-memory-first-ontology.md).

```bash
# 1) Write governing memory (shared pool)
curl -sS -X POST http://localhost:8123/v1/memory -H 'Content-Type: application/json' \
  -d '{"kind":"constraint","authority":8,"statement":"No duplicate query builders for list endpoints.","tags":["demo","api","go"]}' | jq .

curl -sS -X POST http://localhost:8123/v1/memory -H 'Content-Type: application/json' \
  -d '{"kind":"failure","authority":9,"statement":"Duplicated query builders caused inconsistent pagination.","tags":["demo","api"]}' | jq .

# 2) Recall (compile) — tags + retrieval_query
curl -sS -X POST http://localhost:8123/v1/recall/compile -H 'Content-Type: application/json' \
  -d '{"tags":["demo","api"],"retrieval_query":"resume listing endpoint work"}' | jq .

# 3) Pre-change enforcement — proposal_text required
curl -sS -X POST http://localhost:8123/v1/enforcement/evaluate -H 'Content-Type: application/json' \
  -d '{"proposal_text":"Add a second custom SQL query path for list endpoints to move faster."}' | jq .
```

Some advanced flows (evidence linkage, certain curation paths) may still use **optional** correlation keys in the HTTP layer for legacy foreign keys; they are **not** part of the memory doctrine and are **not** required for recall or enforcement above.

**Recall compile, drift check, curation** — use `curl` (or any HTTP client) against `/v1/recall/compile`, `/v1/drift/check`, `/v1/curation/pending` with JSON bodies. Examples:

**GET recall bundle** (read-only; same compile path as `POST /v1/recall/compile`; URL must include trailing slash on some clients):

```bash
curl -sG 'http://localhost:8123/v1/recall/' \
  --data-urlencode 'tags=demo,api' | jq .
```

```bash
curl -s -X POST http://localhost:8123/v1/recall/compile -H 'Content-Type: application/json' \
  -d @recall-request.json | jq .

curl -s -X POST http://localhost:8123/v1/drift/check -H 'Content-Type: application/json' \
  -d @drift-request.json | jq .

curl -s "http://localhost:8123/v1/curation/pending" | jq .
```

**Curation digest (structured capture → materialize)** — `POST /v1/curation/digest` (optional `dry_run`; **`work_summary`** required), `POST /v1/curation/candidates/{id}/materialize`; config under `curation.digest_*` and `promotion.*`. See [../docs/curation-loop.md](../docs/curation-loop.md).

```bash
curl -sS -X POST http://localhost:8123/v1/curation/digest -H 'Content-Type: application/json' \
  -d '{"work_summary":"Shipped digest API with tests. Enough length for server validation.","curation_answers":{"decision":"Prefer dry_run for previews."},"options":{"dry_run":true}}' | jq .
```

**Pre-change enforcement** — `POST /v1/enforcement/evaluate`: compare a bounded **proposal** to **binding** trusted memory (high authority, non-advisory kinds); returns `allow` / `require_review` / `block` / `block_overrideable`. Not the same as **`/v1/drift/check`** (string heuristics + negative pattern overlap) or **`/v1/curation/evaluate`** (candidate salience). Config **`enforcement.*`** (RC1 default on; set **`enabled: false`** to disable). See [../docs/pre-change-enforcement.md](../docs/pre-change-enforcement.md).

**Advisory episodic similarity (optional)** — `POST /v1/advisory-episodes` and `POST /v1/advisory-episodes/similar` (lexical + tag “similar cases”); **subordinate** to canonical recall. Config **`similarity.*`** (shipped default **on**; set **`enabled: false`** to disable). See [../docs/episodic-similarity.md](../docs/episodic-similarity.md). Automated REST proofs: embedded **`proof-episodic-*.json`** (`make proof-rest`) and **`make proof-episodic`** — [../evidence/episodic-proof.md](../evidence/episodic-proof.md).

**LSP-backed recall (Phase 8)** — requires **`lsp.enabled: true`** in config and **gopls** available to the server for auto symbols / **`reference_count`**. See [docs/lsp-features.md](docs/lsp-features.md).

```bash
# Compile: auto-fill symbols from documentSymbol (omit "symbols" or send [])
curl -s -X POST http://localhost:8123/v1/recall/compile -H 'Content-Type: application/json' \
  -d '{"repo_root":"/abs/path/to/repo","lsp_focus_path":"internal/foo.go"}' | jq .

# Run-multi: same LSP fields forwarded to compile-multi; check debug.orchestration for lsp_recall_* flags
curl -s -X POST http://localhost:8123/v1/recall/run-multi -H 'Content-Type: application/json' \
  -d '{"query":"Implement feature safely","repo_root":"/abs/path/to/repo","lsp_focus_path":"internal/foo.go"}' | jq .
```

See [docs/ai-recall-cursor-focus.md](../docs/ai-recall-cursor-focus.md) for Cursor integration and focus-layer context (POC verify flow: [docs/cursor-verify-recall.md](../docs/cursor-verify-recall.md)).

### Cognitive ingest (MCL — audit + normalized facts; M3 reinforces duplicates)

Accepted ingests persist `canonical_fact_extractions` and return `canonical_facts`. **M3:** duplicate hash → **reinforce** (cap 1.0), `merge_actions.reinforce`. **M4:** same subject + high token-Jaccard on predicate/object → **similar_unify** (batch or prior DB row with smaller hash); opposing or divergent facts in one request → `debug.conflicts_detected` (ingest still accepted). **M5:** load `temp_contributor_profiles.trust_weight` by **`temp_contributor_id`** (default **1.0**); reject **noise** if any fact fails `confidence × trust_weight` floor or total reasoning-trace character minimum (`debug.trust_weight_applied`). **M6:** each accepted fact gets **`priority_score`** (0–1) from signal/frequency/recency/agreement; see `debug.priority_formula_version` (`m6-v1`). **M7 (optional bridge, default off):** set `ingest.auto_promote: true` in config, then either send **`"propose_promotion": true`** on cognition (with both gates) or **`POST /v1/ingest/{ingestion_id}/commit`** after review; see `debug.promotion`. There is **no** silent auto-promote.

```bash
curl -s -X POST http://localhost:8123/v1/ingest/cognition -H 'Content-Type: application/json' \
  -d '{
    "temp_contributor_id":"cursor-agent-1",
    "query":"Why use transactions?",
    "reasoning_trace":["User asked about ACID","Checked prior decisions"],
    "extracted_facts":[{"subject":"api","predicate":"must","object":"use transactions for writes","confidence":0.9}],
    "confidence":0.85,
    "source_refs":["wo-123"],
    "context_window_hash":"sha256:..."
  }' | jq .
```

By default, ingest does **not** create `memories` rows. Use **`/v1/memory/promote`**, run-multi promotion when policy allows, or the **M7** gated paths above when `ingest.auto_promote` is enabled.

### Backend synthesis (optional run-multi LLM)

Default posture: **`synthesis.enabled: false`** — the **client or agent LLM** should perform synthesis when possible. To have the control-plane generate run-multi variant text in-process, enable **`synthesis`** and choose **`ollama`**, **`openai`**, or **`anthropic`** (see [docs/backend-synthesis.md](docs/backend-synthesis.md)). There is **no** separate reasoner service or `/v1/reasoning` route.

### Authoritative run-multi + promotion

```bash
curl -s -X POST http://localhost:8123/v1/recall/run-multi -H 'Content-Type: application/json' \
  -d '{"query":"Implement feature safely","merge":true,"promote":true}' | jq .

# With optional evidence UUIDs (see Phase 9.2):
curl -s -X POST http://localhost:8123/v1/recall/run-multi -H 'Content-Type: application/json' \
  -d '{"query":"Implement feature safely","merge":true,"promote":true,"evidence_ids":["EVIDENCE_UUID"]}' | jq .

curl -s -X POST http://localhost:8123/v1/memory/promote -H 'Content-Type: application/json' \
  -d '{"type":"decision","content":"Use one canonical query path","confidence":0.9}' | jq .

curl -s -X POST http://localhost:8123/v1/memory/promote -H 'Content-Type: application/json' \
  -d '{"type":"decision","content":"Use one canonical query path","confidence":0.9,"evidence_ids":["EVIDENCE_UUID"],"require_review":false}' | jq .
```

**Phase 9.1 — promotion policy:** `promotion` in config (`configs/config.example.yaml`) supports **`min_promote_confidence`** (0–1; **0** = off). When set (e.g. `0.45`), run-multi **`confidence`** must be ≥ that value after merge/signal gates before **`POST /v1/memory/promote`** runs.

**Evidence-in-recall (optional):** With **`recall.evidence_in_bundle.enabled: true`**, each **`MemoryItem`** may include **`supporting_evidence`** (compact **`EvidenceRef`** rows). See [../docs/evidence-in-recall.md](../docs/evidence-in-recall.md). Default **off**; Redis cache keys include evidence settings.

**Evidence (9.2):** `POST /v1/recall/run-multi` and **`POST /v1/memory/promote`** accept optional **`evidence_ids`** (UUID array). Each ID must reference an **`evidence_records`** row the server can associate with the request; after a successful promote, links are written to **`memory_evidence_links`**. Config **`require_evidence`**, **`min_evidence_links`**, **`min_evidence_score`** gate run-multi promotion (see **`debug.promotion_decision.gates`**). Rejection codes include **`evidence_required`**, **`evidence_invalid`**, **`evidence_score_low`**, **`evidence_policy_unavailable`** (score gate without evidence service).

**Scoring alignment (9.3):** Optional **`min_policy_composite`** (0–1; **0** = off) applies after merge/signal/confidence and evidence gates: a weighted blend of **`run-multi confidence`**, **normalized merge `total_signal`** (`total_signal / signal_norm_divisor`, capped at 1; divisor **0** in config means **15** at evaluation time), and **`evidence_avg_score`** when evidence was scored (nil/missing IDs → **0** for the evidence term). Default weights **0.4 / 0.3 / 0.3** when all three **`weight_*`** are zero. **`debug.promotion_decision.policy_inputs`** echoes **`run_confidence`**, **`total_signal`**, **`evidence_avg_score`**; **`gates`** may include **`policy_composite`** and **`min_policy_composite`**. Rejection code **`policy_composite_low`** when the blend is below the threshold.

**Review queue (9.4):** Config **`require_review: true`** (or **`POST /v1/memory/promote`** with **`"require_review": true`**) creates promoted memories with **`status`** **`pending`** (see **`api.StatusPending`**). Default **`require_review: false`** preserves **active** rows. **`POST /v1/memory/search`** defaults to **`status: active`** — pending promoted rows do not appear until approved (e.g. **`Repo.UpdateStatus`** → **`active`** in a future/admin flow or direct DB). **`PromoteResponse`** includes **`status`**; run-multi **`debug.promotion_decision`** includes **`gates.require_review`** and, on success, **`memory_status`**. **Decay:** existing **`ttl_seconds`** / **`deprecated_at`** on **`memories`** remain the operator surface for expiration.

**Historical plan index (archival, non-normative):** see `archive/memory-bank/plans/`. Quick reference:

| `debug.promotion_decision.rejection_code` | When |
|------------------------------------------|------|
| `ok` | Promoted successfully |
| `promote_not_attempted` | `promote` false |
| `merge_required` | `promote` true but `merge` false |
| `merge_empty` / `merge_fallback` / `merge_drift` | Merge output or drift gate |
| `signal_low` | Below high-signal threshold |
| `confidence_below_minimum` | Below `min_promote_confidence` |
| `evidence_required` / `evidence_invalid` / `evidence_score_low` / `evidence_policy_unavailable` | Evidence gates (9.2) |
| `policy_composite_low` | Below `min_policy_composite` (9.3) |
| `promoter_unconfigured` | No `MemoryPromoter` |
| `promote_failed` / `promote_declined` | Memory service returned error or declined |

**Strict promotion profile (example)** — override in env-specific YAML; all keys optional:

```yaml
promotion:
  require_evidence: true
  min_evidence_links: 1
  min_evidence_score: 0.5
  require_review: true
  min_promote_confidence: 0.45
  # min_policy_composite: 0.55   # optional 9.3 composite gate
```

Responses with **`"promote": true`** include **`debug.promotion_decision.rejection_code`** (see table above) alongside human **`reason`**.

With **`merge": true`**, the response includes **`debug.merge`**: `segments_in`, `conflicts_dropped`, **`uniques_capped`** (Phase 5.2), and **`attribution`** lines (`agreement` / `unique` / `dropped_conflict` with contributing variants).

Optional **Phase 5.2–5.3** request fields (all off by default): **`merge_strict_conflicts`** (stricter grey-zone conflict pairs, e.g. should vs should-not), **`merge_max_unique_bullets`** (cap uniques after filters), **`merge_dedupe_similar_uniques`** (collapse near-duplicate uniques), **`merge_drop_unique_similar_to_agreement`** (0–1, e.g. `0.85` drops uniques redundant with `[CORE AGREEMENTS]`).

### Slow-path + multi-compile (Phase 7)

- **LSP auto symbols (Phase 8.1):** With **`lsp.enabled`**, **`POST /v1/recall/compile`** and **`POST /v1/recall/compile-multi`** can omit **`symbols`** when **`repo_root`** and **`lsp_focus_path`** are set; the server calls gopls **documentSymbol** and fills names (cap **`lsp.auto_symbol_max`**). If the client sends a non-empty **`symbols`** array, it is **not** replaced.
- **Reference count (Phase 8.2):** When there is symbol overlap with pattern payloads (**`matched_symbols`**), the bundle may set **`reference_count`** to the max reference count across matched symbols (per-symbol count capped by **`lsp.reference_expansion_limit`** when &gt; 0). Requires **`repo_root`**, **`lsp_focus_path`**, and **`lsp.enabled`**. See [docs/lsp-features.md](docs/lsp-features.md).

- **`POST /v1/recall/compile-multi`**: optional **`changed_files_count`** — when **`slow_path.enabled`** and you do **not** set **`slow_path_required`**, the server runs the same risk/slow-path logic as **`POST /v1/recall/preflight`** (including Redis preflight cache when configured). Optional **`slow_path.extra_variants_when_slow`** adds N to the variant count while slow-path is active (still capped at three default strategies).
- **`POST /v1/recall/run-multi`**: optional **`target_id`**, **`context_id`**, **`variants`**, **`strategy`**, **`tags`**, **`symbols`**, **`max_per_kind` / `max_total` / `max_tokens`**, **`changed_files_count`** (passed through to compile-multi for that inference), plus explicit **`slow_path_required`** / **`slow_path_reasons`** / **`recommended_expansion`**. **Phase 8.3:** same LSP fields as compile-multi — **`repo_root`**, **`lsp_focus_path`**, **`lsp_focus_line`**, **`lsp_focus_column`** — forwarded to the runner’s compile-multi request. **`debug.orchestration`** includes **`lsp_recall_repo_root_set`**, **`lsp_recall_focus_path_set`**, **`lsp_recall_focus_position_set`** (booleans only). With **`merge": true`**, the post-merge drift check uses the same **`slow_path_required`** / **`tags`** behavior as per-variant drift (and a second drift round-trip when the drift service returns **`requires_followup_check`**); see **`merge_drift_slow_path`** in orchestration when slow-path was applied to that check.

## Build

```bash
make build
# Or build a single binary:
go build -o controlplane ./cmd/controlplane
```

**Docker image** (from repo root, context `control-plane/`):

```bash
docker build -t recall-controlplane -f control-plane/Dockerfile control-plane
```

## Run

```bash
CONFIG=configs/config.local.yaml ./controlplane
# Or with the example config (no auth):
./controlplane
```

Then: `curl http://localhost:8123/healthz`

## Scripts

- **scripts/migrate.sh** — Obsolete stub (exits with a message): schema apply happens in **server boot** only.
- **scripts/dev-up.sh** — Prints first-run steps (create DB, make build, start servers, health checks).
