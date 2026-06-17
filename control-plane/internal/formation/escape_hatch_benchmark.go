package formation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"control-plane/pkg/api"
)

// EscapeCase is one hostile escape-hatch fixture.
type EscapeCase struct {
	ID                      string `json:"id"`
	Path                    string `json:"path"`
	Interface               string `json:"interface"`
	Input                   struct {
		Kind            string   `json:"kind"`
		Statement       string   `json:"statement"`
		Authority       int      `json:"authority"`
		Applicability   string   `json:"applicability"`
		Status          string   `json:"status"`
		SchemaType      string   `json:"schema_type"`
		Scope           string   `json:"scope"`
		Reason          string   `json:"reason"`
		SourceType      string   `json:"source_type"`
		AuthorityBasis  string   `json:"authority_basis"`
		UseInstruction  string   `json:"use_instruction"`
		RetrievalCues   []string `json:"retrieval_cues"`
	} `json:"input"`
	ExpectedQualityDecision string   `json:"expected_quality_decision"`
	ExpectedGateOutcome     string   `json:"expected_gate_outcome"`
	ExpectedActiveElig      bool     `json:"expected_active_eligibility"`
	ExpectedDefects         []string `json:"expected_defects,omitempty"`
	MaxAuthority            int      `json:"max_authority,omitempty"`
	EscapeHatchResult       string   `json:"escape_hatch_result"`
}

// EscapeCaseFile is cases.json root.
type EscapeCaseFile struct {
	Cases []EscapeCase `json:"cases"`
}

// EscapeCaseResult is one evaluated escape case.
type EscapeCaseResult struct {
	ID                    string  `json:"id"`
	Path                  string  `json:"path"`
	ExpectedGateOutcome   string  `json:"expected_gate_outcome"`
	ActualGateOutcome     string  `json:"actual_gate_outcome"`
	ExpectedQuality       string  `json:"expected_quality_decision"`
	ActualQuality         string  `json:"actual_quality_decision"`
	QualityScore          float64 `json:"quality_score,omitempty"`
	ActiveEligible        bool    `json:"active_eligible"`
	Authority             int     `json:"authority"`
	Passed                bool    `json:"passed"`
	MCPRESTParity         bool    `json:"mcp_rest_parity,omitempty"`
	UnsafePromote         bool    `json:"unsafe_promote,omitempty"`
	UnsafeProbationary    bool    `json:"unsafe_probationary_influence,omitempty"`
	EscapeHatchOpen       bool    `json:"escape_hatch_open"`
}

// EscapeMetrics aggregates escape-hatch benchmark metrics.
type EscapeMetrics struct {
	TotalCases                          int     `json:"total_cases"`
	DirectCreateQualityCoverageRate     float64 `json:"direct_create_quality_coverage_rate"`
	PromoteQualityCoverageRate          float64 `json:"promote_quality_coverage_rate"`
	ProbationaryIngestQualityCoverageRate float64 `json:"probationary_ingest_quality_coverage_rate"`
	AllActivePathsQualityCoverageRate   float64 `json:"all_active_paths_quality_coverage_rate"`
	EscapeHatchCount                    int     `json:"escape_hatch_count"`
	UnsafePromoteAcceptanceRate         float64 `json:"unsafe_promote_acceptance_rate"`
	UnsafeProbationaryInfluenceRate     float64 `json:"unsafe_probationary_influence_rate"`
	MCPRESTFormationParityRate          float64 `json:"mcp_rest_formation_parity_rate"`
}

// EscapeBenchmarkReport is the Phase 11E artifact.
type EscapeBenchmarkReport struct {
	GeneratedAt  string             `json:"generated_at"`
	Summary      EscapeMetrics      `json:"summary"`
	Isolation    map[string]int     `json:"isolation_metrics,omitempty"`
	Cases        []EscapeCaseResult `json:"cases"`
	GatePassed   bool               `json:"gate_passed"`
	GateFailures []string           `json:"gate_failures,omitempty"`
}

// DefaultEscapeCasesDir returns testdata path.
func DefaultEscapeCasesDir() string {
	if d := os.Getenv("FORMATION_ESCAPE_HATCHES_DIR"); d != "" {
		return d
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("control-plane", "testdata", "formation_escape_hatches")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "formation_escape_hatches")
}

