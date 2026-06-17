package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SuiteSummary aggregates benchmark run metrics.
type SuiteSummary struct {
	GeneratedAt        time.Time          `json:"generated_at"`
	GitCommit          string             `json:"git_commit,omitempty"`
	CaseCount          int                `json:"case_count"`
	PassedCount        int                `json:"passed_count"`
	FailedCount        int                `json:"failed_count"`
	OverallRecallAtK   float64            `json:"overall_recall_at_k"`
	OverallPrecisionAtK float64           `json:"overall_precision_at_k"`
	ForbiddenHitRate   float64            `json:"forbidden_hit_rate"`
	DomainConfusion    map[string]map[string]int `json:"domain_confusion"`
	SemanticPath       string             `json:"semantic_retrieval_path"`
	RetrievalMode            string  `json:"retrieval_mode,omitempty"`
	LifecycleViolationRate   float64 `json:"lifecycle_violation_rate"`
	DateBoundViolationRate   float64 `json:"date_bound_violation_rate"`
	UtilityViolationRate     float64 `json:"utility_violation_rate"`
	ModeViolationRate        float64 `json:"mode_violation_rate"`
	EmbedderAvailable        bool    `json:"embedder_available"`
	Cases                    []CaseResult `json:"cases"`
}

// Summarize builds suite-level metrics from case results.
func Summarize(results []CaseResult) SuiteSummary {
	sum := SuiteSummary{
		GeneratedAt: time.Now().UTC(),
		GitCommit:   gitCommit(),
		CaseCount:   len(results),
		DomainConfusion: DomainConfusionMatrix(results),
	}
	var recallSum, precSum, forbSum, lifeSum, dateSum, utilSum, modeSum float64
	for _, r := range results {
		if r.Passed {
			sum.PassedCount++
		} else {
			sum.FailedCount++
		}
		recallSum += r.Metrics.RecallAtK
		precSum += r.Metrics.PrecisionAtK
		forbSum += r.Metrics.ForbiddenHitRate
		v := r.Metrics.Violations
		lifeSum += v.LifecycleViolationRate
		dateSum += v.DateBoundViolationRate
		utilSum += v.UtilityViolationRate
		modeSum += v.ModeViolationRate
		if r.Semantic != nil && sum.SemanticPath == "" {
			sum.SemanticPath = r.Semantic.Path
			sum.EmbedderAvailable = r.Semantic.EmbedderAvailable
			if r.Semantic.Path == "semantic_hybrid" {
				sum.RetrievalMode = RetrievalModeHybrid
			} else {
				sum.RetrievalMode = RetrievalModeLexical
			}
		}
	}
	if len(results) > 0 {
		n := float64(len(results))
		sum.OverallRecallAtK = recallSum / n
		sum.OverallPrecisionAtK = precSum / n
		sum.ForbiddenHitRate = forbSum / n
		sum.LifecycleViolationRate = lifeSum / n
		sum.DateBoundViolationRate = dateSum / n
		sum.UtilityViolationRate = utilSum / n
		sum.ModeViolationRate = modeSum / n
	}
	if sum.SemanticPath == "" {
		sum.SemanticPath = "lexical_only"
	}
	if sum.RetrievalMode == "" {
		sum.RetrievalMode = RetrievalModeLexical
	}
	sum.Cases = results
	return sum
}

