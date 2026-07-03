# Plan: Enforcement for agent-driven curation chores

**Status:** draft plan (implementation not started)  
**Depends on:** agent-driven curation chores backend (migration 0018, `list_chores`, `resolve_chore`, `mcp_context.housekeeping`) — **deployed on orac / 10.1.1.79**  
**Principle:** *Optional in the recall bundle* (never ranked, never injected into the bundle) ≠ *optional for connected agents*. Rules, skills, hooks, MCP `initialize` instructions, and verify scripts are how we make housekeeping mandatory when chores exist.

---

## Goal

Extend the existing **recall → work → record** loop with a **housekeeping** step so visiting agents maintain the shared pool:

```text
wakeup_context (session) ─┐
recall_context (task)     ├─► if housekeeping / open chores → resolve_chore (when able) → work → record_experience
list_chores (explicit)  ─┘
```

Agents that cannot judge defer with an auditable reason in `record_experience`; agents that skip when `housekeeping` was present fail compliance (Phase 2b, below).

---

## Canonical source of truth (single edit, many wrappers)

All platform packs point at one file. **Phase A** updates this first; integration copies only add platform-specific wiring notes.

| File | Change |
|------|--------|
| [`integrations/pluribus-instructions.md`](../integrations/pluribus-instructions.md) | New **§ Housekeeping (when chores exist)** between Recall and Record |

### Proposed § Housekeeping (canonical text)

**If** `recall_context` / `wakeup_context` returns `mcp_context.housekeeping` or `housekeeping`, **or** `list_chores` returns non-empty `chores`:

1. Read the chore (type, statements, allowed `actions`).
2. **If you can judge:** call **`resolve_chore`** with `chore_id`, `action`, and **`agent_id`** (required; use a stable client id, e.g. `cursor:<user>`, `claude-code:<session>`, `vscode-copilot:<machine>`).
3. **If you cannot judge** (insufficient context, conflicting constraints, need human): do **not** vote randomly; note in the next **`record_experience`** why you deferred.
4. **Corroboration:** your vote alone does not apply the action; `min_resolvers` distinct agents must agree. A memory's **author** never counts toward the threshold — use a **different** `agent_id` than the row's author when possible.
5. **Actions:** `quarantine_review` → `release` (→ `pending`, never `active`) or `delete`; `contradiction` → `keep_subject` / `keep_related` / `coexist`; `duplicate_pair` → `consolidate` or `distinct`.

**Do not** skip housekeeping because substantive work feels urgent — one chore vote takes one tool call; ignoring poison or contradictions hurts every future agent.

**Do not** treat empty chores as failure — when the pool is clean, this step is a no-op.

---

## Phase A — Repo rules & skills (Cursor + workspace)

| Item | Action |
|------|--------|
| [`.cursor/rules/pluribus.mdc`](../../.cursor/rules/pluribus.mdc) | Add step 1b: if `housekeeping` present → `resolve_chore` or defer-with-reason |
| [`integrations/cursor/pluribus.mdc`](../../integrations/cursor/pluribus.mdc) | Mirror `.cursor/rules` (pack copy) |
| [`integrations/cursor/skills/pluribus/SKILL.md`](../../integrations/cursor/skills/pluribus/SKILL.md) | Extend loop to 4 steps; link `list_chores` / `resolve_chore` |
| [`integrations/cursor/skill.md`](../../integrations/cursor/skill.md), [`rules.md`](../../integrations/cursor/rules.md), [`prompts.md`](../../integrations/cursor/prompts.md) | Point at new § in `pluribus-instructions.md` |
| [`integrations/cursor/ENFORCEMENT-TIER.md`](../../integrations/cursor/ENFORCEMENT-TIER.md) | Document housekeeping as Tier-2 rule (same as recall/record); note Tier-3 gap unchanged |
| [`integrations/cursor/README.md`](../../integrations/cursor/README.md) | One paragraph on hive maintenance loop |

**New skill (optional but recommended):** `integrations/cursor/skills/pluribus-housekeeping/SKILL.md` — focused triggers, action matrix, deferral examples (mirrors Claude Code plugin skill pattern).

---

## Phase B — All integration packs (thin wrappers)

Each pack keeps **one** behavioral copy in `pluribus-instructions.md`; update table rows only where packs duplicate loop text locally.

