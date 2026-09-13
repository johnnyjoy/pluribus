package mcp

// MemoryLoopInstructions is MCP initialize-time guidance (bias before tool selection).
// First lines must stay under 2048 characters (Claude MCP description/instruction truncate).
const MemoryLoopInstructions = `Recall then record. Clock questions: pass RFC3339 occurred_after/occurred_before on recall_context (server does not parse “last Friday”). Write one dense claim for a later agent, not an essay.

Loop:
- Optional at session start: wakeup_context. Use recall_context when you have situation text.
- After recall: apply grounded [id] statements from agent_grounding that constrain this task; discard the rest. Do not treat the JSON dump as context.
- If housekeeping / list_chores: resolve_chore (chore_id, action, agent_id) or defer with reason in record_experience.
- Used a memory: pass its id as used_memory_ids on record_experience (upvote). Do not also call memory_feedback helpful for the same IDs. Misled/wrong/outdated: memory_feedback with reason.
- record_experience after outcomes. Optional occurred_at (RFC3339) when you know when it happened.

Memories do not fade. Raise access when a memory helped; lower only when it misled.`

// InitializeResult builds the MCP initialize response (protocol + capabilities).
func InitializeResult(serverName, version string) map[string]any {
	return map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"tools":     map[string]any{},
			"prompts":   map[string]any{},
			"resources": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    serverName,
			"version": version,
		},
		"instructions": MemoryLoopInstructions,
	}
}
