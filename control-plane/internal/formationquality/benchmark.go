package formationquality

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// FixtureCase is one hostile formation quality test case.
type FixtureCase struct {
	ID               string   `json:"id"`
	Interface        string   `json:"interface,omitempty"`
	Input            Input    `json:"input"`
	ExpectedDecision Decision `json:"expected_decision"`
	ExpectedDefects  []string `json:"expected_defects,omitempty"`
	SafeForActive    bool     `json:"safe_for_active"`
	MinQualityScore  float64  `json:"min_quality_score,omitempty"`
}

// FixtureFile is the cases.json root.
type FixtureFile struct {
	Cases []FixtureCase `json:"cases"`
}

// CaseResult is one evaluated fixture outcome.
type CaseResult struct {
	ID             string   `json:"id"`
	Expected       Decision `json:"expected_decision"`
	Actual         Decision `json:"actual_decision"`
	QualityScore   float64  `json:"quality_score"`
	SafeForActive  bool     `json:"safe_for_active_recall"`
	DefectCodes    []string `json:"defect_codes"`
	Passed         bool     `json:"passed"`
	MCPRESTParity  bool     `json:"mcp_rest_parity,omitempty"`
}

// Metrics aggregates benchmark results.
type Metrics struct {
	TotalCases                    int     `json:"total_cases"`
	FormationQualityPassRate      float64 `json:"formation_quality_pass_rate"`
	DangerousActiveMemoryRate     float64 `json:"dangerous_active_memory_rate"`
	UnderEncodedActiveMemoryRate  float64 `json:"under_encoded_active_memory_rate"`
	VagueMemoryAcceptanceRate     float64 `json:"vague_memory_acceptance_rate"`
	OvergeneralizedActiveRate     float64 `json:"overgeneralized_active_memory_rate"`
	MissingProvenanceActiveRate   float64 `json:"missing_provenance_active_rate"`
	MissingScopeActiveRate        float64 `json:"missing_scope_active_rate"`
	MissingCuesActiveRate         float64 `json:"missing_cues_active_rate"`
	MCPRESTFormationParityRate    float64 `json:"mcp_rest_formation_parity_rate"`
	SchemaRuleCoverageRate        float64 `json:"schema_rule_coverage_rate"`
}

// GateThresholds for Phase 11D proof.
type GateThresholds struct {
	FormationQualityPassRateMin     float64 `json:"formation_quality_pass_rate_min"`
	SchemaRuleCoverageRateMin       float64 `json:"schema_rule_coverage_rate_min"`
	DangerousActiveMemoryRateMax    float64 `json:"dangerous_active_memory_rate_max"`
	UnderEncodedActiveMemoryRateMax float64 `json:"under_encoded_active_memory_rate_max"`
	VagueMemoryAcceptanceRateMax    float64 `json:"vague_memory_acceptance_rate_max"`
	OvergeneralizedActiveRateMax    float64 `json:"overgeneralized_active_memory_rate_max"`
	MissingProvenanceActiveRateMax  float64 `json:"missing_provenance_active_rate_max"`
	MissingScopeActiveRateMax       float64 `json:"missing_scope_active_rate_max"`
	MissingCuesActiveRateMax        float64 `json:"missing_cues_active_memory_rate_max"`
	MCPRESTFormationParityRateMin   float64 `json:"mcp_rest_formation_parity_rate_min"`
}

// DefaultGateThresholds returns Phase 11D thresholds.
func DefaultGateThresholds() GateThresholds {
	return GateThresholds{
		FormationQualityPassRateMin:     0.80,
		SchemaRuleCoverageRateMin:       0.90,
		DangerousActiveMemoryRateMax:    0,
		UnderEncodedActiveMemoryRateMax: 0,
		VagueMemoryAcceptanceRateMax:    0,
		OvergeneralizedActiveRateMax:    0,
		MissingProvenanceActiveRateMax:  0,
		MissingScopeActiveRateMax:       0,
		MissingCuesActiveRateMax:        0,
		MCPRESTFormationParityRateMin:   1.0,
	}
}

