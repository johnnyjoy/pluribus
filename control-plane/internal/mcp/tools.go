package mcp

const toolDoctrineHint = " Memory-first; see Recall repo docs/memory-doctrine.md; do not assume silo/partition IDs."

const (
	layer1DefaultLoop = " [L1 — default loop; sufficient for recall + episodic learning when config enables auto-distill]"
	layer2Optional    = " [L2 — optional; pull-based; not required for improvement over time]"
)

// wakeupContextInputSchema is strict: only limit overrides; POST body matches WakeupRequest subset.
var wakeupContextInputSchema = map[string]any{
	"type":                 "object",
	"additionalProperties": false,
	"properties": map[string]any{
		"max_state": map[string]any{
			"type":        "integer",
			"description": "Cap L0 identity rows (kind state); server default when omitted.",
		},
		"max_per_kind": map[string]any{
			"type":        "integer",
			"description": "Per primary bucket cap before L1 merge; server default when omitted.",
		},
		"max_governing_total": map[string]any{
			"type":        "integer",
			"description": "Cap L1 governing_memory rows after applicability filter; server default when omitted.",
		},
	},
}

// ToolDefinitions returns MCP tool descriptors from the canonical registry (stdio and HTTP MCP).
func ToolDefinitions() []map[string]any {
	reg := toolRegistry()
	out := make([]map[string]any, len(reg))
	for i, t := range reg {
		desc := t.Description + toolDoctrineHint
		out[i] = map[string]any{
			"name":        t.Name,
			"description": desc,
			"inputSchema": t.InputSchema,
		}
	}
	return out
}
