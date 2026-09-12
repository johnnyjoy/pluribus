# Compliance Telemetry (Phase 2)

Pluribus records **MCP-level loop telemetry** and evaluates sessions against [agent-loop-contract.md](agent-loop-contract.md).

## What is logged

- MCP methods: `initialize`, `tools/list`
- Tool lifecycle: `mcp_tool_call_started`, completed/failed, plus loop roles (`recall_called`, `enforcement_called`, `record_experience_called`, …)
- Session metadata: client name/version, transport, repo root hint, correlation ID
- Redacted request summaries and stable request hashes

## What is not logged

- Raw API keys or secrets
- Full tool argument bodies (hashes + short summaries only)
- File edits outside MCP
- Chat transcripts

## Session model

- **HTTP MCP:** `X-Pluribus-Session-Id` header; generated on `initialize` if absent; returned as `result.pluribus.session_id`
- **Correlation:** `X-Pluribus-Correlation-Id` header or tool `correlation_id` field
- **Repo hint:** `X-Pluribus-Repo-Root` header

## Storage

Postgres tables (migration `0009_agent_loop_compliance.sql`):

- `agent_sessions`
- `agent_loop_events`
- `agent_loop_evaluations`

Tests use in-memory fallback when `DB` is nil.

## REST API

| Method | Path |
|--------|------|
| GET | `/v1/compliance/summary` |
| GET | `/v1/compliance/sessions` |
| GET | `/v1/compliance/sessions/{id}` |
| GET | `/v1/compliance/sessions/{id}/events` |
| POST | `/v1/compliance/evaluate` — body: `{"session_id":"<uuid>","recall_max_age_ms":optional}` |

## MCP tools

- `compliance_summary`
- `compliance_session_get`
- `compliance_session_events`
- `compliance_evaluate`

## CLI / Make

```bash
make test-agent-loop          # unit + hostile + scenario tests
make proof-agent-loop         # tests only (no Docker)
make proof-agent-loop-docker  # tests + Docker MCP smoke
make proof-agent-loop-all     # full proof
```

Script: `scripts/proof-agent-loop-compliance.sh`

## Evaluator limitations

- Based on **MCP-visible** calls only
- Failed tool calls do not count as compliant loop steps
- Stdio adapter direct REST proxy may not appear in HTTP MCP telemetry — prefer `POST /v1/mcp` for auditable sessions
- Does not score recall ranking or enforcement NLU quality

## Product honesty

Pluribus **measures** agent-loop compliance and **reports** missing steps. It does **not** guarantee agents follow the loop unless the client runtime enforces tool calls.
