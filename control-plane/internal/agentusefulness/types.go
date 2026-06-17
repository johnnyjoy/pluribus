package agentusefulness

import (
	"time"

	"control-plane/internal/memory"
	"control-plane/internal/recall"
	"control-plane/internal/utility"
	"control-plane/pkg/api"
)

// FixtureMemory is one labeled memory in the usefulness corpus.
type FixtureMemory struct {
	Label              string        `json:"label"`
	Kind               string        `json:"kind"`
	Statement          string        `json:"statement"`
	Authority          int           `json:"authority"`
	Tags               []string      `json:"tags"`
	Status             string        `json:"status,omitempty"`
	Applicability      string        `json:"applicability,omitempty"`
	OccurredAt         string        `json:"occurred_at,omitempty"`
	CreatedAt          string        `json:"created_at,omitempty"`
	UtilityScore       float64       `json:"utility_score,omitempty"`
	RefutedCount       int           `json:"refuted_count,omitempty"`
	WrongCount         int           `json:"wrong_count,omitempty"`
	OutdatedCount      int           `json:"outdated_count,omitempty"`
	Supersedes         string        `json:"supersedes_label,omitempty"`
	MemorySchemaType   string        `json:"memory_schema_type,omitempty"`
	EncodingCues       *EncodingCues `json:"encoding_cues,omitempty"`
	RetrievalTerms     []string      `json:"retrieval_terms,omitempty"`
	Scope              string        `json:"scope,omitempty"`
	NegativeScope      []string      `json:"negative_scope,omitempty"`
	InterferenceGroup  string        `json:"interference_group,omitempty"`
}

// TaskFixture is one deterministic agent usefulness scenario.
type TaskFixture struct {
	ID                      string            `json:"id"`
	Title                   string            `json:"title"`
	Scenario                string            `json:"scenario"`
	TaskPrompt              string            `json:"task_prompt"`
	DomainTags              []string          `json:"domain_tags"`
	RecallQuery             string            `json:"recall_query"`
	RecallMode              string            `json:"recall_mode,omitempty"`
	IncludeStatus           []string          `json:"include_status,omitempty"`
	OccurredAfter           string            `json:"occurred_after,omitempty"`
	OccurredBefore          string            `json:"occurred_before,omitempty"`
	MemoryLabels            []string          `json:"memory_labels"`
	DecoyLabels             []string          `json:"decoy_labels,omitempty"`
	ExpectedRecalledLabels  []string          `json:"expected_recalled_labels"`
	ForbiddenRecalledLabels []string          `json:"forbidden_recalled_labels,omitempty"`
	ExpectedUsedLabels      []string          `json:"expected_used_labels"`
	ExpectedIgnoredLabels   []string          `json:"expected_ignored_labels,omitempty"`
	FactContributions       map[string][]string `json:"fact_contributions"`
	ExpectedAnswerFacts     []string          `json:"expected_answer_facts"`
	ForbiddenAnswerFacts    []string          `json:"forbidden_answer_facts,omitempty"`
	NoMemoryAnswerFacts     []string          `json:"no_memory_answer_facts,omitempty"`
	RequiresMemoryHelp        bool              `json:"requires_memory_help"`
	CheckMCPRESTParity        bool              `json:"check_mcp_rest_parity,omitempty"`
	ScoringRubric             string            `json:"scoring_rubric,omitempty"`
	ResearchPrinciple         string            `json:"research_principle,omitempty"`
	NearMissMemoryLabels      []string          `json:"near_miss_memory_labels,omitempty"`
	ExpectedSuppressedLabels  []string          `json:"expected_suppressed_memory_ids,omitempty"`
	ExpectedUseReason         string            `json:"expected_use_reason,omitempty"`
	ExpectedIgnoreReason      string            `json:"expected_ignore_reason,omitempty"`
	MinEncodingCueMatch       float64           `json:"min_encoding_cue_match,omitempty"`
}

// LoadedCorpus maps labels to runtime memory objects.
type LoadedCorpus struct {
	LabelToID      map[string]string
	IDToLabel      map[string]string
	Objects        []memory.MemoryObject
	Utility        map[string]utility.Score
	Tasks          []TaskFixture
	MemoryFixtures []FixtureMemory
}

// InterfaceREST or InterfaceMCP selects recall path.
const (
	InterfaceREST = "rest"
	InterfaceMCP  = "mcp"
)

