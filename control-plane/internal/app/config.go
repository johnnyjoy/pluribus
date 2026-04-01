package app

import (
	"fmt"
	"os"

	"control-plane/internal/memory"
	"control-plane/internal/recall"
	"control-plane/internal/synthesis"

	"gopkg.in/yaml.v3"
)

// Config holds control-plane configuration (from config.example.yaml).
type Config struct {
	Server    ServerConfig     `yaml:"server"`
	Synthesis synthesis.Config `yaml:"synthesis"`
	Startup   StartupConfig    `yaml:"startup"`
	Postgres  PostgresConfig   `yaml:"postgres"`
	Redis     RedisConfig      `yaml:"redis"`
	Evidence  EvidenceConfig   `yaml:"evidence"`
	Recall    RecallConfig     `yaml:"recall"`
	Curation  CurationConfig   `yaml:"curation"`
	Memory    MemoryConfig     `yaml:"memory"`
	Drift     DriftConfig      `yaml:"drift"`
	LSP       *LSPConfig       `yaml:"lsp,omitempty"` // Task 101: gate and tune LSP-based recall/drift
	SlowPath  SlowPathConfig   `yaml:"slow_path"`
	// Ingest gates optional MCL → memory promotion bridge (M7). Default: auto_promote false.
	Ingest IngestConfig `yaml:"ingest"`
	// optional promotion policy for run-multi (evidence gates in later slices).
	Promotion PromotionConfig `yaml:"promotion"`
	// Similarity: optional subordinate advisory-episode retrieval (not canonical memory).
	Similarity SimilarityConfig `yaml:"similarity,omitempty"`
	// Distillation: optional advisory episode → candidate_events (not canonical memory until curated).
	Distillation DistillationConfig `yaml:"distillation,omitempty"`
	// Enforcement: pre-change gate — trusted binding memory vs proposals (RC1: on when omitted; set enabled: false to disable).
	Enforcement EnforcementConfig `yaml:"enforcement,omitempty"`
	// MCP: optional JSON-RPC over HTTP at POST /v1/mcp. Omitted or disabled=false → endpoint on (default).
	MCP *MCPConfig `yaml:"mcp,omitempty"`
	// Lexical: optional pg_textsearch BM25 path (experimental). Canonical memory unchanged; see docs/experiments/pg-textsearch-evaluation.md.
	Lexical *LexicalConfig `yaml:"lexical,omitempty"`
}

// LexicalConfig gates experimental BM25 retrieval against a projection table (not canonical memories).
type LexicalConfig struct {
	// ExperimentalHTTP when true registers POST /v1/experimental/lexical/search (default false).
	ExperimentalHTTP bool `yaml:"experimental_http"`
	// ProjectionTable is the BM25-indexed table (e.g. lexical_memory_projection). Lowercase identifier only.
	ProjectionTable string `yaml:"projection_table"`
}

// MCPConfig toggles MCP JSON-RPC over HTTP (POST /v1/mcp). See docs/mcp-service-first.md.
type MCPConfig struct {
	// Disabled skips the MCP wrapper (no MCP endpoint; POST /v1/mcp is not handled by MCP).
	Disabled bool `yaml:"disabled"`
	// MemoryFormation gates MCP tools that create advisory episodes / surface candidates (optional).
	MemoryFormation *MCPMemoryFormationConfig `yaml:"memory_formation,omitempty"`
}

