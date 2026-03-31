# Evaluation and verification

## How do I prove this system works?

**Answer:** at the **REST boundary**, with Postgres **+ pgvector**, on a **clean database**:

```bash
cd control-plane
TEST_PG_DSN='postgres://USER:PASS@HOST:PORT/DB?sslmode=disable' make proof-rest
```

That runs the **adversarial REST proof harness** (`internal/eval/scenarios/proof-*.json`): HTTP-only steps, structured `[PROOF]` logs, and **two full in-process passes** with an identical pass/fail signature (determinism). See [evidence/memory-proof.md](../evidence/memory-proof.md) for the receipt artifact.

Episodic advisory behavior is covered by multiple **`proof-episodic-*.json`** scenarios and **`TestEpisodicProofSprintREST_Postgres`** (including optional **automatic** distillation on advisory ingest when configured) — see [episodic-similarity.md](episodic-similarity.md) and the inventory in [evidence/episodic-proof.md](../evidence/episodic-proof.md).

**Canonical consolidation** (deterministic convergence on materialize when **`promotion.canonical_consolidation.enabled`** is true) is verified by **`internal/curation/consolidation_integration_test.go`** (integration-tagged; **`TEST_PG_DSN`**). The **`POST /v1/curation/candidates/{id}/materialize`** response exposes **`created`**, **`strengthened`**, and **`consolidated_into_memory_id`** so clients can see whether a new row was inserted or an existing canonical memory was reinforced; see [memory-doctrine.md](memory-doctrine.md) and [curation-loop.md](curation-loop.md).

**Full episodic pipeline proof** (explicit distill and/or **auto-from-advisory** when enabled in sprint servers → review → materialize → recall + enforcement boundary, plus adversarial Go scenarios): **`make proof-episodic`** from repo root or **`cd control-plane && make proof-episodic`** — runs `TestProofHarnessREST_Postgres` **and** `TestEpisodicProofSprintREST_Postgres`. Inventory and limits: [evidence/episodic-proof.md](../evidence/episodic-proof.md).

### Clean database (enforced)

Before boot, the integration test checks that **`public.memories` does not exist**. If it does, you get an immediate error: proof requires a **fresh** database so baseline migrate and scenarios stay deterministic. Use a new DB name or `dropdb`/`createdb`; see [scripts/proof-fresh-db.sh](../scripts/proof-fresh-db.sh).

There is **no** versioned upgrade path in the product yet (pre-release); boot only **replays** baseline SQL for greenfield DBs.

### Pre-change enforcement (bounded, rule-based)

`POST /v1/enforcement/evaluate` uses **fixed matchers** (e.g. `normative_conflict`, `anti_pattern_overlap`, `negative_pattern`) over normalized text — **not** open-ended natural-language reasoning. Every **`200`** response includes **`evaluation_engine`** and **`evaluation_note`** so clients cannot mistake the gate for general NL enforcement. Proof scenarios **`proof-enforcement-block-001`** (modeled conflict) and **`proof-enforcement-nl-not-enforced-001`** (unmodelled NL constraint → **allow** with empty **`triggered_memories`**) document the boundary.

### Testing doctrine (do not regress)

- **Terminology** for episodes vs candidates (ingest channel vs distill mode) is fixed in [memory-doctrine.md](memory-doctrine.md) (Terminology); wire field names **`source`** and **`pluribus_distill_origin`** are unchanged.

- **REST** is the **canonical service boundary** — proof of core memory/recall/enforcement behavior happens **here first**.
- **MCP** (`POST /v1/mcp`) is an **adapter** over the same product; **`internal/mcp`** unit tests cover tool → HTTP mapping. **Memory formation through MCP** (JSON-RPC on the real router, advisory ingest → auto-distill → curation tools, dedup, optional stdio smoke) is covered by **integration** tests in **`control-plane/cmd/controlplane/mcp_memory_formation_integration_test.go`** — run with Postgres, e.g.  
  `cd control-plane && TEST_PG_DSN="$TEST_PG_DSN" go test -tags=integration -count=1 -p=1 ./cmd/controlplane/... -run 'TestIntegration_HTTP_MCP|TestIntegration_stdio'`.  
  **`TestIntegration_HTTP_MCP_parityToolsRegistered`** asserts **`tools/list`** includes agent-parity tools (episodic similarity, explicit distill, curation review/reject/auto-promote, contradictions, evidence, memory relationships). **`TestIntegration_HTTP_MCP_memoryContextResolve`** and **`TestIntegration_HTTP_MCP_memoryLogIfRelevant`** cover the autonomous-memory path (deterministic recall wrapper + opportunistic ingest). **Dual-layer MCP scenarios (see [mcp-poc-contract.md](mcp-poc-contract.md) Dual-layer):** **A** — **`TestIntegration_HTTP_MCP_episodeAutoDistillAndCuration`** (ingest → auto-distill → pending, no explicit distill tool); **B** — **`TestIntegration_HTTP_MCP_memoryContextResolve`** (primary recall returns bundle + `mcp_context`); **C** — **`TestIntegration_HTTP_MCP_noAutoDistillNoCandidate`** vs parity tools (optional inspection without affecting default loop). Full memory behavior remains proven at REST + Postgres; MCP integration tests add **HTTP MCP** as a first-class proof surface.
