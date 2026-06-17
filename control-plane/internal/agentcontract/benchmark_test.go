package agentcontract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"control-plane/internal/mcp"
	"control-plane/internal/recall"
	"control-plane/pkg/api"
)

type ContractCaseMemory struct {
	MemoryID       string    `json:"memory_id"`
	Statement      string    `json:"statement"`
	SchemaType     string    `json:"schema_type"`
	LifecycleRole  string    `json:"lifecycle_role"`
	Status         string    `json:"status"`
	Applicability  string    `json:"applicability"`
	Scope          string    `json:"scope"`
	NegativeScope  []string  `json:"negative_scope"`
	UseInstruction string    `json:"use_instruction"`
	MisuseWarning  string    `json:"misuse_warning"`
	SourceType     string    `json:"source_type"`
	AuthorityBasis string   `json:"authority_basis"`

	UtilityScore  *float64 `json:"utility_score"`
	QualityState  string   `json:"quality_state"`
	QualityScore  *float64 `json:"quality_score"`
	SafeForActiveRecall *bool `json:"safe_for_active_recall"`

	SupersededBy string `json:"superseded_by"`
}

type ContractCaseExpected struct {
	ContractPassed bool     `json:"contract_passed"`
	ExpectedUseDecision string `json:"expected_use_decision"`
	ExpectedIgnoreReason string `json:"expected_ignore_reason"`
	ExpectedMissingRequiredFieldsContains []string `json:"expected_missing_required_fields_contains"`
	ExpectedUnsafeOmissionsContains        []string `json:"expected_unsafe_omissions_contains"`
}

type ContractCase struct {
	ID           string                 `json:"id"`
	Interface    string                 `json:"interface"`
	RecallMode   string                 `json:"recall_mode"`
	TaskTags     []string               `json:"task_tags"`
	InputMemories []ContractCaseMemory `json:"input_memories"`
	Expected     ContractCaseExpected   `json:"expected"`
}

type ContractCasesFile struct {
	Cases []ContractCase `json:"cases"`
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	dir := filepath.Dir(file)
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "control-plane", "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Dir(file)
}

func loadContractCases(t *testing.T) []ContractCase {
	t.Helper()
	root := repoRoot()
	path := filepath.Join(root, "control-plane", "testdata", "agent_memory_contract", "cases.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read cases.json: %v", err)
	}
	var f ContractCasesFile
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatalf("parse cases.json: %v", err)
	}
	return f.Cases
}

func parseRecallMode(s string) recall.RecallMode {
	switch s {
	case "historical":
		return recall.RecallModeHistorical
	case "current", "":
		return recall.RecallModeCurrent
	default:
		return recall.RecallModeCurrent
	}
}

func parseApplicability(s string) api.Applicability {
	switch s {
	case "advisory":
		return api.ApplicabilityAdvisory
	case "governing", "":
		return api.ApplicabilityGoverning
	default:
		return api.ApplicabilityGoverning
	}
}

func buildBundleFromCase(c ContractCase) *recall.RecallBundle {
	b := &recall.RecallBundle{}
	for _, cm := range c.InputMemories {
		var util *float64 = cm.UtilityScore
		var qscore *float64 = cm.QualityScore
		it := recall.MemoryItem{
			ID:              cm.MemoryID,
			Kind:            "constraint",
			Statement:      cm.Statement,
			Authority:      3,
			Applicability:  parseApplicability(cm.Applicability),
			Status:         cm.Status,
			LifecycleRole:  cm.LifecycleRole,
			Scope:          cm.Scope,
			NegativeScope:  cm.NegativeScope,
			UseInstruction: cm.UseInstruction,
			MisuseWarning:  cm.MisuseWarning,
			SourceType:     cm.SourceType,
			AuthorityBasis: cm.AuthorityBasis,
			SchemaType:     cm.SchemaType,
			UtilityScore:   util,
			QualityState:   cm.QualityState,
			QualityScore:   qscore,
			SafeForActiveRecall: cm.SafeForActiveRecall,
			SupersededBy:   cm.SupersededBy,
		}
		// Contract evaluator collects from all buckets; keep it simple: all in governing constraints.
		b.GoverningConstraints = append(b.GoverningConstraints, it)
	}
	return b
}

func unsafeOmissionReasonFromContract(ev ContractEvaluation) string {
	if len(ev.UnsafeOmissions) == 0 {
		return "contract_unsafe"
	}
	return ev.UnsafeOmissions[0]
}

