package mcp

// AgentLoopRole classifies MCP tools for agent workflow documentation.
type AgentLoopRole string

const (
	LoopSessionStart AgentLoopRole = "session_start"
	LoopPreChange    AgentLoopRole = "pre_change"
	LoopPostOutcome  AgentLoopRole = "post_outcome"
	LoopCuration     AgentLoopRole = "curation"
	LoopDiagnostic   AgentLoopRole = "diagnostic"
	LoopAdmin        AgentLoopRole = "admin"
	LoopExperimental AgentLoopRole = "experimental"
	LoopNone         AgentLoopRole = "none"
)

// RiskLevel classifies mutating or sensitive tools.
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// ToolSpec is the canonical MCP tool registration (schema, docs, routing metadata).
type ToolSpec struct {
	Name        string
	Aliases     []string
	Description string
	InputSchema map[string]any
	LoopRole    AgentLoopRole
	Risk        RiskLevel
	Backend     string
	Output      string
	Mutates          bool
	TestCoverage     string // unit | integration | both | none (legacy doc field)
	CallCoverage       CallCoverageCategory
	CallCoverageNote   string // test file or proof script reference
	CallCoverageReason string // required when CallCoverage is unsafe_or_impossible_with_justification
}

var (
	schemaRecallContext = func() map[string]any {
		p := schemaTaskTextProps()
		for k, v := range schemaRepoRootProps() {
			p[k] = v
		}
		p["mode"] = propEnumString("Optional routing override.", "constraint", "failure", "pattern", "episodic", "thread", "continuity")
		p["tags"] = schemaTags()
		p["entities"] = propStringArray("Optional entity tokens merged into tags.")
		p["correlation_id"] = propString("Optional session correlation (e.g. mcp:session:…).")
		p["session_id"] = propString("Alias for correlation_id.")
		p["recall_mode"] = propEnumString("Lifecycle recall mode: current (active + rare pending, weighted lower) or historical (superseded/archived context).", "current", "historical")
		p["occurred_after"] = propString("Optional RFC3339 lower bound on memory effective time (occurred_at, else created_at).")
		p["occurred_before"] = propString("Optional RFC3339 exclusive upper bound on effective time.")
		p["agent_id"] = propString("Optional agent or client identifier for attribution.")
		return schemaObject(p, nil)
	}()

	schemaRecordExperience = schemaObject(map[string]any{
		"summary":        propString("Required. What happened — outcome, failure, pattern, or decision (min length enforced server-side)."),
		"tags":           schemaTags(),
		"correlation_id": propString("Optional session correlation."),
		"event_kind":     propString("Optional; becomes mcp:event:<kind> tag."),
		"entities":       propStringArray("Optional entity tokens."),
		"agent_id":       propString("Optional agent or client identifier for attribution (persisted with the episode)."),
		"repo_root":      propString("Optional workspace path; basename becomes a situational tag."),
	}, []string{"summary"})

	schemaMemoryLogIfRelevant = schemaObject(map[string]any{
		"text_block":     propString("Text to scan for learning signals."),
		"correlation_id": propString("Optional session correlation."),
		"tags":           schemaTags(),
	}, []string{"text_block"})

	schemaEnforcement = schemaObject(map[string]any{
		"proposal_text": propString("Required. The planned change or action text to evaluate (max 32768 bytes)."),
		"intent":        propString("Optional intent label (e.g. change, datastore)."),
	}, []string{"proposal_text"})

	schemaRecallCompilePassthrough = schemaObject(map[string]any{
		"retrieval_query": propString("Situation text for recall."),
		"tags":            schemaTags(),
		"mode":            propString("Compile mode (continuity, thread, …)."),
		"recall_mode":     propEnumString("Lifecycle recall: current (active + rare pending) or historical.", "current", "historical"),
		"include_status":  propStringArray("Optional status filter (active, superseded, archived)."),
		"occurred_after":  propString("Optional RFC3339 lower bound (occurred_at, else created_at)."),
		"occurred_before": propString("Optional RFC3339 exclusive upper bound on effective time."),
		"repo_root":       propString("Optional repo path for situational affinity."),
	}, nil)

	schemaRecallGet = schemaObject(map[string]any{
		"retrieval_query": propString("Situation text (query param)."),
		"query":           propString("Alias for retrieval_query."),
		"tags":            schemaTags(),
		"max_per_kind":    propInt("Per-kind cap.", 1, 100),
		"max_total":       propInt("Total item cap.", 1, 500),
		"max_tokens":      propInt("Approx token budget.", 1, 100000),
	}, nil)

	schemaRecallRunMulti = schemaObject(map[string]any{
		"retrieval_query": propString("Primary situation text."),
		"tags":            schemaTags(),
		"variants":        propAnyObject("Run-multi variant configuration."),
	}, []string{"retrieval_query"})

	schemaMemoryCreate = schemaObject(map[string]any{
		"kind":          propEnumString("Memory kind.", "constraint", "decision", "pattern", "failure", "state"),
		"statement":     propString("Canonical distilled statement."),
		"authority":     propInt("Authority 1–10.", 1, 10),
		"tags":          schemaTags(),
		"applicability": propEnumString("governing or advisory.", "governing", "advisory"),
		"payload":       propAnyObject("Optional structured payload."),
		"supersedes_id": propUUID("Optional UUID of memory this row replaces (marks prior row superseded)."),
		"agent_id":      propString("Optional agent or client identifier for attribution (persisted on the memory)."),
	}, []string{"kind", "statement"})

	schemaMemoryRemediate = schemaObject(map[string]any{
		"memory_id": propUUID("UUID of the memory to remediate."),
		"id":        propUUID("Alias for memory_id."),
		"reason":    propString("Why the memory is being quarantined/deleted (auditable)."),
	}, nil)

	schemaMemoryPromote = schemaObject(map[string]any{
		"statement": propString("Statement to promote."),
		"kind":      propString("Target kind."),
		"tags":      schemaTags(),
	}, []string{"statement", "kind"})

	schemaMemoryFeedback = schemaObject(map[string]any{
		"memory_id":         propUUID("UUID of memory to rate."),
		"event_type":        propEnumString("helpful, harmful, wrong, outdated, or irrelevant.", "helpful", "harmful", "wrong", "outdated", "irrelevant"),
		"reason":            propString("Required for harmful/wrong/outdated — why this feedback applies."),
		"source":            propString("Feedback source (default agent)."),
		"source_tool":       propString("Tool that surfaced the memory (e.g. recall_context)."),
		"source_session_id": propString("Optional session id."),
		"correlation_id":    propString("Optional recall correlation id."),
		"recall_bundle_id":  propString("Optional recall bundle id."),
		"evidence_id":       propUUID("Optional linked evidence UUID."),
		"payload":           propAnyObject("Optional structured metadata."),
	}, []string{"memory_id", "event_type"})

	schemaCurationDigest = schemaObject(map[string]any{
		"work_summary": propString("Required bounded narrative to distill into candidates."),
		"tags":         schemaTags(),
	}, []string{"work_summary"})

	schemaCandidateID = schemaObject(map[string]any{
		"candidate_id": propUUID("UUID of pending candidate."),
		"id":           propUUID("Alias for candidate_id."),
	}, nil)

	schemaCurationStrengthened = schemaObject(map[string]any{
		"min_support": propInt("Minimum support count (default 2).", 1, 100),
	}, nil)

	schemaChoreList = schemaObject(map[string]any{
		"limit": propInt("Max open chores to return (default 20).", 1, 100),
	}, nil)

	schemaChoreResolve = schemaObject(map[string]any{
		"chore_id": propUUID("UUID of the curation chore to vote on."),
		"id":       propUUID("Alias for chore_id."),
		"action":   propString("Vote action. contradiction: keep_subject|keep_related|coexist. quarantine_review: release|delete. duplicate_pair: consolidate|distinct."),
		"agent_id": propString("Required agent identifier; the action applies only after min_resolvers DISTINCT agents vote for it (the memory's own author never counts)."),
		"reason":   propString("Optional short justification (auditable)."),
	}, []string{"action", "agent_id"})

	schemaEpisodeSimilar = schemaObject(map[string]any{
		"query":           propString("Search text."),
		"summary_text":    propString("Alias for query."),
		"tags":            schemaTags(),
		"max_results":     propInt("Max results.", 1, 50),
		"occurred_after":  propString("ISO8601 lower bound."),
		"occurred_before": propString("ISO8601 upper bound."),
	}, nil)

	schemaEpisodeDistill = schemaObject(map[string]any{
		"episode_id": propUUID("Advisory episode UUID."),
		"summary":    propString("Summary when episode_id omitted."),
		"tags":       schemaTags(),
		"entities":   propStringArray("Optional entities."),
	}, nil)

	schemaRecallAdvanced = schemaObject(map[string]any{
		"query":            propString("Situation text."),
		"retrieval_query":  propString("Alias for query."),
		"mode":             propEnumString("Recall shaping.", "continuity", "constraint", "pattern", "episodic"),
		"tags":             schemaTags(),
		"symbols":          propStringArray("Optional code symbols."),
		"repo_root":        propString("Optional repo path."),
	}, []string{"query"})

	schemaPreflight = schemaObject(map[string]any{
		"changed_files_count": propInt("Number of changed files.", 0, 10000),
		"tags":                schemaTags(),
	}, nil)

	schemaContradictionDetect = schemaObject(map[string]any{
		"memory_id":        propUUID("First memory UUID."),
		"conflict_with_id": propUUID("Second memory UUID."),
	}, []string{"memory_id", "conflict_with_id"})

	schemaContradictionList = schemaObject(map[string]any{
		"resolution_state": propEnumString("Filter by resolution.", "unresolved", "override", "deprecated", "narrow_exception"),
		"memory_id":        propUUID("Filter by memory UUID."),
		"limit":            propInt("Max rows.", 1, 500),
	}, nil)

	schemaEvidenceAttach = schemaObject(map[string]any{
		"memory_id":     propUUID("Memory to link evidence to."),
		"evidence_text": propString("Plaintext evidence content."),
		"text":          propString("Alias for evidence_text."),
		"content":       propString("Alias for evidence_text."),
		"kind":          propString("Evidence kind (default note)."),
	}, nil)

	schemaEvidenceList = schemaObject(map[string]any{
		"memory_id": propUUID("Filter by memory UUID."),
		"kind":      propString("Filter by evidence kind."),
	}, nil)

	schemaRelationshipsGet = schemaObject(map[string]any{
		"memory_id": propUUID("Memory UUID."),
	}, []string{"memory_id"})

	schemaRelationshipsCreate = schemaObject(map[string]any{
		"from_memory_id":    propUUID("Source memory UUID."),
		"to_memory_id":      propUUID("Target memory UUID."),
		"relationship_type": propString("Edge type (e.g. contradicts, supersedes)."),
	}, []string{"from_memory_id", "to_memory_id", "relationship_type"})

	schemaComplianceSessionID = schemaObject(map[string]any{
		"session_id": propUUID("Agent session UUID from initialize pluribus.session_id or X-Pluribus-Session-Id header."),
	}, []string{"session_id"})

	schemaComplianceEvaluate = schemaObject(map[string]any{
		"session_id":        propUUID("Session UUID to evaluate."),
		"recall_max_age_ms": propInt("Optional recall staleness window in milliseconds (default 3600000).", 1, 86400000),
	}, []string{"session_id"})

	schemaTelemetrySessionStart = schemaObject(map[string]any{
		"session_id":  propString("Optional existing session UUID."),
		"interface":   propString("rest or mcp."),
		"agent_id":    propString("Agent or client identifier."),
		"client_name": propString("Client name."),
		"tags":        propStringArray("Context tags."),
	}, []string{"interface"})

	schemaTelemetrySessionID = schemaObject(map[string]any{
		"session_id": propUUID("Telemetry session UUID."),
	}, []string{"session_id"})

	schemaTelemetryMemoryID = schemaObject(map[string]any{
		"memory_id": propString("Memory identifier."),
	}, []string{"memory_id"})
)

