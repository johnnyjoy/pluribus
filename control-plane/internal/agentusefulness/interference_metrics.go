package agentusefulness

import "fmt"

// CognitiveMetrics extends Phase 11B suite metrics with research-backed measures.
type CognitiveMetrics struct {
	SuiteMetrics

	EncodingCueMatchRate              float64 `json:"encoding_cue_match_rate"`
	UnderEncodedMemoryRate            float64 `json:"under_encoded_memory_rate"`
	SchemaApplicationSuccessRate      float64 `json:"schema_application_success_rate"`
	ContextViolationRate              float64 `json:"context_violation_rate"`
	InterferenceFailureRate           float64 `json:"interference_failure_rate"`
	NearMissSuppressionRate           float64 `json:"near_miss_suppression_rate"`
	WrongContextUseRate               float64 `json:"wrong_context_use_rate"`
	HistoricalAsCurrentMisuseRate     float64 `json:"historical_as_current_misuse_rate"`
	ExperienceReplayErrorRate         float64 `json:"experience_replay_error_rate"`
	RecalledIgnoredHelpFalsePositive  float64 `json:"recalled_ignored_help_false_positive_rate"`
	UsedNoImprovementHelpFalsePositive float64 `json:"used_no_improvement_help_false_positive_rate"`
	ResearchPrincipleCoverageRate     float64 `json:"research_principle_coverage_rate"`
}

// CognitiveGateThresholds adds hard-zero safety thresholds for Phase 11C.
type CognitiveGateThresholds struct {
	GateThresholds
	EncodingCueMatchRateMin              float64 `json:"encoding_cue_match_rate_min"`
	SchemaApplicationSuccessRateMin      float64 `json:"schema_application_success_rate_min"`
	NearMissSuppressionRateMin           float64 `json:"near_miss_suppression_rate_min"`
	ResearchPrincipleCoverageRateMin     float64 `json:"research_principle_coverage_rate_min"`
	ContextViolationRateMax              float64 `json:"context_violation_rate_max"`
	InterferenceFailureRateMax           float64 `json:"interference_failure_rate_max"`
	WrongContextUseRateMax               float64 `json:"wrong_context_use_rate_max"`
	HistoricalAsCurrentMisuseRateMax     float64 `json:"historical_as_current_misuse_rate_max"`
	ExperienceReplayErrorRateMax         float64 `json:"experience_replay_error_rate_max"`
	RecalledIgnoredHelpFalsePositiveMax  float64 `json:"recalled_ignored_help_false_positive_rate_max"`
	UsedNoImprovementHelpFalsePositiveMax float64 `json:"used_no_improvement_help_false_positive_rate_max"`
}

// DefaultCognitiveGateThresholds returns Phase 11C thresholds.
func DefaultCognitiveGateThresholds() CognitiveGateThresholds {
	return CognitiveGateThresholds{
		GateThresholds:                       DefaultGateThresholds(),
		EncodingCueMatchRateMin:              0.80,
		SchemaApplicationSuccessRateMin:      0.80,
		NearMissSuppressionRateMin:           0.80,
		ResearchPrincipleCoverageRateMin:     0.80,
		ContextViolationRateMax:              0,
		InterferenceFailureRateMax:           0,
		WrongContextUseRateMax:               0,
		HistoricalAsCurrentMisuseRateMax:     0,
		ExperienceReplayErrorRateMax:         0,
		RecalledIgnoredHelpFalsePositiveMax:  0,
		UsedNoImprovementHelpFalsePositiveMax: 0,
	}
}

// CognitiveBenchmarkReport is the Phase 11C artifact.
type CognitiveBenchmarkReport struct {
	GeneratedAt     string           `json:"generated_at"`
	ResearchSources []ResearchSource `json:"research_sources"`
	Summary         CognitiveMetrics `json:"summary"`
	Tasks           []TaskResult     `json:"tasks"`
	Thresholds      CognitiveGateThresholds `json:"thresholds"`
	GatePassed      bool             `json:"gate_passed"`
	GateFailures    []string         `json:"gate_failures,omitempty"`
}

