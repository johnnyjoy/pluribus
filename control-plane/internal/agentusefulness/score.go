package agentusefulness

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// ScoreRun evaluates deterministic rubric for one run.
func ScoreRun(t TaskFixture, mode string, facts []string, manifest *RecallManifest, trace MemoryUseTrace) (CaseScore, []string) {
	var failures []string
	score := CaseScore{
		ExpectedFactsPresent:        hasAllFacts(facts, t.ExpectedAnswerFacts),
		ForbiddenFactsAbsent:        !hasAnyFact(facts, t.ForbiddenAnswerFacts),
		ExpectedMemoriesRecalled:    true,
		ExpectedMemoriesUsed:        true,
		ForbiddenMemoriesRecalled:   true,
		ForbiddenMemoriesUsed:       true,
		LifecycleRolesCorrect:       len(trace.MisusedLabels) == 0,
		DateBoundsRespected:         true,
		UtilityConstraintsRespected: len(trace.MisusedLabels) == 0,
	}

	if mode == RunModeNoMemory {
		score.AnswerPass = score.ExpectedFactsPresent && score.ForbiddenFactsAbsent
		if t.RequiresMemoryHelp && score.AnswerPass {
			failures = append(failures, "no-memory run unexpectedly passed but memory help required")
		}
		return score, failures
	}

	recalled := RecalledLabelSet(manifest)
	for _, lbl := range t.ExpectedRecalledLabels {
		if _, ok := recalled[lbl]; !ok {
			score.ExpectedMemoriesRecalled = false
			failures = append(failures, fmt.Sprintf("expected recalled memory missing: %s", lbl))
		}
	}
	for _, lbl := range t.ForbiddenRecalledLabels {
		if _, ok := recalled[lbl]; !ok {
			continue
		}
		score.ForbiddenMemoriesRecalled = false
		failures = append(failures, fmt.Sprintf("forbidden memory recalled: %s", lbl))
	}

	used := labelSet(trace.UsedLabels)
	for _, lbl := range t.ExpectedUsedLabels {
		if !used[lbl] {
			score.ExpectedMemoriesUsed = false
			failures = append(failures, fmt.Sprintf("expected used memory missing: %s", lbl))
		}
	}
	for _, lbl := range t.ForbiddenRecalledLabels {
		if used[lbl] {
			score.ForbiddenMemoriesUsed = false
			failures = append(failures, fmt.Sprintf("forbidden memory used: %s", lbl))
		}
	}

	if len(trace.MisusedLabels) > 0 {
		score.MemoryMisused = true
		score.LifecycleRolesCorrect = false
		for _, lbl := range trace.MisusedLabels {
			failures = append(failures, fmt.Sprintf("misused memory: %s", lbl))
		}
	}

	score.AnswerPass = score.ExpectedFactsPresent && score.ForbiddenFactsAbsent &&
		score.ExpectedMemoriesRecalled && score.ExpectedMemoriesUsed &&
		score.ForbiddenMemoriesRecalled && score.ForbiddenMemoriesUsed &&
		score.LifecycleRolesCorrect

	if t.RequiresMemoryHelp {
		score.MemoryHelped = score.AnswerPass && score.ExpectedMemoriesUsed && score.ExpectedMemoriesRecalled &&
			!score.MemoryMisused && score.ForbiddenMemoriesUsed && len(trace.UsedLabels) > 0
		if score.MemoryHelped && len(trace.UsedLabels) == 0 {
			score.MemoryHelped = false
			failures = append(failures, "memory marked helpful without used memories")
		}
		if len(trace.UsedLabels) > 0 && !score.AnswerPass {
			score.MemoryHurt = hasAnyFact(facts, t.ForbiddenAnswerFacts) || score.MemoryMisused
		}
		for _, lbl := range t.ExpectedIgnoredLabels {
			if used[lbl] {
				score.MemoryIrrelevant = true
			}
		}
	}

	return score, failures
}

// BuildOutcomeFeedback emits outcome-linked feedback artifacts for a run.
func BuildOutcomeFeedback(t TaskFixture, run RunResult, lc *LoadedCorpus) []OutcomeFeedbackEvent {
	var out []OutcomeFeedbackEvent
	base := OutcomeFeedbackEvent{
		TaskID:      t.ID,
		RunID:       run.RunID,
		Interface:   run.Interface,
		Mode:        run.Mode,
		AnswerFacts: append([]string(nil), run.AnswerFacts...),
		ScorePass:   run.Score.AnswerPass,
	}

	if run.Mode == RunModeNoMemory {
		ev := base
		ev.OutcomeLabel = "baseline_no_memory"
		if run.Score.AnswerPass {
			ev.Reason = "no-memory baseline passed"
		} else {
			ev.Reason = "no-memory baseline incomplete"
		}
		out = append(out, ev)
		return out
	}

	for _, lbl := range run.UseTrace.UsedLabels {
		ev := base
		ev.MemoryLabel = lbl
		ev.MemoryID = lc.IDForLabel(lbl)
		ev.Used = true
		switch {
		case run.Score.MemoryHurt:
			ev.OutcomeLabel = "hurt"
			ev.Reason = "used memory correlated with failed or forbidden output"
		case run.Score.MemoryHelped && t.RequiresMemoryHelp:
			ev.OutcomeLabel = "helped"
			ev.Reason = "recalled, used, output passed rubric vs no-memory baseline"
		case run.Score.AnswerPass:
			ev.OutcomeLabel = "used_pass"
			ev.Reason = "used and output passed"
		default:
			ev.OutcomeLabel = "used_fail"
			ev.Reason = "used but output failed rubric"
		}
		out = append(out, ev)
	}

	for _, lbl := range run.UseTrace.IgnoredLabels {
		if !containsLabel(t.ExpectedIgnoredLabels, lbl) {
			continue
		}
		ev := base
		ev.MemoryLabel = lbl
		ev.MemoryID = lc.IDForLabel(lbl)
		ev.Used = false
		ev.OutcomeLabel = "irrelevant"
		ev.Reason = run.UseTrace.IgnoreReasons[lbl]
		out = append(out, ev)
	}

	for _, lbl := range run.UseTrace.MisusedLabels {
		ev := base
		ev.MemoryLabel = lbl
		ev.MemoryID = lc.IDForLabel(lbl)
		ev.Used = true
		ev.OutcomeLabel = "misused"
		ev.Reason = run.UseTrace.IgnoreReasons[lbl]
		out = append(out, ev)
	}

	return out
}