// MCPMemoryFormationConfig gates mcp_episode_ingest and related tools (low-noise; omit section for server defaults).
type MCPMemoryFormationConfig struct {
	// EpisodeIngestEnabled when false, mcp_episode_ingest returns a clear error. Omitted = true (default).
	EpisodeIngestEnabled *bool `yaml:"episode_ingest_enabled,omitempty"`
	// MinSummaryChars minimum summary length for mcp_episode_ingest (0 = use server default 12).
	MinSummaryChars int `yaml:"min_summary_chars"`
	// RequireSignalKeyword when false, skip keyword gate. Omitted = true.
	RequireSignalKeyword *bool `yaml:"require_signal_keyword,omitempty"`
	// DedupEnabled when false, skip time-window dedup for source=mcp advisory episodes. Omitted = true.
	DedupEnabled *bool `yaml:"dedup_enabled,omitempty"`
	// DedupWindowSeconds minimum gap before a duplicate MCP episode (same summary + session) inserts again; 0 = server default 120.
	DedupWindowSeconds int `yaml:"dedup_window_seconds"`
}

// MCPEnabled returns whether POST /v1/mcp should be served. Default true when mcp section omitted.
func (c *Config) MCPEnabled() bool {
	if c == nil || c.MCP == nil {
		return true
	}
	return !c.MCP.Disabled
}

// EnforcementConfig controls POST /v1/enforcement/evaluate (trusted memory as active protection).
// Enabled is nil = RC1 default on; use explicit false to disable.
type EnforcementConfig struct {
	Enabled *bool `yaml:"enabled,omitempty"`
	// MinBindingAuthority is the minimum memory authority (0–10) for a row to participate as binding. Default 7.
	MinBindingAuthority int `yaml:"min_binding_authority"`
	// MaxBindingMemories caps rows loaded for evaluation. Default 120.
	MaxBindingMemories int `yaml:"max_binding_memories"`
	// FailureOverlapThreshold is min word-overlap ratio (statement vs proposal) to flag a failure as review. Default 0.25.
	FailureOverlapThreshold float64 `yaml:"failure_overlap_threshold"`
	// PatternBlockScore is the drift-style severity score above which a negative pattern maps to block_overrideable. Default 4 (high).
	PatternBlockScore float64 `yaml:"pattern_block_score"`
	// MaxEvidencePerMemory caps evidence records attached per triggered memory in the response. Default 3.
	MaxEvidencePerMemory int `yaml:"max_evidence_per_memory"`
}

// IsEnabled returns whether enforcement is active. Nil Enabled means true (RC1 default).
func (e *EnforcementConfig) IsEnabled() bool {
	if e == nil {
		return false
	}
	if e.Enabled == nil {
		return true
	}
	return *e.Enabled
}

// DistillationConfig gates POST /v1/episodes/distill (deterministic keyword → candidate rows).
type DistillationConfig struct {
	Enabled bool `yaml:"enabled"`
	// AutoFromAdvisoryEpisodes runs the same distillation as POST /v1/episodes/distill after each successful POST /v1/advisory-episodes (candidate-only; never canonical). Default false.
	AutoFromAdvisoryEpisodes bool `yaml:"auto_from_advisory_episodes"`
	MinStatementChars        int  `yaml:"min_statement_chars"` // 0 = default 20 in distillation service
}

// SimilarityConfig controls subordinate advisory episodes (lexical + tag resemblance, in-process).
// Enabled is nil = default on (POST /v1/advisory-episodes and MCP record_experience need this); use explicit false to disable.
type SimilarityConfig struct {
	Enabled *bool `yaml:"enabled,omitempty"`
	MaxSummaryBytes int  `yaml:"max_summary_bytes"`
	MaxEpisodesScan int  `yaml:"max_episodes_scan"`
	MaxResults      int  `yaml:"max_results"`
	// MinResemblance is the minimum combined score [0,1] to return a case (lexical + optional tags).
	MinResemblance float64 `yaml:"min_resemblance"`
	// PruneRejectedOlderThanHours when > 0, once at server startup delete rejected advisory_experiences older than this many hours (0 = off).
	PruneRejectedOlderThanHours int `yaml:"prune_rejected_older_than_hours,omitempty"`
}