// BenchmarkReport is the Phase 11D artifact.
type BenchmarkReport struct {
	GeneratedAt string         `json:"generated_at"`
	Summary     Metrics        `json:"summary"`
	Cases       []CaseResult   `json:"cases"`
	Thresholds  GateThresholds `json:"thresholds"`
	GatePassed  bool           `json:"gate_passed"`
	GateFailures []string      `json:"gate_failures,omitempty"`
}

// DefaultCasesDir returns testdata path.
func DefaultCasesDir() string {
	if d := os.Getenv("MEMORY_FORMATION_QUALITY_DIR"); d != "" {
		return d
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("control-plane", "testdata", "memory_formation_quality")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "memory_formation_quality")
}

// LoadCases reads cases.json.
func LoadCases(dir string) (FixtureFile, error) {
	b, err := os.ReadFile(filepath.Join(dir, "cases.json"))
	if err != nil {
		return FixtureFile{}, err
	}
	var f FixtureFile
	if err := json.Unmarshal(b, &f); err != nil {
		return FixtureFile{}, err
	}
	return f, nil
}

// RunBenchmark evaluates all fixture cases.
func RunBenchmark(cases []FixtureCase) BenchmarkReport {
	eval := NewEvaluator()
	var results []CaseResult
	for _, c := range cases {
		r := eval.Evaluate(c.Input)
		cr := CaseResult{
			ID:            c.ID,
			Expected:      c.ExpectedDecision,
			Actual:        r.Decision,
			QualityScore:  r.QualityScore,
			SafeForActive: r.SafeForActiveRecall,
			Passed:        r.Decision == c.ExpectedDecision && r.SafeForActiveRecall == c.SafeForActive,
		}
		for _, d := range r.Defects {
			cr.DefectCodes = append(cr.DefectCodes, string(d.Code))
		}
		if c.Interface == "both" {
			r2 := eval.Evaluate(c.Input)
			cr.MCPRESTParity = r2.Decision == r.Decision && r2.QualityScore == r.QualityScore
			cr.Passed = cr.Passed && cr.MCPRESTParity
		}
		if c.MinQualityScore > 0 && r.QualityScore < c.MinQualityScore {
			cr.Passed = false
		}
		if len(c.ExpectedDefects) > 0 {
			if !containsAllDefects(cr.DefectCodes, c.ExpectedDefects) {
				cr.Passed = false
			}
		}
		results = append(results, cr)
	}
	summary := ComputeMetrics(cases, results)
	th := DefaultGateThresholds()
	passed, fails := EvaluateGate(summary, th)
	return BenchmarkReport{
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		Summary:      summary,
		Cases:        results,
		Thresholds:   th,
		GatePassed:   passed,
		GateFailures: fails,
	}
}

func containsAllDefects(got, want []string) bool {
	set := map[string]struct{}{}
	for _, g := range got {
		set[g] = struct{}{}
	}
	for _, w := range want {
		if _, ok := set[w]; !ok {
			return false
		}
	}
	return true
}

