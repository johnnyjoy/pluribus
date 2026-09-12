# Agent Memory Usefulness Harness

Phase 11B adds a **deterministic, CI-safe** harness that measures whether Pluribus memory improves agent-like task performance. This is **not** an LLM judge, **not** external SaaS, and **not** proof of real-world agent intelligence.

## Definitions

| Term | Meaning |
|------|---------|
| **Stored memory** | Row exists in the memory pool |
| **Relevant memory** | Recall compile returned the memory in the bundle |
| **Useful memory** | Recalled **and** explicitly used **and** output improved vs no-memory baseline **and** rubric passed |

**Memory helped** requires all of:

1. Task expects memory help (`requires_memory_help`)
2. No-memory baseline fails answer rubric
3. Memory run passes answer rubric
4. Expected memories recalled and used
5. No forbidden memory used, no lifecycle misuse

**Recalled-but-ignored is not helpful.**

## Cognitive memory principles (engineering constraints)

Phase 11C extends this harness with research-backed cognitive cases. See [cognitive-memory-engineering.md](cognitive-memory-engineering.md).

- **Encoding specificity** — `encoding_cues`, `EvaluateCueMatch`, under-encoded detection
- **Schema-based memory** — `memory_schema_type` hostile cases (constraint, failure, warning, procedure)
- **Retrieval practice** — false-positive helpfulness metrics; recalled-ignored and used-no-improvement cases
- **Context-dependent recall** — project/system scope rules in simulator
- **Interference control** — near-miss, high-authority, high-utility wrong-scope metrics
- **Reconstructive risk** — lifecycle_role, status, utility_score in manifest (gated)
- **Experience-following risk** — stale `wrong_count` / `outdated_count` suppression
- **Lifecycle governance** — refuted historical not current guidance

Commands:

```bash
make test-cognitive-memory-usefulness
make proof-cognitive-memory-benefit
```

Artifact: `artifacts/cognitive-memory-usefulness-benchmark.json`

## Architecture

```text
task fixture
  → no-memory simulator (empty facts)
  → memory REST compile → deterministic agent simulator → answer facts
  → memory MCP recall_context body → same compile → same simulator
  → score rubric + outcome_feedback.json artifact
```

Package: `control-plane/internal/agentusefulness/`

Fixtures: `control-plane/testdata/agent_memory_usefulness/`

Artifact: `artifacts/agent-memory-usefulness-benchmark.json`

## Deterministic agent simulator

The simulator is **not an AI**. It:

1. Reads recalled bundle items
2. Applies explicit selection rules (lifecycle, domain, refuted utility, fixture expected use/ignore)
3. Emits fact tokens from `fact_contributions`
4. Records **memory-use trace** with reasons

## MCP and REST/API parity

Both interfaces are primary. The harness runs:

- **REST** — direct `recall.Compiler.Compile` with fixture `CompileRequest`
- **MCP** — `buildMemoryContextResolveCompileBody` (same as `recall_context`) then compile

Parity cases require equivalent recalled memory IDs and answer outcomes.

## Outcome-linked feedback

Benchmark emits `outcome_feedback` events per run with:

- `task_id`, `run_id`, `interface`, `mode`
- `memory_id`, `memory_label`, `used`
- `answer_facts`, `score_pass`, `outcome_label` (`helped`, `hurt`, `irrelevant`, `misused`, …)

This is a **benchmark artifact** — not verified production outcome proof.

## Make targets

```bash
make test-agent-memory-usefulness          # unit/integration tests
make agent-memory-usefulness-benchmark     # write artifact (AGENT_MEMORY_USEFULNESS_BENCHMARK=1)
make proof-agent-memory-effectiveness      # gate + artifact (CI)
```

## Thresholds (safety hard zeros)

| Metric | Threshold |
|--------|-----------|
| `memory_help_rate` | ≥ 0.50 (help-eligible tasks only) |
| `memory_harm_rate` | 0 |
| `memory_misuse_rate` | 0 |
| `forbidden_memory_use_rate` | 0 |
| `lifecycle_misuse_rate` | 0 |
| `mcp_rest_parity_rate` | 1.0 (parity-required cases) |

## Honest limits

This harness **does not** prove:

- Real LLM agents improve in production
- Utility feedback equals outcome verification
- Semantic/vector recall solves usefulness

It **does** prove Pluribus can **test** memory/no-memory differences deterministically through MCP and REST/API.

See also: [recall-quality.md](./recall-quality.md), [agent-interface-boundary.md](./agent-interface-boundary.md), [reports/phase11-memory-usefulness-agentic-effectiveness-audit.md](./reports/phase11-memory-usefulness-agentic-effectiveness-audit.md).
