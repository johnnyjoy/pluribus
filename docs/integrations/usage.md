# Integration usage — adoption and behavior

**Goal:** Turn **MCP connected** into **memory used** — recall before substantive work, record after meaningful outcomes, promotion when appropriate.

**Canonical deep dive** (failure modes, automation, REST fallback): **[ensuring-agent-usage.md](../usage/ensuring-agent-usage.md)**. **First run:** **[get-started.md](../get-started.md)**.

---

## 1. Behavioral loop (non-negotiable)

1. **`recall_context`** (or alias **`memory_context_resolve`**) — before planning or editing when the task matches **[`pluribus-instructions.md`](../../integrations/pluribus-instructions.md)** triggers.  
2. **Plan / act** — work informed by the bundle.  
3. **`record_experience`** (or alias **`mcp_episode_ingest`**) — after fixes, decisions, or non-obvious outcomes.  
4. **Curation** — optional pull tools (`curation_pending`, `curation_materialize`, …) when promoting learning; see [curation-loop.md](../curation-loop.md).

**Rules + snippets** are what make step 1 and 3 happen; tools alone are not enough. Platform packs: **[integrations/README.md](../../integrations/README.md)**.

---

## 2. MCP (primary) vs REST (supporting)

| Path | Use for |
|------|---------|
| **MCP** `POST /v1/mcp` | Agents in editors — tools in the loop ([mcp-usage.md](../mcp-usage.md)). |
| **REST** same host | Scripts, CI, admin, batch ([http-api-index.md](../http-api-index.md)). |
| **VS Code extension** | **[`integrations/vscode/extension/`](../../integrations/vscode/extension/)** — same REST endpoints for **manual** recall, advisory record, and pending queue (sidebar + Output); use when you want Pluribus visible without relying on the agent to call MCP. |

Service behavior is proven at REST first; MCP is a thin adapter ([mcp-service-first.md](../mcp-service-first.md)).

---

## 3. Failure modes (short)

| Symptom | Likely cause |
|---------|----------------|
| Tools never called | No user/repo **rules**; paste **`pluribus-instructions.md`** or copy **`pluribus.mdc`**. |
| 403 on advisory ingest | **`similarity.enabled`** off in config — see [episodic-similarity.md](../episodic-similarity.md). |
| Episodes but thin **memories** | Candidates not reviewed/materialized — [curation-loop.md](../curation-loop.md). |
| “Empty recall” forever | No durable rows yet; still run **record** after real outcomes. |

Full list: **[ensuring-agent-usage.md §7](../usage/ensuring-agent-usage.md#7-common-failure-modes)**.

---

## 4. Verification checklist

| Check | How |
|-------|-----|
| MCP alive | `GET /healthz`, `GET /readyz`; client lists Pluribus tools after restart. |
| Advisory path | Rows in **`advisory_episodes`** after **`record_experience`** (similarity on). |
| Candidates | `curation_pending` or **`GET /v1/curation/pending`**. |
| Durable memory | **`memories`** + **`memories_tags`** growth for your tags. |
| Recall used | Agent logs or server traces showing **`recall_context`** on non-trivial tasks. |

---

## 5. Skills model (four intents, one pack per platform)

Behavior intents map to MCP tools; each editor ships **one** **`skills/pluribus/SKILL.md`** (not four duplicate skills). See **[skills-model.md](skills-model.md)**.

---

## 6. Related

| Doc | Purpose |
|-----|---------|
| [matrix.md](matrix.md) | Platform comparison + tiers |
| [mcp-usage.md](../mcp-usage.md) | Client JSON, Cursor, Claude, troubleshooting |
| [memory-doctrine.md](../memory-doctrine.md) | Product truth |
