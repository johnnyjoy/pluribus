# Pluribus — OpenClaw skill templates

1. **Session ground** — On task start, `memory_context_resolve` + optional `correlation_id` for continuity.  
2. **Session capture** — On milestone or failure, `mcp_episode_ingest` with concise summary + tags.  
3. **Promotion path** — Only when operator validates: review candidate → materialize; never treat advisory as canon by default.  
4. **Inspect** — `memory_detect_contradictions` / `memory_relationships_get` for debugging recall—not every turn.