// ComputeSuiteMetrics aggregates task results.
func ComputeSuiteMetrics(tasks []TaskResult) SuiteMetrics {
	m := SuiteMetrics{TotalTasks: len(tasks)}
	if len(tasks) == 0 {
		return m
	}

	var helpEligible, helped, harmed, misused, irrelevantCases, parityReq, parityPass, improved int
	var forbiddenUse, lifecycleMisuse int

	for _, tr := range tasks {
		if tr.RequiresMemoryHelp {
			helpEligible++
			if tr.MemoryHelped {
				helped++
			}
			if !tr.NoMemory.Score.AnswerPass && tr.MemoryREST.Score.AnswerPass {
				improved++
			}
		}
		if tr.MemoryREST.Score.MemoryHurt || tr.MemoryMCP.Score.MemoryHurt {
			harmed++
		}
		if tr.MemoryREST.Score.MemoryMisused || tr.MemoryMCP.Score.MemoryMisused {
			misused++
			lifecycleMisuse++
		}
		if tr.MemoryREST.Score.MemoryIrrelevant {
			irrelevantCases++
		}
		if !tr.MemoryREST.Score.ForbiddenMemoriesUsed || !tr.MemoryMCP.Score.ForbiddenMemoriesUsed {
			forbiddenUse++
		}
		if tr.ParityRequired {
			parityReq++
			if tr.MCPRESTParity {
				parityPass++
			}
		}
	}

	m.HelpEligibleTasks = helpEligible
	m.HelpedTasks = helped
	m.HarmedTasks = harmed
	m.MisusedTasks = misused
	m.ParityRequiredTasks = parityReq
	m.ParityPassedTasks = parityPass

	if helpEligible > 0 {
		m.MemoryHelpRate = float64(helped) / float64(helpEligible)
		m.AnswerImprovementRate = float64(improved) / float64(helpEligible)
	}
	n := float64(len(tasks))
	m.MemoryHarmRate = float64(harmed) / n
	m.MemoryMisuseRate = float64(misused) / n
	m.MemoryIrrelevantRate = float64(irrelevantCases) / n
	m.ForbiddenMemoryUseRate = float64(forbiddenUse) / n
	m.LifecycleMisuseRate = float64(lifecycleMisuse) / n
	if parityReq > 0 {
		m.MCPRESTParityRate = float64(parityPass) / float64(parityReq)
	} else {
		m.MCPRESTParityRate = 1
	}
	return m
}

// EvaluateGate checks thresholds against metrics.
func EvaluateGate(m SuiteMetrics, th GateThresholds) (bool, []string) {
	var fails []string
	if m.HelpEligibleTasks > 0 && m.MemoryHelpRate < th.MemoryHelpRateMin {
		fails = append(fails, fmt.Sprintf("memory_help_rate=%.3f min=%.3f", m.MemoryHelpRate, th.MemoryHelpRateMin))
	}
	if m.MemoryHarmRate > th.MemoryHarmRateMax {
		fails = append(fails, fmt.Sprintf("memory_harm_rate=%.3f max=%.3f", m.MemoryHarmRate, th.MemoryHarmRateMax))
	}
	if m.MemoryMisuseRate > th.MemoryMisuseRateMax {
		fails = append(fails, fmt.Sprintf("memory_misuse_rate=%.3f max=%.3f", m.MemoryMisuseRate, th.MemoryMisuseRateMax))
	}
	if m.ForbiddenMemoryUseRate > th.ForbiddenMemoryUseRateMax {
		fails = append(fails, fmt.Sprintf("forbidden_memory_use_rate=%.3f max=%.3f", m.ForbiddenMemoryUseRate, th.ForbiddenMemoryUseRateMax))
	}
	if m.LifecycleMisuseRate > th.LifecycleMisuseRateMax {
		fails = append(fails, fmt.Sprintf("lifecycle_misuse_rate=%.3f max=%.3f", m.LifecycleMisuseRate, th.LifecycleMisuseRateMax))
	}
	if m.MCPRESTParityRate < th.MCPRESTParityRateMin {
		fails = append(fails, fmt.Sprintf("mcp_rest_parity_rate=%.3f min=%.3f", m.MCPRESTParityRate, th.MCPRESTParityRateMin))
	}
	sort.Strings(fails)
	return len(fails) == 0, fails
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