| Pack | Files to touch |
|------|----------------|
| **Generic MCP** | `skill.md`, `skills/pluribus/SKILL.md`, `README.md`, `examples.json` (add chore resolve example) |
| **Claude Code** | `skill.md`, `skills/pluribus/SKILL.md`, `CLAUDE.template.md` |
| **Claude Code plugin** | `hooks/session-start.sh` (inject `housekeeping` from wakeup JSON if present), **new** `skills/resolve-chore/SKILL.md`, update `recall-context` skill cross-link, `README.md`, `plugin.json` skills list if enumerated |
| **Claude Desktop** | `skill.md`, `custom-instructions.template.md` |
| **VS Code / Copilot** | `skill.md`, `extension/README.md`, `github-copilot-instructions.template.md` |
| **Continue / OpenCode / Zed / OpenClaw** | respective `skill.md` + policy templates |
| **Root** | [`integrations/README.md`](../../integrations/README.md) — loop diagram includes housekeeping |

---

## Phase C — MCP server surface (bias at connect time)

Agents read `initialize.instructions` before tool selection. Align with canonical §.

| File | Change |
|------|--------|
| [`control-plane/internal/mcp/init.go`](../../control-plane/internal/mcp/init.go) `MemoryLoopInstructions` | Add bullet: when `housekeeping` present, call `resolve_chore` or `list_chores` |
| [`control-plane/internal/mcp/tool_registry.go`](../../control-plane/internal/mcp/tool_registry.go) | Fix `list_chores` description: remove "ignoring chores loses nothing" → "when chores exist, agents should resolve or defer with reason (see initialize instructions)" |
| [`control-plane/internal/mcp/behavior_test.go`](../../control-plane/internal/mcp/behavior_test.go), [`handler_test.go`](../../control-plane/internal/mcp/handler_test.go) | Assert instructions mention housekeeping / resolve_chore |

Regenerate doc: `UPDATE_MCP_TOOLS_DOC=1 go test ./internal/mcp/ -run TestMCPToolsDocMatchesRegistry` from `control-plane/`.

---

## Phase D — Doctrine & operator docs (terminology split)

| File | Change |
|------|--------|
| [`docs/memory-doctrine.md`](../memory-doctrine.md) § "The hive maintains itself" | Clarify: *optional in bundle* vs *mandatory for connected agents per pluribus-instructions* |
| [`docs/usage/ensuring-agent-usage.md`](../usage/ensuring-agent-usage.md) | §4 behavioral loop adds housekeeping step; §3 minimal expectations |
| [`docs/usage/snippets/agent-rules.md`](../usage/snippets/agent-rules.md) | Add housekeeping bullets (copy from canonical §) |
| [`docs/usage/snippets/mcp-usage-guidance.md`](../usage/snippets/mcp-usage-guidance.md) | Mention `housekeeping` field and chore tools |
| [`docs/mcp-usage.md`](../mcp-usage.md) | Extend recommended order |
| [`docs/get-started.md`](../get-started.md) | Loop string includes housekeeping when chores exist |

**Fix stale snippet:** `agent-rules.md` currently says there is no "delete the bad memory" story — update to quarantine/chore/soft-delete path.

---

## Phase E — Hooks & session injection

| Hook / entry | Change |
|--------------|--------|
| [`integrations/claude-code-plugin/hooks/session-start.sh`](../../integrations/claude-code-plugin/hooks/session-start.sh) | After wakeup JSON parse, if `.housekeeping` exists, append **### Pluribus housekeeping** block with full line + explicit `resolve_chore` nudge |
| [`integrations/claude-code-plugin/hooks/user-prompt-recall.sh`](../../integrations/claude-code-plugin/hooks/user-prompt-recall.sh) | If compile/wakeup preview exposes housekeeping, surface one line (bounded) |
| **VS Code extension** [`integrations/vscode/extension/src/extension.ts`](../../integrations/vscode/extension/src/extension.ts) | If extension injects session context from wakeup, include housekeeping (read-only; no auto-vote) |

Hooks **inject context only** — they do not call `resolve_chore` automatically (judgment stays with the agent; corroboration requires distinct agent ids).

---

## Phase F — Compliance & telemetry (measurement, not blocking)

Today [`compliance/evaluator.go`](../../control-plane/internal/compliance/evaluator.go) checks `recall_before_work`, `record_after_outcome`, enforcement. Extend in a **follow-up PR** (optional Phase F — can ship A–E first):

| Item | Proposal |
|------|----------|
| `compliance/classify.go` | Classify `resolve_chore`, `list_chores` as curation/housekeeping tools |
| Evaluator | New soft step `housekeeping_addressed_when_present`: if any recall/wakeup in session returned housekeeping (requires telemetry payload capture — **may need** redacted storage of `housekeeping` presence flag on recall events) |
| `compliance_evaluate` | Surface missing housekeeping in `missing_steps` as **advisory** first (warn, not fail) → tighten to fail after rules rollout |

