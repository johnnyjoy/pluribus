package eval

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"control-plane/internal/recall"
)

// RunAllProofScenariosREST runs embedded scenarios/proof-*.json against a live baseURL (httptest or real).
// This is the adversarial REST proof harness; it does not use in-process recall shortcuts.
func RunAllProofScenariosREST(ctx context.Context, baseURL string, hc *http.Client) (*ProofHarnessReport, error) {
	return RunProofHarnessREST(ctx, baseURL, hc)
}

// RunProofHarnessRESTDeterminismTwice runs the harness twice and requires identical pass/fail signatures (see proof_rest_runner.go).
func RunProofHarnessRESTDeterminismTwice(ctx context.Context, baseURL string, hc *http.Client) (*ProofHarnessReport, error) {
	return RunProofHarnessRESTDeterminism(ctx, baseURL, hc)
}

// RunReport aggregates dual-mode scenario results and sprint-level trigger metrics.
type RunReport struct {
	DualReports []DualScenarioReport
	AllPassed   bool
	// AggregateTriggerMetrics: over-triggering / coverage (Workstream 4).
	TotalTriggerCount       int
	ScenariosWithAnyTrigger int
	RedundantTriggerCount   int
	AggregateBetter         int
	AggregateSame           int
	AggregateWorse          int
}

func RunAllScenarios() (*RunReport, error) {
	scenarios, err := LoadScenarios()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	report := &RunReport{AllPassed: true}
	for _, s := range scenarios {
		writes := RunInitialScenarioExtraction(s)
		extraction := ValidateExtraction(s, writes)
		svc := newEvalRecallService(writes)

		bundleEx, err := runScenarioExplicit(ctx, svc, s)
		if err != nil {
			return nil, err
		}
		bundleTr, meta, err := runScenarioTriggered(ctx, svc, s)
		if err != nil {
			return nil, err
		}

		recallEx := ValidateRecall(s, bundleEx)
		outEx := RunFollowupBehavior(s, bundleEx)
		behEx := ValidateBehavior(s, bundleEx, outEx)

		recallTr := ValidateRecall(s, bundleTr)
		outTr := RunFollowupBehavior(s, bundleTr)
		behTr := ValidateBehavior(s, bundleTr, outTr)

		req := buildCompileRequest(s)
		explicitQ := strings.TrimSpace(req.RetrievalQuery)
		obs := buildTriggerObserved(meta, explicitQ, recallEx, recallTr, behEx, behTr)

		delta := computeDelta(recallEx, recallTr, behEx, behTr)
		stressEx, err := runStressScenario(ctx, svc, s, false)
		if err != nil {
			return nil, err
		}
		stressTr, err := runStressScenario(ctx, svc, s, true)
		if err != nil {
			return nil, err
		}
		stress := mergeStressReports(stressEx, stressTr)

		dr := DualScenarioReport{
			ScenarioID: s.ID,
			Extraction: extraction,
			Explicit: ArmReport{
				Recall:   recallEx,
				Behavior: behEx,
			},
			Triggered: ArmReport{
				Recall:   recallTr,
				Behavior: behTr,
			},
			Trigger: obs,
			Delta:   delta,
			Stress:  stress,
		}
		report.DualReports = append(report.DualReports, dr)

		report.TotalTriggerCount += obs.TriggersFired
		if obs.TriggersFired > 0 {
			report.ScenariosWithAnyTrigger++
		}
		if obs.RedundantTrigger {
			report.RedundantTriggerCount++
		}
		switch delta.Improvement {
		case "yes":
			report.AggregateBetter++
		case "no":
			report.AggregateWorse++
		default:
			report.AggregateSame++
		}

		ok := extraction.Pass && recallEx.Pass && behEx.Pass && recallTr.Pass && behTr.Pass && stressPass(stress)
		if !ok {
			report.AllPassed = false
		}
	}
	return report, nil
}

func buildTriggerObserved(
	meta *recall.TriggerMetadata,
	explicitQ string,
	recallEx, recallTr CheckResult,
	behEx, behTr BehaviorResult,
) TriggerObserved {
	obs := TriggerObserved{ExplicitQuery: explicitQ}
	if meta == nil {
		return obs
	}
	obs.SkippedReason = meta.SkippedReason
	obs.EffectiveQuery = strings.TrimSpace(meta.RetrievalQueryEffective)
	obs.TriggersFired = len(meta.Triggers)
	for _, t := range meta.Triggers {
		obs.Kinds = append(obs.Kinds, string(t.Kind))
	}
	obs.Capped = meta.SkippedReason == "max_triggers_per_request"
	// Redundant: triggers fired but recall+behavior outcomes match explicit arm (possible noise).
	if obs.TriggersFired > 0 &&
		recallEx.Pass == recallTr.Pass &&
		behEx.Pass == behTr.Pass &&
		behEx.AvoidedFailure == behTr.AvoidedFailure &&
		behEx.AppliedPattern == behTr.AppliedPattern &&
		behEx.AlignedWithDecision == behTr.AlignedWithDecision {
		obs.RedundantTrigger = true
	}
	return obs
}