type AgentContractMetrics struct {
	ContractPassRate            float64 `json:"contract_pass_rate"`
	BundleContractScore         float64 `json:"bundle_contract_score"`
	MemoryContractScoreAvg      float64 `json:"memory_contract_score_avg"`
	FlattenedTextOnlyRate        float64 `json:"flattened_text_only_rate"`
	MissingLifecycleRoleRate     float64 `json:"missing_lifecycle_role_rate"`
	MissingScopeForGuidanceRate  float64 `json:"missing_scope_for_guidance_rate"`
	MissingUseInstructionRate    float64 `json:"missing_use_instruction_rate"`
	MissingMisuseWarningRate     float64 `json:"missing_misuse_warning_rate"`
	MissingProvenanceRate         float64 `json:"missing_provenance_rate"`
	MissingQualityStateRate      float64 `json:"missing_quality_state_rate"`

	SupersededAsCurrentRate      float64 `json:"superseded_as_current_rate"`
	RefutedAsCurrentRate         float64 `json:"refuted_as_current_rate"`
	HistoricalAsCurrentRate      float64 `json:"historical_as_current_rate"`

	WrongScopeUseRate             float64 `json:"wrong_scope_use_rate"`
	NegativeScopeViolationRate   float64 `json:"negative_scope_violation_rate"`

	MCPRESTContractParityRate    float64 `json:"mcp_rest_contract_parity_rate"`
}