// ComputeCognitiveMetrics aggregates research-backed metrics from task results.
func ComputeCognitiveMetrics(tasks []TaskFixture, results []TaskResult, lc *LoadedCorpus) CognitiveMetrics {
	base := ComputeSuiteMetrics(results)
	cm := CognitiveMetrics{SuiteMetrics: base}

	if len(tasks) == 0 {
		return cm
	}

	principleSet := map[string]struct{}{}
	for _, p := range RequiredResearchPrinciples() {
		principleSet[p] = struct{}{}
	}
	covered := CoveredPrinciples(tasks)
	var coveredCount int
	for p := range principleSet {
		if covered[p] > 0 {
			coveredCount++
		}
	}
	if len(principleSet) > 0 {
		cm.ResearchPrincipleCoverageRate = float64(coveredCount) / float64(len(principleSet))
	}

	var encodingEligible, encodingMatch, underEncoded, underEncodedEligible, schemaEligible, schemaSuccess int
	var nearMissEligible, nearMissSuppressed, contextViolations, interferenceFailures int
	var wrongContextUse, historicalMisuse, experienceReplayErrors int
	var recalledIgnoredFP, usedNoImprovementFP int

	taskByID := map[string]TaskFixture{}
	for _, t := range tasks {
		taskByID[t.ID] = t
	}

	for _, tr := range results {
		t := taskByID[tr.TaskID]
		rest := tr.MemoryREST

		// Schema application: help-required schema tasks that passed.
		if t.ResearchPrinciple == PrincipleSchemaMemory && t.RequiresMemoryHelp {
			schemaEligible++
			if tr.MemoryHelped {
				schemaSuccess++
			}
		}

		// Encoding specificity: score only memories expected to carry adequate cues.
		if t.ResearchPrinciple == PrincipleEncodingSpecificity {
			for _, lbl := range t.ExpectedUsedLabels {
				mem := memoryFixtureByLabel(lc, lbl)
				if mem == nil {
					continue
				}
				encodingEligible++
				match := EvaluateCueMatch(t, *mem)
				if match.MatchScore >= MinCueMatchThreshold && !match.WrongContext {
					encodingMatch++
				}
			}
			if t.ID == "encoding_specificity_missing_cues_fail_to_help" {
				for _, lbl := range append(t.MemoryLabels, t.ExpectedIgnoredLabels...) {
					mem := memoryFixtureByLabel(lc, lbl)
					if mem == nil {
						continue
					}
					match := EvaluateCueMatch(t, *mem)
					if match.UnderEncoded {
						underEncodedEligible++
						underEncoded++
					}
				}
			}
		}

		// Near-miss / interference suppression.
		for _, lbl := range t.ExpectedIgnoredLabels {
			if containsLabel(t.NearMissMemoryLabels, lbl) || containsLabel(t.ExpectedSuppressedLabels, lbl) {
				nearMissEligible++
				if containsLabel(rest.UseTrace.IgnoredLabels, lbl) && !containsLabel(rest.UseTrace.UsedLabels, lbl) {
					nearMissSuppressed++
				}
			}
		}

		// Context / interference failures from REST run.
		if !rest.Score.ForbiddenMemoriesUsed {
			wrongContextUse++
		}
		if rest.Score.MemoryMisused {
			interferenceFailures++
			if t.ResearchPrinciple == PrincipleExperienceFollowing {
				experienceReplayErrors++
			}
			if stringsContainsAny(rest.UseTrace.MisusedLabels, t.ExpectedIgnoredLabels) {
				historicalMisuse++
			}
		}
		for _, lbl := range rest.UseTrace.UsedLabels {
			if containsLabel(t.ForbiddenRecalledLabels, lbl) || containsLabel(t.ExpectedSuppressedLabels, lbl) {
				contextViolations++
			}
		}

		// False positive helpfulness detection.
		if rest.Score.MemoryHelped {
			for _, lbl := range t.ExpectedIgnoredLabels {
				if containsLabel(rest.UseTrace.UsedLabels, lbl) {
					recalledIgnoredFP++
				}
			}
		}
		if rest.Score.MemoryHelped && tr.NoMemory.Score.AnswerPass {
			usedNoImprovementFP++
		}
	}

	n := float64(len(results))
	if encodingEligible > 0 {
		cm.EncodingCueMatchRate = float64(encodingMatch) / float64(encodingEligible)
	}
	if underEncodedEligible > 0 {
		cm.UnderEncodedMemoryRate = float64(underEncoded) / float64(underEncodedEligible)
	}
	if schemaEligible > 0 {
		cm.SchemaApplicationSuccessRate = float64(schemaSuccess) / float64(schemaEligible)
	}
	if nearMissEligible > 0 {
		cm.NearMissSuppressionRate = float64(nearMissSuppressed) / float64(nearMissEligible)
	}
	cm.ContextViolationRate = float64(contextViolations) / n
	cm.InterferenceFailureRate = float64(interferenceFailures) / n
	cm.WrongContextUseRate = float64(wrongContextUse) / n
	cm.HistoricalAsCurrentMisuseRate = float64(historicalMisuse) / n
	cm.ExperienceReplayErrorRate = float64(experienceReplayErrors) / n
	cm.RecalledIgnoredHelpFalsePositive = float64(recalledIgnoredFP) / n
	cm.UsedNoImprovementHelpFalsePositive = float64(usedNoImprovementFP) / n

	return cm
}

