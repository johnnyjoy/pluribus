# Audit — Does Pluribus meet the hive-mind long-term-memory objective? (July 2026)

**Objective under audit:** Give AI agents durable long-term memory over MCP, shared hive-mind style: a lesson recorded by any agent is retrievable by any other agent connected to the same Pluribus server, from any point in time.

**Method:** Full code exploration of `control-plane/` (retrieval, formation, curation, MCP surface, eval), documentation review, and **live probes against the running server** (`http://10.1.1.79:8123`). Every finding below is backed by file references or live observation.

---

## Verdict

**The architecture is sound and much of the right code already exists — but the objective is currently NOT being met in production, primarily for operational reasons, not architectural ones.**

The single biggest problem is not missing code. It is that **the deployed server is running an old build that predates the relevance-first ranking work**, semantic retrieval is disabled, and the memory pool has no tagging discipline. The repo at HEAD already fixes the worst retrieval failure observed live.

---

## Evidence from the live server (2026-07-02)

1. **Recall returns wrong-project memories.** `POST /v1/recall/compile` with
   `retrieval_query: "Pluribus semantic retrieval pgvector embedding recall ranking configuration"`
   returned buckets filled almost entirely with **phprax** memories (a different project), each with
   `justification: {reason: "authority", score: 2.5–3.0}`. The 6 Pluribus-tagged memories that exist
   in the pool were mostly not surfaced.

2. **The current repo code would NOT do this.** A scorer probe test against HEAD
   (`internal/recall` `computeScoreComponents`, relevance-first + wrong-domain penalty) scored the
   same phprax memory **0.0000** and a Pluribus-relevant memory **3.4993** for that query. The
   deployed binary's authority-dominant scores (2.5/3.0, reason always `authority`) match the
   pre-Phase-4 formula. **Conclusion: deployment is stale.** `active.md` already says:
   *"Next: Deploy/restart control plane."* That step was never done.

3. **Semantic retrieval is dead in production.** Live response:
   `"semantic_retrieval": {"attempted": true, "path": "lexical_only", "fallback_reason": "no_embedder"}`.
   Shipped `configs/config.yaml` has `semantic_retrieval.enabled: false` and no embedding endpoint
   is configured. All recall is keyword/ILIKE based — paraphrases and synonyms miss.

4. **The pool is small and contaminated.** Lexical search caps at 20 rows; the pool is dominated by
   phprax memories with no `project:*` tags. Recall is intentionally unscoped
   (`internal/recall/compiler.go` line 184: *"Recall is not project-partitioned: search is unscoped"*),
   so without tagging discipline, every project's agents wade through every other project's memories.

---

## Gap analysis vs the objective

| # | Objective component | Status | Root cause |
|---|--------------------|--------|-----------|
| G1 | Relevant recall (right memory at the right time) | **Failing live; fixed at HEAD but undeployed** | Stale deployment; semantic off; no embedder |
| G2 | Cross-agent sharing (hive mind) | **Works** on a shared server (proven in `proof-scenarios/simulated-multi-agent-continuity.yaml`) | — |
| G3 | Cross-project isolation (no contamination) | **Failing by default** | Global pool by doctrine; tags are optional and client-supplied; integrations don't auto-tag `project:*` |
| G4 | Agent attribution ("who learned this") | **Missing** | `memories` has no author column; `record_experience` does not forward `agent_id`; single shared API key (`internal/httpx/auth.go`) |
| G5 | Semantic recall (paraphrase robustness) | **Built but off** | pgvector schema + hybrid path exist (`internal/memory/semantic*.go`, `migrations/0011`); config-gated, no embedding endpoint, no backfill pipeline |
| G6 | Memory quality over time (curation, decay) | **Partially built, not looped** | `auto-promote` is an opt-in batch endpoint (default off, nothing calls it); TTL expire is a manual HTTP call; no scheduler exists in the server; utility feedback demotion requires agents to volunteer `memory_feedback` |
| G7 | Cross-agent lesson fusion (dedup/merge) | **Exact-key only** | Dedup = `(kind, statement_key)` exact match; two phrasings of the same lesson become two rows (pattern Jaccard merge is the only exception) |
| G8 | Simple agent UX (loop actually happens) | **Weak** | 55 MCP tools, 4 alias pairs, no tier filtering (`internal/mcp/tool_registry.go`); loop is advisory in all integrations — measured, never enforced |
| G9 | Proof the memory helps real agents | **Missing** | Eval harness is deterministic/simulated (project admits: *"does not prove real LLM agents improve"*); in-process eval stubs return the full pool for any query, overstating retrieval quality |
| G10 | Docs coherence | **Sprawling** | ~101 docs; `pluribus-operational-guide.md` says semantic is "on by default" (wrong); broken links to `memory-bank/` and missing files |

