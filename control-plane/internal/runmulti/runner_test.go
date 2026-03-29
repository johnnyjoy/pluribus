package runmulti

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// mockLLM returns output based on [VARIANT] in the prompt (for integration test).
type mockLLM struct {
	variantFromPrompt func(prompt string) string
}

func (m *mockLLM) Generate(ctx context.Context, prompt string) (string, error) {
	v := "unknown"
	if m.variantFromPrompt != nil {
		v = m.variantFromPrompt(prompt)
	}
	return "output-" + v, nil
}

func TestRunner_Run_integration_selection(t *testing.T) {
	// Mock compile-multi: return 3 bundles
	compileHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/recall/compile-multi" || r.Method != http.MethodPost {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		resp := CompileMultiResponse{
			Bundles: []VariantBundleMirror{
				{Variant: "balanced", Bundle: RecallBundleMirror{}},
				{Variant: "failure_heavy", Bundle: RecallBundleMirror{}},
				{Variant: "authority_heavy", Bundle: RecallBundleMirror{}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Mock drift: proposal "output-balanced" -> violations (rejected); "output-failure_heavy" -> score 0; "output-authority_heavy" -> score 1
	driftHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/drift/check" || r.Method != http.MethodPost {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		var req DriftCheckRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var result DriftResult
		switch {
		case strings.Contains(req.Proposal, "output-balanced"):
			result = DriftResult{Passed: false, Violations: []DriftIssue{{Code: "constraint", Statement: "x"}}, RiskLevel: "low"}
		case strings.Contains(req.Proposal, "output-failure_heavy"):
			result = DriftResult{Passed: true, Violations: nil, Warnings: []string{}, RiskLevel: "low"}
		case strings.Contains(req.Proposal, "output-authority_heavy"):
			result = DriftResult{Passed: true, Violations: nil, Warnings: []string{"w1"}, RiskLevel: "low"}
		default:
			result = DriftResult{Passed: true, RiskLevel: "low"}
		}
		_ = json.NewEncoder(w).Encode(result)
	})

	mux := http.NewServeMux()
	mux.Handle("/v1/recall/compile-multi", compileHandler)
	mux.Handle("/v1/drift/check", driftHandler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	llm := &mockLLM{
		variantFromPrompt: func(prompt string) string {
			if i := strings.Index(prompt, "[VARIANT]"); i >= 0 {
				rest := prompt[i+len("[VARIANT]"):]
				if j := strings.Index(rest, "["); j >= 0 {
					rest = rest[:j]
				}
				return strings.TrimSpace(rest)
			}
			return "unknown"
		},
	}

	runner := NewRunner(srv.URL, llm)
	result, err := runner.Run(context.Background(), RunMultiInput{
		Variants:  3,
		Strategy:  "default",
		Prompt:    "Do the thing.",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(result.Runs) != 3 {
		t.Fatalf("expected 3 runs, got %d", len(result.Runs))
	}

	// One run should be rejected (balanced with violations)
	var rejected int
	for _, r := range result.Runs {
		if r.Rejected {
			rejected++
		}
	}
	if rejected < 1 {
		t.Errorf("expected at least 1 rejected run, got %d", rejected)
	}

	// Selected should be the valid run with lowest score (failure_heavy with score 0)
	if result.Selected == nil {
		t.Fatal("expected non-nil selected")
	}
	if result.Selected.Rejected {
		t.Errorf("selected should not be rejected, got variant %s", result.Selected.Variant)
	}
	if result.Selected.Score != 0 {
		t.Errorf("selected score = %v, want 0 (failure_heavy)", result.Selected.Score)
	}
	if result.Selected.Variant != "failure_heavy" {
		t.Errorf("selected variant = %s, want failure_heavy", result.Selected.Variant)
	}
}

func TestRunner_Run_preflightAppliesSlowPathToCompileMulti(t *testing.T) {
	var gotCompile json.RawMessage
	compileHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/recall/compile-multi" || r.Method != http.MethodPost {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		b, _ := io.ReadAll(r.Body)
		gotCompile = json.RawMessage(b)
		resp := CompileMultiResponse{
			Bundles: []VariantBundleMirror{{Variant: "balanced", Bundle: RecallBundleMirror{}}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	preflightHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/recall/preflight" || r.Method != http.MethodPost {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		out := PreflightResultMirror{
			RiskLevel:         "high",
			RequiredActions:   []string{"deep_recall", "drift_check"},
			RiskScore:         1.0,
			SlowPathRequired:  true,
			SlowPathReasons:   []string{"risk score exceeds high_risk_threshold"},
			RecommendedExpansion: &RecommendedExpansionMirror{
				ConstraintsDelta: 4, FailuresDelta: 4, PatternsDelta: 2,
			},
		}
		_ = json.NewEncoder(w).Encode(out)
	})

	driftHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/drift/check" || r.Method != http.MethodPost {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(DriftResult{Passed: true, RiskLevel: "low"})
	})

	mux := http.NewServeMux()
	mux.Handle("/v1/recall/compile-multi", compileHandler)
	mux.Handle("/v1/recall/preflight", preflightHandler)
	mux.Handle("/v1/drift/check", driftHandler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	llm := &mockLLM{variantFromPrompt: func(string) string { return "balanced" }}
	runner := NewRunner(srv.URL, llm)
	n := 11
	_, err := runner.Run(context.Background(), RunMultiInput{
		Variants:              1,
		Strategy:              "default",
		Prompt:                "x",
		PreflightChangedFiles: &n,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	var decoded CompileMultiRequest
	if err := json.Unmarshal(gotCompile, &decoded); err != nil {
		t.Fatalf("compile-multi body: %v", err)
	}
	if !decoded.SlowPathRequired {
		t.Error("compile-multi: want slow_path_required true from preflight")
	}
	if decoded.RecommendedExpansion == nil || decoded.RecommendedExpansion.ConstraintsDelta != 4 {
		t.Errorf("compile-multi recommended_expansion = %+v", decoded.RecommendedExpansion)
	}
}

func TestRunner_Run_driftSlowPathRequiresSecondCheck(t *testing.T) {
	var driftMu sync.Mutex
	driftCallsByProposal := make(map[string]int)

	compileHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := CompileMultiResponse{
			Bundles: []VariantBundleMirror{{Variant: "balanced", Bundle: RecallBundleMirror{}}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	driftHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req DriftCheckRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		driftMu.Lock()
		driftCallsByProposal[req.Proposal]++
		n := driftCallsByProposal[req.Proposal]
		driftMu.Unlock()

		var res DriftResult
		switch {
		case req.IsFollowupCheck:
			res = DriftResult{Passed: true, RiskLevel: "low", Violations: nil}
		case req.SlowPathRequired && n == 1:
			res = DriftResult{
				Passed: true, RiskLevel: "low",
				RequiresFollowupCheck: true,
				FollowupReason:        "slow-path requires second drift check before execution",
			}
		default:
			res = DriftResult{Passed: true, RiskLevel: "low"}
		}
		_ = json.NewEncoder(w).Encode(res)
	})

	mux := http.NewServeMux()
	mux.Handle("/v1/recall/compile-multi", compileHandler)
	mux.Handle("/v1/drift/check", driftHandler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	llm := &mockLLM{variantFromPrompt: func(string) string { return "balanced" }}
	runner := NewRunner(srv.URL, llm)

	_, err := runner.Run(context.Background(), RunMultiInput{
		Variants:  1,
		Strategy:  "default",
		Prompt:    "x",
		SlowPathRequired: true,
		SlowPathReasons:  []string{"explicit"},
		RecommendedExpansion: &RecommendedExpansionMirror{ConstraintsDelta: 1, FailuresDelta: 1, PatternsDelta: 1},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	proposal := "output-balanced"
	driftMu.Lock()
	c := driftCallsByProposal[proposal]
	driftMu.Unlock()
	if c != 2 {
		t.Errorf("drift calls for %q = %d, want 2 (follow-up)", proposal, c)
	}
}

func TestRunner_Run_explicitSlowPathOverridesPreflight(t *testing.T) {
	var gotCompile json.RawMessage
	compileHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotCompile = json.RawMessage(b)
		_ = json.NewEncoder(w).Encode(CompileMultiResponse{
			Bundles: []VariantBundleMirror{{Variant: "balanced", Bundle: RecallBundleMirror{}}},
		})
	})

	preflightHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out := PreflightResultMirror{
			RiskLevel:        "high",
			SlowPathRequired: true,
			RecommendedExpansion: &RecommendedExpansionMirror{
				ConstraintsDelta: 99, FailuresDelta: 99, PatternsDelta: 99,
			},
		}
		_ = json.NewEncoder(w).Encode(out)
	})

	driftHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(DriftResult{Passed: true, RiskLevel: "low"})
	})

	mux := http.NewServeMux()
	mux.Handle("/v1/recall/compile-multi", compileHandler)
	mux.Handle("/v1/recall/preflight", preflightHandler)
	mux.Handle("/v1/drift/check", driftHandler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	llm := &mockLLM{variantFromPrompt: func(string) string { return "balanced" }}
	runner := NewRunner(srv.URL, llm)
	n := 11
	_, err := runner.Run(context.Background(), RunMultiInput{
		Variants:              1,
		Strategy:              "default",
		Prompt:                "x",
		PreflightChangedFiles: &n,
		SlowPathRequired:      true,
		SlowPathReasons:       []string{"client"},
		RecommendedExpansion:  &RecommendedExpansionMirror{ConstraintsDelta: 2, FailuresDelta: 2, PatternsDelta: 1},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	var decoded CompileMultiRequest
	if err := json.Unmarshal(gotCompile, &decoded); err != nil {
		t.Fatalf("decode compile body: %v", err)
	}
	if decoded.RecommendedExpansion == nil || decoded.RecommendedExpansion.ConstraintsDelta != 2 {
		t.Errorf("want explicit expansion constraints_delta=2, got %+v", decoded.RecommendedExpansion)
	}
}

func TestRunner_Run_compileBodyIncludesChangedFilesCompilePath(t *testing.T) {
	var gotBody []byte
	compileHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = b
		_ = json.NewEncoder(w).Encode(CompileMultiResponse{
			Bundles: []VariantBundleMirror{{Variant: "balanced", Bundle: RecallBundleMirror{}}},
		})
	})
	driftHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(DriftResult{Passed: true, RiskLevel: "low"})
	})
	mux := http.NewServeMux()
	mux.Handle("/v1/recall/compile-multi", compileHandler)
	mux.Handle("/v1/drift/check", driftHandler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	n := 8
	runner := NewRunner(srv.URL, &mockLLM{variantFromPrompt: func(string) string { return "balanced" }})
	_, err := runner.Run(context.Background(), RunMultiInput{
		Variants:               1,
		Strategy:               "default",
		Prompt:                 "x",
		CompileChangedFilesCount: &n,
	})
	if err != nil {
		t.Fatal(err)
	}
	var dec CompileMultiRequest
	if err := json.Unmarshal(gotBody, &dec); err != nil {
		t.Fatal(err)
	}
	if dec.ChangedFilesCount == nil || *dec.ChangedFilesCount != 8 {
		t.Fatalf("compile-multi changed_files_count = %v, want 8", dec.ChangedFilesCount)
	}
}

