# Automatic Recall Telemetry

Phase 11J closes the gap between **explicit telemetry submission** (Phase 11I) and **automatic recall exposure observation**.

## Principle

Automatic recall telemetry records **exposure**, not use.

- Recall telemetry must **not** create positive utility by itself.
- Returning `recall_event_id` does **not** imply obedience.
- Agents must still submit decisions/output for evaluation.

## Default behavior (Option B)

Telemetry auto-enables when `telemetry_session_id` or `session_id` is supplied on a recall request.

- `telemetry: true` forces on; `telemetry: false` forces off.
- Without a session id, recall behavior is unchanged (backward compatible).

## REST surfaces (automatic hook)

| Surface | Telemetry field location |
|---------|-------------------------|
| `POST /v1/recall/compile` | `telemetry` on `RecallBundle` |
| `POST /v1/recall/compile-multi` | `telemetry` on `CompileMultiResponse` |
| `GET /v1/recall/` | `telemetry` on `RecallBundle` (query: `telemetry_session_id`) |
| `POST /v1/recall/wakeup` | `telemetry` on `WakeupResponse` |
| `POST /v1/recall/run-multi` | `telemetry` on `RunMultiResponse` when `memories_used` non-empty |

## MCP surfaces (via REST parity)

| Tool | Hook path |
|------|-----------|
| `recall_context` / `memory_context_resolve` | `POST /v1/recall/compile` + `mcp_context` correlation fields |
| `recall_compile` | `POST /v1/recall/compile` |
| `recall_get` | `GET /v1/recall/` |
| `wakeup_context` | `POST /v1/recall/wakeup` |
| `memory_recall_advanced` | `POST /v1/recall/compile` |
| `recall_run_multi` | `POST /v1/recall/run-multi` |

## Correlation fields

```json
{
  "telemetry_enabled": true,
  "telemetry_session_id": "<uuid>",
  "recall_event_id": "<uuid>",
  "recall_bundle_id": "bundle_<stable-hash>",
  "recall_request_hash": "<16-byte-hex>"
}
```

### Generation

- **`recall_event_id`**: new UUID per persisted recall event (idempotent replay returns existing id).
- **`recall_bundle_id`**: SHA-256 of canonical bundle JSON (`bundle_<hex>`).
- **`recall_request_hash`**: SHA-256 of `{session_id, request}` canonical JSON.
- **`telemetry_session_id`**: client-supplied or auto-created session on first auto recall.

## Lifecycle

1. Client supplies `telemetry_session_id` / `session_id` (or calls `POST /v1/agent/telemetry/session/start` first).
2. Client calls recall (REST or MCP).
3. Pluribus persists `agent_recall_events` automatically.
4. Response includes correlation fields under `telemetry`.
5. Client submits decisions/output referencing `recall_event_id`.
6. `evaluate` loads persisted bundle/recalled IDs (client cannot substitute a fake bundle).
7. Violations and utility candidates persist (`safe_to_apply=false` by default).

## Evaluation hardening

- Unknown `recall_event_id` → rejected.
- `recall_event_id` from another session → rejected.
- Evaluator uses **persisted** `recalled_memory_ids` and `recall_bundle_json`, not client-supplied recall payloads.

## Idempotency

Duplicate recall with the same session + `recall_request_hash` returns the existing `recall_event_id`.

## Postgres vs in-memory

- **CI unit proofs**: in-memory `memStore` (Phase 11I preserved).
- **Regression / `TEST_PG_DSN`**: `agenttelemetry.Repo` against migration `0012_agent_memory_use_telemetry.sql`.
- `make regression` runs Postgres proof gate inside the regression runner.

## Utility safety

- `recall_only_positive_utility_rate` must remain **0**.
- `auto_recall_auto_utility_mutation_rate` must remain **0**.

## Explicit non-scope

- No LLM judge.
- No utility auto-apply.
- No semantic/vector work.