**What is genuinely good and should not be rebuilt:** the memory-first ontology, the relevance-first scorer with hostile tests, the formation gates, the 28-case labeled recall benchmark with CI gates, the proof-scenario harness, the migration/upgrade runbooks, and the strict wire-truth discipline (`json` tags as contract).

---

## Remediation plan (prioritized)

### P0 — Operational (hours, no code; biggest single win)

1. **Deploy HEAD to the live server.** Follow `docs/local-server-upgrade-runbook.md`
   (backup → image build → migrate dry-run → restart → `scripts/smoke/local-post-upgrade-verify.sh`).
   Re-run the probe query above; phprax rows must drop out or score ≤0.05.
2. **Enable semantic retrieval** in the deployed config: `recall.semantic_retrieval.enabled: true` +
   embedding endpoint/model/key (OpenAI-compatible; `text-embedding-3-small` is the coded default).
3. **Adopt tagging discipline now:** every client (Cursor rule, VS Code orchestrator, Claude Code hook)
   passes `project:<repo-basename>` tags on both `record_experience` and `recall_context`, plus
   `repo_root`. This is the doctrine-compatible answer to contamination (soft scope, no partitions).
4. **Backfill tags on the existing pool** (~small; tag phprax rows `project:phprax`, pluribus rows
   `project:pluribus`) via `PUT /v1/memory/{id}/attributes`.

### P1 — Retrieval quality (the product's core)

5. **Embedding backfill job:** one-shot command (`cmd/`) that embeds all rows where
   `embedding IS NULL` or stale per `0011_embedding_metadata`. Currently memories written while
   semantic was off have no vectors, so enabling hybrid does nothing for old memories.
6. **Auto-inject `project:*`/`repo_root`-derived tag server-side at formation** when the client
   provides `repo_root` (kept as a tag, not a partition — doctrine-safe). Removes reliance on
   client discipline for G3.
7. **Wire the BM25/`pg_textsearch` lexical layer** (`internal/lexical`, currently experimental)
   into candidate retrieval as a replacement for the ILIKE keyword bridge, behind config. The eval
   harness for it already exists (`make pg-textsearch-eval`).

### P2 — Memory lifecycle becomes a loop, not a pile of endpoints

8. **In-server scheduler** (config-gated goroutine): periodically run TTL expire, advisory-reject
   prune, auto-promote batch, and embedding-staleness sweep. Everything it calls already exists;
   nothing calls it today.
9. **Semantic near-dup consolidation at write:** when a new memory embeds within cosine ≥0.93 of an
   existing active row of the same kind, reinforce (salience/support-count) instead of inserting.
   Extends the existing pattern-Jaccard merge to all kinds; closes G7.
10. **Automatic utility signal:** when recall surfaces a memory and the session's telemetry shows it
    was never applied (Phase 11I data already persists this), emit a low-weight negative utility
    event. Closes the loop the docs call "future gap".

### P3 — Agent experience and trust