func TestAgentMemoryContractBenchmarkArtifact(t *testing.T) {
	if os.Getenv("AGENT_MEMORY_CONTRACT_BENCHMARK") != "1" {
		t.Skip("set AGENT_MEMORY_CONTRACT_BENCHMARK=1")
	}
	cases := loadContractCases(t)
	if len(cases) < 20 {
		t.Fatalf("need >=20 cases, got %d", len(cases))
	}

	var contractPassedCount float64
	var bundleScoreSum float64
	var memoryScoreSum float64
	var memoryScoreCount float64

	var flattenedTextOnlyPass float64
	var flattenedTextOnlyDenom float64

	var missingLifecycleRoleUnsafePass float64
	var missingLifecycleRoleDenom float64

	var missingScopeUnsafePass float64
	var missingScopeDenom float64

	var missingUseUnsafePass float64
	var missingUseDenom float64

	var missingMisuseUnsafePass float64
	var missingMisuseDenom float64

	var missingProvenanceUnsafePass float64
	var missingProvenanceDenom float64

	var missingQualityStateUnsafePass float64
	var missingQualityStateDenom float64

	var supersededAsCurrent float64
	var supersededAsCurrentDenom float64
	var refutedAsCurrent float64
	var refutedAsCurrentDenom float64
	var historicalAsCurrent float64
	var historicalAsCurrentDenom float64

	var wrongScopeUseRate float64
	var wrongScopeUseRateDenom float64
	var negativeScopeViolation float64
	var negativeScopeViolationDenom float64

	var parityMatch float64
	var parityDenom float64

	for _, c := range cases {
		bundle := buildBundleFromCase(c)
		mode := parseRecallMode(c.RecallMode)

		restEv := EvaluateBundleContract(bundle, mode, false)
		var mcpEv ContractEvaluation
		if c.Interface == "both" {
			mcpBundle := mcpRecallBundleFromRESTBundle(t, bundle)
			mcpEv = EvaluateBundleContract(mcpBundle, mode, false)
		} else {
			mcpEv = EvaluateBundleContract(bundle, mode, c.Interface == "mcp")
		}

		primaryEv := restEv
		if c.Interface == "mcp" {
			primaryEv = mcpEv
		}

		contractPassed := primaryEv.ContractPassed
		if contractPassed {
			contractPassedCount++
		}
		bundleScoreSum += primaryEv.BundleContractScore
		for _, score := range primaryEv.MemoryContractScores {
			memoryScoreSum += score
			memoryScoreCount++
		}

		// Flattened text-only detection: count only the cases where we simulate flattened MCP.
		if c.Interface == "mcp" {
			flattenedTextOnlyDenom++
			if primaryEv.ContractPassed {
				flattenedTextOnlyPass++
			}
		}

		// Missing field unsafe-pass rates.
		for _, it := range c.InputMemories {
			// Missing lifecycle role.
			if it.LifecycleRole == "" {
				missingLifecycleRoleDenom++
				if primaryEv.ContractPassed {
					missingLifecycleRoleUnsafePass++
				}
			}
			// Missing scope for guidance.
			if it.LifecycleRole == recall.LifecycleCurrentGuidance && it.Scope == "" {
				missingScopeDenom++
				if primaryEv.ContractPassed {
					missingScopeUnsafePass++
				}
			}
			// Missing use instruction for guidance.
			if it.LifecycleRole == recall.LifecycleCurrentGuidance && it.UseInstruction == "" {
				missingUseDenom++
				if primaryEv.ContractPassed {
					missingUseUnsafePass++
				}
			}
			// Missing misuse warning for historical.
			if it.LifecycleRole == recall.LifecycleHistoricalContext && it.MisuseWarning == "" {
				missingMisuseDenom++
				if primaryEv.ContractPassed {
					missingMisuseUnsafePass++
				}
			}
			// Missing provenance.
			if it.SourceType == "" || it.AuthorityBasis == "" {
				missingProvenanceDenom++
				if primaryEv.ContractPassed {
					missingProvenanceUnsafePass++
				}
			}
			// Missing quality state.
			if it.LifecycleRole == recall.LifecycleCurrentGuidance && it.QualityState == "" {
				missingQualityStateDenom++
				if primaryEv.ContractPassed {
					missingQualityStateUnsafePass++
				}
			}
		}

		// Discipline correctness rates: compute based on simulated agent decision.
		for _, it := range c.InputMemories {
			// Use discipline on primary interface evaluation.
			disc := DecideUseDiscipline(itToMemoryItem(it), c.TaskTags)

			// Override: if the contract evaluator rejected this memory item in current guidance,
			// do not let it guide action.
			score := primaryEv.MemoryContractScores[it.MemoryID]
			if it.LifecycleRole == recall.LifecycleCurrentGuidance && score < 1.0 {
				disc.Decision = "ignore"
				disc.Reason = unsafeOmissionReasonFromContract(primaryEv)
			}

			// Wrong-scope and negative-scope violations.
			scopeMatch := false
			for _, tt := range c.TaskTags {
				if tt == it.Scope {
					scopeMatch = true
					break
				}
			}
			if disc.Decision == "use" {
				if !scopeMatch {
					wrongScopeUseRate++
					wrongScopeUseRateDenom++
				}
				negHit := false
				for _, ns := range it.NegativeScope {
					for _, tt := range c.TaskTags {
						if ns == tt {
							negHit = true
						}
					}
				}
				if negHit {
					negativeScopeViolation++
					negativeScopeViolationDenom++
				}
			}

			// historical/superseded/refuted used as current guidance:
			if disc.Decision == "use" {
				switch it.LifecycleRole {
				case recall.LifecycleSupersededContext:
					supersededAsCurrent++
					supersededAsCurrentDenom++
				case recall.LifecycleRefutedContext:
					refutedAsCurrent++
					refutedAsCurrentDenom++
				case recall.LifecycleHistoricalContext:
					historicalAsCurrent++
					historicalAsCurrentDenom++
				}
			}
		}

		// MCP/REST parity: only for interface "both".
		if c.Interface == "both" {
			parityDenom++
			if restEv.ContractPassed == mcpEv.ContractPassed &&
				len(restEv.UnsafeOmissions) == len(mcpEv.UnsafeOmissions) {
				parityMatch++
			}
		}
	}

	total := float64(len(cases))

	metrics := AgentContractMetrics{
		ContractPassRate:        contractPassedCount / total,
		BundleContractScore:     bundleScoreSum / total,
		MemoryContractScoreAvg: memoryScoreSum / (memoryScoreCount + 1e-9),

		FlattenedTextOnlyRate:        safeDiv(flattenedTextOnlyPass, flattenedTextOnlyDenom),
		MissingLifecycleRoleRate:     safeDiv(missingLifecycleRoleUnsafePass, missingLifecycleRoleDenom),
		MissingScopeForGuidanceRate: safeDiv(missingScopeUnsafePass, missingScopeDenom),
		MissingUseInstructionRate:    safeDiv(missingUseUnsafePass, missingUseDenom),
		MissingMisuseWarningRate:     safeDiv(missingMisuseUnsafePass, missingMisuseDenom),
		MissingProvenanceRate:         safeDiv(missingProvenanceUnsafePass, missingProvenanceDenom),
		MissingQualityStateRate:      safeDiv(missingQualityStateUnsafePass, missingQualityStateDenom),

		SupersededAsCurrentRate:      safeDiv(supersededAsCurrent, supersededAsCurrentDenom),
		RefutedAsCurrentRate:         safeDiv(refutedAsCurrent, refutedAsCurrentDenom),
		HistoricalAsCurrentRate:      safeDiv(historicalAsCurrent, historicalAsCurrentDenom),

		WrongScopeUseRate:           safeDiv(wrongScopeUseRate, wrongScopeUseRateDenom),
		NegativeScopeViolationRate: safeDiv(negativeScopeViolation, negativeScopeViolationDenom),

		MCPRESTContractParityRate: safeDiv(parityMatch, parityDenom),
	}

	artifactPath := filepath.Join(repoRoot(), "artifacts", "agent-facing-memory-contract-benchmark.json")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func itToMemoryItem(it ContractCaseMemory) recall.MemoryItem {
	var util *float64 = it.UtilityScore
	var qscore *float64 = it.QualityScore
	return recall.MemoryItem{
		ID: it.MemoryID,
		Kind: "constraint",
		Statement: it.Statement,
		Authority: 3,
		LifecycleRole: it.LifecycleRole,
		Status: it.Status,
		Applicability: parseApplicability(it.Applicability),
		SchemaType: it.SchemaType,
		Scope: it.Scope,
		NegativeScope: it.NegativeScope,
		UseInstruction: it.UseInstruction,
		MisuseWarning: it.MisuseWarning,
		SourceType: it.SourceType,
		AuthorityBasis: it.AuthorityBasis,
		UtilityScore: util,
		QualityState: it.QualityState,
		QualityScore: qscore,
		SafeForActiveRecall: it.SafeForActiveRecall,
		SupersededBy: it.SupersededBy,
	}
}

func mcpRecallBundleFromRESTBundle(t *testing.T, bundle *recall.RecallBundle) *recall.RecallBundle {
	t.Helper()

	rawBundle, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/recall/compile" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(rawBundle)
	}))
	defer srv.Close()

	client := srv.Client()

	// The MCP tool wrapper validates that recall_context receives a task or task_description.
	// Our stub compile handler ignores the body and always returns rawBundle.
	params := json.RawMessage(`{
		"name": "recall_context",
		"arguments": {
			"task": "agent contract parity check",
			"tags": ["parity"]
		}
	}`)

	toolResp, err := mcp.HandleToolsCall(client, srv.URL, "", params, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	raw, err := json.Marshal(toolResp)
	if err != nil {
		t.Fatal(err)
	}

	type mcpContentItem struct {
		Type string `json:"type"`
		JSON struct {
			McpContext   json.RawMessage `json:"mcp_context"`
			RecallBundle json.RawMessage `json:"recall_bundle"`
		} `json:"json"`
	}
	var out struct {
		Content []mcpContentItem `json:"content"`
	}

	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Content) == 0 {
		t.Fatal("mcp tools/call response missing content")
	}
	if out.Content[0].Type != "json" {
		t.Fatalf("expected mcp content[0].type=json, got %q", out.Content[0].Type)
	}
	if len(out.Content[0].JSON.RecallBundle) == 0 {
		t.Fatal("mcp content[0].json missing recall_bundle")
	}

	var got recall.RecallBundle
	if err := json.Unmarshal(out.Content[0].JSON.RecallBundle, &got); err != nil {
		t.Fatal(err)
	}
	return &got
}