// IsEnabled returns whether advisory similarity ingest and retrieval are active. Nil Enabled means true.
func (c *SimilarityConfig) IsEnabled() bool {
	if c == nil {
		return false
	}
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// BoolPtr returns a pointer to v (YAML *bool fields in tests and overrides).
func BoolPtr(v bool) *bool { return &v }

// StartupConfig controls service startup behavior before HTTP server begins serving.
type StartupConfig struct {
	// DBWaitTimeoutSeconds total time to wait for Postgres before failing startup.
	DBWaitTimeoutSeconds int `yaml:"db_wait_timeout_seconds"`
	// DBWaitIntervalMillis interval between DB ping attempts during startup wait.
	DBWaitIntervalMillis int `yaml:"db_wait_interval_millis"`
}

// PromotionConfig gates run-multi memory promotion.
type PromotionConfig struct {
	// RequireEvidence when true will require evidence links (Phase 9.2+). Default false.
	RequireEvidence bool `yaml:"require_evidence"`
	// MinEvidenceLinks minimum links after promote (Phase 9.2+). Default 0.
	MinEvidenceLinks int `yaml:"min_evidence_links"`
	// MinEvidenceScore minimum aggregate evidence score (Phase 9.2+). Default 0 = off.
	MinEvidenceScore float64 `yaml:"min_evidence_score"`
	// RequireReview when true creates promoted rows pending review (Phase 9.4+). Default false.
	RequireReview bool `yaml:"require_review"`
	// MinPromoteConfidence in [0,1]. When > 0, run-multi confidence must be >= this to promote. Default 0 = off.
	MinPromoteConfidence float64 `yaml:"min_promote_confidence"`
	// MinPolicyComposite in [0,1]. When > 0, weighted blend of confidence + signal + evidence must pass (Phase 9.3). Default 0 = off.
	MinPolicyComposite float64 `yaml:"min_policy_composite"`
	// WeightConfidence / WeightSignal / WeightEvidence — for composite; all zero => 0.4 / 0.3 / 0.3.
	WeightConfidence float64 `yaml:"weight_confidence"`
	WeightSignal     float64 `yaml:"weight_signal"`
	WeightEvidence   float64 `yaml:"weight_evidence"`
	// SignalNormDivisor scales merge total_signal into [0,1] (divide then cap). Default 15 when 0.
	SignalNormDivisor float64 `yaml:"signal_norm_divisor"`
	// AutoPromote when true allows POST /v1/curation/auto-promote to materialize eligible pending candidates (default false).
	AutoPromote bool `yaml:"auto_promote"`
	// AutoMinSupportCount minimum distill_support_count (or 1 when unset in proposal) for auto-promote. Default 4 when 0.
	AutoMinSupportCount int `yaml:"min_support_count"`
	// AutoMinSalience minimum candidate salience_score for auto-promote. Default 0.7 when 0.
	AutoMinSalience float64 `yaml:"min_salience"`
	// AutoAllowedKinds memory kinds allowed for auto-promote (e.g. failure, pattern). Empty = use server default list.
	AutoAllowedKinds []string `yaml:"allowed_kinds"`
	// CanonicalConsolidation merges similar digest promotions into existing canonical rows (deterministic; default off).
	CanonicalConsolidation *memory.CanonicalConsolidationConfig `yaml:"canonical_consolidation,omitempty"`
}

// IngestConfig controls the cognitive ingest gateway (MCL) and optional promotion bridge.
type IngestConfig struct {
	// AutoPromote enables server-internal calls to memory.Promote when explicitly requested
	// (client propose_promotion on ingest, or POST /v1/ingest/:id/commit). Default false.
	AutoPromote bool `yaml:"auto_promote"`
}

// LSPConfig gates and tunes LSP use for recall symbol boost and drift reference risk (Task 101).
// When Enabled is false, LSP is not used (no drift reference checks, no symbol-overlap boost in recall).
type LSPConfig struct {
	Enabled                    bool    `yaml:"enabled"`                       // when false, skip all LSP-based behavior
	RecallSymbolBoost          float64 `yaml:"recall_symbol_boost"`           // weight for symbol overlap in recall (0 = use recall.ranking.weight_symbol_overlap)
	ReferenceExpansionLimit    int     `yaml:"reference_expansion_limit"`     // cap refs considered per symbol (0 = no cap)
	HighRiskReferenceThreshold int     `yaml:"high_risk_reference_threshold"` // drift: ref count above this => high risk / block (0 = off)
	// AutoSymbolMax caps names taken from LSP documentSymbol when recall auto-fills symbols. 0 = default 64.
	AutoSymbolMax int `yaml:"auto_symbol_max"`
}

// SlowPathConfig controls the slow-path policy when drift risk is high (Task 92).
// When enabled and risk >= HighRiskThreshold, recall expands limits and optionally requires a second drift check.
type SlowPathConfig struct {
	Enabled                 bool    `yaml:"enabled"`
	HighRiskThreshold       float64 `yaml:"high_risk_threshold"`        // risk score >= this triggers slow path; default 1.0 (high)
	ExpandConstraintsBy     int     `yaml:"expand_constraints_by"`      // add N constraints in slow path; default 4
	ExpandFailuresBy        int     `yaml:"expand_failures_by"`         // add N failures; default 4
	ExpandPatternsBy        int     `yaml:"expand_patterns_by"`         // add N patterns; default 2
	RequireSecondDriftCheck bool    `yaml:"require_second_drift_check"` // require second drift check before execution; default true
	// ExtraVariantsWhenSlow adds N to compile-multi variant count when slow-path is active (max 3 for default strategies). Default 0.
	ExtraVariantsWhenSlow int `yaml:"extra_variants_when_slow"`
}

// DriftConfig holds drift check options (Task 76: fuzzy failure; Task 100: LSP reference risk).
type DriftConfig struct {
	FailureFuzzyThreshold         float64 `yaml:"failure_fuzzy_threshold"`           // 0 = off; 0.8 = match when word overlap >= 80%
	LSPHighRiskReferenceThreshold int     `yaml:"lsp_high_risk_reference_threshold"` // Task 100: 0 = off; if symbol ref count > this, escalate to high risk / block
	PatternHighBlocks             bool    `yaml:"pattern_high_blocks"`               // high/catastrophic negative pattern match blocks execution
}

// MemoryConfig holds memory-related config (e.g. lifecycle authority adjustment).
type MemoryConfig struct {
	Lifecycle *MemoryLifecycleConfig `yaml:"lifecycle,omitempty"`
	// Dedup: exact canonical duplicate rejection on create (Phase C). Omitted → enabled (on).
	Dedup *MemoryDedupConfig `yaml:"dedup,omitempty"`
	// PatternGeneralization: near-duplicate pattern merge on create (optional; default off).
	PatternGeneralization *MemoryPatternGeneralizationConfig `yaml:"pattern_generalization,omitempty"`
	// PatternElevation: cluster eligible patterns into one dominant row (optional; default off).
	PatternElevation *memory.PatternElevationConfig `yaml:"pattern_elevation,omitempty"`
	// RecallReinforcement caps authority bumps from recall/success hooks (optional).
	RecallReinforcement *memory.RecallReinforcementConfig `yaml:"recall_reinforcement,omitempty"`
}

// MemoryDedupConfig toggles statement_key duplicate detection before insert.
type MemoryDedupConfig struct {
	Enabled *bool `yaml:"enabled,omitempty"`
	// NearDupJaccardThreshold: Jaccard on statement_canonical tokens for recall-only collapse (Phase F).
	// 0 = do not collapse near-dups (exact statement_key collapse still applies when ranking is on).
	// Typical: 0.92 (creative C2).
	NearDupJaccardThreshold float64 `yaml:"near_dup_jaccard_threshold,omitempty"`
}

// MemoryLifecycleConfig holds deltas for authority adjustment (Task 74) and expiration (Task 75).
type MemoryLifecycleConfig struct {
	AuthorityPositiveDelta       float64 `yaml:"authority_positive_delta"`       // validation: default 0.1
	AuthorityNegativeDelta       float64 `yaml:"authority_negative_delta"`       // contradiction/failure: default 0.2
	ExpirationAuthorityThreshold int     `yaml:"expiration_authority_threshold"` // max authority (0-10) to consider for TTL expiry; 2 = 0.2 logical
}

// MemoryPatternGeneralizationConfig gates near-duplicate pattern reinforcement on memory create.
type MemoryPatternGeneralizationConfig struct {
	Enabled                    bool    `yaml:"enabled"`
	MergeJaccardMin            float64 `yaml:"merge_jaccard_min"`
	MinTagOverlapFraction      float64 `yaml:"min_tag_overlap_fraction"`
	MaxCandidatesScan          int     `yaml:"max_candidates_scan"`
	MaxSupportingStatementKeys int     `yaml:"max_supporting_statement_keys"`
	NegationGuard              bool    `yaml:"negation_guard"`
}

type ServerConfig struct {
	Bind string `yaml:"bind"`
}

type PostgresConfig struct {
	DSN string `yaml:"dsn"`
}

type EvidenceConfig struct {
	RootPath        string  `yaml:"root_path"`
	AuthorityFactor float64 `yaml:"authority_factor"` // evidence_score * this added to memory authority on link; 0 = off; default 0.1 (Task 79)
}

type RecallConfig struct {
	DefaultMaxItemsPerKind int                  `yaml:"default_max_items_per_kind"`
	DefaultMaxTotal        int                  `yaml:"default_max_total"`  // 0 = no cap beyond per-kind
	DefaultMaxTokens       int                  `yaml:"default_max_tokens"` // 0 = no token cap
	CacheTTLSeconds        int                  `yaml:"cache_ttl_seconds"`  // 0 = no cache or use Redis default
	Ranking                *RecallRankingConfig `yaml:"ranking,omitempty"`
	// RIU (Recall Intelligence Upgrade): structured scoring + contradiction policy. Default off.
	RIU *RecallRIUConfig `yaml:"riu,omitempty"`
	// load promoted experiences from JSONL into recall (default off).
	ExperiencesEnabled        bool   `yaml:"experiences_enabled"`
	ExperiencesPath           string `yaml:"experiences_path"`
	ExperiencesLimit          int    `yaml:"experiences_limit"`
	ExperiencesAuthorityBoost int    `yaml:"experiences_authority_boost"`
	// EvidenceInBundle attaches compact supporting_evidence to recalled MemoryItem (default off).
	EvidenceInBundle *recall.EvidenceInBundleConfig `yaml:"evidence_in_bundle,omitempty"`
	// TriggeredRecall enables heuristic triggered recall (opt-in per request via enable_triggered_recall).
	TriggeredRecall *recall.TriggerRecallConfig `yaml:"triggered_recall,omitempty"`
	// SemanticRetrieval pgvector + embedding API for semantic candidates (default on after LoadConfig; set enabled: false to disable).
	SemanticRetrieval *memory.SemanticRetrievalConfig `yaml:"semantic_retrieval,omitempty"`
	// BehaviorValidation configures run-multi overlap validation (constraint / failure / decision). Omit for defaults.
	BehaviorValidation *recall.BehaviorValidationConfig `yaml:"behavior_validation,omitempty"`
	// LogRankTopN logs top N globally ranked memories after compile ranking (0 = off). Tuning/debug only.
	LogRankTopN int `yaml:"log_rank_top_n"`
}

// RecallRankingConfig holds configurable weights for recall weighted ranking (Phase 1: config-only).
type RecallRankingConfig struct {
	WeightAuthority         float64 `yaml:"weight_authority"`          // default 1.0
	WeightRecency           float64 `yaml:"weight_recency"`            // default 0.5
	WeightScopeMatch        float64 `yaml:"weight_scope_match"`        // default 1.0
	WeightTagMatch          float64 `yaml:"weight_tag_match"`          // default 1.0
	WeightFailureOverlap    float64 `yaml:"weight_failure_overlap"`    // default 0.5; extra for failure-kind when tags overlap
	WeightSymbolOverlap     float64 `yaml:"weight_symbol_overlap"`     // Task 100: default 0.5; boost when task/memory symbols overlap
	WeightPatternPriority   float64 `yaml:"weight_pattern_priority"`   // explicit pattern additive weight; default 0 (off)
	WeightLexicalSimilarity float64 `yaml:"weight_lexical_similarity"` // additive lexical overlap weight for situation matching; default 0.15
	// WeightPatternGeneralization boosts patterns that carry generalization metadata (merged / reinforced). Default 0 (off).
	WeightPatternGeneralization float64 `yaml:"weight_pattern_generalization"`
	// WeightFailureSeverity boosts failure rows via keyword heuristic on statement (default 0 = off).
	WeightFailureSeverity float64 `yaml:"weight_failure_severity"`
	// WeightCrossContextSalience boosts rows with payload.salience.distinct_contexts (default 0 = off).
	WeightCrossContextSalience float64 `yaml:"weight_cross_context_salience"`
	// WeightCrossContextSalienceK is k in log1p(distinct)/k (0 = server default 3).
	WeightCrossContextSalienceK float64 `yaml:"weight_cross_context_salience_k"`
	// WeightCrossAgentSalience boosts rows with payload.salience.distinct_agents (default 0 = off).
	WeightCrossAgentSalience float64 `yaml:"weight_cross_agent_salience"`
	// WeightCrossAgentSalienceK is k for distinct_agents (0 = server default 3).
	WeightCrossAgentSalienceK float64 `yaml:"weight_cross_agent_salience_k"`
	// WeightSemanticSimilarity scales pgvector cosine similarity for hybrid ranking.
	// Omitted (nil) defaults to server default (0.4). Explicit 0 disables the term and logs a warning.
	WeightSemanticSimilarity *float64 `yaml:"weight_semantic_similarity,omitempty"`
	// WeightElevationSuppression penalizes superseded patterns when the elevated row is in the same candidate set (default 0 = off).
	WeightElevationSuppression float64 `yaml:"weight_elevation_suppression"`
}

// RecallRIUConfig configures Recall Intelligence Upgrade (RIU). Omit or enabled=false for ranking-only recall.
type RecallRIUConfig struct {
	Enabled                    bool    `yaml:"enabled"`
	ContradictionPolicy        string  `yaml:"contradiction_policy"` // exclude | warn | bounded_pair
	BoundedPairMax             int     `yaml:"bounded_pair_max"`     // max pairs in bundle; 0 = server default
	WeightApplicability        float64 `yaml:"weight_applicability"`
	WeightTransferable         float64 `yaml:"weight_transferable"`
	WeightLineageProxy         float64 `yaml:"weight_lineage_proxy"`
	WeightContradictionPenalty float64 `yaml:"weight_contradiction_penalty"`
}

// RedisConfig is optional. When disabled, no cache is used (recall/preflight always compute).
type RedisConfig struct {
	Enabled       bool   `yaml:"enabled"`
	Addr          string `yaml:"addr"`
	Password      string `yaml:"password"`
	DB            int    `yaml:"db"`
	DefaultTTLSec int    `yaml:"default_ttl_seconds"`
}

type CurationConfig struct {
	CandidateThreshold float64 `yaml:"candidate_threshold"`
	ReviewThreshold    float64 `yaml:"review_threshold"`
	PromoteThreshold   float64 `yaml:"promote_threshold"`
	// Digest (POST /v1/curation/digest) bounds; 0 = use server defaults.
	DigestMaxProposals        int `yaml:"digest_max_proposals"`
	DigestWorkSummaryMaxBytes int `yaml:"digest_work_summary_max_bytes"`
	DigestStatementMaxBytes   int `yaml:"digest_statement_max_bytes"`
	DigestReasonMaxBytes      int `yaml:"digest_reason_max_bytes"`
}

// SlowPathEnabled returns true when slow-path policy is enabled (recall/drift can use this to gate behavior).
func (c *Config) SlowPathEnabled() bool {
	return c != nil && c.SlowPath.Enabled
}

// applySlowPathDefaults sets default values for SlowPath when not specified in YAML.
func applySlowPathDefaults(sp *SlowPathConfig) {
	if sp.HighRiskThreshold <= 0 {
		sp.HighRiskThreshold = 1.0
	}
	if sp.ExpandConstraintsBy <= 0 {
		sp.ExpandConstraintsBy = 4
	}
	if sp.ExpandFailuresBy <= 0 {
		sp.ExpandFailuresBy = 4
	}
	if sp.ExpandPatternsBy <= 0 {
		sp.ExpandPatternsBy = 2
	}
}

// LoadConfig reads config from path (e.g. configs/config.local.yaml for dev, CONFIG env). Uses config.example.yaml if path is missing.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		path = "configs/config.example.yaml"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid YAML in config file %s: %w", path, err)
	}
	applyStartupDefaults(&cfg.Startup)
	synthesis.ApplyDefaults(&cfg.Synthesis)
	applySlowPathDefaults(&cfg.SlowPath)
	if err := cfg.Synthesis.Validate(); err != nil {
		return nil, err
	}
	applyRecallExperiencesDefaults(&cfg.Recall)
	applyRecallRIUDefaults(cfg.Recall.RIU)
	applyTriggeredRecallDefaults(&cfg.Recall)
	recall.ApplyEvidenceInBundleDefaults(cfg.Recall.EvidenceInBundle)
	applyBehaviorValidationDefaults(cfg.Recall.BehaviorValidation)
	applyRecallSemanticAndRankingDefaults(&cfg.Recall)
	applyLSPDefaults(cfg.LSP)
	applyPromotionDefaults(&cfg.Promotion)
	applyCurationDefaults(&cfg.Curation)
	applySimilarityDefaults(&cfg.Similarity)
	applyEnforcementDefaults(&cfg.Enforcement)
	return &cfg, nil
}

