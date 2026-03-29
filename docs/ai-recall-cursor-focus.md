# AI Recall — Cursor Focus Reference

Short reference extracted from **AI Coding and Recall** (and the PRD) to keep Cursor aligned with the Recall task. For the full rule, see [.cursor/rules/recall-focus.mdc](../.cursor/rules/recall-focus.mdc).

---

## 1. Retrieval order (session ritual)

Every coding session should load context in this order:

1. **Constitution** — stable project law (what must never drift).
2. **Current active work order** — what we’re doing right now.
3. **Recent decisions** relevant to the domain/task.
4. **Evidence** for affected files (tests, logs, diffs, benchmarks).
5. **Codebase search** only after the above.

**Hierarchy:** Constitution > Current Work Order > Recent Decisions > Evidence > Codebase Search.

---

## 2. Memory types (explicit)

| Type        | Purpose                |
|------------|-------------------------|
| decision   | Chosen path             |
| constraint | Rule                    |
| pattern    | Reusable success        |
| failure    | Known bad outcome       |
| evidence   | Proof artifact          |

Good memory: decision, reason, constraint, affected files, evidence, revisit condition.  
Bad memory: long rambly summaries, repeated prose, vague aspirations, giant transcripts.

---

## 3. Immediate memory pipeline

Immediate memory is **not stored**; it is **assembled**:

1. **Retrieve** candidates (by task, tags, domain).
2. **Score** (authority, recency, tag overlap, evidence strength).
3. **Compose** (situation-shaped relevance under authority).
4. **Prune** (drop weak, superseded, contradictory).
5. **Bind** to current task.

Result = “best applicable memory package” for the current step. Ordering must prevent anecdote from overriding law.

---

## 4. Stay on Target

- **Before acting:** Load goal, constraints, recent decisions, state.
- **After acting:** Did we violate constraints? Contradict decisions? Repeat a known failure? Make forward progress?
- Memory is what makes “Stay on Target” possible beyond a single session.

---

## 5. Anti-patterns (what not to promote)

- Chatter, one-off remarks, speculative “maybe we could…”
- Raw logs or every step/diff without curation.
- Stale fragments, weaker alternatives, superseded rules.

**Promote when:** Explicit decision, failure, repeated pattern, user signal, cross-file impact — and when evidence supports it.

---

## 6. Key files in this repo

- **PRD:** `scripts/prd.txt`
- **Control-plane design:** `docs/control-plane-design-and-starter.md`
- **Pluribus vs editor LSP:** `docs/pluribus-lsp-mcp-boundary.md` (MCP/HTTP for agents; gopls stays in the editor)
- **Tasks:** `.taskmaster/tasks/tasks.json`
- **Cursor rule:** `.cursor/rules/recall-focus.mdc`
- **Memory Bank:** `memory-bank/`