func safeDiv(num, denom float64) float64 {
	if denom == 0 {
		return 0
	}
	return num / denom
}

func TestProofAgentMemoryContractHardThresholds(t *testing.T) {
	if os.Getenv("PROOF_AGENT_MEMORY_CONTRACT") != "1" {
		t.Skip("set PROOF_AGENT_MEMORY_CONTRACT=1")
	}
	cases := loadContractCases(t)

	metrics := computeAgentContractMetrics(t, cases)

	// Hard safety thresholds (must not relax).
	if metrics.FlattenedTextOnlyRate != 0 {
		t.Fatalf("flattened_text_only_rate=%.3f want 0", metrics.FlattenedTextOnlyRate)
	}
	if metrics.MissingLifecycleRoleRate != 0 {
		t.Fatalf("missing_lifecycle_role_rate=%.3f want 0", metrics.MissingLifecycleRoleRate)
	}
	if metrics.MissingScopeForGuidanceRate != 0 {
		t.Fatalf("missing_scope_for_guidance_rate=%.3f want 0", metrics.MissingScopeForGuidanceRate)
	}
	if metrics.SupersededAsCurrentRate != 0 {
		t.Fatalf("superseded_as_current_rate=%.3f want 0", metrics.SupersededAsCurrentRate)
	}
	if metrics.RefutedAsCurrentRate != 0 {
		t.Fatalf("refuted_as_current_rate=%.3f want 0", metrics.RefutedAsCurrentRate)
	}
	if metrics.HistoricalAsCurrentRate != 0 {
		t.Fatalf("historical_as_current_rate=%.3f want 0", metrics.HistoricalAsCurrentRate)
	}
	if metrics.WrongScopeUseRate != 0 {
		t.Fatalf("wrong_scope_use_rate=%.3f want 0", metrics.WrongScopeUseRate)
	}
	if metrics.NegativeScopeViolationRate != 0 {
		t.Fatalf("negative_scope_violation_rate=%.3f want 0", metrics.NegativeScopeViolationRate)
	}
	if metrics.MCPRESTContractParityRate != 1.0 {
		t.Fatalf("mcp_rest_contract_parity_rate=%.3f want 1.0", metrics.MCPRESTContractParityRate)
	}

	// Positive thresholds.
	if metrics.ContractPassRate < 0.90 {
		t.Fatalf("contract_pass_rate=%.3f want >=0.90", metrics.ContractPassRate)
	}
	if metrics.BundleContractScore < 0.90 {
		t.Fatalf("bundle_contract_score=%.3f want >=0.90", metrics.BundleContractScore)
	}
	// Floating point math can produce values like 0.89999999995 when the
	// underlying score is intended to be exactly 0.90. Allow a tight epsilon.
	if metrics.MemoryContractScoreAvg+1e-9 < 0.90 {
		t.Fatalf("memory_contract_score_avg=%.3f want >=0.90", metrics.MemoryContractScoreAvg)
	}
}