func TestRunner_Run_compileMultiIncludesLSPRecallFields(t *testing.T) {
	var gotBody []byte
	compileHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = b
		_ = json.NewEncoder(w).Encode(CompileMultiResponse{
			Bundles: []VariantBundleMirror{{Variant: "balanced", Bundle: RecallBundleMirror{}}},
		})
	})
	driftHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(DriftResult{Passed: true, RiskLevel: "low"})
	})
	mux := http.NewServeMux()
	mux.Handle("/v1/recall/compile-multi", compileHandler)
	mux.Handle("/v1/drift/check", driftHandler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	runner := NewRunner(srv.URL, &mockLLM{variantFromPrompt: func(string) string { return "balanced" }})
	_, err := runner.Run(context.Background(), RunMultiInput{
		Variants:       1,
		Strategy:       "default",
		Prompt:         "x",
		RepoRoot:       "/repo",
		LSPFocusPath:   "pkg/a.go",
		LSPFocusLine:   3,
		LSPFocusColumn: 7,
	})
	if err != nil {
		t.Fatal(err)
	}
	var dec CompileMultiRequest
	if err := json.Unmarshal(gotBody, &dec); err != nil {
		t.Fatal(err)
	}
	if dec.RepoRoot != "/repo" || dec.LSPFocusPath != "pkg/a.go" || dec.LSPFocusLine != 3 || dec.LSPFocusColumn != 7 {
		t.Fatalf("compile-multi LSP fields: %+v", dec)
	}
}
