package recall

import (
	"fmt"
	"strings"
	"time"

	"control-plane/internal/memory"
	"control-plane/internal/merge"

	"github.com/google/uuid"
)

// CompileRequest is the payload for POST /v1/recall/compile.
type CompileRequest struct {
	// AgentID optional opaque client identifier for salience / reinforcement only (never used for search or ranking filters).
	AgentID string `json:"agent_id,omitempty"`
	// RetrievalQuery drives candidate retrieval and lexical similarity (situation / intent text).
	RetrievalQuery string `json:"retrieval_query,omitempty"`
	// ProposalText optional; used with enable_triggered_recall for heuristic trigger detection.
	ProposalText string `json:"proposal_text,omitempty"`
	// EnableTriggeredRecall when true runs DetectTriggers and merges RetrievalQuery before compile (opt-in).
	EnableTriggeredRecall bool `json:"enable_triggered_recall,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Symbols     []string  `json:"symbols,omitempty"`       // Task 100: LSP symbol names for overlap scoring
	// when lsp.enabled and symbols omitted, server may auto-fill from documentSymbol.
	RepoRoot       string `json:"repo_root,omitempty"`
	LSPFocusPath   string `json:"lsp_focus_path,omitempty"`
	LSPFocusLine   int    `json:"lsp_focus_line,omitempty"`   // reserved for narrowing (8.2+)
	LSPFocusColumn int    `json:"lsp_focus_column,omitempty"` // reserved
	MaxPerKind  int       `json:"max_per_kind,omitempty"`  // default 5
	MaxTotal    int       `json:"max_total,omitempty"`     // RIE: cap total items (0 = no cap)
	MaxTokens   int       `json:"max_tokens,omitempty"`   // RIE: cap total tokens (0 = no cap)
	// Task 94: when set, compiler expands limits and annotates bundle with slow-path metadata.
	SlowPathRequired   bool                  `json:"slow_path_required,omitempty"`
	SlowPathReasons    []string              `json:"slow_path_reasons,omitempty"`
	RecommendedExpansion *RecommendedExpansion `json:"recommended_expansion,omitempty"`
	// VariantModifier optionally overrides limits and ranking for this compile.
	VariantModifier *VariantModifier `json:"variant_modifier,omitempty"`
	// Mode controls grouped recall view shaping on the same memory pool.
	// - "continuity" (default): balanced continuity/constraints/experience
	// - "thread": bias toward current-thread continuity slice
	Mode string `json:"mode,omitempty"`
	// CorrelationID optional client session id; boosts memories tagged mcp:session:<id> in ranking (does not filter global pool).
	CorrelationID string `json:"correlation_id,omitempty"`
}

// TriggerMetadata is included on RecallBundle when enable_triggered_recall is used (see triggered.go).
type TriggerMetadata struct {
	Triggers                []TriggerDecision `json:"triggers,omitempty"`
	RetrievalQueryEffective string            `json:"retrieval_query_effective,omitempty"`
	SkippedReason           string            `json:"skipped_reason,omitempty"`
}

// SemanticRetrievalDebug records how semantic (embedding) candidate retrieval behaved for this compile.
// Present when recall.semantic_retrieval is enabled and the request carried non-empty situation text.
type SemanticRetrievalDebug struct {
	Attempted      bool   `json:"attempted"`
	Path           string `json:"path"` // semantic_hybrid | lexical_only
	FallbackReason string `json:"fallback_reason,omitempty"`
}

// RecallBundle is the compiled immediate memory for a request (governing memory from the shared pool).
type RecallBundle struct {
	GoverningConstraints []MemoryItem     `json:"governing_constraints"`
	Decisions            []MemoryItem     `json:"decisions"`
	KnownFailures        []MemoryItem     `json:"known_failures"`
	ApplicablePatterns   []MemoryItem     `json:"applicable_patterns"`
	// Task 94: set when slow-path expansion was applied.
	SlowPathApplied  bool          `json:"slow_path_applied,omitempty"`
	SlowPathReasons  []string      `json:"slow_path_reasons,omitempty"`
	BaseLimits       *BucketLimits `json:"base_limits,omitempty"`
	ExpandedLimits   *BucketLimits `json:"expanded_limits,omitempty"`
	// Task 101: LSP/symbol diagnostics (when compile symbols overlap object-lesson payload symbols).
	MatchedSymbols       []string `json:"matched_symbols,omitempty"`
	ReferenceCount       int      `json:"reference_count,omitempty"`       // optional; 0 if not computed
	SymbolRelevanceReason string `json:"symbol_relevance_reason,omitempty"` // short explanation when symbol overlap applied
	// RIU (Recall Intelligence Upgrade): unresolved contradiction pairs surfaced when policy is bounded_pair.
	ContradictionSets []ContradictionSet `json:"contradiction_sets,omitempty"`
	// ConflictResolutions records winner/loser per unresolved pair when pair-wise resolution ran (Phase E, creative C3).
	ConflictResolutions []ConflictResolution `json:"conflict_resolutions,omitempty"`
	// EvidenceBudgetApplied is true when global max_per_bundle stopped attaching evidence before all memories were considered.
	EvidenceBudgetApplied bool `json:"evidence_budget_applied,omitempty"`
	// Continuity groups state + decision memories (resume / thread context).
	Continuity []MemoryItem `json:"continuity"`
	// Constraints groups failure + constraint memories (guardrails).
	Constraints []MemoryItem `json:"constraints"`
	// Experience groups reusable patterns.
	Experience []MemoryItem `json:"experience"`
	// AgentGrounding is plain-text Continuity / Constraints / Experience for agent prompts (derived from grouped slices or buckets).
	AgentGrounding *AgentGrounding `json:"agent_grounding,omitempty"`
	// TriggerMetadata is set when POST /v1/recall/compile uses enable_triggered_recall and triggered_recall is enabled in config.
	TriggerMetadata *TriggerMetadata `json:"trigger_metadata,omitempty"`
	// SemanticRetrieval is set when semantic retrieval is enabled and retrieval_query (or effective situation text) was non-empty.
	SemanticRetrieval *SemanticRetrievalDebug `json:"semantic_retrieval,omitempty"`
	// RecallPreamble is a single neutral line when any memory is present (deterministic; not marketing copy).
	RecallPreamble string `json:"recall_preamble,omitempty"`
}

// EvidenceInBundleConfig controls bounded supporting evidence in recall bundles (YAML: recall.evidence_in_bundle).
type EvidenceInBundleConfig struct {
	Enabled         bool `yaml:"enabled" json:"enabled"`
	MaxPerMemory    int  `yaml:"max_per_memory" json:"max_per_memory"`
	MaxPerBundle    int  `yaml:"max_per_bundle" json:"max_per_bundle"`
	SummaryMaxChars int  `yaml:"summary_max_chars" json:"summary_max_chars"`
}

// EvidenceRef is compact supporting metadata for a linked evidence record (no artifact body).
type EvidenceRef struct {
	ID      string `json:"id"`
	Kind    string `json:"kind,omitempty"`
	Title   string `json:"title,omitempty"`
	Summary string `json:"summary,omitempty"`
	Ref     string `json:"ref,omitempty"` // storage path hint; fetch full detail via GET /v1/evidence/{id} if needed
}

// BucketLimits records per-bucket limits used for compilation.
type BucketLimits struct {
	Constraints   int `json:"constraints"`
	Decisions     int `json:"decisions"`
	Failures      int `json:"failures"`
	Patterns      int `json:"patterns"`
}

// ContradictionSet is a bounded pair of memory IDs in unresolved contradiction (RIU bounded_pair mode).
type ContradictionSet struct {
	PairID string   `json:"pair_id,omitempty"`
	Items  []string `json:"items"`
	Reason string   `json:"reason,omitempty"`
}

// ConflictResolution is one pairwise resolution for POST /v1/recall/compile (Phase E).
type ConflictResolution struct {
	WinnerMemoryID string `json:"winner_memory_id"`
	LoserMemoryID  string `json:"loser_memory_id"`
	Reason         string `json:"reason,omitempty"`
}

// ContradictionPolicy controls how memories involved in unresolved contradictions are handled (RIU).
type ContradictionPolicy string

const (
	ContradictionPolicyExclude     ContradictionPolicy = "exclude"      // default: drop from recall (RIE default)
	ContradictionPolicyWarn        ContradictionPolicy = "warn"         // include with penalty + status
	ContradictionPolicyBoundedPair ContradictionPolicy = "bounded_pair" // warn + surface pairs in ContradictionSets
)

// ParseContradictionPolicy parses YAML/config values into ContradictionPolicy.
func ParseContradictionPolicy(s string) ContradictionPolicy {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "warn":
		return ContradictionPolicyWarn
	case "bounded_pair", "bounded-pair", "boundedpair":
		return ContradictionPolicyBoundedPair
	default:
		return ContradictionPolicyExclude
	}
}

// RIUWeights holds additive / penalty weights for Recall Intelligence Upgrade (RIU) scoring.
type RIUWeights struct {
	Applicability        float64 // tag + enum applicability
	Transferable         float64 // global + domain/tag alignment
	LineageProxy         float64 // proxy from authority / recency when lineage tables are absent
	ContradictionPenalty float64 // subtract when memory is in unresolved contradiction (warn / bounded_pair)
}

// DefaultRIUWeights returns tunable defaults (RIU off in config uses none of this until enabled).
func DefaultRIUWeights() RIUWeights {
	return RIUWeights{
		Applicability:        0.4,
		Transferable:         0.35,
		LineageProxy:         0.15,
		ContradictionPenalty: 0.5,
	}
}

// RIUConfig gates RIU behavior on the compiler. Nil or Enabled=false preserves ranking-only (pre-RIU) behavior.
type RIUConfig struct {
	Enabled        bool
	Policy         ContradictionPolicy
	Weights        RIUWeights
	BoundedPairMax int // max pairs in ContradictionSets; 0 = default 16
}

// RecallCandidate wraps compute-once fields for RIU scoring (Pluribus RIU plan §5.1).
type RecallCandidate struct {
	Object              memory.MemoryObject
	Transferable        bool
	ContradictionStatus string // none | unresolved
}

// RIUScoreBreakdown is attached to MemoryItem when RIU is enabled (inspectable components).
type RIUScoreBreakdown struct {
	RankingScore         float64 `json:"ranking_score"`
	ApplicabilityScore   float64 `json:"applicability_score"`
	TransferableScore    float64 `json:"transferable_score"`
	LineageProxyScore    float64 `json:"lineage_proxy_score"`
	ContradictionPenalty float64 `json:"contradiction_penalty"`
	TotalScore           float64 `json:"total_score"`
	Transferable         bool    `json:"transferable"`
	ContradictionStatus  string  `json:"contradiction_status"`
}

// MemoryItem is a minimal memory view for the bundle (statement, authority, kind).
type MemoryItem struct {
	ID            string            `json:"id"`
	Kind          string            `json:"kind"`
	Statement     string            `json:"statement"`
	Authority     int                `json:"authority"`
	// OccurredAt is set when the underlying memory row has event time (canonical occurred_at).
	OccurredAt   *time.Time         `json:"occurred_at,omitempty"`
	Justification *JustificationMeta `json:"justification,omitempty"`
	RIU           *RIUScoreBreakdown `json:"riu,omitempty"`
	// SupportingEvidence is optional bounded proof for this memory (evidence-in-recall; memory remains authoritative).
	SupportingEvidence []EvidenceRef `json:"supporting_evidence,omitempty"`
	// WhyMatters is a short deterministic line: role of this memory + rank rationale (+ evidence when hydrated).
	WhyMatters string `json:"why_matters,omitempty"`
	// SessionLocal is true when this row matched correlation_id session tagging (mcp:session:*) for ranking.
	SessionLocal bool `json:"session_local,omitempty"`
}

// JustificationMeta explains why a memory was recalled (RIE / Task 73).
type JustificationMeta struct {
	Reason string  `json:"reason"`
	Score  float64 `json:"score"`
}

// PreflightRequest is the payload for POST /v1/recall/preflight.
type PreflightRequest struct {
	ChangedFilesCount int      `json:"changed_files_count,omitempty"`
	Tags             []string  `json:"tags,omitempty"`
}

// PreflightResult is the response from Preflight (risk level and required actions).
type PreflightResult struct {
	RiskLevel         string                `json:"risk_level"`          // "low", "medium", "high"
	RequiredActions   []string              `json:"required_actions"`    // e.g. "deep_recall", "drift_check"
	RiskScore         float64               `json:"risk_score,omitempty"` // numeric 0–1 for comparison with slow_path threshold (Task 93)
	SlowPathRequired  bool                  `json:"slow_path_required,omitempty"`
	SlowPathReasons   []string              `json:"slow_path_reasons,omitempty"`
	RecommendedExpansion *RecommendedExpansion `json:"recommended_expansion,omitempty"`
}

// RecommendedExpansion holds deltas to apply to recall bucket limits when slow-path is required (Task 93).
type RecommendedExpansion struct {
	ConstraintsDelta int `json:"constraints_delta"`
	FailuresDelta    int `json:"failures_delta"`
	PatternsDelta    int `json:"patterns_delta"`
}

// SlowPathPreflightConfig is the slow-path policy used by Preflight (populated from app.Config.SlowPath in main).
type SlowPathPreflightConfig struct {
	Enabled             bool
	HighRiskThreshold   float64
	ExpandConstraintsBy int
	ExpandFailuresBy    int
	ExpandPatternsBy    int
	// ExtraVariantsWhenSlow adds N to compile-multi variant count when slow-path is active (capped by default strategy list length).
	ExtraVariantsWhenSlow int
}

// --- Multi-context recall ---

// CompileMultiRequest is the payload for POST /v1/recall/compile-multi.
type CompileMultiRequest struct {
	Tags      []string  `json:"tags,omitempty"`
	Symbols   []string  `json:"symbols,omitempty"`
	// same as CompileRequest — optional LSP context for auto symbols.
	RepoRoot       string `json:"repo_root,omitempty"`
	LSPFocusPath   string `json:"lsp_focus_path,omitempty"`
	LSPFocusLine   int    `json:"lsp_focus_line,omitempty"`
	LSPFocusColumn int    `json:"lsp_focus_column,omitempty"`
	Variants  int       `json:"variants,omitempty"`  // default 3
	Strategy  string    `json:"strategy,omitempty"`  // "default" = balanced, failure_heavy, authority_heavy
	// RIE caps (optional); 0 = compiler default behavior for unset.
	MaxPerKind int `json:"max_per_kind,omitempty"`
	MaxTotal   int `json:"max_total,omitempty"`
	MaxTokens  int `json:"max_tokens,omitempty"`
	// Slow-path expansion for all variants (balanced variant keeps expanded limits; failure_heavy/authority_heavy may override per-bucket via VariantModifier — same as Compile).
	SlowPathRequired       bool                  `json:"slow_path_required,omitempty"`
	SlowPathReasons        []string              `json:"slow_path_reasons,omitempty"`
	RecommendedExpansion *RecommendedExpansion `json:"recommended_expansion,omitempty"`
	// ChangedFilesCount when set (non-nil) and slow_path is not explicitly requested triggers the same preflight+slow-path logic as POST /v1/recall/preflight (uses preflight cache when configured).
	ChangedFilesCount *int `json:"changed_files_count,omitempty"`
	// RetrievalQuery forwarded to each variant compile (same as CompileRequest.retrieval_query).
	RetrievalQuery string `json:"retrieval_query,omitempty"`
	// AgentID optional opaque client identifier for salience / reinforcement only (not used for recall search).
	AgentID string `json:"agent_id,omitempty"`
}

// CompileMultiResponse is the response from compile-multi.
type CompileMultiResponse struct {
	Bundles []VariantBundle `json:"bundles"`
}

// VariantBundle pairs a variant name with its recall bundle.
type VariantBundle struct {
	Variant string       `json:"variant"`
	Bundle  RecallBundle `json:"bundle"`
}

// VariantModifier is applied to a single compile to produce a variant (limits and optional ranking).
type VariantModifier struct {
	Limits  *VariantLimits  `json:"limits,omitempty"`  // nil = use defaults from maxPerKind
	Ranking *RankingWeights `json:"ranking,omitempty"` // nil = use Compiler.Ranking
}

// VariantLimits overrides per-kind limits. Nil field = use default; non-nil = use value (min 0).
type VariantLimits struct {
	Constraints   *int `json:"constraints,omitempty"`
	Decisions     *int `json:"decisions,omitempty"`
	Failures      *int `json:"failures,omitempty"`
	Patterns      *int `json:"patterns,omitempty"`
}

// RunMultiRequest is the payload for POST /v1/recall/run-multi (Pluribus Phase A).
type RunMultiRequest struct {
	Query     string          `json:"query"`
	Inputs    []string        `json:"inputs,omitempty"`
	Merge     bool            `json:"merge,omitempty"`
	Promote   bool            `json:"promote,omitempty"`
	Tags      []string        `json:"tags,omitempty"`
	// AgentID optional opaque client identifier for salience / reinforcement only (not used for recall search).
	AgentID   string          `json:"agent_id,omitempty"`
	Variants  int             `json:"variants,omitempty"`  // default 3 when 0
	Strategy  string          `json:"strategy,omitempty"`  // default "default"
	Symbols   []string        `json:"symbols,omitempty"`
	// RIE / compile-multi
	MaxPerKind int `json:"max_per_kind,omitempty"`
	MaxTotal   int `json:"max_total,omitempty"`
	MaxTokens  int `json:"max_tokens,omitempty"`
	// Optional slow-path for compile-multi / drift (explicit). Overrides preflight-derived slow-path when both apply to runner.
	SlowPathRequired     bool                  `json:"slow_path_required,omitempty"`
	SlowPathReasons      []string              `json:"slow_path_reasons,omitempty"`
	RecommendedExpansion *RecommendedExpansion `json:"recommended_expansion,omitempty"`
	// When set, runner calls preflight with this count (HTTP) unless slow_path is explicitly set above.
	ChangedFilesCount *int `json:"changed_files_count,omitempty"`
	// LSP context for compile-multi (auto symbols + reference_count); forwarded to runner HTTP body.
	RepoRoot       string `json:"repo_root,omitempty"`
	LSPFocusPath   string `json:"lsp_focus_path,omitempty"`
	LSPFocusLine   int    `json:"lsp_focus_line,omitempty"`
	LSPFocusColumn int    `json:"lsp_focus_column,omitempty"`
	// Merge tuning (only when merge=true); all zero/false = default merge behavior.
	MergeStrictConflicts              bool    `json:"merge_strict_conflicts,omitempty"` // Phase 5.3: stricter pairwise conflict marking
	MergeMaxUniqueBullets             int     `json:"merge_max_unique_bullets,omitempty"`
	MergeDedupeSimilarUniques         bool    `json:"merge_dedupe_similar_uniques,omitempty"`
	MergeDropUniqueSimilarToAgreement float64 `json:"merge_drop_unique_similar_to_agreement,omitempty"` // 0 = off; e.g. 0.85 drops redundant uniques vs CORE
	// Optional evidence UUIDs to link to the promoted memory (global evidence pool).
	EvidenceIDs []uuid.UUID `json:"evidence_ids,omitempty"`
	// EnableTriggeredRecall enriches compile-multi retrieval_query from heuristics (server; requires recall.triggered_recall.enabled).
	EnableTriggeredRecall bool `json:"enable_triggered_recall,omitempty"`
	// RetrievalQuery optional situation / intent text forwarded to compile-multi (merged with triggered recall when enabled).
	RetrievalQuery string `json:"retrieval_query,omitempty"`
}

// RunMultiDebug is mandatory observability for run-multi (Pluribus phase-next).
// Top-level maps are always JSON-encoded (may be empty) for a stable client contract.
type RunMultiDebug struct {
	SignalBreakdown   map[string]any `json:"signal_breakdown"`
	FilterReasons     map[string]any `json:"filter_reasons"`
	PromotionDecision map[string]any `json:"promotion_decision"`
	// Orchestration summarizes run-multi → compile-multi inputs.
	Orchestration map[string]any `json:"orchestration"`
	// Merge is set when merge=true; structured merge observability.
	Merge *merge.MergeDebug `json:"merge,omitempty"`
}

// RunMultiResponse is the response for POST /v1/recall/run-multi (Pluribus Phase A).
type RunMultiResponse struct {
	Selected       map[string]any     `json:"selected,omitempty"`
	Scores         map[string]float64 `json:"scores,omitempty"`
	Merged         map[string]any     `json:"merged,omitempty"`
	Promoted       bool               `json:"promoted"`
	Blocked        bool               `json:"blocked,omitempty"` // true when behavior validation failed (constraint/failure/decision)
	MemoriesUsed   []string           `json:"memories_used,omitempty"`
	Excluded       []string           `json:"excluded,omitempty"`
	Contradictions []string           `json:"contradictions,omitempty"`
	Confidence     float64            `json:"confidence"`
	// Debug is always present (use newRunMultiDebug); do not omitempty.
	Debug RunMultiDebug `json:"debug"`
}

// Validate validates contract-level requirements for RunMultiRequest.
func (r RunMultiRequest) Validate() error {
	if r.Query == "" {
		return fmt.Errorf("query is required")
	}
	if r.Promote && !r.Merge {
		return fmt.Errorf("promote requires merge=true")
	}
	return nil
}
