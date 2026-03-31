# Pluribus — OpenCode behavior templates

Use as **`.opencode/skills/`** markdown, **`AGENTS.md`** sections, or inline checklists. OpenCode discovers skills from config directories—see [OpenCode config](https://dev.opencode.ai/docs/config).

## 1. Use memory for context

**Trigger:** Starting non-trivial reasoning, design, or code change.  
**Action:** Call **`memory_context_resolve`** via **`pluribus`** tools with the task description. Use **`mcp_context`** + **`recall_bundle`** before planning.  
**Success:** Plans respect governing constraints, failures, and patterns when recall returns them.

## 2. Log meaningful experience

**Trigger:** After incidents, decisions, fixes, or repeated failures.  
**Action:** **`mcp_episode_ingest`** with a short summary; optional **`correlation_id`**, **`tags`**, **`event_kind`**.  
**Success:** Advisory episode recorded; **distill mode** may create candidates—canon still requires promotion.

## 3. Review candidate learning (optional)

**Trigger:** Recall is thin or before promotion.  
**Action:** **`curation_pending`**, **`curation_review_candidate`** as needed.  
**Success:** Clear separation of pending vs canon.

## 4. Promote high-confidence knowledge (optional)

**Trigger:** Candidate validated and policy allows.  
**Action:** **`curation_promote_candidate`** / materialize per server gates.  
**Success:** Durable memory with traceability—not raw chat.

## 5. Inspect contradictions / relationships (optional)

**Trigger:** Recall shows tension or graph context is needed.  
**Action:** **`memory_detect_contradictions`**, **`memory_relationships_get`**.  
**Success:** Structured check without blocking ingest/recall.