func computeAgentContractMetrics(t *testing.T, cases []ContractCase) AgentContractMetrics {
	var contractPassedCount float64
	var bundleScoreSum float64
	var memoryScoreSum float64
	var memoryScoreCount float64

	var flattenedTextOnlyPass float64
	var flattenedTextOnlyDenom float64

	var missingLifecycleRoleUnsafePass float64
	var missingLifecycleRoleDenom float64

	var missingScopeUnsafePass float64
	var missingScopeDenom float64

	var missingUseUnsafePass float64
	var missingUseDenom float64

	var missingMisuseUnsafePass float64
	var missingMisuseDenom float64

	var missingProvenanceUnsafePass float64
	var missingProvenanceDenom float64

	var missingQualityStateUnsafePass float64
	var missingQualityStateDenom float64

	var supersededAsCurrent float64
	var supersededAsCurrentDenom float64
	var refutedAsCurrent float64
	var refutedAsCurrentDenom float64
	var historicalAsCurrent float64
	var historicalAsCurrentDenom float64

	var wrongScopeUseRate float64
	var wrongScopeUseRateDenom float64
	var negativeScopeViolation float64
	var negativeScopeViolationDenom float64

	var parityMatch float64
	var parityDenom float64

	for _, c := range cases {
		bundle := buildBundleFromCase(c)
		mode := parseRecallMode(c.RecallMode)

		restEv := EvaluateBundleContract(bundle, mode, false)
		var mcpEv ContractEvaluation
		if c.Interface == "both" {
			mcpBundle := mcpRecallBundleFromRESTBundle(t, bundle)
			mcpEv = EvaluateBundleContract(mcpBundle, mode, false)
		} else {
			mcpEv = EvaluateBundleContract(bundle, mode, c.Interface == "mcp")
		}

		primaryEv := restEv
		if c.Interface == "mcp" {
			primaryEv = mcpEv
		}

		if primaryEv.ContractPassed {
			contractPassedCount++
		}

		bundleScoreSum += primaryEv.BundleContractScore
		for _, score := range primaryEv.MemoryContractScores {
			memoryScoreSum += score
			memoryScoreCount++
		}

		if c.Interface == "mcp" {
			flattenedTextOnlyDenom++
			if primaryEv.ContractPassed {
				flattenedTextOnlyPass++
			}
		}

		for _, it := range c.InputMemories {
			if it.LifecycleRole == "" {
				missingLifecycleRoleDenom++
				if primaryEv.ContractPassed {
					missingLifecycleRoleUnsafePass++
				}
			}
			if it.LifecycleRole == recall.LifecycleCurrentGuidance && it.Scope == "" {
				missingScopeDenom++
				if primaryEv.ContractPassed {
					missingScopeUnsafePass++
				}
			}
			if it.LifecycleRole == recall.LifecycleCurrentGuidance && it.UseInstruction == "" {
				missingUseDenom++
				if primaryEv.ContractPassed {
					missingUseUnsafePass++
				}
			}
			if it.LifecycleRole == recall.LifecycleHistoricalContext && it.MisuseWarning == "" {
				missingMisuseDenom++
				if primaryEv.ContractPassed {
					missingMisuseUnsafePass++
				}
			}
			if it.SourceType == "" || it.AuthorityBasis == "" {
				missingProvenanceDenom++
				if primaryEv.ContractPassed {
					missingProvenanceUnsafePass++
				}
			}
			if it.LifecycleRole == recall.LifecycleCurrentGuidance && it.QualityState == "" {
				missingQualityStateDenom++
				if primaryEv.ContractPassed {
					missingQualityStateUnsafePass++
				}
			}
		}

		// Discipline correctness rates.
		for _, it := range c.InputMemories {
			disc := DecideUseDiscipline(itToMemoryItem(it), c.TaskTags)

			score := primaryEv.MemoryContractScores[it.MemoryID]
			if it.LifecycleRole == recall.LifecycleCurrentGuidance && score < 1.0 {
				disc.Decision = "ignore"
				disc.Reason = unsafeOmissionReasonFromContract(primaryEv)
			}

			scopeMatch := false
			for _, tt := range c.TaskTags {
				if tt == it.Scope {
					scopeMatch = true
					break
				}
			}

			if disc.Decision == "use" {
				wrongScopeUseRateDenom++
				if !scopeMatch {
					wrongScopeUseRate++
				}

				negHit := false
				for _, ns := range it.NegativeScope {
					for _, tt := range c.TaskTags {
						if ns == tt {
							negHit = true
							break
						}
					}
				}
				if negHit {
					negativeScopeViolationDenom++
					negativeScopeViolation++
				}

				// Superseded/refuted/historical may never be used as current guidance.
				switch it.LifecycleRole {
				case recall.LifecycleSupersededContext:
					supersededAsCurrentDenom++
					supersededAsCurrent++
				case recall.LifecycleRefutedContext:
					refutedAsCurrentDenom++
					refutedAsCurrent++
				case recall.LifecycleHistoricalContext:
					historicalAsCurrentDenom++
					historicalAsCurrent++
				}
			}
		}

		if c.Interface == "both" {
			parityDenom++
			if restEv.ContractPassed == mcpEv.ContractPassed &&
				len(restEv.UnsafeOmissions) == len(mcpEv.UnsafeOmissions) {
				parityMatch++
			}
		}
	}

	total := float64(len(cases))
	return AgentContractMetrics{
		ContractPassRate:            contractPassedCount / total,
		BundleContractScore:         bundleScoreSum / total,
		MemoryContractScoreAvg:     memoryScoreSum / (memoryScoreCount + 1e-9),
		FlattenedTextOnlyRate:       safeDiv(flattenedTextOnlyPass, flattenedTextOnlyDenom),
		MissingLifecycleRoleRate:    safeDiv(missingLifecycleRoleUnsafePass, missingLifecycleRoleDenom),
		MissingScopeForGuidanceRate: safeDiv(missingScopeUnsafePass, missingScopeDenom),
		MissingUseInstructionRate:   safeDiv(missingUseUnsafePass, missingUseDenom),
		MissingMisuseWarningRate:    safeDiv(missingMisuseUnsafePass, missingMisuseDenom),
		MissingProvenanceRate:      safeDiv(missingProvenanceUnsafePass, missingProvenanceDenom),
		MissingQualityStateRate:   safeDiv(missingQualityStateUnsafePass, missingQualityStateDenom),
		SupersededAsCurrentRate:    safeDiv(supersededAsCurrent, supersededAsCurrentDenom),
		RefutedAsCurrentRate:       safeDiv(refutedAsCurrent, refutedAsCurrentDenom),
		HistoricalAsCurrentRate:    safeDiv(historicalAsCurrent, historicalAsCurrentDenom),
		WrongScopeUseRate:          safeDiv(wrongScopeUseRate, wrongScopeUseRateDenom),
		NegativeScopeViolationRate: safeDiv(negativeScopeViolation, negativeScopeViolationDenom),
		MCPRESTContractParityRate:  safeDiv(parityMatch, parityDenom),
	}
}

