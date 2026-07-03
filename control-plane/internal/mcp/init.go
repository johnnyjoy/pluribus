package mcp

// MemoryLoopInstructions is MCP initialize-time guidance (bias before tool selection).
const MemoryLoopInstructions = `Pluribus is your memory system.

For best results:
- Optional at session start: wakeup_context for compact L0/L1 (POST /v1/recall/wakeup); use recall_context when you have task text for deeper situational recall.
- Use recall_context before complex reasoning or multi-step actions.
- When recall_context or wakeup_context returns housekeeping (or list_chores is non-empty): call resolve_chore with chore_id, action, and agent_id if you can judge; otherwise defer with reason in record_experience. Empty chores = no-op.
- Use record_experience after meaningful outcomes, fixes, failures, or discoveries.

This helps you avoid repeating mistakes and reuse proven solutions.`

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
