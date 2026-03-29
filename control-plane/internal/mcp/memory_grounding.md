# Pluribus memory-grounding protocol

**Canonical lifecycle (mnemonic):** **Recall → Act → Validate → Update → Repeat**.

Use this protocol **before substantive work**.  
Its purpose is to **reconstruct governing context from durable memory** across agents, machines, and sessions — not to redefine what memory is “about.” Memory is **global and tag-shaped**; **recall** is selective retrieval from that shared pool using **retrieval_query**, **tags**, and memory kind — not hidden partition IDs.

## Mission
Load relevant memory first, and only then begin substantive technical work.

## Rules
1. **Treat recalled governing memory as the planning baseline.**  
   In particular, pay close attention to:
   - governing constraints
   - decisions
   - known failures

2. **Do not propose substantive edits, migrations, architectural changes, or workflow changes until recall is complete.**

3. **If the situation or execution context changes materially, refresh recall once before continuing.**

## Required sequence

### Step 1 — Load memory
Call the appropriate recall path for the current situation.

Responses include **`agent_grounding`** (Continuity / Constraints / Experience as plain text) when supported — use it as the system-shaped summary; prompts alone do not enforce recall.

Preferred default:
- **`recall_get`** with **`retrieval_query`** (or **`query`**) plus **`tags`** when you have a clear situation description
- **`recall_compile`** with **`retrieval_query`** and **`tags`** when you need a POST body or richer options

Use a compile variant when tags or intent materially improve retrieval for the current work.

### Step 2 — Ground in recalled truth
Read the returned memory bundle and extract the items that govern the current work, especially:
- constraints that must be obeyed
- decisions that should not be rediscovered or contradicted
- failures that must not be repeated

Use these as the planning baseline.

### Step 3 — Act (substantive work)
Only after Steps 1–2 are complete may you:
- propose code edits
- plan migrations
- propose architecture
- make workflow-shaping technical decisions

### Step 4 — Validate risky proposals
For risky changes, run `enforcement_evaluate` and follow `validation.next_action`:
- `proceed` → continue
- `revise` → revise proposal, then re-run recall/validation
- `reject` → stop this action path

### Step 5 — Update + Repeat
Digest/materialize validated learning, then repeat the loop as context evolves.

### Step 6 — Refresh when context shifts
If the work meaningfully shifts direction, subsystem, or intent, refresh recall once before continuing.

## No guessing rule
- If continuity, constraints, or experience are incomplete, call recall again.
- Do not infer missing structure from chat momentum.
- Do not invent silo or ownership assumptions about where knowledge “lives.”

## Canon and authority model
When using Pluribus, keep these distinctions clear:

- **Governing memory**: authoritative enough to shape planning and block contradictions
- **Advisory memory**: informative, but not binding
- **Candidates**: proposed learnings, not canon
- **Evidence**: support for memory, not policy by itself
- **History/transcript**: not the same as memory

Do not confuse these classes.

## Anti-failure checks
Before continuing, ensure all of the following are true:

- You loaded recall before substantive recommendations.
- You reviewed governing constraints, decisions, and known failures.
- You are not about to contradict known memory truth due to recency or local chat momentum.

## Resource
See:
- `pluribus://discipline/lifecycle`
- `pluribus://discipline/canon-vs-advisory`
- `pluribus://discipline/history-not-memory`
- `pluribus://architecture/active-context-vs-durable-store`

---

## Pluribus doctrine (MCP)

Do not assume any project, task, or container.
This system is memory-first.
Canonical authority in-repo: `docs/memory-doctrine.md` (Recall repository).
Do not guess missing governing memory — recall again instead.
