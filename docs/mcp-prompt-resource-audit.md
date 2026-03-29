# MCP prompt & resource audit (Pluribus)

**Purpose:** Record the behavioral contract for **`prompts/*`** and **`resources/*`** on the MCP surface, gaps addressed in **SurfaceVersion 1.1.0**, and truth alignment vs shipped HTTP/API docs.

**Code:** `control-plane/internal/mcp/` — embedded markdown (`*.md`), `resources.go`, `prompts.go`, `surface_version.go`.

---

## Prompt inventory

| Name | Job | Body |
|------|-----|------|
| `pluribus_memory_grounding` | Recall → ground in governing memory → work; anti-fork | `task_start.md` |
| `pluribus_pre_change_enforcement` | Risky vs trivial; `enforcement_evaluate` gate | `pre_change.md` |
| `pluribus_memory_curation` | Meaningful learning; digest → materialize; candidate ≠ canon | `post_task.md` |
| `pluribus_canon_vs_advisory` | Authority classes only; pointers to resources | `canon_vs_advisory.md` |

**Descriptions:** `PromptDefinitions()` one-liners include **`SurfaceVersion`** and stay aligned with each body’s mission (not a duplicate of the full text).

---

## Resource inventory (six canonical URIs)

| URI | Job |
|-----|-----|
| `pluribus://discipline/doctrine` | Compact defaults, anti-skip, pointers to A–E |
| `pluribus://discipline/lifecycle` | Ordered phase map + tools per phase |
| `pluribus://discipline/project-identity` | Shared continuity; resolve before create; anti-fork |
| `pluribus://discipline/canon-vs-advisory` | Governing vs advisory vs candidate vs evidence vs transcript |
| `pluribus://architecture/active-context-vs-durable-store` | Recall bundle vs DB; reference, don’t inline |
| `pluribus://discipline/history-not-memory` | Transcript/log ≠ durable operational memory |

**Alternate URIs** (same bodies): `pluribus://discipline/canon-advisory` → canon; `pluribus://discipline/architecture-notes` → architecture.

**Implementation note:** Resource markdown lives in Go raw string literals; inline code fences use **bold** for identifiers (Go cannot embed backticks inside raw strings).

---

## Pre-audit gaps → fixes (Q1)

| Item | Weakness | Fix |
|------|----------|-----|
| Prompt descriptions vs bodies | Could drift from embedded files | Descriptions cite **SurfaceVersion**; L1 tests lock substrings |
| Pre/post/canon | Less explicit on risk/trivial/meaningful | Full rewrites in `pre_change.md`, `post_task.md`, `canon_vs_advisory.md` |
| Resources | Missing project identity & history-not-memory | New URIs B and E; doctrine points to full set |
| Doctrine vs lifecycle | Overlap risk | Doctrine = defaults/ritual; lifecycle = ordered steps only |
| Proof | Prompt layer under-tested | `internal/mcp` substring tests + this audit + proof doc |

---

## Truth alignment (B3)

Cross-read against server behavior and docs. **No contradictions found** at audit time; prompts **describe** tools without changing server rules.

| Check | Reference | Result |
|-------|-----------|--------|
| Tool names and HTTP mapping | [mcp-poc-contract.md](mcp-poc-contract.md) | Prompts name `recall_get`, `recall_compile`, `enforcement_evaluate`, `curation_digest`, `curation_materialize` consistently with the contract table |
| Pre-change gate | [pre-change-enforcement.md](pre-change-enforcement.md) | `pre_change.md`: binding memory vs proposal; 403 when enforcement disabled matches contract |
| Curation loop | [curation-loop.md](curation-loop.md) | `post_task.md`: digest → review → materialize; candidates pending |
| Discipline / lifecycle | [mcp-discipline-doctrine.md](mcp-discipline-doctrine.md) | Locked mnemonic and steps match doctrine intent; resources link to longer doc |
| Authority | Enforcement + curation docs | Canon vs advisory vs candidate vs evidence aligned with handler semantics (server is source of truth for membership) |

**If server behavior changes:** update HTTP docs and contract first, then adjust embedded prompts/resources and bump **SurfaceVersion** per [mcp-prompt-resource-versioning.md](mcp-prompt-resource-versioning.md).

---

## References

- [mcp-service-first.md](mcp-service-first.md)
- [mcp-prompt-resource-proof.md](mcp-prompt-resource-proof.md)
- [mcp-prompt-resource-versioning.md](mcp-prompt-resource-versioning.md)