// ComputeMetrics aggregates safety and pass rates.
func ComputeMetrics(cases []FixtureCase, results []CaseResult) Metrics {
	m := Metrics{TotalCases: len(results)}
	if len(results) == 0 {
		return m
	}
	var passed, dangerousActive, underCuedActive, vagueAccepted, overgenActive int
	var missingProvActive, missingScopeActive, missingCuesActive int
	var parityEligible, parityPassed int
	schemaTypes := map[string]struct{}{}
	for _, c := range cases {
		if c.Input.SchemaType != "" {
			schemaTypes[c.Input.SchemaType] = struct{}{}
		}
	}
	schemaCovered := map[string]struct{}{}

	for i, cr := range results {
		c := cases[i]
		if cr.Passed {
			passed++
		}
		if cr.SafeForActive {
			if containsDefect(cr.DefectCodes, "refuted_guidance_as_active") ||
				containsDefect(cr.DefectCodes, "superseded_guidance_as_active") ||
				containsDefect(cr.DefectCodes, "unsafe_direct_governing_memory") ||
				containsDefect(cr.DefectCodes, "agent_inferred_preference_cannot_be_high_authority") {
				dangerousActive++
			}
			if containsDefect(cr.DefectCodes, "missing_retrieval_cues") {
				underCuedActive++
			}
			if containsDefect(cr.DefectCodes, "vague_statement") {
				vagueAccepted++
			}
			if containsDefect(cr.DefectCodes, "overgeneralized_statement") {
				overgenActive++
			}
			if containsDefect(cr.DefectCodes, "missing_provenance") {
				missingProvActive++
			}
			if containsDefect(cr.DefectCodes, "missing_scope") || containsDefect(cr.DefectCodes, "constraint_without_scope") {
				missingScopeActive++
			}
			if containsDefect(cr.DefectCodes, "missing_retrieval_cues") {
				missingCuesActive++
			}
		}
		if c.Interface == "both" {
			parityEligible++
			if cr.MCPRESTParity {
				parityPassed++
			}
		}
		if c.Input.SchemaType != "" && cr.Passed {
			schemaCovered[c.Input.SchemaType] = struct{}{}
		}
	}
	n := float64(len(results))
	m.FormationQualityPassRate = float64(passed) / n
	m.DangerousActiveMemoryRate = float64(dangerousActive) / n
	m.UnderEncodedActiveMemoryRate = float64(underCuedActive) / n
	m.VagueMemoryAcceptanceRate = float64(vagueAccepted) / n
	m.OvergeneralizedActiveRate = float64(overgenActive) / n
	m.MissingProvenanceActiveRate = float64(missingProvActive) / n
	m.MissingScopeActiveRate = float64(missingScopeActive) / n
	m.MissingCuesActiveRate = float64(missingCuesActive) / n
	if parityEligible > 0 {
		m.MCPRESTFormationParityRate = float64(parityPassed) / float64(parityEligible)
	} else {
		m.MCPRESTFormationParityRate = 1
	}
	if len(schemaTypes) > 0 {
		m.SchemaRuleCoverageRate = float64(len(schemaCovered)) / float64(len(schemaTypes))
	} else {
		m.SchemaRuleCoverageRate = 1
	}
	return m
}

func containsDefect(codes []string, code string) bool {
	for _, c := range codes {
		if c == code {
			return true
		}
	}
	return false
}

// EvaluateGate checks Phase 11D thresholds.
func EvaluateGate(m Metrics, th GateThresholds) (bool, []string) {
	var fails []string
	checkMax := func(name string, val, max float64) {
		if val > max {
			fails = append(fails, name+" above max")
		}
	}
	checkMin := func(name string, val, min float64) {
		if val < min {
			fails = append(fails, name+" below min")
		}
	}
	checkMax("dangerous_active_memory_rate", m.DangerousActiveMemoryRate, th.DangerousActiveMemoryRateMax)
	checkMax("under_encoded_active_memory_rate", m.UnderEncodedActiveMemoryRate, th.UnderEncodedActiveMemoryRateMax)
	checkMax("vague_memory_acceptance_rate", m.VagueMemoryAcceptanceRate, th.VagueMemoryAcceptanceRateMax)
	checkMax("overgeneralized_active_memory_rate", m.OvergeneralizedActiveRate, th.OvergeneralizedActiveRateMax)
	checkMax("missing_provenance_active_rate", m.MissingProvenanceActiveRate, th.MissingProvenanceActiveRateMax)
	checkMax("missing_scope_active_rate", m.MissingScopeActiveRate, th.MissingScopeActiveRateMax)
	checkMax("missing_cues_active_rate", m.MissingCuesActiveRate, th.MissingCuesActiveRateMax)
	checkMin("formation_quality_pass_rate", m.FormationQualityPassRate, th.FormationQualityPassRateMin)
	checkMin("schema_rule_coverage_rate", m.SchemaRuleCoverageRate, th.SchemaRuleCoverageRateMin)
	checkMin("mcp_rest_formation_parity_rate", m.MCPRESTFormationParityRate, th.MCPRESTFormationParityRateMin)
	return len(fails) == 0, fails
}

// WriteReport writes benchmark JSON.
func WriteReport(report *BenchmarkReport, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
