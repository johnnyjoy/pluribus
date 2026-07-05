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

// SuiteHonestyNotes are limitations that apply to every integration proof run regardless of scenario PASS.
var SuiteHonestyNotes = []string{
	"HTTP REST clients only — not MCP tool invocation, agent_id attribution, resolve_chore, or memory_feedback unless a scenario explicitly says otherwise.",
	"Local CI uses proof-friendly formation defaults (seeded memories land active); deployed receipts use the server's real formation policy — PASS in one environment does not guarantee the other.",
	"Substring/kind assertions are necessary but not sufficient for ranking quality, latency SLOs, or long-pool noise resistance.",
	"Green receipts do not prove agents will follow integration skills; static grep checks in verify-housekeeping-enforcement.sh are not behavioral proof.",
}

// CollectHonestyReport builds a markdown section documenting known proof gaps for a run.
func CollectHonestyReport(scenarios []Scenario, rows []ResultRow) string {
	byID := make(map[string]Scenario, len(scenarios))
	for _, sc := range scenarios {
		byID[sc.ID] = sc
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## Honesty — what PASS does not mean\n\n")
	fmt.Fprintf(&b, "### Suite-wide (always)\n\n")
	for _, note := range SuiteHonestyNotes {
		fmt.Fprintf(&b, "- %s\n", note)
	}
	fmt.Fprintf(&b, "\n### Per scenario\n\n")
	for _, r := range rows {
		sc, ok := byID[r.ScenarioID]
		if !ok || len(sc.DoesNotProve) == 0 {
			continue
		}
		res := "FAIL"
		if r.Pass {
			res = "PASS"
		}
		fmt.Fprintf(&b, "#### `%s` (%s)\n\n", r.ScenarioID, res)
		fmt.Fprintf(&b, "_Claim:_ %s\n\n", sc.BenefitClaim)
		for _, line := range sc.DoesNotProve {
			fmt.Fprintf(&b, "- Does **not** prove: %s\n", line)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// WriteMarkdownSummary writes a simple pass/fail table (optional; set RECALL_PROOF_RESULTS_OUT).
func WriteMarkdownSummary(path, environment string, rows []ResultRow, scenarios []Scenario) error {
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
	if len(scenarios) > 0 {
		b.WriteString("\n")
		b.WriteString(CollectHonestyReport(scenarios, rows))
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
