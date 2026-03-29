package mcp

const toolDoctrineHint = " Memory-first; see Recall repo docs/memory-doctrine.md; do not assume silo/partition IDs."

// ToolDefinitions returns MCP tool descriptors (stdio and HTTP MCP use the same list).
func ToolDefinitions() []map[string]any {
	schemaObj := map[string]any{"type": "object"}
	return []map[string]any{
		{"name": "health", "description": "GET /healthz — liveness/debug only; no workflow effect." + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "recall_compile", "description": "POST /v1/recall/compile — recall-first retrieval in body form. Use before action to reconstruct continuity, constraints, and experience." + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "recall_get", "description": "GET /v1/recall/ — recall-first retrieval from shared memory. Use before action; output guides behavior via continuity/constraints/experience." + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "recall_run_multi", "description": "POST /v1/recall/run-multi — orchestration endpoint for variant generation/selection; optional advanced path after recall + enforcement steps." + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "memory_create", "description": "POST /v1/memory — write durable behavioral memory only (state, decision, failure, pattern, constraint). Use for explicit canonical entries." + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "memory_promote", "description": "POST /v1/memory/promote — promote structured output to memory via server policy; advanced/optional versus curation_digest materialization flow." + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "curation_digest", "description": "POST /v1/curation/digest — capture durable learning into pending curation candidates. Required: work_summary (bounded). Global memory pool; tags optional on digest body per API." + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "curation_materialize", "description": "POST /v1/curation/candidates/{id}/materialize — promote one validated curation candidate to durable memory." + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "enforcement_evaluate", "description": "POST /v1/enforcement/evaluate — validate step for risky proposals against binding memory. Do not proceed when next_action is revise/reject." + toolDoctrineHint, "inputSchema": schemaObj},
	}
}