- **LSP** is **optional enrichment** for recall; it is **not** the memory contract and **not** the primary proof surface.

### Why not `go test -count=2` on the same DB?

Boot applies embedded migrations to the live database. A second **process** test run against the same DB can hit **non-reentrant** DDL paths. The harness instead runs **two full scenario passes inside one test** (`RunProofHarnessRESTDeterminism`) on one server — same accumulation model, comparable signatures, no duplicate migrate.

### Semantic retrieval and fallback

Semantic candidate retrieval is **best-effort**. The **authoritative baseline** remains **lexical + tags + authority** ranking; vectors only expand candidates when the embed path succeeds.

- When semantic retrieval is **enabled** and the request includes situation text, **`POST /v1/recall/compile`** includes **`semantic_retrieval`** on the bundle: `attempted`, `path` (`semantic_hybrid` \| `lexical_only`), and **`fallback_reason`** when the path is lexical-only (e.g. `no_embedder`, `dimension_mismatch`, `embedding_failed`, `semantic_retrieval_disabled`).
- Logs: **`[SEMANTIC FALLBACK] reason=<code>`** for every lexical-only outcome after a semantic attempt; **`[SEMANTIC ERROR]`** when an embed or vector search **errors** during an attempted hybrid path (failure is still visible, then lexical continues).
- Match **`embedding_dimensions`** to your model and the DB column (**`vector(1536)`** in baseline migration).

Proof scenario **`proof-semantic-fallback-001`** locks this behavior (lexical search + compile debug + fallback reason).

### Database: Postgres + pgvector

Use an image that provides the **`vector`** extension (e.g. **`pgvector/pgvector:pg18`**, as in the repo root `docker-compose.yml`). Plain **`postgres:*-alpine`** without pgvector will fail at migration time.

---

## Other Makefile targets (supporting)

From **repository root**:

```bash
make eval
make stress-eval
make test
make regression
make test-drive
```

| Command | Purpose |
|---|---|
| `make test` | Core control-plane package tests (`go test ./...`) |
| `make eval` | Deterministic eval harness in `internal/eval` (not the adversarial REST proof suite) |
| `make stress-eval` | Stress-oriented eval execution/log output |
| `make regression` | **CI batch gate:** Docker Postgres + `go test -tags=integration -count=1 ./...` (includes YAML proof scenarios; memory relationship REST tests in `cmd/controlplane` when `TEST_PG_DSN` is set) |
| `make test-drive` | Fast confidence path (`test` + `eval`) |
| `cd control-plane && make proof-rest` | **Canonical memory-substrate proof** — REST-only `proof-*.json` + two-pass determinism |
| `make proof-episodic` | **Episodic lane stress proof** — all `proof-*.json` (two-pass determinism) + `TestEpisodicProofSprintREST_Postgres` adversarial subtests (see [evidence/episodic-proof.md](../evidence/episodic-proof.md)) |
| `scripts/proof-episodic.sh` | Same as `make proof-episodic` after checking `TEST_PG_DSN` is set |

---

## Artifacts

Eval/stress commands emit lightweight artifacts:

- `artifacts/eval-report.txt`
- `artifacts/eval-report.json`
- `artifacts/stress-report.txt`
- `artifacts/stress-report.json`

JSON artifacts are intentionally minimal pointers to text outputs for machine tooling.

---

## How to interpret eval/stress results (non-proof harness)

Eval output includes:

- aggregate trigger metrics
- explicit vs triggered arm comparisons
- behavior/recall pass-fail
- stress continuity/failure/pattern/drift summaries

Primary signal:

- harness should report passing scenarios and no regression from explicit to triggered behavior.

If failures appear:

1. identify failing scenario ID,
2. inspect per-scenario report section,
3. verify whether failure is extraction, recall ranking, behavior validation, or stress drift.

---

## API and integration testing

For host-managed DB integration checks:

```bash
# set TEST_PG_DSN first
make api-test
make integration-test
```

---

## References

- [rest-test-matrix.md](rest-test-matrix.md) — REST behavior matrix
- [proof-scenarios.md](proof-scenarios.md) — YAML scenario suite (runs in `make regression`)
- [pluribus-proof-index.md](pluribus-proof-index.md) — proof bundle index
- [pluribus-operational-guide.md](pluribus-operational-guide.md) — CI and operations