// computeDelta prefers behavior lift: "yes" only if explicit behavior fails and triggered passes.
func computeDelta(recallEx, recallTr CheckResult, behEx, behTr BehaviorResult) DeltaResult {
	exAll := recallEx.Pass && behEx.Pass
	trAll := recallTr.Pass && behTr.Pass
	if !exAll && trAll {
		return DeltaResult{Improvement: "yes", Notes: "triggered arm passed recall+behavior when explicit arm did not"}
	}
	if exAll && !trAll {
		return DeltaResult{Improvement: "no", Notes: "triggered arm failed recall+behavior when explicit arm passed"}
	}
	if exAll && trAll {
		sameFlags := behEx.AvoidedFailure == behTr.AvoidedFailure &&
			behEx.AppliedPattern == behTr.AppliedPattern &&
			behEx.AlignedWithDecision == behTr.AlignedWithDecision
		if sameFlags && recallEx.Pass == recallTr.Pass {
			return DeltaResult{Improvement: "same", Notes: "both arms pass with identical behavior flags and recall check"}
		}
		return DeltaResult{Improvement: "same", Notes: "both arms pass; details differ"}
	}
	return DeltaResult{Improvement: "same", Notes: "both arms fail at least one check"}
}

func FormatScenarioReport(r ScenarioReport) string {
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "[SCENARIO] %s\n\n", r.ScenarioID)
	fmt.Fprintf(sb, "[EXTRACTION]\n%s\n", passFail(r.Extraction.Pass))
	for _, d := range r.Extraction.Details {
		fmt.Fprintf(sb, "details: %s\n", d)
	}
	fmt.Fprintf(sb, "\n[RECALL]\n%s\n", passFail(r.Recall.Pass))
	for _, d := range r.Recall.Details {
		fmt.Fprintf(sb, "missing: %s\n", d)
	}
	fmt.Fprintf(sb, "\n[BEHAVIOR]\n%s\n", passFail(r.Behavior.Pass))
	fmt.Fprintf(sb, "avoided_failure: %t\n", r.Behavior.AvoidedFailure)
	fmt.Fprintf(sb, "applied_pattern: %t\n", r.Behavior.AppliedPattern)
	fmt.Fprintf(sb, "aligned_with_decision: %t\n", r.Behavior.AlignedWithDecision)
	for _, d := range r.Behavior.Details {
		fmt.Fprintf(sb, "details: %s\n", d)
	}
	return sb.String()
}

