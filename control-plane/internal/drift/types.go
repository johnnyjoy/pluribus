package drift

// SymbolPosition identifies a symbol location for LSP reference lookup (Task 100).
type SymbolPosition struct {
	Path   string `json:"path"`   // file path relative to repo root or absolute
	Line   int    `json:"line"`   // 1-based line
	Column int    `json:"column"` // 1-based column (UTF-8 code units)
}

// CheckRequest is the payload for POST /v1/drift/check.
type CheckRequest struct {
	Proposal string   `json:"proposal"` // text to check (e.g. LLM output)
	Tags     []string `json:"tags,omitempty"`
	// StructuralSignals optional (Task 77): file-change/dependency/boundary context for risk escalation.
	StructuralSignals *StructuralSignals `json:"structural_signals,omitempty"`
	// Task 95: when true, preflight indicated slow-path; drift may set RequiresFollowupCheck.
	SlowPathRequired bool `json:"slow_path_required,omitempty"`
	// When true, this is the second (follow-up) drift check for the same proposal; do not set RequiresFollowupCheck.
	IsFollowupCheck bool `json:"is_followup_check,omitempty"`
	// Task 100: repo root path for LSP; when set with TouchedSymbols and LSP configured, reference count can escalate risk.
	RepoRoot string `json:"repo_root,omitempty"`
	// Task 100: symbols touched by the proposal; reference count above LSPHighRiskReferenceThreshold escalates risk.
	TouchedSymbols []SymbolPosition `json:"touched_symbols,omitempty"`
}

// StructuralSignals are inputs for risk assessment (change count, dependency impact, boundary violations).
type StructuralSignals struct {
	ChangeCount            int     `json:"change_count"`              // number of files changed
	DependencyChangeScore  float64 `json:"dependency_change_score"`    // 0..1 impact score
	BoundaryViolationCount int     `json:"boundary_violation_count"`  // module boundary violations
}

// CheckResult is the response from a drift check.
type CheckResult struct {
	Passed        bool         `json:"passed"`
	Violations    []DriftIssue `json:"violations"`
	Warnings      []string     `json:"warnings,omitempty"`
	PatternViolationCount int `json:"pattern_violation_count,omitempty"`
	PatternWarningCount   int `json:"pattern_warning_count,omitempty"`
	RiskLevel     string       `json:"risk_level,omitempty"`     // "low"|"medium"|"high" when structural signals provided (Task 77)
	BlockExecution bool       `json:"block_execution,omitempty"` // true when risk is high
	// Task 95: when true, a second drift check is required before allowing execution (slow-path).
	RequiresFollowupCheck bool   `json:"requires_followup_check,omitempty"`
	FollowupReason        string `json:"followup_reason,omitempty"`
}

// DriftIssue is a single violation (constraint, failure, failure_pattern).
type DriftIssue struct {
	Code      string  `json:"code"`      // "constraint", "failure", "failure_pattern"
	Statement string  `json:"statement"`
	Score     float64 `json:"score,omitempty"` // optional similarity/overlap score for fuzzy patterns
}
