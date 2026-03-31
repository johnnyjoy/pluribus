# Ensuring agents actually use Pluribus

Operational guide for people running **real** agents and editors against a Pluribus control plane. Not marketing—**defaults, rules, and verification**.

**Related:** [memory-doctrine.md](../memory-doctrine.md) · [mcp-usage.md](../mcp-usage.md) · [integrations/README.md](../integrations/README.md)

**Copy-paste snippets:** [snippets/agent-rules.md](snippets/agent-rules.md) · [snippets/mcp-usage-guidance.md](snippets/mcp-usage-guidance.md) · [snippets/rest-usage-guidance.md](snippets/rest-usage-guidance.md)

---

## 1. Why usage matters

- **Sessions end.** Anything not written to Pluribus (or promoted from candidates) evaporates from the agent’s reach next chat.
- **Prompts don’t persist.** Instructions in a one-off system message are not institutional memory.
- **RAG over documents** is not the same as **governed memory**: decisions, failures, and constraints need typed rows and authority—not just similarity to chunks.
- Pluribus only changes outcomes if agents **recall before acting** and **log after learning**. Unused infrastructure looks like “we deployed memory” and still get amnesia.

---

## 2. MCP (primary path)

### What MCP gives you

- **Tools inside the agent loop** (**`recall_context`**, **`record_experience`**, enforcement, curation, …; aliases **`memory_context_resolve`**, **`mcp_episode_ingest`**)—same semantics as REST, routed via **`POST /v1/mcp`**.
- **Continuous availability** whenever the client keeps the MCP connection to a reachable control plane.

### What you do

1. **Enable MCP** in the editor/agent and point at **`http://<host>:8123/v1/mcp`** (HTTP) or run stdio **`pluribus-mcp`** with **`CONTROL_PLANE_URL`**—see [mcp-usage.md](../mcp-usage.md).
2. **Confirm reachability:** **`GET /healthz`**, **`GET /readyz`** on the API base; fix networking before tuning prompts.
3. **Ship rules** that *require* recall and episodic capture for non-trivial work—see §3 and [snippets/agent-rules.md](snippets/agent-rules.md).

Details: [snippets/mcp-usage-guidance.md](snippets/mcp-usage-guidance.md), [mcp-service-first.md](../mcp-service-first.md).

---

## 3. Agent rules (critical)

Rules are how you turn “MCP connected” into “MCP **used**.” Models default to skipping tools unless instructed.

### Minimal expectations

- **Recall before** architectural or multi-step decisions: **`recall_context`** (or **`memory_context_resolve`**) with raw task text.
- **Consult memory** when requirements are ambiguous or historically noisy (tags + retrieval_query).
- **Log outcomes** after incidents, fixes, or explicit decisions: **`record_experience`** (or **`mcp_episode_ingest`**) — summary — or REST **`POST /v1/advisory-episodes`**.
- **Treat memory as shared:** tag for situation, not for “my chat.”

### Tone

Short, imperative, repeatable—see [snippets/agent-rules.md](snippets/agent-rules.md). Platform-specific packs: [integrations/](../../integrations/) per editor.

**Doctrine:** Do not reintroduce **project / task / workspace / scope** as memory partitions—[anti-regression.md](../anti-regression.md).

---

## 4. What good looks like (behavioral loop)

1. **Task starts** → agent has **task text** and **tags** (situation).
2. **Recall** → **`recall_context`** / bundle consumed; **no** plan that ignores binding constraints when recall returned them.
3. **Work** → edits, tests, commits.
4. **Episode** → advisory summary ingested via **`record_experience`** (MCP) or REST.
5. **Distillation** (server policy) → **candidates** with **distill mode** metadata; **`pluribus_distill_origin`** on candidates—see [memory-doctrine.md](../memory-doctrine.md) · **ingest channel** on episodes.
6. **Promotion** → validated candidates become **durable memory**; recall and enforcement improve over time.

If step 2 or 4 is always skipped, you have a connected database and a **disconnected** agent.

---

## 5. REST (fallback / advanced)

When MCP is **not** wired (batch jobs, headless scripts, CI, or a client that only speaks HTTP JSON), use REST on the **same** base URL.

- **Recall:** `POST /v1/recall/compile`
- **Advisory episodes:** `POST /v1/advisory-episodes` (**`summary`** required; **`source`** = ingest channel)
- **Explicit distill:** `POST /v1/episodes/distill` (gated by **`distillation.enabled`**)
- **Durable memory:** `POST /v1/memory` / `POST /v1/memories` when you need explicit rows

Minimal examples: [snippets/rest-usage-guidance.md](snippets/rest-usage-guidance.md). Full map: [http-api-index.md](../http-api-index.md).

---

## 6. Automation vs manual

| Mode | What happens |
|------|----------------|
| **Stronger** | MCP tools in loop + **rules** + server **auto-distill** from advisory episodes (when enabled) + periodic **curation review** / **materialize** |
| **Weaker but valid** | Manual REST ingest + manual distill + manual promotion—works only if someone **does** it |

Manual-only without discipline produces **empty recall** and **nothing promoted**.

---

## 7. Common failure modes

- **MCP connected, tools never called** — no rules; model treats Pluribus as optional.
- **No rules in repo** — every new session forgets your “standard.”
- **Only episodic noise** — summaries with no learning signals; distillation may produce nothing useful.
- **Everything stays advisory** — candidates never reviewed; durable memory stays thin.
- **403 on episodic paths** — deployment may disable similarity/episodic features; use REST diagnostics and ops config, not prompt tweaks.

---

## 8. Practical advice

- **Start with MCP + rules** (see [integrations/README.md](../integrations/README.md)); add REST scripts when you need automation.
- **Don’t over-build** before you see **recall bundles** and **episodes** appearing in normal work.
- **Verify** by inspecting:
  - **Advisory episodes** (ingest working)
  - **Pending candidates** (`GET /v1/curation/pending` or MCP **`curation_pending`**)
  - **Durable memory growth** over time (tags you care about)

---

## Quick links

| Need | Where |
|------|--------|
| Tool order | [mcp-usage.md](../mcp-usage.md#recall-driven-workflow-recommended-order) |
| Editor configs | [integrations/README.md](../integrations/README.md) |
| Auth | [authentication.md](../authentication.md) |
| Curation | [curation-loop.md](../curation-loop.md) |