func memoryFixtureByLabel(lc *LoadedCorpus, label string) *FixtureMemory {
	if lc == nil {
		return nil
	}
	for _, m := range lc.MemoryFixtures {
		if m.Label == label {
			cp := m
			return &cp
		}
	}
	return nil
}

func stringsContainsAny(hay []string, needles []string) bool {
	for _, n := range needles {
		if containsLabel(hay, n) {
			return true
		}
	}
	return false
}

// EvaluateCognitiveGate checks Phase 11C thresholds.
func EvaluateCognitiveGate(m CognitiveMetrics, th CognitiveGateThresholds) (bool, []string) {
	_, fails := EvaluateGate(m.SuiteMetrics, th.GateThresholds)

	checks := []struct {
		val, max float64
		name     string
		hardZero bool
		min      bool
	}{
		{m.ContextViolationRate, th.ContextViolationRateMax, "context_violation_rate", true, false},
		{m.InterferenceFailureRate, th.InterferenceFailureRateMax, "interference_failure_rate", true, false},
		{m.WrongContextUseRate, th.WrongContextUseRateMax, "wrong_context_use_rate", true, false},
		{m.HistoricalAsCurrentMisuseRate, th.HistoricalAsCurrentMisuseRateMax, "historical_as_current_misuse_rate", true, false},
		{m.ExperienceReplayErrorRate, th.ExperienceReplayErrorRateMax, "experience_replay_error_rate", true, false},
		{m.RecalledIgnoredHelpFalsePositive, th.RecalledIgnoredHelpFalsePositiveMax, "recalled_ignored_help_false_positive_rate", true, false},
		{m.UsedNoImprovementHelpFalsePositive, th.UsedNoImprovementHelpFalsePositiveMax, "used_no_improvement_help_false_positive_rate", true, false},
	}
	for _, c := range checks {
		if c.val > c.max {
			fails = append(fails, formatRateFail(c.name, c.val, c.max, false))
		}
	}
	if m.EncodingCueMatchRate < th.EncodingCueMatchRateMin {
		fails = append(fails, formatRateFail("encoding_cue_match_rate", m.EncodingCueMatchRate, th.EncodingCueMatchRateMin, true))
	}
	if m.SchemaApplicationSuccessRate < th.SchemaApplicationSuccessRateMin {
		fails = append(fails, formatRateFail("schema_application_success_rate", m.SchemaApplicationSuccessRate, th.SchemaApplicationSuccessRateMin, true))
	}
	if m.NearMissSuppressionRate < th.NearMissSuppressionRateMin {
		fails = append(fails, formatRateFail("near_miss_suppression_rate", m.NearMissSuppressionRate, th.NearMissSuppressionRateMin, true))
	}
	if m.ResearchPrincipleCoverageRate < th.ResearchPrincipleCoverageRateMin {
		fails = append(fails, formatRateFail("research_principle_coverage_rate", m.ResearchPrincipleCoverageRate, th.ResearchPrincipleCoverageRateMin, true))
	}
	return len(fails) == 0, fails
}

func formatRateFail(name string, val, bound float64, isMin bool) string {
	if isMin {
		return name + "=below_min:" + fmtFloat(val) + "<" + fmtFloat(bound)
	}
	return name + "=above_max:" + fmtFloat(val) + ">" + fmtFloat(bound)
}

func fmtFloat(v float64) string {
	return fmt.Sprintf("%.3f", v)
}