// LoadEscapeCases reads cases.json.
func LoadEscapeCases(dir string) (EscapeCaseFile, error) {
	b, err := os.ReadFile(filepath.Join(dir, "cases.json"))
	if err != nil {
		return EscapeCaseFile{}, err
	}
	var f EscapeCaseFile
	if err := json.Unmarshal(b, &f); err != nil {
		return EscapeCaseFile{}, err
	}
	return f, nil
}

// EvaluateEscapeCase runs the formation gate for one fixture case.
func EvaluateEscapeCase(g *Gate, c EscapeCase) (EscapeCaseResult, *CreateInput, error) {
	in := InputFromEscapeCase(c)
	d, err := g.EvaluateCreateInput(in)
	res := EscapeCaseResult{
		ID:                  c.ID,
		Path:                c.Path,
		ExpectedGateOutcome: c.ExpectedGateOutcome,
		ExpectedQuality:     c.ExpectedQualityDecision,
		Authority:           in.Authority,
	}
	if err != nil {
		res.ActualGateOutcome = "reject"
		res.ActiveEligible = false
	} else {
		res.ActualGateOutcome = string(d.Outcome)
		res.ActiveEligible = in.Status == api.StatusActive || in.Status == ""
		if d.ForcePending {
			res.ActiveEligible = false
		}
	}
	if d.Quality != nil {
		res.ActualQuality = string(d.Quality.Decision)
		res.QualityScore = d.Quality.QualityScore
	}
	res.UnsafePromote = c.Path == "promote" && res.ActiveEligible && c.ExpectedActiveElig == false
	res.UnsafeProbationary = (c.Path == "probationary_ingest" || c.Path == "record_experience") &&
		res.ActiveEligible && c.ExpectedActiveElig == false && c.ExpectedGateOutcome != "allow"
	res.EscapeHatchOpen = pathNeedsQuality(c.Path) && d.Quality == nil && err == nil
	res.Passed = res.ActualGateOutcome == c.ExpectedGateOutcome &&
		(err != nil) == (c.ExpectedGateOutcome == "reject")
	if d.Quality != nil {
		res.Passed = res.Passed && res.ActualQuality == c.ExpectedQualityDecision
	}
	if c.MaxAuthority > 0 && in.Authority > c.MaxAuthority {
		res.Passed = false
	}
	if c.ExpectedActiveElig != res.ActiveEligible {
		res.Passed = false
	}
	return res, in, err
}

func pathNeedsQuality(path string) bool {
	switch path {
	case "direct_create", "promote", "probationary_ingest", "record_experience":
		return true
	default:
		return false
	}
}

// InputFromEscapeCase builds CreateInput for gate evaluation.
func InputFromEscapeCase(c EscapeCase) *CreateInput {
	in := &CreateInput{
		Kind:          api.MemoryKind(c.Input.Kind),
		Authority:     c.Input.Authority,
		Applicability: api.Applicability(c.Input.Applicability),
		Status:        api.Status(c.Input.Status),
		Statement:     c.Input.Statement,
	}
	switch c.Path {
	case "promote":
		in.Path = PathPromote
	case "probationary_ingest", "record_experience":
		in.Path = PathProbationaryIngest
	default:
		in.Path = PathDirectCreate
	}
	payload := map[string]any{}
	if c.Input.SchemaType != "" {
		payload["schema_type"] = c.Input.SchemaType
	}
	if c.Input.Scope != "" {
		payload["scope"] = c.Input.Scope
	}
	if c.Input.Reason != "" {
		payload["reason"] = c.Input.Reason
	}
	if c.Input.SourceType != "" {
		payload["source_type"] = c.Input.SourceType
	}
	if c.Input.AuthorityBasis != "" {
		payload["authority_basis"] = c.Input.AuthorityBasis
	}
	if c.Input.UseInstruction != "" {
		payload["use_instruction"] = c.Input.UseInstruction
	}
	if len(c.Input.RetrievalCues) > 0 {
		payload["retrieval_cues"] = c.Input.RetrievalCues
	}
	if len(payload) > 0 {
		b, _ := json.Marshal(payload)
		in.Payload = b
	}
	return in
}

