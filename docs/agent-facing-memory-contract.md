# Agent-Facing Memory Contract (Phase 11F)

This document defines the *agent-facing contract* that Pluribus must expose when it recalls memory for AI agents via:

1. **MCP** (tool calls such as `recall_context`)
2. **REST/API** (compile endpoints such as `POST /v1/recall/compile`)

The contract is not “extra metadata.” It is required safety structure that allows an agent to decide:

- whether a recalled memory is **current guidance** vs **historical context**
- whether it is **within scope** (and therefore safe to act on)
- what **use instruction** to follow (and what to avoid)
- how to handle **quality** and **provenance**
- how to handle **supersession/refutation**

Pluribus does **not** assume the agent will read this doc. The Pluribus service must return a contract that is machine-checkable.

---

## Contract Envelope (shared semantics)

### REST/API representation

- The contract is returned inside the `RecallBundle` JSON returned by:
  - `POST /v1/recall/compile`
  - `POST /v1/recall/compile-multi`
  - `GET /v1/recall/` (bundle-shaped recall via query parameters)

### MCP representation

- The contract is returned by `recall_context` (and aliases such as `memory_context_resolve`) under:
  - `content[0].type == "json"`
  - `content[0].json.recall_bundle` (the recall bundle object)

- A legacy field is also present:
  - `content[0].text` (string containing JSON for backward compatibility)

**Contract safety rule:** agents must prefer `content[0].json` (structured JSON) rather than parsing only `content[0].text`.

---

## Memory Item Contract (required vs optional)

Each recalled `MemoryItem` must expose at least the following *contract fields*.

### Required fields (by evaluator)

The Phase 11F deterministic evaluator requires:

- `memory_id` / `id`
- `statement`
- `schema_type`
- `lifecycle_role`
- `status`
- **Current guidance fields** (when `lifecycle_role == current_guidance`)
  - `scope` (non-empty)
  - `negative_scope` (non-empty)
  - `use_instruction` (non-empty)
- **Historical context fields** (when `lifecycle_role == historical_context`)
  - `misuse_warning` (non-empty)
- `authority_basis` / provenance basis summary (non-empty)
- `source_type` / provenance source summary (non-empty)
- `utility_score` (non-nil)
- `quality_state` (non-empty)
- `quality_score` (non-nil)

### Supersession marker

- When `lifecycle_role == superseded_context`, `superseded_by` must be non-empty.

### Refutation marker

- When `lifecycle_role == refuted_context`, the `lifecycle_role` itself acts as the refutation marker in this phase.

---

## Lifecycle categories and agent treatment rules

The Phase 11F deterministic use-discipline rules interpret `lifecycle_role` as follows:

### `current_guidance`

The agent may guide action **only if all** of the following are true:

1. `quality_state == accept_active`
2. `use_instruction` is present (non-empty)
3. `negative_scope` is present (non-empty) and does **not** match any task tag
4. `scope` is present and matches at least one task tag

If any condition fails, the agent must:

- return **ignore** (or equivalent) and use the `Reason` from the deterministic decision output.

### `historical_context`

The agent may use it as background, but must not guide current action.

The contract must expose `misuse_warning`, which is used to prevent repeating mistakes.

### `superseded_context`, `refuted_context`, `archived_context`

The agent must treat these as **not current guidance** (ignore).

---

## Quality and utility semantics

The contract exposes:

- `utility_score`: a numeric score (non-nil in current contract enforcement)
- `quality_state` and `quality_score`:
  - `quality_state` must be non-empty
  - guidance is only considered safe when `quality_state == accept_active`

Quality defects are represented through contract evaluation (presence/absence) and are persisted at formation time into the payload so recall can expose them deterministically.

---

## Contract completeness evaluator and discipline (deterministic)

Phase 11F uses deterministic, non-LLM logic:

- `EvaluateBundleContract` checks contract completeness and emits:
  - `contract_passed`
  - `missing_required_fields`
  - `unsafe_omissions`

- `DecideUseDiscipline` emits:
  - `decision` in `{use, historical_only, ignore, unsafe}`
  - `reason` (machine-readable and stable)

Agents (or agent simulators in tests) must follow the discipline decision:

- **Never** treat `historical_only`, `ignore`, or `unsafe` as guidance for current action.

---

## Flattened text-only rejection (safety hardening)

If an MCP client parses only `content[0].text` (flattened text-only behavior),
contract evaluation must reject as:

- `flattened_text_only_memory`

This prevents accidental “JSON-looking text” from being treated as safe structured contract data.