// FormatDualScenarioReport prints charter-style explicit vs triggered comparison.
func FormatDualScenarioReport(d DualScenarioReport) string {
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "[SCENARIO] %s\n\n", d.ScenarioID)
	fmt.Fprintf(sb, "[EXTRACTION]\n%s\n", passFail(d.Extraction.Pass))
	for _, x := range d.Extraction.Details {
		fmt.Fprintf(sb, "details: %s\n", x)
	}

	fmt.Fprintf(sb, "\n[EXPLICIT RECALL]\n")
	fmt.Fprintf(sb, "recall: %s\n", passFail(d.Explicit.Recall.Pass))
	for _, x := range d.Explicit.Recall.Details {
		fmt.Fprintf(sb, "missing: %s\n", x)
	}
	fmt.Fprintf(sb, "behavior: %s\n", passFail(d.Explicit.Behavior.Pass))
	fmt.Fprintf(sb, "continuity restored (decision aligned heuristic): avoided_failure=%t applied_pattern=%t aligned_with_decision=%t\n",
		d.Explicit.Behavior.AvoidedFailure, d.Explicit.Behavior.AppliedPattern, d.Explicit.Behavior.AlignedWithDecision)

	fmt.Fprintf(sb, "\n[TRIGGERED RECALL]\n")
	fmt.Fprintf(sb, "recall: %s\n", passFail(d.Triggered.Recall.Pass))
	for _, x := range d.Triggered.Recall.Details {
		fmt.Fprintf(sb, "missing: %s\n", x)
	}
	fmt.Fprintf(sb, "behavior: %s\n", passFail(d.Triggered.Behavior.Pass))
	fmt.Fprintf(sb, "continuity restored (decision aligned heuristic): avoided_failure=%t applied_pattern=%t aligned_with_decision=%t\n",
		d.Triggered.Behavior.AvoidedFailure, d.Triggered.Behavior.AppliedPattern, d.Triggered.Behavior.AlignedWithDecision)

	fmt.Fprintf(sb, "\n[TRIGGER OBSERVED]\n")
	fmt.Fprintf(sb, "triggers_fired: %d kinds=%v skipped_reason=%q capped=%t redundant=%t\n",
		d.Trigger.TriggersFired, d.Trigger.Kinds, d.Trigger.SkippedReason, d.Trigger.Capped, d.Trigger.RedundantTrigger)
	fmt.Fprintf(sb, "explicit_query: %q\n", d.Trigger.ExplicitQuery)
	fmt.Fprintf(sb, "effective_query: %q\n", d.Trigger.EffectiveQuery)

	fmt.Fprintf(sb, "\n[DELTA]\n")
	fmt.Fprintf(sb, "improvement: %s\n", d.Delta.Improvement)
	fmt.Fprintf(sb, "notes: %s\n", d.Delta.Notes)
	if d.Stress != nil {
		fmt.Fprintf(sb, "\n[SCENARIO]\n")
		fmt.Fprintf(sb, "continuity maintained: %s\n", yesNo(d.Stress.ContinuityMaintained))
		fmt.Fprintf(sb, "failure avoided: %s\n", yesNo(d.Stress.FailureAvoided))
		fmt.Fprintf(sb, "pattern reused: %s\n", yesNo(d.Stress.PatternReused))
		fmt.Fprintf(sb, "drift detected: %s\n", yesNo(d.Stress.DriftDetected))
		if len(d.Stress.Issues) > 0 {
			fmt.Fprintf(sb, "issues:\n")
			for _, issue := range d.Stress.Issues {
				fmt.Fprintf(sb, "- %s\n", issue)
			}
		}
		for _, st := range d.Stress.StepTraces {
			fmt.Fprintf(sb, "\n[WORKFLOW STEP]\n")
			fmt.Fprintf(sb, "task: %s\n", st.Task)
			fmt.Fprintf(sb, "action: %s\n", st.Action)
			fmt.Fprintf(sb, "recall_used: %s\n", yesNo(st.RecallUsed))
			if strings.Contains(strings.ToLower(st.Action), "resume gap:") {
				fmt.Fprintf(sb, "\n[RESUME]\n")
				fmt.Fprintf(sb, "restored_state: %d\n", st.RestoredState)
				fmt.Fprintf(sb, "restored_constraints: %d\n", st.RestoredConstraints)
			}
			for _, drift := range st.DriftEvents {
				fmt.Fprintf(sb, "\n[DRIFT]\n")
				fmt.Fprintf(sb, "type: %s\n", drift.Type)
				fmt.Fprintf(sb, "cause: %s\n", drift.Cause)
			}
		}
	}
	return sb.String()
}

// FormatRunReportSummary prints aggregate metrics after all scenarios.
func FormatRunReportSummary(r *RunReport) string {
	if r == nil {
		return ""
	}
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "[AGGREGATE]\n")
	fmt.Fprintf(sb, "total_triggers: %d scenarios_with_triggers: %d redundant_trigger_rows: %d\n",
		r.TotalTriggerCount, r.ScenariosWithAnyTrigger, r.RedundantTriggerCount)
	fmt.Fprintf(sb, "delta_better: %d delta_same: %d delta_worse: %d all_passed: %t\n",
		r.AggregateBetter, r.AggregateSame, r.AggregateWorse, r.AllPassed)
	return sb.String()
}

func passFail(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}

func yesNo(ok bool) string {
	if ok {
		return "yes"
	}
	return "no"
}

func mergeStressReports(a, b *StressReport) *StressReport {
	if a == nil && b == nil {
		return nil
	}
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	out := &StressReport{
		ScenarioType:         firstNonEmpty(a.ScenarioType, b.ScenarioType),
		ContinuityMaintained: a.ContinuityMaintained && b.ContinuityMaintained,
		FailureAvoided:       a.FailureAvoided && b.FailureAvoided,
		PatternReused:        a.PatternReused && b.PatternReused,
		DriftDetected:        a.DriftDetected || b.DriftDetected,
		Issues:               append(append([]string{}, a.Issues...), b.Issues...),
		StepTraces:           append(append([]StepTrace{}, a.StepTraces...), b.StepTraces...),
	}
	return out
}

func stressPass(s *StressReport) bool {
	if s == nil {
		return true
	}
	return s.ContinuityMaintained && s.FailureAvoided && s.PatternReused
}
