# Memory-Use Telemetry (Phase 11I–11J)

Phase **11J** adds automatic recall hooks and Postgres-backed persistence proof. See [automatic-recall-telemetry.md](automatic-recall-telemetry.md).

Operational memory-use telemetry bridges Phase 11H obedience evaluation to **persisted, queryable** MCP/REST agent-loop sessions.

## Core doctrine

- **Recall is not use.** Exposure of a recall bundle is recorded separately from memory-use decisions.
- **Agent self-report alone is not proof.** `obedience_passed` on evaluate is always computed by the deterministic Phase 11H evaluator.
- **Use requires linkage** to recalled memory IDs and output facts/actions.
- **Violations are persisted**, not only logged.
- **Utility updates must not blindly reward recall frequency.**

## Lifecycle

1. `recall_requested` — agent begins recall (optional external correlation)
2. `recall_bundle_returned` — `POST /v1/agent/telemetry/recall` or `agent_telemetry_record_recall`
3. `memory_decision_reported` — `POST /v1/agent/telemetry/decision` / `agent_telemetry_record_decision`
4. `output_reported` — `POST /v1/agent/telemetry/output` / `agent_telemetry_record_output`
5. `obedience_evaluated` — `POST /v1/agent/telemetry/evaluate` / `agent_telemetry_evaluate` (evaluator required)
6. `outcome_recorded` — evaluation + violations persisted
7. `utility_candidate_generated` — evaluator-validated candidates only; **not auto-applied**

## REST API (`/v1/agent/telemetry/*`)

| Endpoint | Purpose |
|----------|---------|
| `POST /session/start` | Start telemetry session (`interface`: `rest` \| `mcp`) |
| `POST /recall` | Persist recall exposure + bundle JSON |
| `POST /decision` | Persist per-memory decisions |
| `POST /output` | Persist output facts/actions + citations |
| `POST /evaluate` | Run deterministic evaluator; persist evaluation, violations, utility candidates |
| `GET /session/{session_id}` | Full session summary |
| `GET /memory/{memory_id}` | Per-memory aggregates |
| `GET /violations` | Query violations (optional `memory_id`, `violation_code`) |
| `GET /utility-candidates` | Query utility candidates (optional `memory_id`) |

## MCP tools

Structured JSON tools (parity with REST):

- `agent_telemetry_start_session`
- `agent_telemetry_record_recall`
- `agent_telemetry_record_decision`
- `agent_telemetry_record_output`
- `agent_telemetry_evaluate`
- `agent_telemetry_get_session`
- `agent_telemetry_get_memory`
- `agent_telemetry_get_violations`
- `agent_telemetry_get_utility_candidates`

## Persistence (Postgres migration `0012_agent_memory_use_telemetry.sql`)

- `agent_telemetry_sessions`
- `agent_recall_events`
- `agent_memory_decisions`
- `agent_output_events`
- `agent_obedience_evaluations`
- `agent_memory_use_violations`
- `agent_utility_candidates`

In-memory store mirrors Postgres for deterministic CI (same pattern as compliance).

## Utility candidates

Signal types: `used_correctly`, `ignored_correctly`, `historical_only_correctly`, `misused`, `unsafe_use`, `unsupported_claim`, `helped_output`, `harmed_output`.

Rules:

- Only evaluator-validated telemetry may generate candidates.
- Raw recall alone **cannot** generate positive utility.
- Raw self-reported use without evaluation **cannot** generate positive utility.
- `safe_to_apply` defaults to **false**; memory utility scores are **not** mutated automatically.

## Retention and privacy

Telemetry stores memory IDs, contract-field citations, and machine-checkable output facts — not raw LLM transcripts. Sessions are scoped by `session_id` and optional tags. No external SaaS or LLM judge is used.

## Proof gates

```bash
make test-agent-telemetry-persistence
make agent-telemetry-persistence-benchmark
make proof-agent-telemetry-persistence
```

Fixtures: `control-plane/testdata/agent_telemetry_persistence/cases.json` (28 hostile cases).