**Prerequisite spike:** confirm whether MCP telemetry already stores enough of `recall_context` result to detect `mcp_context.housekeeping` without storing full bundle.

---

## Phase G — Verification gates (CI / scripts)

Extend existing integration verify scripts — **fail closed on missing enforcement text**.

| Script | New checks |
|--------|------------|
| [`scripts/integrations/verify-mcp-surface.sh`](../../scripts/integrations/verify-mcp-surface.sh) | `MemoryLoopInstructions` or generated doc mentions `housekeeping` / `resolve_chore` |
| [`scripts/integrations/verify-cursor-pack.sh`](../../scripts/integrations/verify-cursor-pack.sh) | Rules/skill contain housekeeping step |
| [`scripts/integrations/verify-claude-code-plugin.sh`](../../scripts/integrations/verify-claude-code-plugin.sh) | session-start mentions housekeeping; new resolve-chore skill exists |
| [`scripts/integrations/verify-generic-mcp.sh`](../../scripts/integrations/verify-generic-mcp.sh) | examples.json includes chore resolve sample |
| **New** `scripts/integrations/verify-housekeeping-enforcement.sh` | Grep canonical + N packs for required strings (`housekeeping`, `resolve_chore`, `agent_id`) |

Wire into [`Makefile`](../../Makefile) target alongside existing `verify-integrations-*`.

**Live proof script (optional):** extend `scripts/integrations/verify-mcp-surface.sh --live` or add `scripts/proof-chore-enforcement.sh` that quarantines a test row, asserts housekeeping in wakeup, votes twice, asserts resolved — **run against staging only**, not production orac.

---

## Phase H — `agent_id` convention (enforcement enabler)

Corroboration requires **distinct** agents. Document in canonical § and [`docs/pluribus-operational-guide.md`](../pluribus-operational-guide.md):

| Client | Suggested `agent_id` |
|--------|----------------------|
| Cursor | `cursor:<hostname>` or user-configured `PLURIBUS_AGENT_ID` |
| Claude Code plugin | `claude-code:<hostname>` |
| VS Code Copilot | `vscode:<hostname>` |
| CI / scripts | `ci:<job>:<run_id>` |

Optional: read `PLURIBUS_AGENT_ID` in MCP proxy from env for stdio adapter (`cmd/pluribus-mcp`) so clients don't forget.

---

## Implementation order

1. **Phase A** — `pluribus-instructions.md` + Cursor rules/skills (fastest feedback in this repo)
2. **Phase C** — MCP `initialize` instructions + tool description fix (all clients on connect)
3. **Phase B** — integration pack wrappers (parallelizable)
4. **Phase D** — docs/snippets/doctrine terminology
5. **Phase E** — hooks inject housekeeping text
6. **Phase G** — verify scripts + Makefile (prevents drift)
7. **Phase F** — compliance telemetry (after rules stable)
8. **Phase H** — agent_id convention + env helper

Estimated scope: **~25–35 files**, mostly copy + verify grep checks; **no backend behavior change** except MCP instruction string and tool descriptions.

---

## Acceptance criteria

- [ ] Any agent following **only** `pluribus-instructions.md` knows to call `resolve_chore` when `housekeeping` is present
- [ ] Cursor + Claude Code plugin session start surfaces housekeeping when orac has open chores
- [ ] `verify-housekeeping-enforcement.sh` passes in CI
- [ ] MCP `initialize.instructions` mentions housekeeping (unit tests updated)
- [ ] Doctrine distinguishes bundle-optional vs agent-mandatory
- [ ] Live orac test: quarantine → housekeeping in wakeup → two `agent_id`s → chore resolved (documented in operational guide, not automated against prod by default)

---

## Out of scope (this plan)

- Auto-calling `resolve_chore` from hooks (no backend LLM, no auto-vote)
- Lowering `chore_min_resolvers` on orac (operator config, not enforcement docs)
- New chore types or backend picker changes
- Cursor Tier-5 native tool blocking (documented gap in ENFORCEMENT-TIER.md remains)

---

## Open decisions (confirm before implementation)

1. **Deferral without vote:** Is `record_experience` mention sufficient, or require a no-op `list_chores` + explicit "deferred" tag?
2. **Compliance strictness:** Advisory missing step for 1 release, then fail?
3. **Dedicated skill vs inline:** One `pluribus-housekeeping` skill per platform vs only extending `pluribus` skill?
4. **Workspace rule at repo root:** Update [`.cursor/rules/pluribus.mdc`](../../.cursor/rules/pluribus.mdc) only, or also ship via Cursor plugin manifest when that lands?