// RunModeNoMemory or RunModeMemory.
const (
	RunModeNoMemory = "no_memory"
	RunModeMemory   = "memory"
)

// RecalledMemoryEntry is one row in a recall manifest.
type RecalledMemoryEntry struct {
	MemoryID      string  `json:"memory_id"`
	Label         string  `json:"label"`
	Rank          int     `json:"rank"`
	Score         float64 `json:"score,omitempty"`
	LifecycleRole string  `json:"lifecycle_role,omitempty"`
	Status        string  `json:"status,omitempty"`
	UtilityScore  float64 `json:"utility_score,omitempty"`
	Why           string  `json:"why,omitempty"`
}

// RecallManifest captures what Pluribus returned for a run.
type RecallManifest struct {
	TaskID        string                `json:"task_id"`
	RunID         string                `json:"run_id"`
	Interface     string                `json:"interface"`
	Mode          string                `json:"mode"`
	RecallRequest recall.CompileRequest `json:"recall_request"`
	Recalled      []RecalledMemoryEntry `json:"recalled_memories"`
}

// MemoryUseTrace records explicit simulator use decisions.
type MemoryUseTrace struct {
	UsedMemoryIDs    []string          `json:"used_memory_ids"`
	UsedLabels       []string          `json:"used_labels"`
	IgnoredMemoryIDs []string          `json:"ignored_memory_ids"`
	IgnoredLabels    []string          `json:"ignored_labels"`
	MisusedMemoryIDs []string          `json:"misused_memory_ids"`
	MisusedLabels    []string          `json:"misused_labels"`
	UseReasons       map[string]string `json:"use_reasons"`
	IgnoreReasons    map[string]string `json:"ignore_reasons"`
}

// OutcomeFeedbackEvent links recall, use, and scored outcome.
type OutcomeFeedbackEvent struct {
	TaskID         string   `json:"task_id"`
	RunID          string   `json:"run_id"`
	Interface      string   `json:"interface"`
	Mode           string   `json:"mode"`
	MemoryLabel    string   `json:"memory_label,omitempty"`
	MemoryID       string   `json:"memory_id,omitempty"`
	Used           bool     `json:"used"`
	AnswerFacts    []string `json:"answer_facts"`
	ScorePass      bool     `json:"score_pass"`
	OutcomeLabel   string   `json:"outcome_label"`
	Reason         string   `json:"reason,omitempty"`
}

// CaseScore is deterministic rubric output for one run variant.
type CaseScore struct {
	AnswerPass                 bool `json:"answer_pass"`
	ExpectedFactsPresent       bool `json:"expected_facts_present"`
	ForbiddenFactsAbsent       bool `json:"forbidden_facts_absent"`
	ExpectedMemoriesRecalled   bool `json:"expected_memories_recalled"`
	ExpectedMemoriesUsed       bool `json:"expected_memories_used"`
	ForbiddenMemoriesRecalled  bool `json:"forbidden_memories_not_recalled"`
	ForbiddenMemoriesUsed      bool `json:"forbidden_memories_not_used"`
	LifecycleRolesCorrect      bool `json:"lifecycle_roles_correct"`
	DateBoundsRespected        bool `json:"date_bounds_respected"`
	UtilityConstraintsRespected bool `json:"utility_constraints_respected"`
	MemoryHelped               bool `json:"memory_helped"`
	MemoryHurt                 bool `json:"memory_hurt"`
	MemoryIrrelevant           bool `json:"memory_irrelevant"`
	MemoryMisused              bool `json:"memory_misused"`
}

// RunResult is one harness execution (no-memory, REST memory, MCP memory).
type RunResult struct {
	TaskID          string                 `json:"task_id"`
	RunID           string                 `json:"run_id"`
	Interface       string                 `json:"interface"`
	Mode            string                 `json:"mode"`
	AnswerFacts     []string               `json:"answer_facts"`
	Manifest        *RecallManifest        `json:"manifest,omitempty"`
	UseTrace        MemoryUseTrace         `json:"memory_use_trace"`
	OutcomeFeedback []OutcomeFeedbackEvent `json:"outcome_feedback"`
	Score           CaseScore              `json:"score"`
	Failures        []string               `json:"failures,omitempty"`
}

