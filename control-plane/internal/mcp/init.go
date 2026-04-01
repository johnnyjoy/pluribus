package mcp

// MemoryLoopInstructions is MCP initialize-time guidance (bias before tool selection).
const MemoryLoopInstructions = `Pluribus is your memory system.

For best results:
- Use recall_context before complex reasoning or multi-step actions.
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
