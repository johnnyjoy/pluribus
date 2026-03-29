package runmulti

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RunMultiInput is the input for a multi-run execution.
type RunMultiInput struct {
	Variants  int
	Strategy  string
	Prompt    string
	Tags      []string
	Symbols   []string

	// RIE / compile-multi
	MaxPerKind int
	MaxTotal   int
	MaxTokens  int

	// Explicit slow-path for compile-multi. When set with RecommendedExpansion, takes precedence over preflight-derived slow-path.
	SlowPathRequired     bool
	SlowPathReasons      []string
	RecommendedExpansion *RecommendedExpansionMirror

	// PreflightChangedFiles when non-nil triggers POST /v1/recall/preflight; slow-path from response is applied only if SlowPathRequired is not already true on input.
	PreflightChangedFiles *int
	// CompileChangedFilesCount when non-nil is sent as compile-multi changed_files_count (server infers slow-path). Ignored on JSON when PreflightChangedFiles is set (preflight supplies slow-path).
	CompileChangedFilesCount *int

	// forwarded to compile-multi JSON for LSP auto symbols / reference_count.
	RepoRoot       string
	LSPFocusPath   string
	LSPFocusLine   int
	LSPFocusColumn int
	// RetrievalQuery optional; set by server run-multi when enable_triggered_recall enriches compile-multi.
	RetrievalQuery string
}

// Runner runs the multi-context loop (compile-multi → LLM per variant → drift → score → select).
type Runner struct {
	BaseURL    string
	HTTPClient *http.Client
	LLM        LLMCaller
}

// NewRunner creates a runner with the given base URL and LLM caller.
func NewRunner(baseURL string, llm LLMCaller) *Runner {
	baseURL = strings.TrimSuffix(baseURL, "/")
	if llm == nil {
		llm = &nopLLM{}
	}
	return &Runner{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 2 * time.Minute},
		LLM:        llm,
	}
}

type nopLLM struct{}

func (nopLLM) Generate(context.Context, string) (string, error) {
	return "", fmt.Errorf("runmulti: backend synthesis is not configured (set synthesis.enabled and provider in config, or use client-side synthesis)")
}

// Run executes the full loop and returns runs + selected.
func (r *Runner) Run(ctx context.Context, input RunMultiInput) (*RunMultiResult, error) {
	if input.Variants <= 0 {
		input.Variants = 3
	}
	if input.Strategy == "" {
		input.Strategy = "default"
	}

	compileReq := r.buildCompileMultiRequest(input)

	if input.PreflightChangedFiles != nil {
		pfReq := PreflightRequestMirror{
			ChangedFilesCount: *input.PreflightChangedFiles,
			Tags:              input.Tags,
		}
		pf, err := PostPreflight(ctx, r.BaseURL, pfReq, r.HTTPClient)
		if err != nil {
			return nil, fmt.Errorf("preflight: %w", err)
		}
		if !compileReq.SlowPathRequired && pf.SlowPathRequired {
			compileReq.SlowPathRequired = true
			compileReq.SlowPathReasons = append([]string(nil), pf.SlowPathReasons...)
			if pf.RecommendedExpansion != nil {
				compileReq.RecommendedExpansion = pf.RecommendedExpansion
			}
		}
	}

	slowPathForDrift := compileReq.SlowPathRequired

	compileResp, err := r.callCompileMulti(ctx, compileReq)
	if err != nil {
		return nil, fmt.Errorf("compile-multi: %w", err)
	}

	// 2. For each bundle: build context, LLM, drift, score (parallel)
	runs := make([]RunResult, len(compileResp.Bundles))
	var wg sync.WaitGroup
	for i := range compileResp.Bundles {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			runs[i] = r.runOne(ctx, compileResp.Bundles[i], input.Prompt, input.Tags, slowPathForDrift)
		}()
	}
	wg.Wait()

	// 3. Select best
	selected := SelectBest(runs)
	out := &RunMultiResult{Runs: runs, Selected: selected}
	return out, nil
}

func (r *Runner) buildCompileMultiRequest(input RunMultiInput) CompileMultiRequest {
	req := CompileMultiRequest{
		Tags:       input.Tags,
		Symbols:    input.Symbols,
		Variants:   input.Variants,
		Strategy:   input.Strategy,
		MaxPerKind: input.MaxPerKind,
		MaxTotal:   input.MaxTotal,
		MaxTokens:  input.MaxTokens,
	}
	if input.SlowPathRequired {
		req.SlowPathRequired = true
		req.SlowPathReasons = append([]string(nil), input.SlowPathReasons...)
		req.RecommendedExpansion = input.RecommendedExpansion // may be nil; recall compile still forwards for drift policy
	}
	if input.PreflightChangedFiles == nil {
		req.ChangedFilesCount = input.CompileChangedFilesCount
	}
	req.RepoRoot = input.RepoRoot
	req.LSPFocusPath = input.LSPFocusPath
	req.LSPFocusLine = input.LSPFocusLine
	req.LSPFocusColumn = input.LSPFocusColumn
	req.RetrievalQuery = input.RetrievalQuery
	return req
}

func (r *Runner) runOne(ctx context.Context, vb VariantBundleMirror, userPrompt string, tags []string, slowPathForDrift bool) RunResult {
	res := RunResult{Variant: vb.Variant}

	fullPrompt := BuildContext(&vb.Bundle, vb.Variant, userPrompt)
	output, err := r.LLM.Generate(ctx, fullPrompt)
	if err != nil {
		res.Output = ""
		res.Drift = DriftResult{}
		res.Score = 999
		res.Rejected = true
		return res
	}
	res.Output = output

	driftResp, err := PostDriftCheckSlowPathOptional(ctx, r.BaseURL, output, r.HTTPClient, tags, slowPathForDrift)
	if err != nil {
		res.Drift = DriftResult{}
		res.Score = 999
		res.Rejected = true
		return res
	}
	res.Drift = driftResp
	res.Score, res.Rejected = ScoreRun(&driftResp)
	return res
}

func (r *Runner) callCompileMulti(ctx context.Context, req CompileMultiRequest) (*CompileMultiResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, r.BaseURL+"/v1/recall/compile-multi", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := r.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	slurp, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", resp.Status, string(slurp))
	}
	var out CompileMultiResponse
	if err := json.Unmarshal(slurp, &out); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &out, nil
}