func gitCommit() string {
	b, err := os.ReadFile(filepath.Join(findRepoRoot(), ".git", "HEAD"))
	if err != nil {
		return ""
	}
	ref := strings.TrimSpace(string(b))
	if strings.HasPrefix(ref, "ref: ") {
		refPath := filepath.Join(findRepoRoot(), ".git", strings.TrimPrefix(ref, "ref: "))
		b2, err := os.ReadFile(strings.TrimSpace(refPath))
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(b2))[:min(12, len(strings.TrimSpace(string(b2))))]
	}
	if len(ref) >= 12 {
		return ref[:12]
	}
	return ref
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// WriteBaselineJSON writes machine-readable baseline to path.
func WriteBaselineJSON(path string, sum SuiteSummary) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(sum, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// LoadBaselineJSON reads a prior baseline for regression compare.
func LoadBaselineJSON(path string) (*SuiteSummary, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sum SuiteSummary
	if err := json.Unmarshal(b, &sum); err != nil {
		return nil, err
	}
	return &sum, nil
}

// FormatCaseFailure returns human-readable failure block for one case.
func FormatCaseFailure(r CaseResult) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("CASE FAILED: %s\n\n", r.Case.ID))
	b.WriteString(fmt.Sprintf("Query:\n  %s\n\n", r.Case.Query))
	if len(r.Metrics.ExpectedMissing) > 0 {
		b.WriteString("Expected labels missing:\n")
		for _, lbl := range r.Metrics.ExpectedMissing {
			b.WriteString(fmt.Sprintf("  - %s\n", lbl))
		}
		b.WriteString("\n")
	}
	if len(r.Metrics.ForbiddenReturned) > 0 {
		b.WriteString("Forbidden labels returned:\n")
		for _, lbl := range r.Metrics.ForbiddenReturned {
			for _, h := range r.Hits {
				if h.Label == lbl {
					b.WriteString(fmt.Sprintf("  - %s at rank %d\n", lbl, h.Rank))
					break
				}
			}
		}
		b.WriteString("\n")
	}
	b.WriteString(fmt.Sprintf("Top %d returned:\n", len(r.Hits)))
	for _, h := range r.Hits {
		flag := ""
		if h.Forbidden {
			flag = " FORBIDDEN"
		}
		b.WriteString(fmt.Sprintf("  %d. %s score=%.2f kind=%s domain=%s%s\n",
			h.Rank, h.Label, h.Score, h.Kind, h.Domain, flag))
	}
	b.WriteString(fmt.Sprintf("\nMetrics:\n  recall_at_%d=%.2f required=%.2f\n  precision_at_%d=%.2f required=%.2f\n  forbidden_hit_count=%d max=%d\n",
		r.Case.K, r.Metrics.RecallAtK, r.Case.MinimumRecallAtK,
		r.Case.K, r.Metrics.PrecisionAtK, r.Case.MinimumPrecisionAtK,
		r.Metrics.ForbiddenHitCount, r.Case.MaximumForbiddenHits))
	if r.FailReason != "" {
		b.WriteString(fmt.Sprintf("  reason: %s\n", r.FailReason))
	}
	return b.String()
}

// WriteMarkdownReport writes phase3-style markdown report.
func WriteMarkdownReport(path string, sum SuiteSummary, judgment string) error {
	var b strings.Builder
	b.WriteString("# Phase 3 Recall Benchmark Report\n\n")
	b.WriteString("## Final Judgment\n\n")
	b.WriteString(judgment + "\n\n")
	b.WriteString("## Summary\n\n")
	b.WriteString(fmt.Sprintf("- Cases: %d (passed %d, failed %d)\n", sum.CaseCount, sum.PassedCount, sum.FailedCount))
	b.WriteString(fmt.Sprintf("- Overall Recall@K: %.2f\n", sum.OverallRecallAtK))
	b.WriteString(fmt.Sprintf("- Overall Precision@K: %.2f\n", sum.OverallPrecisionAtK))
	b.WriteString(fmt.Sprintf("- Forbidden-hit rate (avg): %.2f\n", sum.ForbiddenHitRate))
	b.WriteString(fmt.Sprintf("- semantic retrieval enabled: false (benchmark mode)\n"))
	b.WriteString(fmt.Sprintf("- embedder available: %v\n", sum.EmbedderAvailable))
	b.WriteString(fmt.Sprintf("- retrieval mode: %s\n\n", sum.SemanticPath))
	b.WriteString("## Domain Confusion Matrix\n\n")
	b.WriteString(formatMatrix(sum.DomainConfusion))
	b.WriteString("\n## Worst Recall Failures\n\n")
	for _, r := range worstCases(sum.Cases, 5) {
		b.WriteString("```text\n")
		b.WriteString(FormatCaseFailure(r))
		b.WriteString("```\n\n")
	}
	b.WriteString("## Proof Commands\n\n")
	b.WriteString("```bash\nmake test-recall-benchmark\nmake recall-benchmark-baseline\nmake recall-benchmark-report\n```\n")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func formatMatrix(m map[string]map[string]int) string {
	domains := map[string]bool{}
	for exp, rets := range m {
		domains[exp] = true
		for d := range rets {
			domains[d] = true
		}
	}
	order := []string{"pluribus", "dornan_pro", "onguard", "nginx_apache", "homelab", "comfyui", "fitness", "finance", "generic_noise", "unknown"}
	var cols []string
	for _, d := range order {
		if domains[d] {
			cols = append(cols, d)
		}
	}
	var b strings.Builder
	b.WriteString("| Expected \\\\ Returned | " + strings.Join(cols, " | ") + " |\n")
	b.WriteString("|" + strings.Repeat("---|", len(cols)+1) + "\n")
	for _, exp := range order {
		if !domains[exp] {
			continue
		}
		row := []string{exp}
		for _, col := range cols {
			v := 0
			if m[exp] != nil {
				v = m[exp][col]
			}
			row = append(row, fmt.Sprintf("%d", v))
		}
		b.WriteString("| " + strings.Join(row, " | ") + " |\n")
	}
	return b.String()
}

func worstCases(cases []CaseResult, n int) []CaseResult {
	var failed []CaseResult
	for _, c := range cases {
		if !c.Passed {
			failed = append(failed, c)
		}
	}
	if len(failed) > n {
		return failed[:n]
	}
	return failed
}