11. **Tool tiering:** `PLURIBUS_TOOLS=core|standard|all` env (mirror Taskmaster's pattern). `core` =
    `wakeup_context`, `recall_context`, `record_experience`, `health` (+`enforcement_evaluate`).
    55 tools in every client's context window actively hurts loop compliance.
12. **Agent attribution:** forward `agent_id` from MCP session into advisory episodes and probationary
    memory payload (`payload.provenance.agent`), and issue per-agent API keys (table + middleware).
    Attribution, not partitioning — doctrine-safe.
13. **Docs pass:** fix `pluribus-operational-guide.md` semantic claim, dead `memory-bank/` links,
    missing-file references; add one canonical "state of the system" page generated from config.

### P4 — Proof it works (defensibility)

14. **Live-LLM eval:** small harness that runs a real cheap LLM twice on N tasks (with/without
    Pluribus recall) and diffs outcomes; publish as an artifact. This is the missing evidence for
    "memory makes agents better".
15. **True multi-host continuity smoke** (already on the post-release roadmap): two MCP clients on
    different machines, record→recall across them, in CI against a shared server.

---

## Delegation prompts for lower-cost LLMs

Each task below is self-contained. Give the model the repo at `/projects/pluribus`, the prompt, and nothing else. Tasks marked ⚙ need Docker/Postgres to verify; others are pure code+unit-test.

**Two project decisions constrain every task below — bake them into any delegated work:**

1. **Local embedder.** Semantic retrieval uses a local Ollama-compatible endpoint (`nomic-embed-text`, 768 dims), not a hosted API. It is **on** in the shipped `control-plane/configs/config.yaml` (`semantic_retrieval.enabled: true`, endpoint `http://host.docker.internal:11434/v1`; use `http://127.0.0.1:11434/v1` for bare-metal). The column dimension is fixed at 768 by `migrations/0015_local_embedder_dimensions.sql`. Any embedding work must degrade gracefully to lexical when the embedder is down (`[SEMANTIC FALLBACK]` logs) and must not assume 1536-dim OpenAI vectors.
2. **Shadow-low-authority posture with a harmful-advice quarantine gate.** New lessons from `record_experience` surface immediately at low authority (they are *not* held pending by default). The safety valve is the deterministic screen `formation.HarmfulAdviceReason` (`control-plane/internal/formation/harm_screen.go`), which forces safety-negating imperative advice to `status=quarantined` (stored, never recalled) at ingest in `internal/vet/service.go`. Do not reintroduce a blanket "everything pending" gate, and do not weaken or bypass the quarantine screen.

**Remediation status (2026-07-02):** the Phase 1–4 remediation already landed Tasks **B, C-equivalent, D, E** and the embedder decision. The prompts below are annotated `[DONE …]` where the work now exists in-tree; treat those as *audit/extend* prompts rather than greenfield. A/F/G/H remain open.

### Task A (P0.4) — Backfill project tags on existing memories — *small model OK*

> In the Pluribus repo, write a shell script `scripts/backfill-project-tags.sh` that: (1) calls `POST /v1/memories/search` against `$PLURIBUS_URL` for each keyword in a provided list, (2) for every returned memory id whose statement starts with a known project marker (e.g. "phprax" → tag `project:phprax`, "Pluribus"/"Phase" → `project:pluribus`), calls `PUT /v1/memory/{id}/attributes` adding that tag while preserving existing tags. Read the exact attributes wire shape from `control-plane/internal/memory/handlers.go` and `types.go` before writing the call. Idempotent: skip memories that already have a `project:*` tag. Include a `--dry-run` flag that prints planned changes. Do not modify Go code.

### Task B (P1.5) — Embedding backfill — *mid model* — **[DONE — audit/extend only]**

> **Already shipped** as an HTTP endpoint, not a CLI: `POST /v1/memory/embeddings/backfill` (see `control-plane/internal/memory/embedding_backfill.go`, wired in `internal/apiserver/router.go`), which scans `embedding IS NULL` rows, embeds with the configured local `Embedder`, and returns `{scanned,embedded,failed,skipped,remaining}`. Verified live 2026-07-02 (3 rows embedded, remaining=0). **If further work is delegated:** add a thin `cmd/embed-backfill/main.go` that loops the endpoint (or calls the service directly) for large pools, with a `--limit` batch flag and progress logging. Reuse the local-embedder config (768-dim `nomic-embed-text`); keep graceful lexical fallback. Add unit tests with a fake embedder. Read `docs/semantic-retrieval-local.md` and `docs/embedding-staleness-policy.md` first.

### Task C (P1.6) — Server-side repo tag injection — *mid model*

> In `control-plane/internal/similarity/` and `internal/mcp/proxy.go` (`buildAdvisoryEpisodeMCPBody`): when a `record_experience`/advisory-episode request includes `repo_root` (add it to the accepted MCP arguments), derive `project:<basename(repo_root)>` and append it to tags before persisting. Same for `recall_context` (`internal/mcp/context_resolve.go` `buildMemoryContextResolveCompileBody`): when `repo_root` is present, append the derived `project:*` tag to compile tags. This must remain a tag (soft scope) — do NOT add SQL partitions or required inputs; read the doctrine warning at the top of `internal/recall/compiler.go` and `internal/enforcement/types.go` first, and the guardrails tests in `internal/guardrails` must stay green. Add unit tests covering: basename derivation, no repo_root (no tag), tag dedup.

### Task D (P2.8) — Config-gated maintenance scheduler — *mid model* — **[DONE — audit/extend only]**

> **Already shipped** as `internal/curationloop/scheduler.go` (wired in `internal/apiserver/router.go`), gated by the `curation_scheduler` config section (`enabled`, `interval_minutes` default 60, `initial_delay_seconds` default 30). Each pass calls `memorySvc.ExpireMemories` and — only when `promotion.auto_promote` is true — `curationSvc.AutoPromoteBatchCount`, with count logging and context cancellation; unit tests in `scheduler_test.go`. Enabled in the shipped `config.yaml` and confirmed live (`[CURATION LOOP] scheduled interval=1h0m0s auto_promote=false`). **If further work is delegated:** add rejected-advisory pruning as a third per-pass job behind a `prune_rejected` boolean (service method in `internal/similarity`), and probationary time-decay demotion (the `probationary_expire_days` lifecycle setting already exists — wire it into the pass). Keep all jobs no-op unless their gate is set. Add fake-clock unit tests; document new sub-toggles in `configs/config.example.yaml`.

### Task E (P3.11) — MCP tool tiering — *small/mid model*

> In `control-plane/internal/mcp/tools.go` and `tool_registry.go`: add tier filtering to `ToolDefinitions()`. New env/config `PLURIBUS_TOOLS` with values `core` (wakeup_context, recall_context, memory_context_resolve, record_experience, mcp_episode_ingest, enforcement_evaluate, health), `standard` (core + curation_* + memory_feedback + memory_create + memory_promote + **memory_delete + memory_quarantine**, the C3 remediation tools added in this remediation), `all` (everything, default — preserves current behavior). Filter applies to `tools/list` only; `tools/call` still accepts all names (a hidden tool that is called should still work). Note the registry is now **57** tools (`tool_call_coverage_test.go` asserts the count) and includes the two remediation tools plus `agent_id` on `memory_create` — keep those in the right tiers. Add unit tests for each tier's exact tool list, and update `docs/mcp-tools.md` and `integrations/generic-mcp/README.md` counts. Existing tests including `internal/mcp/tool_call_coverage_test.go` and `spec_compliance_test.go` must pass.

### Task F (P3.13) — Documentation truth pass — *small model, no code*

> In the Pluribus repo `docs/`: (1) semantic retrieval is now **on** in the shipped `control-plane/configs/config.yaml` (`semantic_retrieval.enabled: true`, local `nomic-embed-text` at `host.docker.internal:11434`). Make every doc consistent with that — `pluribus-operational-guide.md`, `recall-quality.md`, and `semantic-retrieval-local.md` must agree that the default is on with a local embedder and lexical fallback when it is down. Do **not** revert docs to "disabled by default". (2) Find and fix all links in `docs/**/*.md` that point to `memory-bank/…` (moved to `archive/memory-bank/`) or to files that don't exist in the repo (e.g. `foundational-beta-benefit-baseline.md` referenced from `deployment-poc.md`) — repoint to archive, or remove the reference with a one-line note. (3) Confirm the durability doctrine reads "the database is sacred; upgrades are forward-only in-place migrations" (the fresh-DB doctrine was reversed in this remediation) and flag any doc still describing fresh-DB upgrades. (4) Produce a short report of every doc claim you could not verify against code. Do not change any Go code and do not change doctrine content in `memory-doctrine.md`.

### Task G (P2.9) — Semantic near-dup consolidation — *strong-mid model, ⚙*

> In `control-plane/internal/memory/service.go` Create path: after exact dedup and before insert, when semantic retrieval is enabled and an embedding was computed, run `SearchSimilar` (see `repo.go`) restricted to same kind + active status; if the best match has cosine similarity ≥ a new config `memory.dedup.semantic_consolidate_threshold` (default 0, meaning off), do not insert — instead reinforce the existing memory (increment salience via the existing `mergeSaliencePayload` mechanics in `salience.go`, append provenance of the new source into payload, and return the existing memory with a `consolidated: true` flag on the response). Follow the shape of the existing `tryMergeNearDuplicatePattern`. **Respect corroboration (decision 2 + Phase 3):** consolidation must not let an author self-promote — reuse the same-agent guard already in Create (compare `req.AgentID` to `existing.AgentID`; salience merges but authority only rises for a distinct agent). Use the local 768-dim embedder and keep graceful fallback (no consolidation when the embedder is down). Config default MUST be off. Add unit tests with a fake embedder and integration test under the `integration` build tag. Read `docs/memory-lifecycle-semantics.md` and keep `make test` green.

### Task H (P4.14) — Live-LLM benefit harness — *strong model designs, cheap model implements*

> Design doc + script: `scripts/live-llm-benefit-eval.sh` + `docs/experiments/live-llm-benefit.md`. Protocol: take 10 task fixtures from `control-plane/testdata/agent_memory_usefulness/tasks.json`; for each, call an OpenAI-compatible chat endpoint twice — arm A with only the task text, arm B with task text + the recall bundle from `POST /v1/recall/compile` — and grade both answers against the fixture's expected memory applications with a deterministic string-match rubric (no LLM judge in v1). Emit `artifacts/live-llm-benefit.json` with per-task and aggregate win rates. Environment: `PLURIBUS_URL`, `LLM_ENDPOINT`, `LLM_MODEL`, `LLM_API_KEY`. No Go changes.

**Sequencing (post-remediation):** B, D, and the embedder decision are **done**; C (repo-root → `project:*` tag) landed as part of H2 arg-acceptance — verify before re-delegating. Open work: **A** and **F** immediately (no risk), then **E** (tool tiering, now must place the two new remediation tools), then **G** (semantic near-dup, now vectors exist after backfill), **H** anytime.

**Do-not-delegate list (keep on a strong model or human):** changing ranking weights/formula in `internal/recall/relevance_scoring.go` (hostile-test-guarded, high regression risk), doctrine changes, migration authoring beyond additive columns, and anything touching `internal/guardrails` assertions.

---

## What NOT to do

- **Do not add project/hive/silo partitions to the schema.** The doctrine ("global pool, tag-first")
  is repeatedly enforced in code comments, guardrails tests, and docs. Contamination is solved by
  tags + ranking (P0.3, P1.6), not partitions — reversing doctrine would invalidate a large body of
  tests and docs for no retrieval benefit.
- **Do not rewrite the scorer.** HEAD's relevance-first scorer already passes hostile tests and fixed
  the observed failure. Deploy it before judging it.
- **Do not add more MCP tools.** The surface needs subtraction (tiering), not addition.