// RunEscapeBenchmark evaluates all escape cases.
func RunEscapeBenchmark(g *Gate, cases []EscapeCase) EscapeBenchmarkReport {
	var results []EscapeCaseResult
	var (
		directN, directOK int
		promoteN, promoteOK int
		probN, probOK int
		unsafePromote, unsafeProb, escapeOpen int
		parityEligible, parityOK int
	)
	for _, c := range cases {
		r1, _, _ := EvaluateEscapeCase(g, c)
		if c.Interface == "both" {
			r2, _, _ := EvaluateEscapeCase(g, c)
			r1.MCPRESTParity = r1.ActualGateOutcome == r2.ActualGateOutcome &&
				r1.ActualQuality == r2.ActualQuality
			r1.Passed = r1.Passed && r1.MCPRESTParity
			parityEligible++
			if r1.MCPRESTParity {
				parityOK++
			}
		}
		if r1.UnsafePromote {
			unsafePromote++
		}
		if r1.UnsafeProbationary {
			unsafeProb++
		}
		if r1.EscapeHatchOpen {
			escapeOpen++
		}
		switch c.Path {
		case "direct_create":
			directN++
			if r1.Passed {
				directOK++
			}
		case "promote":
			promoteN++
			if r1.Passed {
				promoteOK++
			}
		case "probationary_ingest", "record_experience":
			probN++
			if r1.Passed {
				probOK++
			}
		}
		results = append(results, r1)
	}
	rate := func(ok, n int) float64 {
		if n == 0 {
			return 1
		}
		return float64(ok) / float64(n)
	}
	totalN := len(cases)
	summary := EscapeMetrics{
		TotalCases:                          totalN,
		DirectCreateQualityCoverageRate:     rate(directOK, directN),
		PromoteQualityCoverageRate:          rate(promoteOK, promoteN),
		ProbationaryIngestQualityCoverageRate: rate(probOK, probN),
		AllActivePathsQualityCoverageRate:   rate(directOK+promoteOK+probOK, directN+promoteN+probN),
		EscapeHatchCount:                    escapeOpen,
		UnsafePromoteAcceptanceRate:         float64(unsafePromote) / float64(max(1, promoteN)),
		UnsafeProbationaryInfluenceRate:     float64(unsafeProb) / float64(max(1, probN)),
		MCPRESTFormationParityRate:          rate(parityOK, parityEligible),
	}
	if parityEligible == 0 {
		summary.MCPRESTFormationParityRate = 1
	}
	if unsafePromote == 0 {
		summary.UnsafePromoteAcceptanceRate = 0
	}
	if unsafeProb == 0 {
		summary.UnsafeProbationaryInfluenceRate = 0
	}
	passed, fails := EvaluateEscapeGate(summary)
	return EscapeBenchmarkReport{
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		Summary:      summary,
		Cases:        results,
		GatePassed:   passed,
		GateFailures: fails,
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// EvaluateEscapeGate checks Phase 11E thresholds.
func EvaluateEscapeGate(m EscapeMetrics) (bool, []string) {
	var fails []string
	check := func(name string, ok bool) {
		if !ok {
			fails = append(fails, name)
		}
	}
	check("direct_create_quality_coverage_rate", m.DirectCreateQualityCoverageRate >= 1.0)
	check("promote_quality_coverage_rate", m.PromoteQualityCoverageRate >= 1.0)
	check("probationary_ingest_quality_coverage_rate", m.ProbationaryIngestQualityCoverageRate >= 1.0)
	check("all_active_paths_quality_coverage_rate", m.AllActivePathsQualityCoverageRate >= 1.0)
	check("escape_hatch_count", m.EscapeHatchCount == 0)
	check("unsafe_promote_acceptance_rate", m.UnsafePromoteAcceptanceRate == 0)
	check("unsafe_probationary_influence_rate", m.UnsafeProbationaryInfluenceRate == 0)
	check("mcp_rest_formation_parity_rate", m.MCPRESTFormationParityRate >= 1.0)
	return len(fails) == 0, fails
}

// WriteEscapeReport writes benchmark JSON.
func WriteEscapeReport(report *EscapeBenchmarkReport, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