func applyEnforcementDefaults(e *EnforcementConfig) {
	if e == nil {
		return
	}
	if e.MinBindingAuthority <= 0 {
		e.MinBindingAuthority = 7
	}
	if e.MaxBindingMemories <= 0 {
		e.MaxBindingMemories = 120
	}
	if e.FailureOverlapThreshold <= 0 {
		e.FailureOverlapThreshold = 0.25
	}
	if e.PatternBlockScore <= 0 {
		e.PatternBlockScore = 4.0
	}
	if e.MaxEvidencePerMemory <= 0 {
		e.MaxEvidencePerMemory = 3
	}
}

func applySimilarityDefaults(c *SimilarityConfig) {
	if c == nil {
		return
	}
	if c.MaxSummaryBytes <= 0 {
		c.MaxSummaryBytes = 2048
	}
	if c.MaxEpisodesScan <= 0 {
		c.MaxEpisodesScan = 500
	}
	if c.MaxResults <= 0 {
		c.MaxResults = 5
	}
	if c.MinResemblance <= 0 {
		c.MinResemblance = 0.08
	}
}

func applyCurationDefaults(c *CurationConfig) {
	if c == nil {
		return
	}
	if c.DigestMaxProposals <= 0 {
		c.DigestMaxProposals = 5
	}
	if c.DigestWorkSummaryMaxBytes <= 0 {
		c.DigestWorkSummaryMaxBytes = 8192
	}
	if c.DigestStatementMaxBytes <= 0 {
		c.DigestStatementMaxBytes = 2048
	}
	if c.DigestReasonMaxBytes <= 0 {
		c.DigestReasonMaxBytes = 1024
	}
}

