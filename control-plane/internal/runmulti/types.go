// Package runmulti implements the multi-run execution loop.
// compile-multi → LLM per variant → drift check → score → select best.

package runmulti

// RunResult holds the result of one variant run (LLM output + drift + score).
type RunResult struct {
	Variant   string     `json:"variant"`
	Output    string     `json:"output,omitempty"`
	Drift     DriftResult `json:"drift,omitempty"`
	Score     float64    `json:"score"`
	Rejected  bool       `json:"rejected"`
}

// RunMultiResult is the structured output of run-multi (all runs + selected).
type RunMultiResult struct {
	Runs     []RunResult  `json:"runs"`
	Selected *RunResult   `json:"selected,omitempty"`
}

// DriftResult mirrors drift.CheckResult for JSON (CLI does not import control-plane drift).
type DriftResult struct {
	Passed         bool        `json:"passed"`
	Violations     []DriftIssue `json:"violations,omitempty"`
	Warnings       []string    `json:"warnings,omitempty"`
	RiskLevel      string      `json:"risk_level,omitempty"`
	BlockExecution bool        `json:"block_execution,omitempty"`
	// Second drift pass when slow-path + RequireSecondDriftCheck on server.
	RequiresFollowupCheck bool   `json:"requires_followup_check,omitempty"`
	FollowupReason        string `json:"followup_reason,omitempty"`
}

// DriftIssue mirrors drift.DriftIssue for JSON.
type DriftIssue struct {
	Code      string  `json:"code"`
	Statement string  `json:"statement"`
	Score     float64 `json:"score,omitempty"`
}

// --- API mirrors for HTTP (compile-multi, drift check) ---

// RecommendedExpansionMirror matches recall.RecommendedExpansion JSON.
type RecommendedExpansionMirror struct {
	ConstraintsDelta   int `json:"constraints_delta"`
	FailuresDelta      int `json:"failures_delta"`
	PatternsDelta      int `json:"patterns_delta"`
}

// CompileMultiRequest is the JSON body for POST /v1/recall/compile-multi.
type CompileMultiRequest struct {
	Tags      []string `json:"tags,omitempty"`
	Symbols   []string `json:"symbols,omitempty"`
	Variants  int      `json:"variants,omitempty"`
	Strategy  string   `json:"strategy,omitempty"`
	MaxPerKind int     `json:"max_per_kind,omitempty"`
	MaxTotal   int     `json:"max_total,omitempty"`
	MaxTokens  int     `json:"max_tokens,omitempty"`
	// forwarded to recall compile-multi / compile per variant.
	SlowPathRequired       bool                        `json:"slow_path_required,omitempty"`
	SlowPathReasons        []string                    `json:"slow_path_reasons,omitempty"`
	RecommendedExpansion   *RecommendedExpansionMirror `json:"recommended_expansion,omitempty"`
	// Server-side compile-multi can infer slow-path from this when set (same as recall.CompileMultiRequest).
	ChangedFilesCount *int `json:"changed_files_count,omitempty"`
	// same fields as recall.CompileMultiRequest for LSP-backed recall.
	RepoRoot       string `json:"repo_root,omitempty"`
	LSPFocusPath   string `json:"lsp_focus_path,omitempty"`
	LSPFocusLine   int    `json:"lsp_focus_line,omitempty"`
	LSPFocusColumn int    `json:"lsp_focus_column,omitempty"`
	RetrievalQuery string `json:"retrieval_query,omitempty"`
}

// CompileMultiResponse is the JSON response from compile-multi.
type CompileMultiResponse struct {
	Bundles []VariantBundleMirror `json:"bundles"`
}

// VariantBundleMirror pairs variant name with bundle (API response shape).
type VariantBundleMirror struct {
	Variant string         `json:"variant"`
	Bundle  RecallBundleMirror `json:"bundle"`
}

// RecallBundleMirror is the recall bundle shape for context building (subset of API).
type RecallBundleMirror struct {
	GoverningConstraints []MemoryItemMirror   `json:"governing_constraints"`
	Decisions            []MemoryItemMirror   `json:"decisions"`
	KnownFailures        []MemoryItemMirror   `json:"known_failures"`
	ApplicablePatterns   []MemoryItemMirror   `json:"applicable_patterns"`
}

// MemoryItemMirror has Statement for context building.
type MemoryItemMirror struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Statement string `json:"statement"`
}

// DriftCheckRequest is the JSON body for POST /v1/drift/check.
type DriftCheckRequest struct {
	Proposal         string   `json:"proposal"`
	Tags             []string `json:"tags,omitempty"`
	SlowPathRequired bool     `json:"slow_path_required,omitempty"`
	IsFollowupCheck  bool     `json:"is_followup_check,omitempty"`
}

// --- Preflight ---

// PreflightRequestMirror is the JSON body for POST /v1/recall/preflight.
type PreflightRequestMirror struct {
	ChangedFilesCount int      `json:"changed_files_count,omitempty"`
	Tags              []string `json:"tags,omitempty"`
}

// PreflightResultMirror mirrors recall.PreflightResult JSON.
type PreflightResultMirror struct {
	RiskLevel              string                      `json:"risk_level"`
	RequiredActions        []string                    `json:"required_actions"`
	RiskScore              float64                     `json:"risk_score,omitempty"`
	SlowPathRequired       bool                        `json:"slow_path_required,omitempty"`
	SlowPathReasons        []string                    `json:"slow_path_reasons,omitempty"`
	RecommendedExpansion   *RecommendedExpansionMirror `json:"recommended_expansion,omitempty"`
}
