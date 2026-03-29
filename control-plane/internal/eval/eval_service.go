package eval

import (
	"strings"

	"control-plane/internal/memory"
	"control-plane/internal/recall"
)

// evalTriggerConfig returns measurement-mode triggered recall: enabled with the same per-kind defaults as
// DefaultTriggerRecallConfig (risk/decision/similarity on). Only Enabled was missing when we passed {Enabled: true}
// alone — zero-value bools disabled every kind in filterTriggersByConfig.
func evalTriggerConfig() *recall.TriggerRecallConfig {
	d := recall.DefaultTriggerRecallConfig()
	d.Enabled = true
	return recall.NormalizeTriggerRecall(d)
}

// newEvalRecallService builds an in-process recall Service with deterministic memory search and no cache.
func newEvalRecallService(writes []memory.MemoryObject) *recall.Service {
	return &recall.Service{
		Compiler: &recall.Compiler{
			Memory:  &evalMemorySearcher{objs: writes},
			Ranking: ptrWeights(recall.DefaultRankingWeights()),
		},
		TriggerRecall: evalTriggerConfig(),
	}
}

// explicitSituationQuery is the explicit retrieval string for the "explicit recall" arm:
// use recall_expectations.query when set; otherwise join goal and context (deterministic, reproducible).
func explicitSituationQuery(s Scenario) string {
	if q := strings.TrimSpace(s.RecallExpectations.Query); q != "" {
		return q
	}
	var parts []string
	if g := strings.TrimSpace(s.Goal); g != "" {
		parts = append(parts, g)
	}
	if c := strings.TrimSpace(s.Context); c != "" {
		parts = append(parts, c)
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// buildCompileRequest builds the same CompileRequest for both explicit (Compile) and triggered (CompileTriggered) arms.
// ProposalText is the scenario trap (full text) so risk/decision heuristics see deploy/choice language consistently.
func buildCompileRequest(s Scenario) recall.CompileRequest {
	return recall.CompileRequest{
		MaxPerKind:     10,
		MaxTotal:       50,
		Mode:           "continuity",
		RetrievalQuery: explicitSituationQuery(s),
		ProposalText:   strings.TrimSpace(s.Trap),
	}
}