func applyStartupDefaults(s *StartupConfig) {
	if s == nil {
		return
	}
	if s.DBWaitTimeoutSeconds <= 0 {
		s.DBWaitTimeoutSeconds = 60
	}
	if s.DBWaitIntervalMillis <= 0 {
		s.DBWaitIntervalMillis = 1000
	}
}

func applyPromotionDefaults(p *PromotionConfig) {
	if p == nil {
		return
	}
	if p.MinEvidenceLinks < 0 {
		p.MinEvidenceLinks = 0
	}
	if p.MinEvidenceScore < 0 {
		p.MinEvidenceScore = 0
	}
	if p.MinPromoteConfidence < 0 {
		p.MinPromoteConfidence = 0
	}
	if p.MinPromoteConfidence > 1 {
		p.MinPromoteConfidence = 1
	}
	if p.MinPolicyComposite < 0 {
		p.MinPolicyComposite = 0
	}
	if p.MinPolicyComposite > 1 {
		p.MinPolicyComposite = 1
	}
	if p.SignalNormDivisor < 0 {
		p.SignalNormDivisor = 0
	}
	if p.AutoMinSupportCount < 0 {
		p.AutoMinSupportCount = 0
	}
	if p.AutoMinSalience < 0 {
		p.AutoMinSalience = 0
	}
	if p.AutoMinSalience > 1 {
		p.AutoMinSalience = 1
	}
	// Conservative defaults for optional auto-promote (used only when auto_promote is true).
	if p.AutoMinSupportCount <= 0 {
		p.AutoMinSupportCount = 4
	}
	if p.AutoMinSalience <= 0 {
		p.AutoMinSalience = 0.7
	}
}

