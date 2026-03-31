package mcp

const toolDoctrineHint = " Memory-first; see Recall repo docs/memory-doctrine.md; do not assume silo/partition IDs."

// Layer hints: L1 = default automatic cognitive loop; L2 = optional pull-based inspection, control, analysis (never required).
const (
	layer1DefaultLoop = " [L1 — default loop; sufficient for recall + episodic learning when config enables auto-distill]"
	layer2Optional    = " [L2 — optional; pull-based; not required for improvement over time]"
)

// ToolDefinitions returns MCP tool descriptors (stdio and HTTP MCP use the same list).
// Order is intentional: behavior-first names first, then the default cognitive loop, then depth tools,
// compatibility aliases, health last — see tool_surface.go.
func ToolDefinitions() []map[string]any {
	schemaObj := map[string]any{"type": "object"}
	return []map[string]any{
		{"name": "recall_context", "description": "PRE-ACTION (run before plan/tooling on non-trivial work). WHEN: Before planning, uncertain requirements, architecture, or multi-step work. WHY: Surfaces prior constraints, failures, patterns, and decisions so you do not repeat mistakes. WHAT: POST /v1/recall/compile with deterministic routing from your task text (task or task_description). Response includes mcp_context (why_now, bundle_counts) + recall_bundle. Same tool as memory_context_resolve — use whichever name your client lists." + layer1DefaultLoop + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "record_experience", "description": "POST-ACTION (run after meaningful act). WHEN: After incidents, non-trivial fixes, explicit decisions, or important discoveries. WHY: Writes an advisory episode so future tasks can reuse the outcome; optional server auto-distill → pending candidates. WHAT: POST /v1/advisory-episodes (summary required; optional tags, correlation_id, event_kind, entities). Not canonical memory until reviewed/promoted. Same tool as mcp_episode_ingest." + layer1DefaultLoop + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "memory_log_if_relevant", "description": "WHEN: You have a text block that may contain learning signals but you are unsure. WHY: Deterministic gate ingests only when tokens match (no LLM); otherwise returns skipped JSON. Same HTTP path as record_experience when it fires. Alias: auto_log_episode_if_relevant." + layer1DefaultLoop + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "auto_log_episode_if_relevant", "description": "Alias for memory_log_if_relevant — same arguments (text_block, optional correlation_id, tags)." + layer1DefaultLoop + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "enforcement_evaluate", "description": "WHEN: Before large refactors, policy-sensitive edits, or high-risk proposals. WHY: Deterministic check against binding memory; do not proceed when next_action is revise/reject. WHAT: POST /v1/enforcement/evaluate with proposal_text." + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "recall_compile", "description": "WHEN: You need full CompileRequest control. WHY: Raw recall compile without memory_context_resolve shaping. Prefer recall_context first." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "recall_get", "description": "WHEN: GET-shaped recall is required by your workflow. WHY: Same pool as compile; use before action when appropriate. Prefer recall_context for the default path." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "recall_run_multi", "description": "WHEN: Advanced orchestration (variants / run-multi). WHY: Optional after recall + enforcement. " + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "memory_create", "description": "WHEN: You must author explicit durable memory (decision, constraint, failure, pattern, state). WHY: Direct canonical write — use sparingly vs curation/promotion flows." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "memory_promote", "description": "WHEN: Structured promote path per server policy. WHY: Alternative to digest → materialize; advanced." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "curation_digest", "description": "WHEN: You have bounded narrative work_summary to turn into digest proposals. WHY: Creates pending candidates — not canon until materialize." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "curation_pending", "description": "WHEN: Recall is thin or you want to see what the system is learning. WHY: Lists pending candidates (not canonical). Inspect before promotion." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "curation_promotion_suggestions", "description": "WHEN: You want promotion assistance. WHY: Readiness hints; never auto-promotes by itself." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "curation_strengthened", "description": "WHEN: Looking for candidates reinforced by multiple distill signals. WHY: Optional min_support filter (default 2)." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "curation_review_candidate", "description": "WHEN: Before promoting a specific candidate. WHY: Full review payload (explanation, episodes, promotion preview). Args: candidate_id or id (UUID)." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "curation_materialize", "description": "WHEN: A candidate is validated and should become durable memory. WHY: POST materialize for one UUID. Alias: curation_promote_candidate." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "curation_promote_candidate", "description": "Alias for curation_materialize — same endpoint and arguments." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "curation_reject_candidate", "description": "WHEN: A candidate should not become memory. WHY: Marks rejected. Args: candidate_id or id (UUID)." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "curation_auto_promote", "description": "WHEN: Server has promotion.auto_promote and you want batch materialize. WHY: Respects server gates; may return 403 if disabled." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "episode_search_similar", "description": "WHEN: You want episodic continuity from past advisory text. WHY: Similarity search over episodes. Args: query or summary_text." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "episode_distill_explicit", "description": "WHEN: Auto-distill is off or you need an explicit distill step. WHY: Creates/updates pending candidates from episode_id or summary. Requires distillation.enabled." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "memory_recall_advanced", "description": "WHEN: You need mode-shaped compile (continuity|constraint|pattern|episodic). WHY: Advanced recall; prefer recall_context for defaults." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "memory_preflight_check", "description": "WHEN: Quick risk hint from scope stats. WHY: Not a substitute for enforcement_evaluate on real proposal text." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "memory_detect_contradictions", "description": "WHEN: Comparing two memory UUIDs for conflicts. WHY: Analysis; may create contradiction record." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "memory_list_contradictions", "description": "WHEN: Auditing contradiction queue. WHY: Optional filters." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "evidence_attach", "description": "WHEN: Linking traceability artifact to a memory. WHY: Create evidence + link." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "evidence_list", "description": "WHEN: Listing evidence by memory_id or kind." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "memory_relationships_get", "description": "WHEN: Inspecting graph edges for a memory UUID." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "memory_relationships_create", "description": "WHEN: Recording a typed edge between two memories." + layer2Optional + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "memory_context_resolve", "description": "Compatibility alias for recall_context — identical arguments, response, and HTTP behavior. Prefer recall_context in new prompts." + layer1DefaultLoop + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "mcp_episode_ingest", "description": "Compatibility alias for record_experience — identical arguments and HTTP behavior. Prefer record_experience in new prompts." + layer1DefaultLoop + toolDoctrineHint, "inputSchema": schemaObj},
		{"name": "health", "description": "GET /healthz — liveness only; no recall or ingest effect." + toolDoctrineHint, "inputSchema": schemaObj},
	}
}