// toolRegistry is the single source of truth for MCP tool registration.
func toolRegistry() []ToolSpec {
	recallDesc := "Use at the start of a substantive task, after context changes, or when uncertain whether prior constraints, decisions, failures, or patterns apply. Returns a bounded memory bundle (governing_constraints, failures, patterns, decisions, continuity) plus mcp_context. Mutates nothing. Part of the default agent loop (before complex work). Do not treat unrelated recall hits as binding — check applicability and authority. Same handler as memory_context_resolve." + layer1DefaultLoop
	recordDesc := "Use after solving a problem, encountering a failure, discovering a reusable pattern, or confirming a decision. Mutates state: creates advisory experience and may form probationary memory. Part of the default agent loop (after meaningful outcomes). Weak text may go to reject bucket only. Not a substitute for explicit memory_create for canonical constraints. Same handler as mcp_episode_ingest." + layer1DefaultLoop
	enforceDesc := "Use before large refactors, policy-sensitive edits, or high-risk proposals when binding memory may apply. Mutates nothing. Does NOT run automatically — you must call this tool. Engine is rule-based heuristic v1 (postgres/sqlite keywords, word overlap, negative patterns) — not full semantic NL policy enforcement. If decision is block or next_action is revise/reject, do not proceed without revision."

	return []ToolSpec{
		{Name: "recall_context", Description: recallDesc, InputSchema: schemaRecallContext, LoopRole: LoopSessionStart, Risk: RiskLow, Backend: "POST /v1/recall/compile (wrapped)", Output: "mcp_context + recall_bundle JSON", Mutates: false, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "record_experience", Description: recordDesc, InputSchema: schemaRecordExperience, LoopRole: LoopPostOutcome, Risk: RiskMedium, Backend: "POST /v1/advisory-episodes", Output: "advisory episode + formation status", Mutates: true, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "memory_log_if_relevant", Description: "Use when unsure if a text block contains learning signals. Deterministic gate — ingests only when tokens match; otherwise returns skipped JSON. Mutates only when ingest fires. Same path as record_experience when it fires.", InputSchema: schemaMemoryLogIfRelevant, LoopRole: LoopPostOutcome, Risk: RiskMedium, Backend: "POST /v1/advisory-episodes (conditional)", Output: "skipped or ingest result", Mutates: true, TestCoverage: "unit", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "auto_log_episode_if_relevant", Description: "Alias for memory_log_if_relevant.", InputSchema: schemaMemoryLogIfRelevant, LoopRole: LoopPostOutcome, Risk: RiskMedium, Backend: "POST /v1/advisory-episodes (conditional)", Output: "skipped or ingest result", Mutates: true, TestCoverage: "unit", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "enforcement_evaluate", Description: enforceDesc, InputSchema: schemaEnforcement, LoopRole: LoopPreChange, Risk: RiskLow, Backend: "POST /v1/enforcement/evaluate", Output: "decision, triggered_memories, validation", Mutates: false, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go + proof-mcp-docker.sh"},
		{Name: "recall_compile", Description: "Advanced: raw POST /v1/recall/compile with full CompileRequest body. Prefer recall_context for the default path. Mutates nothing.", InputSchema: schemaRecallCompilePassthrough, LoopRole: LoopNone, Risk: RiskLow, Backend: "POST /v1/recall/compile", Output: "recall bundle JSON", Mutates: false, TestCoverage: "unit", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "recall_get", Description: "GET-shaped recall when your workflow requires it. Prefer recall_context. Mutates nothing.", InputSchema: schemaRecallGet, LoopRole: LoopNone, Risk: RiskLow, Backend: "GET /v1/recall/", Output: "recall bundle JSON", Mutates: false, TestCoverage: "unit", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "wakeup_context", Description: "Use at session start for compact L0/L1 identity and governing memory via POST /v1/recall/wakeup. Mutates nothing. Complements recall_context (which needs task text). Optional caps: max_state, max_per_kind, max_governing_total.", InputSchema: wakeupContextInputSchema, LoopRole: LoopSessionStart, Risk: RiskLow, Backend: "POST /v1/recall/wakeup", Output: "wakeup bundle (identity, governing_memory)", Mutates: false, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go + proof-mcp-docker.sh"},
		{Name: "recall_run_multi", Description: "Advanced multi-variant recall orchestration. Optional after recall + enforcement. Mutates nothing unless promotion paths fire server-side.", InputSchema: schemaRecallRunMulti, LoopRole: LoopExperimental, Risk: RiskMedium, Backend: "POST /v1/recall/run-multi", Output: "run-multi response", Mutates: false, TestCoverage: "integration", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "memory_create", Description: "Author explicit durable memory (constraint, decision, etc.). Mutates canonical store. Use sparingly vs record_experience + curation. High impact — wrong writes pollute shared pool.", InputSchema: schemaMemoryCreate, LoopRole: LoopAdmin, Risk: RiskHigh, Backend: "POST /v1/memory", Output: "memory object", Mutates: true, TestCoverage: "integration", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "memory_feedback", Description: "Submit structured feedback about an existing memory's usefulness, harm, correctness, freshness, or relevance. Updates utility score (separate from authority). Use after recall when a memory helped or misled.", InputSchema: schemaMemoryFeedback, LoopRole: LoopPostOutcome, Risk: RiskMedium, Backend: "POST /v1/memory/{id}/feedback", Output: "feedback response with utility score", Mutates: true, TestCoverage: "unit", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "memory_feedback_test.go"},
		{Name: "memory_promote", Description: "Structured promote path per server policy. Mutates canonical store.", InputSchema: schemaMemoryPromote, LoopRole: LoopAdmin, Risk: RiskHigh, Backend: "POST /v1/memory/promote", Output: "promotion result", Mutates: true, TestCoverage: "integration", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "curation_digest", Description: "Turn bounded work_summary into digest proposals (pending candidates). Mutates candidate store — not canon until materialize.", InputSchema: schemaCurationDigest, LoopRole: LoopCuration, Risk: RiskMedium, Backend: "POST /v1/curation/digest", Output: "digest proposals", Mutates: true, TestCoverage: "integration", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "curation_pending", Description: "List pending candidates (not canonical memory). Mutates nothing.", InputSchema: schemaEmptyObject(), LoopRole: LoopCuration, Risk: RiskLow, Backend: "GET /v1/curation/pending", Output: "candidate list", Mutates: false, TestCoverage: "integration", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go + proof-mcp-docker.sh"},
		{Name: "curation_promotion_suggestions", Description: "Promotion readiness hints. Mutates nothing.", InputSchema: schemaEmptyObject(), LoopRole: LoopCuration, Risk: RiskLow, Backend: "GET /v1/curation/promotion-suggestions", Output: "suggestions", Mutates: false, TestCoverage: "integration", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "curation_strengthened", Description: "Candidates reinforced by multiple distill signals. Mutates nothing.", InputSchema: schemaCurationStrengthened, LoopRole: LoopCuration, Risk: RiskLow, Backend: "GET /v1/curation/strengthened", Output: "candidate list", Mutates: false, TestCoverage: "integration", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "curation_review_candidate", Description: "Review one candidate before promote. Mutates nothing.", InputSchema: schemaCandidateID, LoopRole: LoopCuration, Risk: RiskLow, Backend: "GET /v1/curation/candidates/{id}/review", Output: "review payload", Mutates: false, TestCoverage: "integration", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "curation_materialize", Description: "Promote validated candidate to durable memory. Mutates canonical store.", InputSchema: schemaCandidateID, LoopRole: LoopCuration, Risk: RiskHigh, Backend: "POST /v1/curation/candidates/{id}/materialize", Output: "materialize result", Mutates: true, TestCoverage: "integration", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "curation_promote_candidate", Description: "Alias for curation_materialize.", InputSchema: schemaCandidateID, LoopRole: LoopCuration, Risk: RiskHigh, Backend: "POST /v1/curation/candidates/{id}/materialize", Output: "materialize result", Mutates: true, TestCoverage: "integration", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "curation_reject_candidate", Description: "Reject a candidate so it will not become memory. Mutates candidate status.", InputSchema: schemaCandidateID, LoopRole: LoopCuration, Risk: RiskMedium, Backend: "POST /v1/curation/candidates/{id}/reject", Output: "reject result", Mutates: true, TestCoverage: "integration", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "curation_auto_promote", Description: "Batch auto-promote when server promotion.auto_promote enabled. May return 403 if disabled. Mutates canonical store.", InputSchema: schemaEmptyObject(), LoopRole: LoopCuration, Risk: RiskCritical, Backend: "POST /v1/curation/auto-promote", Output: "auto-promote result", Mutates: true, TestCoverage: "integration", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "list_chores", Description: "List open curation chores the hive would like reviewed (contradiction pairs, quarantined rows, near-duplicate pairs). When chores exist, agents should resolve or defer with reason (see initialize instructions); recall/wakeup also surface one housekeeping line. Mutates nothing.", InputSchema: schemaChoreList, LoopRole: LoopCuration, Risk: RiskLow, Backend: "GET /v1/curation/chores", Output: "chore list with statements and allowed actions", Mutates: false, TestCoverage: "unit", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "resolve_chore", Description: "Vote on a curation chore (from list_chores or the housekeeping line in recall/wakeup responses). One vote per agent per chore; the action applies only after min_resolvers DISTINCT agents agree, and a memory's own author never counts. Applied outcomes are reversible (supersede, pending, soft delete) — nothing a vote does can mint an active memory. Mutates chore votes; may apply corroborated resolution.", InputSchema: schemaChoreResolve, LoopRole: LoopCuration, Risk: RiskMedium, Backend: "POST /v1/curation/chores/{id}/resolve", Output: "vote receipt (recorded, counted, applied, state)", Mutates: true, TestCoverage: "unit", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "episode_search_similar", Description: "Search similar advisory episodes (non-reject bucket). Mutates nothing.", InputSchema: schemaEpisodeSimilar, LoopRole: LoopDiagnostic, Risk: RiskLow, Backend: "POST /v1/advisory-episodes/similar", Output: "similar episodes", Mutates: false, TestCoverage: "integration", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "episode_distill_explicit", Description: "Explicit distill to candidates when auto-distill off. Requires distillation.enabled. Mutates candidate store.", InputSchema: schemaEpisodeDistill, LoopRole: LoopCuration, Risk: RiskMedium, Backend: "POST /v1/episodes/distill", Output: "distill result", Mutates: true, TestCoverage: "integration", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "memory_recall_advanced", Description: "Mode-shaped compile (continuity|constraint|pattern|episodic). Prefer recall_context. Mutates nothing.", InputSchema: schemaRecallAdvanced, LoopRole: LoopNone, Risk: RiskLow, Backend: "POST /v1/recall/compile", Output: "recall bundle", Mutates: false, TestCoverage: "unit", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "memory_preflight_check", Description: "Quick risk hint from scope stats. Not a substitute for enforcement_evaluate on real proposal text. Mutates nothing.", InputSchema: schemaPreflight, LoopRole: LoopDiagnostic, Risk: RiskLow, Backend: "POST /v1/recall/preflight", Output: "preflight stats", Mutates: false, TestCoverage: "unit", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "memory_quarantine", Description: "Operator remediation: remove a memory from ALL recall (any mode) pending review. Non-destructive — row is preserved with status quarantined and an auditable reason. Use for poisoned, harmful, or wrong guidance that must stop surfacing immediately.", InputSchema: schemaMemoryRemediate, LoopRole: LoopAdmin, Risk: RiskHigh, Backend: "POST /v1/memory/{id}/quarantine", Output: "updated memory object", Mutates: true, TestCoverage: "unit", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "memory_delete", Description: "Operator remediation: soft-delete a memory (tombstone status deleted; excluded from all recall including historical). Canonical rows are never hard-deleted. Prefer memory_quarantine when the row may need review.", InputSchema: schemaMemoryRemediate, LoopRole: LoopAdmin, Risk: RiskHigh, Backend: "DELETE /v1/memory/{id}", Output: "updated memory object", Mutates: true, TestCoverage: "unit", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "memory_detect_contradictions", Description: "Compare two memory UUIDs for conflicts. May create contradiction record. Mutates contradiction store.", InputSchema: schemaContradictionDetect, LoopRole: LoopDiagnostic, Risk: RiskMedium, Backend: "POST /v1/contradictions/detect", Output: "detection result", Mutates: true, TestCoverage: "integration", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "memory_list_contradictions", Description: "Audit contradiction queue. Mutates nothing.", InputSchema: schemaContradictionList, LoopRole: LoopDiagnostic, Risk: RiskLow, Backend: "GET /v1/contradictions/", Output: "contradiction list", Mutates: false, TestCoverage: "integration", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "evidence_attach", Description: "Create evidence and link to memory. Mutates evidence store.", InputSchema: schemaEvidenceAttach, LoopRole: LoopAdmin, Risk: RiskMedium, Backend: "POST /v1/evidence + link", Output: "evidence ids", Mutates: true, TestCoverage: "unit", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "evidence_list", Description: "List evidence by memory_id or kind. Mutates nothing.", InputSchema: schemaEvidenceList, LoopRole: LoopDiagnostic, Risk: RiskLow, Backend: "GET /v1/evidence/", Output: "evidence list", Mutates: false, TestCoverage: "unit", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "memory_relationships_get", Description: "Inspect graph edges for a memory. Mutates nothing.", InputSchema: schemaRelationshipsGet, LoopRole: LoopDiagnostic, Risk: RiskLow, Backend: "GET /v1/memory/{id}/relationships", Output: "relationships", Mutates: false, TestCoverage: "integration", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "memory_relationships_create", Description: "Record typed edge between memories. Mutates relationship store.", InputSchema: schemaRelationshipsCreate, LoopRole: LoopAdmin, Risk: RiskMedium, Backend: "POST /v1/memory/relationships", Output: "relationship", Mutates: true, TestCoverage: "integration", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "memory_context_resolve", Description: "Compatibility alias for recall_context — identical behavior.", InputSchema: schemaRecallContext, LoopRole: LoopSessionStart, Risk: RiskLow, Backend: "POST /v1/recall/compile (wrapped)", Output: "mcp_context + recall_bundle", Mutates: false, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "mcp_episode_ingest", Description: "Compatibility alias for record_experience — identical behavior.", InputSchema: schemaRecordExperience, LoopRole: LoopPostOutcome, Risk: RiskMedium, Backend: "POST /v1/advisory-episodes", Output: "advisory episode", Mutates: true, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "compliance_summary", Description: "Phase 2: aggregate loop compliance counts across recent MCP sessions. Mutates nothing. Measures telemetry — does not enforce agent behavior.", InputSchema: schemaEmptyObject(), LoopRole: LoopDiagnostic, Risk: RiskLow, Backend: "GET /v1/compliance/summary", Output: "session counts by compliance status", Mutates: false, TestCoverage: "unit", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go + proof-agent-loop-compliance.sh"},
		{Name: "compliance_session_get", Description: "Phase 2: fetch one agent session row (client, transport, repo_root). Mutates nothing.", InputSchema: schemaComplianceSessionID, LoopRole: LoopDiagnostic, Risk: RiskLow, Backend: "GET /v1/compliance/sessions/{id}", Output: "session object", Mutates: false, TestCoverage: "unit", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "compliance_session_events", Description: "Phase 2: list redacted loop telemetry events for a session. Mutates nothing.", InputSchema: schemaComplianceSessionID, LoopRole: LoopDiagnostic, Risk: RiskLow, Backend: "GET /v1/compliance/sessions/{id}/events", Output: "events array", Mutates: false, TestCoverage: "unit", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
		{Name: "compliance_evaluate", Description: "Phase 2: evaluate whether MCP-visible calls satisfy recall → enforcement → record loop for a session. Mutates evaluation store only.", InputSchema: schemaComplianceEvaluate, LoopRole: LoopDiagnostic, Risk: RiskLow, Backend: "POST /v1/compliance/evaluate", Output: "status, missing_steps, evidence", Mutates: false, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go + proof-agent-loop-compliance.sh"},
		{Name: "agent_telemetry_start_session", Description: "Phase 11I: start persisted memory-use telemetry session. Mutates telemetry store.", InputSchema: schemaTelemetrySessionStart, LoopRole: LoopSessionStart, Risk: RiskLow, Backend: "POST /v1/agent/telemetry/session/start", Output: "telemetry_session JSON", Mutates: true, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "agenttelemetry persistence tests"},
		{Name: "agent_telemetry_record_recall", Description: "Phase 11I: persist recall exposure event. Mutates telemetry store.", InputSchema: schemaObject(map[string]any{"session_id": propUUID("Session UUID."), "task_id": propString("Task id."), "recalled_memory_ids": propStringArray("Recalled memory ids."), "recall_bundle": map[string]any{"type": "object"}}, []string{"session_id", "recalled_memory_ids", "recall_bundle"}), LoopRole: LoopSessionStart, Risk: RiskLow, Backend: "POST /v1/agent/telemetry/recall", Output: "telemetry_recall JSON", Mutates: true, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "agenttelemetry persistence tests"},
		{Name: "agent_telemetry_record_decision", Description: "Phase 11I: persist memory use decisions. Mutates telemetry store.", InputSchema: schemaObject(map[string]any{"session_id": propUUID("Session UUID."), "recall_event_id": propUUID("Recall event UUID."), "decisions": map[string]any{"type": "array"}}, []string{"session_id", "recall_event_id", "decisions"}), LoopRole: LoopPostOutcome, Risk: RiskLow, Backend: "POST /v1/agent/telemetry/decision", Output: "telemetry_decision JSON", Mutates: true, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "agenttelemetry persistence tests"},
		{Name: "agent_telemetry_record_output", Description: "Phase 11I: persist agent output facts/actions. Mutates telemetry store.", InputSchema: schemaObject(map[string]any{"session_id": propUUID("Session UUID."), "recall_event_id": propUUID("Recall event UUID."), "output_facts": propStringArray("Output facts."), "memory_citations": propStringArray("Memory citations.")}, []string{"session_id"}), LoopRole: LoopPostOutcome, Risk: RiskLow, Backend: "POST /v1/agent/telemetry/output", Output: "telemetry_output JSON", Mutates: true, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "agenttelemetry persistence tests"},
		{Name: "agent_telemetry_evaluate", Description: "Phase 11I: run deterministic obedience evaluator and persist violations/utility candidates. Mutates telemetry store.", InputSchema: schemaObject(map[string]any{"session_id": propUUID("Session UUID."), "recall_event_id": propUUID("Recall event UUID."), "output_id": propUUID("Output UUID.")}, []string{"session_id", "recall_event_id"}), LoopRole: LoopDiagnostic, Risk: RiskLow, Backend: "POST /v1/agent/telemetry/evaluate", Output: "telemetry_evaluation JSON", Mutates: true, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "agenttelemetry persistence tests"},
		{Name: "agent_telemetry_get_session", Description: "Phase 11I: query full session telemetry summary. Mutates nothing.", InputSchema: schemaTelemetrySessionID, LoopRole: LoopDiagnostic, Risk: RiskLow, Backend: "GET /v1/agent/telemetry/session/{id}", Output: "telemetry_session_summary JSON", Mutates: false, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "agenttelemetry persistence tests"},
		{Name: "agent_telemetry_get_memory", Description: "Phase 11I: query per-memory telemetry aggregates. Mutates nothing.", InputSchema: schemaTelemetryMemoryID, LoopRole: LoopDiagnostic, Risk: RiskLow, Backend: "GET /v1/agent/telemetry/memory/{id}", Output: "telemetry_memory_summary JSON", Mutates: false, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "agenttelemetry persistence tests"},
		{Name: "agent_telemetry_get_violations", Description: "Phase 11I: query persisted violations. Mutates nothing.", InputSchema: schemaObject(map[string]any{"memory_id": propString("Filter by memory id."), "violation_code": propString("Filter by violation code.")}, nil), LoopRole: LoopDiagnostic, Risk: RiskLow, Backend: "GET /v1/agent/telemetry/violations", Output: "telemetry_violations JSON", Mutates: false, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "agenttelemetry persistence tests"},
		{Name: "agent_telemetry_get_utility_candidates", Description: "Phase 11I: query utility candidates (not auto-applied). Mutates nothing.", InputSchema: schemaObject(map[string]any{"memory_id": propString("Filter by memory id.")}, nil), LoopRole: LoopDiagnostic, Risk: RiskLow, Backend: "GET /v1/agent/telemetry/utility-candidates", Output: "telemetry_utility_candidates JSON", Mutates: false, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "agenttelemetry persistence tests"},
		{Name: "agent_utility_evaluate_candidate", Description: "Phase 11K: evaluate utility candidate through policy without mutation.", InputSchema: schemaObject(map[string]any{"candidate_id": propUUID("Utility candidate UUID.")}, []string{"candidate_id"}), LoopRole: LoopDiagnostic, Risk: RiskLow, Backend: "POST /v1/agent/utility/policy/evaluate-candidate", Output: "policy_decision JSON", Mutates: false, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "utilitypolicy persistence tests"},
		{Name: "agent_utility_apply_candidate", Description: "Phase 11K: apply utility candidate through guarded policy. Mutates utility scores when approved.", InputSchema: schemaObject(map[string]any{"candidate_id": propUUID("Utility candidate UUID."), "applied_by": propString("Actor applying policy.")}, []string{"candidate_id"}), LoopRole: LoopPostOutcome, Risk: RiskMedium, Backend: "POST /v1/agent/utility/policy/apply-candidate", Output: "utility_application JSON", Mutates: true, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "utilitypolicy persistence tests"},
		{Name: "agent_utility_apply_batch", Description: "Phase 11K: batch apply utility candidates through policy caps.", InputSchema: schemaObject(map[string]any{"candidate_ids": propStringArray("Candidate UUIDs."), "applied_by": propString("Actor.")}, []string{"candidate_ids"}), LoopRole: LoopPostOutcome, Risk: RiskMedium, Backend: "POST /v1/agent/utility/policy/apply-batch", Output: "utility_applications JSON", Mutates: true, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "utilitypolicy persistence tests"},
		{Name: "agent_utility_revert_application", Description: "Phase 11K: revert policy utility application by rollback token.", InputSchema: schemaObject(map[string]any{"rollback_token": propString("Rollback token."), "revert_reason": propString("Reason.")}, []string{"rollback_token"}), LoopRole: LoopPostOutcome, Risk: RiskMedium, Backend: "POST /v1/agent/utility/policy/revert-application", Output: "utility_application JSON", Mutates: true, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "utilitypolicy persistence tests"},
		{Name: "agent_utility_get_candidate", Description: "Phase 11K: query utility application for candidate.", InputSchema: schemaObject(map[string]any{"candidate_id": propUUID("Candidate UUID.")}, []string{"candidate_id"}), LoopRole: LoopDiagnostic, Risk: RiskLow, Backend: "GET /v1/agent/utility/policy/candidate/{id}", Output: "utility_application JSON", Mutates: false, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "utilitypolicy persistence tests"},
		{Name: "agent_utility_get_memory_history", Description: "Phase 11K: query utility application history for memory.", InputSchema: schemaObject(map[string]any{"memory_id": propString("Memory id.")}, []string{"memory_id"}), LoopRole: LoopDiagnostic, Risk: RiskLow, Backend: "GET /v1/agent/utility/policy/memory/{id}", Output: "utility_applications JSON", Mutates: false, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "utilitypolicy persistence tests"},
		{Name: "agent_utility_get_applications", Description: "Phase 11K: list recent utility policy applications.", InputSchema: schemaEmptyObject(), LoopRole: LoopDiagnostic, Risk: RiskLow, Backend: "GET /v1/agent/utility/policy/applications", Output: "utility_applications JSON", Mutates: false, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "utilitypolicy persistence tests"},
		{Name: "agent_utility_get_policy_summary", Description: "Phase 11K: policy application summary metrics.", InputSchema: schemaEmptyObject(), LoopRole: LoopDiagnostic, Risk: RiskLow, Backend: "GET /v1/agent/utility/policy/summary", Output: "policy_summary JSON", Mutates: false, TestCoverage: "both", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "utilitypolicy persistence tests"},
		{Name: "health", Description: "Liveness probe (GET /healthz). Mutates nothing. Use to verify MCP transport only — not recall quality.", InputSchema: schemaEmptyObject(), LoopRole: LoopDiagnostic, Risk: RiskLow, Backend: "GET /healthz", Output: "ok JSON", Mutates: false, TestCoverage: "unit", CallCoverage: CoverageUnitToolsCall, CallCoverageNote: "tool_call_coverage_test.go"},
	}
}

// ToolSpecByName returns registration metadata for a tool name.
func ToolSpecByName(name string) (ToolSpec, bool) {
	for _, t := range toolRegistry() {
		if t.Name == name {
			return t, true
		}
	}
	return ToolSpec{}, false
}

// RegisteredToolNames returns sorted tool names from the registry.
func RegisteredToolNames() []string {
	reg := toolRegistry()
	names := make([]string, len(reg))
	for i, t := range reg {
		names[i] = t.Name
	}
	return names
}
