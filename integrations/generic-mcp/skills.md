# Pluribus — portable skills (any MCP agent)

## Skill: Ground from memory

**Input:** User task. **Tool:** `recall_context` (or `memory_context_resolve`). **Output:** Use constraints, failures, patterns before planning.

## Skill: Record episode

**Input:** What happened (summary). **Tool:** `record_experience` (or `mcp_episode_ingest`). **Output:** Advisory row; may feed distill pipeline per server policy.

## Skill: Optional curation

**Input:** Need to inspect pending learning. **Tools:** `curation_pending`, `curation_review_candidate`. **Output:** Understanding only—promotion is explicit.

## Skill: Optional contradictions

**Input:** Suspected conflict in recall. **Tools:** `memory_detect_contradictions`, `memory_list_contradictions`.
