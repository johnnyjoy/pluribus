ARCHIVED — NOT ACTIVE SYSTEM TRUTH

---

# PRD double-check against HTML (AI Coding and Recall)

Text was stripped from `AI Coding and Recall.html` and saved as **`docs/ai-coding-and-recall-extracted.txt`** (canonical extract; HTML stripped). This note records a pass through that extract to verify the PRD in `scripts/prd.txt` matches the original goals and design. *(An older duplicate file `docs/ai-coding-and-recall-extract.txt` was removed in the 2026-03-22 doc-truth cleanup.)*

## Summary: PRD matches the HTML

The PRD is an accurate distillation. The following are confirmed against the extract.

### Goals and purpose

- **HTML:** "The goal is not 'create an AI mind.' The goal is: reduce rediscovery, contradiction, drift, and repeated errors across long tasks" — or "stay on target."
- **PRD:** Same framing: operational memory and recall for AI-assisted coding; "gifted-amnesiac" behavior without recall; solution = small file-based memory + retrieval order + evidence. ✓

- **HTML:** "Give the AI a small, file-based external memory made of rules, active state, decisions, and evidence, and force it to reload that memory before every meaningful coding action."
- **PRD:** Solution section matches. ✓

### Four memory primitives

- **HTML:** Constitution, Current State, Work Orders, Queryable Evidence; same "answers" (What must never drift? Where are we? What are we doing? How do we know?); minimal layout: constitution.md, active.md, decisions.md, workorders/NNNN.md, evidence/.
- **PRD:** Four primitives and minimal file layout match. ✓

### Curated memory, not diary

- **HTML:** Good memory = decision, reason, constraint, affected files, evidence, next action; bad = long rambly summaries, repeated prose, vague aspirations, "we discussed several options," giant transcripts; machine-recallable units.
- **PRD:** Same good/bad and machine-recallable format. ✓

### Retrieval order and hierarchy

- **HTML:** Load constitution → current active work order → recent decisions → evidence → then code; "Constitution > Current Work Order > Recent Decisions > Evidence > Codebase Search."
- **PRD:** Retrieval discipline and hierarchy match. ✓

### Intent-driven recall, not similarity

- **HTML:** "You need intent-driven recall, not similarity-driven recall"; immediate memory = assembled, best applicable, task-shaped; "Recall should be based on: task type, domain, impacted invariants, recent decisions."
- **PRD:** Deterministic recall, "compiled not guessed," recall engine with tags/authority/recency and limit per type. ✓

### Authority and design laws

- **HTML:** LAW (constitution) > DECISIONS > STATE > HYPOTHESIS; "No important state lives in chat"; "The model is stateless"; "Immediate memory is assembled"; "Promotion is selective"; "Evidence beats vibes"; "Speed is conditional."
- **PRD:** No chat state; deterministic recall; control-plane design doc has the same authority and design laws. ✓

### Stay on target (operationalized)

- **HTML:** Goal alignment, constraint adherence, state continuity, error memory, progress directionality; three mechanisms: Target Definition, Target Lock (reconstruct before action), Target Check (after action: violate constraints? contradict? repeat failure? move forward?); "Recall → Act → Validate → Update → Repeat."
- **PRD:** Orchestrator flow (load task → recall → LLM → drift → store → curate); drift detection; risk-based execution / preflight. ✓

### Drift

- **HTML:** Drift check after action; validate against constraints, prior decisions, known failures; POST /drift/check; violations list; "duplicate responsibility," "known failure reintroduced."
- **PRD:** Drift detection with constraints and failures; string match v1. ✓

### Curation and candidates

- **HTML:** Raw activity → candidate → validation → promotion → compression; do not write directly to long-term; salience/scoring (directive words, repeated mention, evidence, etc.); thresholds.
- **PRD:** Curation loop; promotion rules; optional cognitive control plane with candidate capture and promotion. ✓

### Control-plane and services

- **HTML:** Reasoner → Recall + Target + Drift → Memory + Curation + Pattern + Evidence → Tool Gateway → DB/evidence; memory-api, target-api, drift-api, recall compile, **-style CLI**; Go; one binary first.
- **PRD / design:** Same service map at a high level. **Current repo:** **`controlplane`** (single canonical server process; optional **`synthesis`** for run-multi only), **`pluribus-mcp`** (thin MCP → HTTP). There is **no** separate HTTP “tool API” binary; LSP-backed signals live in-process in **`controlplane`**. A dedicated shell CLI binary is **not** shipped today — use **`curl`** / HTTP clients or **MCP** for the same endpoints. ✓ (intent matches; CLI shape evolved.)

## Nuances in the HTML worth keeping

1. **"Immediate memory is a product, not a store"** — f(Intent, Long-term, Medium-term, Short-term) with ranking, filtering, compression, conflict resolution. PRD captures this as "recall engine" and "compiled context."
2. **Multi-speed architecture** — Fast path (cheap append candidate), promotion path (triggered), background consolidation (merge, compress, archive). PRD has "risk-based execution"; control-plane has curation thresholds; optional to add explicit "fast vs slow path" wording.
3. **Executive memory** — "How to choose what to recall; what kinds of memories matter for which task; how to bind recalled knowledge into action; what validation steps must occur before acting." That's the recall compiler + preflight + drift. Already implied; could be named in docs.
4. **"The LLM never owns truth"** — Memory plane owns truth; recall compiles; drift prevents stupidity; tool plane grounds. PRD and design doc align; good to keep stating this.

## Conclusion

The PRD in `scripts/prd.txt` correctly reflects the goals, memory model, retrieval discipline, stay-on-target loop, drift, curation, and service architecture described in the HTML. No material contradictions found. The extract for re-checks is **`docs/ai-coding-and-recall-extracted.txt`**.
