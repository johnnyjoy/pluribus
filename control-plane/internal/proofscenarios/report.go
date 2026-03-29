package proofscenarios

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// ResultRow is one line for proof-scenario-results-latest.md.
type ResultRow struct {
	ScenarioID string
	Pass       bool
	Notes      string
	Duration   time.Duration
}

// WriteMarkdownSummary writes a simple pass/fail table (optional; set RECALL_PROOF_RESULTS_OUT).
func WriteMarkdownSummary(path, environment string, rows []ResultRow) error {
	if path == "" {
		return nil
	}
	var b strings.Builder
	now := time.Now().UTC().Format(time.RFC3339)
	fmt.Fprintf(&b, "# Proof scenario run\n\n")
	fmt.Fprintf(&b, "- **Date (UTC):** %s\n", now)
	fmt.Fprintf(&b, "- **Environment:** %s\n", environment)
	fmt.Fprintf(&b, "\n| Scenario | Result | Duration | Notes |\n")
	fmt.Fprintf(&b, "|----------|--------|----------|-------|\n")
	for _, r := range rows {
		res := "FAIL"
		if r.Pass {
			res = "PASS"
		}
		n := strings.ReplaceAll(r.Notes, "|", "\\|")
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", r.ScenarioID, res, r.Duration.Round(time.Millisecond), n)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
