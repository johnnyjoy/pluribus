ARCHIVED — NOT ACTIVE SYSTEM TRUTH

---

# Pluribus — memory-first semantic cutover report

**Date:** 2026-03-27  
**Scope:** Documentation, MCP prompt names and bodies, resource copy, API/type comments, and high-traffic examples — **no** DB schema or route renames.

## 1. Architecture statement (post-cutover)

**Pluribus is a global, shared behavioral memory system** (durable rows in **`memories`**, recall- and authority-shaped). **Workflow tables** and internal FKs support evidence, curation, and logging — they are **not** memory truth boundaries. Public recall/enforcement/run-multi bodies use **tags** and **retrieval text**, not partition UUIDs on the wire in the current control-plane.

Canonical doctrine: [pluribus-memory-first-ontology.md](pluribus-memory-first-ontology.md).

## 2. MCP prompt renames (breaking for hard-coded prompt names)

| Old MCP prompt name | New name |
|---------------------|----------|
| `pluribus_task_start` | **`pluribus_memory_grounding`** |
| `pluribus_post_task_curation` | **`pluribus_memory_curation`** |

**Unchanged:** `pluribus_pre_change_enforcement`, `pluribus_canon_vs_advisory`.

Go constants in `control-plane/internal/mcp/prompts.go`: `PromptMemoryGrounding`, `PromptMemoryCuration` (replacing `PromptTaskStart`, `PromptPostTaskCuration`).

**SurfaceVersion** bumped to **1.3.0** (`surface_version.go`) because prompt names and embedded bodies changed.

## 3. Embedded prompt and resource wording

- **`task_start.md`** — Retitled **memory-grounding protocol**; replaces task-first ontology with **situation / execution metadata** language; keeps the same lifecycle mnemonic.
- **`post_task.md`** — Retitled **memory curation protocol**; removes “post-task” framing from the title (body already aligned).
- **`resources.go`** (lifecycle resource) — Prompt list uses new names; **task-shaped** → **situation-shaped**; active context wording uses **request** not **task**.

## 4. Code comments (semantics only)

- **`internal/recall/types.go`** — `CompileRequest` documents tag- and retrieval-driven recall (no workspace UUID JSON fields).
- **`internal/memory/types.go`** — promote/create types document evidence linkage fields that exist on the wire.

## 5. Public docs and examples

- **`docs/pluribus-memory-first-ontology.md`** — New canonical ontology (memory truth vs execution metadata).
- **`docs/pluribus-semantic-cutover-report.md`** — This file.
- **`docs/api-contract.md`** — Global **memory-first ontology** subsection; `POST /v1/memory` behavioral note on promote vs create.
- **`docs/pluribus-public-architecture.md`** — Reframed product story, prompts, workspace continuity, feature stack.
- **`docs/quickstart.md`**, **`docs/pluribus-quickstart.md`** — Memory-first `curl` paths with tags + retrieval text.
- **`docs/README.md`** — Index rows for ontology + cutover report.
- **`README.md`** — Doc table links; run-multi example comment.
- **`docs/mcp-service-first.md`**, **`docs/mcp-cursor-functional-workflow.md`**, **`docs/mcp-prompt-resource-audit.md`**, **`docs/mcp-prompt-resource-proof.md`**, **`docs/mcp-prompt-resource-versioning.md`**, **`docs/passive-continuity-architecture.md`**, **`docs/mcp-poc-contract.md`**, **`docs/pre-change-enforcement.md`**, **`docs/pluribus-differentiators.md`**
- **`control-plane/proof-scenarios/functional-quality-workflow.yaml`**

## 6. Tests and type comments

- **`internal/mcp/prompts_test.go`** — Renamed tests; asserts new protocol title and **execution metadata** phrase; **`TestPromptDefinitions_memoryFirstNames`** forbids legacy prompt names in definitions.
- **`internal/mcp/handler_test.go`** — `prompts/get` uses **`PromptMemoryGrounding`**.
- **`internal/recall/types.go`**, **`internal/recall/handlers_getbundle.go`**, **`internal/memory/types.go`**, **`internal/enforcement/types.go`** — Comments align with memory-first, tag-shaped recall and enforcement.

## 7. Follow-up (post-report code)

- **Shipped router** (`internal/apiserver/router.go`) exposes recall, memory, enforcement, curation, drift, etc. **without** `/v1/hives` workspace CRUD on the public tree documented here at archive time.
- **JSON:** `CompileRequest`, `RunMultiRequest`, `EvaluateRequest`, and drift check bodies match strict **`json`** tags — **no** legacy partition UUID keys on those structs.
- **MCP tools (shipped `tools/list`):** memory-first set in `internal/mcp/tools.go` — **no** `project_*` tool names.

**Why:** This report captured a **documentation** cutover; subsequent releases tightened the **wire** to match the memory-first story.

## 8. Future cleanup candidates (non-commitments)

- Consider aliases in MCP **`prompts/list`** for old names (if clients need a transition period) — not implemented in this sprint.
- Long-term: clearer field names (e.g. `workspace_id`, `execution_context_id`) would require coordinated API versioning — **out of scope** here.
