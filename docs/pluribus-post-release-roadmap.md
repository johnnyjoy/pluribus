# Pluribus — post-release roadmap (disciplined)

This note **fences future work** so it does **not** blur the current public release. Items here are **candidates**, not commitments.

---

## Validated direction (post-release, when justified)

| Direction | Notes |
|-----------|--------|
| **True multi-host continuity smoke** | Same slug, two machines, same URL — optional validation after **simulated** proof. |
| **Further proof maturation** | More scenarios; stricter receipts; optional MCP-in-the-loop assertions. |
| **Observability** | Metrics, tracing, audit logs — **when** operators need them. |
| **Explainability** | Better docs/resources for recall, enforcement, curation — **without** product redesign. |

---

## Explicitly deferred (not “next sprint” by default)

| Topic | Reason |
|-------|--------|
| **Embeddings / vector retrieval as authority** | Conflicts with **governed memory + proof** story; would be a **major** product conversation. |
| **Semantic recall redesign** | Same as above. |
| **Orchestration / workflow engines** | Out of product scope. |
| **MCP stdio as canonical** | **Compatibility** path only; HTTP-first remains. |

---

## Optional product extensions (if demand is proven)

| Topic | Position |
|-------|----------|
| **Advisory episodic similarity** | Already **subordinate** — [episodic-similarity.md](episodic-similarity.md). |
| **Get-or-create HTTP** | Listed as optional in [passive-continuity-architecture.md](passive-continuity-architecture.md). |

---

## Rule

**Do not** let “future” items (embeddings, orchestration) appear in **release** messaging as if they were **current** scope — [pluribus-release-scope.md](pluribus-release-scope.md).
