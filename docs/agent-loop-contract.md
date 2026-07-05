# Pluribus Agent Loop Contract (Phase 2)

This document defines the **required agent memory loop** in testable terms. Pluribus **measures** adherence via MCP telemetry; it does **not** force client runtimes to call tools.

## Required loop

```text
Session or substantive task start  →  recall_context (or wakeup_context / memory_context_resolve)
Before material change           →  enforcement_evaluate
After meaningful outcome           →  record_experience (or mcp_episode_ingest)
```

**Formation (Phase 5):** `record_experience` is the canonical write path for agent learnings. Vague junk summaries are rejected; **`memory_create`** is admin/high-risk (authority cap; **active at capped authority** under shipped defaults — **`pending` only for explicit review holds or warehouse mode).

**Utility (Phase 7):** After recall, agents may submit structured feedback via **`memory_feedback`** (or `POST /v1/memory/{id}/feedback`). Event types: `helpful`, `harmful`, `wrong`, `outdated`, `irrelevant`. This updates **utility score** (separate from authority) and can influence recall ranking in a bounded way. Recall alone and duplicate writes do **not** increase authority or utility by default.

**Lifecycle recall (Phase 8):** Default **`recall_context`** uses **`recall_mode: current`** (**active** plus rare **`pending`**, weighted lower). For history, supersession, archival, or “what changed” questions, use **`recall_mode: historical`**. Historical results are labeled (`lifecycle_role`, `status`) and must not be treated as present guidance. See [memory-lifecycle-semantics.md](memory-lifecycle-semantics.md).

## 1. Session

A **session** is a correlated sequence of MCP-visible activity.

| Source | Counts as session |
|--------|-------------------|
| HTTP MCP `initialize` | Yes — server returns `result.pluribus.session_id` |
| Header `X-Pluribus-Session-Id` | Yes — continues or joins session |
| Stdio `pluribus-mcp` process | Partial — REST proxy calls are **not** HTTP MCP; telemetry is strongest on `POST /v1/mcp` |
| Stateless HTTP without headers | Server generates session on `initialize` |

**Weak cases:** repeated `initialize` without header creates new sessions; duplicate session IDs from different clients are stored with warnings at evaluation time; external IDE edits are invisible unless reflected in MCP calls.

## 2. Substantive work

**Substantive** (requires prior recall when visible to Pluribus):

- `recall_context`, `record_experience`, `enforcement_evaluate`
- Memory mutations (`memory_create`, `memory_promote`, …)
- Curation mutations (`curation_digest`, `curation_materialize`, …)
- Most non-diagnostic MCP tools that change or author state

**Not substantive:**

- `health`, `tools/list`, compliance read tools
- Read-only diagnostics (`curation_pending`, `memory_preflight_check`, …)
- Clarifying chat with no MCP tool calls (invisible to Pluribus)

## 3. Material change

**Material** (requires prior `enforcement_evaluate` in session when detected):

- Any mutating MCP tool (`record_experience`, `memory_create`, curation promote/materialize, …)
- High/critical risk admin tools

**Not material:**

- Read-only recall, enforcement itself, diagnostic lists

## 4. Meaningful outcome

**Requires `record_experience`** when session shows substantive MCP work:

- Fixes, decisions, failures, reusable lessons recorded via MCP
- Successful mutating tool calls imply an outcome marker for evaluation

**Do not record:** raw chat, trivial status, duplicates, unlabeled guesses, secrets.

## 5. Ordering and staleness

- Recall must precede substantive MCP work in the **same session**.
- Enforcement must precede material MCP mutations in the **same session**.
- Record must follow substantive work in the **same session**.
- **Default recall max age:** 60 minutes (`DefaultRecallMaxAge`). Older recall before material change → `recall_stale_before_material_change`.

## 6. Compliance statuses

| Status | When |
|--------|------|
| `compliant` | Recall + enforcement (if material) + record present; no missing steps |
| `partially_compliant` | Some loop steps present, others missing |
| `non_compliant` | Substantive/material work with no loop steps |
| `unknown` | No events or insufficient telemetry |
| `not_applicable` | Diagnostic-only MCP session |

## 7. Evidence

Evaluations include: event IDs, timestamps, session ID, tool names, redacted summaries, request hashes, enforcement decision (when present), transport, client name (from `initialize`).

## 8. Phase 2 limits

**Phase 2 measures loop adherence. It does not prove recall quality or enforcement intelligence.**

Clients can bypass Pluribus entirely (local edits without MCP). Documentation and rule packs **guide** behavior; **telemetry** is the audit source of truth for MCP-visible work.
