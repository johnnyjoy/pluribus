ARCHIVED — NOT ACTIVE SYSTEM TRUTH

---

# Hard memory-first product surface cutover (2026-03-27)

Sprint goal: stop teaching Pluribus as a **project/task product**. The implementation was already memory-first; **public surface, MCP contract, onboarding, and Cursor commands** were still contaminated.

---

## 1. What Pluribus is (after this cutover)

- **Global memory system** for agents: durable constraints, decisions, failures, patterns; **recall**; **enforcement**; **curation**.
- **Optional DB-side workflow / FK scaffolding** (targets, tasks, evidence, curation) for **correlation and policy** — **not** “where memory truth lives.” Public JSON for recall compile, enforcement, and run-multi does **not** carry workspace/partition UUIDs on the wire in the current control-plane.

Canonical ontology: [pluribus-memory-first-ontology.md](pluribus-memory-first-ontology.md).

---

## 2. What was removed or demoted (the “poison”)

- **Project-first / task-start ritual** as the default story in operator docs and Cursor commands.
- False claims that **`POST /v1/memory`** requires a **`projects`** row or a partition UUID on the body.
- **MCP contract drift**: docs listing **`project_get_by_slug`**, **`project_list`**, **`project_create`** as MCP tools while **`internal/mcp/tools.go`** did not expose them.
- **Onboarding examples** that opened with **create project → target → task → memory**.
- Wording like **“shared project continuity”** and **“task-shaped”** where **memory / situation-shaped** is accurate.
- **`scope_type`** in benefit-eval fixtures (invalid JSON for strict **`CreateRequest`** decoding).

---

## 3. What was fixed (high signal)

| Area | Change |
|------|--------|
| **Control-plane README** | Memory-first curl flow; optional workspace note for curation. |
| **Walkthroughs** | Tag-based memory + recall + enforcement without legacy workspace-create rituals. |
| **MCP doctrine + functional workflow** | Memory grounding first; HTTP mirrors where MCP does not expose a tool. |
| **`docs/mcp-poc-contract.md`** | Tool table matches **`tools.go`**; lifecycle memory-first. |
| **`docs/mcp-service-first.md`** | Accurate tool and resource lists (five resources; no phantom `project-identity` URI). |
| **`docs/passive-continuity-architecture.md`** | Rewritten for global memory + tag namespaces (archived copy under **`docs/archive/`**). |
| **`docs/cursor-verify-recall.md`** + **`.cursor/commands/ask-cursor-verify-recall.md`** | Verify path without **`project_create`**. |
| **Benefit eval** | Fixture + **`pluribus-benefit-eval.md`** use valid **`memory_create`** payloads; recall via **tags**. |
| **`.cursor/rules/pluribus-eval-execute.mdc`** | Execution rule matches memory-first + HTTP for project when needed. |
| **Proof YAML** | Manual steps aligned with memory-first + shared tags. |
| **Embedded MCP doctrine** (`internal/mcp/resources.go`) | Explicit global-memory + metadata sentence (test-guarded). |
| **Adapter** | **`BuildRecallGetURL`** maps tool args to **`GET /v1/recall/`** query params (`retrieval_query` / `query`, `tags`, `symbols`, limits). |

---

## 4. What remains as scaffolding (intentional)

- **DB FK graph** for evidence, curation, drift logs, etc., may still reference internal workspace rows — this is **persistence**, not the public recall/enforcement JSON contract.
- **Legacy prompt filenames** (`task_start.md`, `post_task.md`) may remain on disk; **prompt names** in MCP are **`pluribus_memory_grounding`** / **`pluribus_memory_curation`** (see semantic cutover report).

---

## 5. Tests / guardrails

- **`TestResourceText_doctrineStatesGlobalMemory`** — doctrine resource contains ontology substrings.
- **`TestBuildRecallGetURL_*`** — recall GET URL encodes tags and retrieval text only (see **`internal/mcp/proxy_test.go`**).
- Existing **`TestToolDefinitions_noProjectCRUDTools`** — ensures **`project_*`** are not re-added to **`tools/list`** without an explicit product decision.

---

## 6. Related reports

- [pluribus-semantic-cutover-report.md](pluribus-semantic-cutover-report.md) — MCP prompt renames and first doc pass.
