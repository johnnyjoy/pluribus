# Anti-regression — memory-first enforcement

**Purpose:** Hard guardrails so Pluribus **cannot silently slide back** into container-based thinking. Use this in **code review**, **docs review**, and **MCP copy review**.

**Authority:** Subordinate to [memory-doctrine.md](memory-doctrine.md); both are enforced in CI via guard tests where applicable.

---

## A. Banned vocabulary (hard list)

Do **not** use these in **user-facing** copy, **MCP prompts**, **onboarding**, or **examples** except when **quoting this ban list** or **memory-doctrine.md**:

| Term | Notes |
|------|--------|
| **project** | As a memory partition, “your project’s memory,” required project IDs for recall. |
| **task** | As a memory partition or “task-scoped recall.” |
| **workspace** | As a memory silo or required workspace for governing memory. |
| **hive** | Same as workspace; historical name. |
| **scope** | As a mandatory container abstraction for memory (e.g. “scope:” as silo truth). |

**Synonyms to reject:** backlog item, ticket, container, silo, partition (when used to mean **memory truth boundary**), ownership container.

---

## B. Banned patterns

1. **Memory partitioning by container** — any design where recall or enforcement **only** searches rows “for” a client-supplied container.
2. **Required container IDs** on core flows: recall compile/get, enforcement evaluate, memory create/search.
3. **Recall filtered by container** as the **default** or **documented** mental model — recall may use tags and query only.
4. **Docs or prompts that teach** “first select your project/workspace, then recall.”
5. **MCP tools** that imply container IDs are **necessary** for grounding.

---

## C. Allowed constructs

| Construct | Use |
|-----------|-----|
| **Tags** | Primary handle for situation and domain (`rules:security`, `domain:api`, …). |
| **Query / retrieval_query** | Describe the situation in natural language. |
| **Situation** | Framing language for “what we’re doing now,” not a silo ID. |
| **Authority** | Ranking and binding strength. |
| **Memory kinds** | constraint, decision, pattern, failure, state, experience, etc. |
| **Agent ID** | Opaque client identity for telemetry/salience where the API allows — **not** a memory partition. |

---

## D. PR review checklist

Every change should pass:

- [ ] **Does it introduce container thinking?** (project/task/workspace/hive/scope as memory boundary)
- [ ] **Does it require container IDs** for recall, enforcement, or core memory?
- [ ] **Does it weaken the memory-first model** (e.g. onboarding that skips recall)?
- [ ] **Does it misrepresent recall** (e.g. “dump everything in this silo”)?
- [ ] **Do MCP prompts still state** “Do not assume any project, task, or container; this system is memory-first”?
- [ ] **If touching examples**, do they show **tags + query** without container IDs?

If any answer is **yes** to the first four in a bad way, **reject or revise**.

---

## E. Automated gates

The control-plane module includes **guard tests** that:

- Assert MCP prompt markdown contains the **doctrine footer** and avoids **legacy wire vocabulary** in those files.
- Assert selected request structs have **no forbidden JSON field names** for core recall/enforcement/memory paths.
- Assert root **README.md** and **CONTRIBUTING.md** stay within the same rules for listed substrings.

See `control-plane/internal/guardrails/` tests. **CI:** `go test ./...` in `control-plane/`.

---

## F. Failure mode

If container language slips back in **without** updating doctrine and **without** failing tests, the sprint failed — fix forward by reverting the copy or tightening the guard.