// TaskResult aggregates all variants for one fixture task.
type TaskResult struct {
	TaskID            string    `json:"task_id"`
	RequiresMemoryHelp bool     `json:"requires_memory_help"`
	ParityRequired    bool      `json:"parity_required"`
	NoMemory          RunResult `json:"no_memory"`
	MemoryREST        RunResult `json:"memory_rest"`
	MemoryMCP         RunResult `json:"memory_mcp"`
	MCPRESTParity     bool      `json:"mcp_rest_parity"`
	MCPRESTParityNote string    `json:"mcp_rest_parity_note,omitempty"`
	MemoryHelped      bool      `json:"memory_helped"`
}

// SuiteMetrics aggregates benchmark metrics.
type SuiteMetrics struct {
	TotalTasks              int     `json:"total_tasks"`
	MemoryHelpRate          float64 `json:"memory_help_rate"`
	MemoryHarmRate          float64 `json:"memory_harm_rate"`
	MemoryIrrelevantRate    float64 `json:"memory_irrelevant_rate"`
	MemoryMisuseRate        float64 `json:"memory_misuse_rate"`
	AnswerImprovementRate   float64 `json:"answer_improvement_rate"`
	MCPRESTParityRate       float64 `json:"mcp_rest_parity_rate"`
	ForbiddenMemoryUseRate  float64 `json:"forbidden_memory_use_rate"`
	LifecycleMisuseRate     float64 `json:"lifecycle_misuse_rate"`
	DateBoundMisuseRate     float64 `json:"date_bound_misuse_rate"`
	UtilityMisuseRate       float64 `json:"utility_misuse_rate"`
	HelpEligibleTasks       int     `json:"help_eligible_tasks"`
	HelpedTasks             int     `json:"helped_tasks"`
	HarmedTasks             int     `json:"harmed_tasks"`
	MisusedTasks            int     `json:"misused_tasks"`
	ParityRequiredTasks     int     `json:"parity_required_tasks"`
	ParityPassedTasks       int     `json:"parity_passed_tasks"`
}

// BenchmarkReport is the full artifact written to artifacts/.
type BenchmarkReport struct {
	GeneratedAt string        `json:"generated_at"`
	Summary     SuiteMetrics  `json:"summary"`
	Tasks       []TaskResult  `json:"tasks"`
	Thresholds  GateThresholds `json:"thresholds"`
	GatePassed  bool          `json:"gate_passed"`
	GateFailures []string     `json:"gate_failures,omitempty"`
}

// GateThresholds for proof-agent-memory-effectiveness.
type GateThresholds struct {
	MemoryHelpRateMin         float64 `json:"memory_help_rate_min"`
	MemoryHarmRateMax         float64 `json:"memory_harm_rate_max"`
	MemoryMisuseRateMax       float64 `json:"memory_misuse_rate_max"`
	ForbiddenMemoryUseRateMax float64 `json:"forbidden_memory_use_rate_max"`
	LifecycleMisuseRateMax    float64 `json:"lifecycle_misuse_rate_max"`
	DateBoundMisuseRateMax    float64 `json:"date_bound_misuse_rate_max"`
	UtilityMisuseRateMax      float64 `json:"utility_misuse_rate_max"`
	MCPRESTParityRateMin      float64 `json:"mcp_rest_parity_rate_min"`
}

// DefaultGateThresholds returns conservative Phase 11B thresholds.
func DefaultGateThresholds() GateThresholds {
	return GateThresholds{
		MemoryHelpRateMin:         0.50,
		MemoryHarmRateMax:         0,
		MemoryMisuseRateMax:       0,
		ForbiddenMemoryUseRateMax: 0,
		LifecycleMisuseRateMax:    0,
		DateBoundMisuseRateMax:    0,
		UtilityMisuseRateMax:      0,
		MCPRESTParityRateMin:      1.0,
	}
}

// parseStatus converts fixture status to api.Status.
func parseStatus(s string) api.Status {
	if s == "" {
		return api.StatusActive
	}
	return api.Status(s)
}

// parseKind converts fixture kind.
func parseKind(s string) api.MemoryKind {
	return api.MemoryKind(s)
}

// parseApplicability converts fixture applicability.
func parseApplicability(s string, authority int) api.Applicability {
	if s != "" {
		return api.Applicability(s)
	}
	if authority >= 8 {
		return api.ApplicabilityGoverning
	}
	return api.ApplicabilityAdvisory
}

// parseRFC3339 optional time.
func parseRFC3339(s string) *time.Time {
	s = trim(s)
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	tt := t.UTC()
	return &tt
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	return s
}
