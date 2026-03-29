# Pre-change enforcement

**Endpoint:** `POST /v1/enforcement/evaluate`  
**Config:** `enforcement` in `control-plane/configs/config.example.yaml` (RC1 default **on** when omitted; set **`enabled: false`** to disable).

## What it does

Evaluates a **bounded proposal** (text + optional intent/tags) against **binding** trusted memory only:

- Kinds: **constraint**, **decision**, **failure**, **object_lesson** (negative lessons use drift-style overlap).
- Rows must be **active**, **authority ≥ `min_binding_authority`**, and **applicability ≠ advisory**.
- Returns a single **`decision`** plus **`triggered_memories`** with **`reason_code`**, snippets, and optional **evidence** links.

## What it is not

- **Not** `/v1/drift/check` — drift returns violations/warnings for execution checks; enforcement returns a **gate decision** enum for pre-change workflows.
- **Not** `/v1/curation/evaluate` — that scores **candidate** text for digest/materialize.
- **Not** episodic similarity, embeddings, or vector retrieval.

## Request (v1)

```json
{
  "proposal_text": "We will migrate durable storage to SQLite.",
  "intent": "datastore",
  "tags": [],
  "rationale": ""
}
```

(Optional fields on **`EvaluateRequest`**: `intent`, `tags`, `rationale`, `goal`, `agent_id` — see `internal/enforcement/types.go` and [http-api-index.md](http-api-index.md).)

## Response (shape)

```json
{
  "decision": "block",
  "explanation": "Proposal introduces SQLite while trusted memory requires Postgres…",
  "triggered_memories": [
    {
      "memory_id": "…",
      "kind": "constraint",
      "authority": 9,
      "statement_snippet": "All durable application data must use Postgres…",
      "reason_code": "normative_conflict",
      "detail": "…",
      "evidence": []
    }
  ],
  "remediation_hints": ["Revise the proposal…"],
  "override": null
}
```

`block_overrideable` sets **`override`** with guidance for explicit review/override (no silent bypass).

## Automated proof (integration)

With **`TEST_PG_DSN`** set, **`go test -tags=integration ./cmd/controlplane -run TestIntegration_enforcementEvaluate_postgresVsSqlite`** exercises **`POST /v1/enforcement/evaluate`** against real Postgres (binding constraint → **`block`** / **`normative_conflict`**; unrelated proposal → **`allow`**). **`make regression`** runs this in Docker.

## Proof scenario (manual)

1. Ensure enforcement is not disabled (**`enforcement.enabled: false`** turns the gate off; omit the key for default-on).
2. Create a **constraint** with authority **≥ `min_binding_authority`** stating Postgres is required for durable data.
3. Call **`POST /v1/enforcement/evaluate`** with a proposal mentioning **SQLite** for datastore work.
4. Expect **`decision": "block"`** and a **`normative_conflict`** trigger.

## MCP (pluribus-mcp)

Thin proxy: **`enforcement_evaluate`** → **`POST /v1/enforcement/evaluate`** — **`arguments`** must match **`EvaluateRequest`** (**`proposal_text`** required). Same semantics as HTTP; see **Pre-change enforcement** in [mcp-poc-contract.md](mcp-poc-contract.md).
For lifecycle sequencing (memory-grounding recall → pre-change gate → memory curation), see [mcp-discipline-doctrine.md](mcp-discipline-doctrine.md).

## Limits

- v1 uses **heuristic** rules (normative phrases, token overlap, negative object lessons) — not a full policy language.
- **`proposal_text`** is capped (see server validation).
