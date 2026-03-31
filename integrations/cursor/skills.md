# Pluribus — behavior templates (Cursor / portable)

Use as Cursor skills or as checklists in prompts.

## 1. Use memory for context

**Trigger:** Starting non-trivial reasoning, design, or code change.  
**Action:** Call **`recall_context`** (or **`memory_context_resolve`**) with the task description (or `task` field). Consume `mcp_context` + `recall_bundle` before proposing plans.  
**Success:** Decisions cite governing constraints/failures/patterns when present.

## 2. Log meaningful experience

**Trigger:** After incidents, decisions, fixes, or repeated failures.  
**Action:** Call **`record_experience`** (or **`mcp_episode_ingest`**) with a short summary; optional **`correlation_id`**, **`tags`**, **`event_kind`**.  
**Success:** Advisory episode recorded; **distill mode** may create candidates if server policy allows—canon requires promotion.

## 3. Review candidate learning (optional)

**Trigger:** When debugging why recall is thin or before promotion.  
**Action:** **`curation_pending`**, **`curation_review_candidate`** as needed.  
**Success:** You understand pending vs canon without treating candidates as law.

## 4. Promote high-confidence knowledge (optional)

**Trigger:** When a candidate is validated and policy allows.  
**Action:** **`curation_promote_candidate`** / materialize—follow server gates.  
**Success:** Durable memory rows with traceability—not raw chat.

## 5. Inspect contradictions / relationships (optional)

**Trigger:** When recall surfaces tension or you need graph context.  
**Action:** **`memory_detect_contradictions`**, **`memory_relationships_get`**.  
**Success:** Structured check without blocking the default ingest/recall loop.