func applyLSPDefaults(l *LSPConfig) {
	if l == nil {
		return
	}
	if l.AutoSymbolMax <= 0 {
		l.AutoSymbolMax = 64
	}
}

func applyRecallSemanticAndRankingDefaults(r *RecallConfig) {
	if r == nil {
		return
	}
	if r.SemanticRetrieval == nil {
		r.SemanticRetrieval = &memory.SemanticRetrievalConfig{}
	}
	sr := r.SemanticRetrieval
	if sr.MaxSemanticCandidates <= 0 {
		sr.MaxSemanticCandidates = 50
	}
	if sr.MinCosineSimilarity <= 0 {
		sr.MinCosineSimilarity = 0.35
	}
	if sr.EmbeddingDimensions <= 0 {
		sr.EmbeddingDimensions = memory.DefaultEmbeddingDimensions
	}
	if r.Ranking == nil {
		r.Ranking = &RecallRankingConfig{}
	}
}

func applyRecallExperiencesDefaults(r *RecallConfig) {
	if r == nil {
		return
	}
	if r.ExperiencesLimit <= 0 {
		r.ExperiencesLimit = 50
	}
	if r.ExperiencesPath == "" {
		r.ExperiencesPath = "data/memory/experiences.jsonl"
	}
}

func applyRecallRIUDefaults(riu *RecallRIUConfig) {
	if riu == nil {
		return
	}
	if riu.BoundedPairMax < 0 {
		riu.BoundedPairMax = 0
	}
}

func applyTriggeredRecallDefaults(r *RecallConfig) {
	if r == nil || r.TriggeredRecall == nil {
		return
	}
	r.TriggeredRecall = recall.NormalizeTriggerRecall(r.TriggeredRecall)
}

func applyBehaviorValidationDefaults(c *recall.BehaviorValidationConfig) {
	if c == nil {
		return
	}
	n := recall.NormalizeBehaviorValidationConfig(c)
	*c = n
}
