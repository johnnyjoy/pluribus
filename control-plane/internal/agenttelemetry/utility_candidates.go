package agenttelemetry

import (
	"control-plane/internal/agentobedience"
)

// generateUtilityCandidates creates auditable utility signals from evaluated telemetry.
// Does not mutate memory utility scores.
func generateUtilityCandidates(
	evalID string,
	eval agentobedience.ObedienceEvaluation,
	tel agentobedience.MemoryUseTelemetry,
) []UtilityCandidate {
	var out []UtilityCandidate
	eid, _ := parseUUID(evalID)

	for _, d := range tel.MemoryDecisions {
		strength := 0.5
		safe := false
		reason := d.Reason
		var signal string
		switch d.Decision {
		case "used":
			if eval.ObediencePassed {
				mr, ok := eval.MemoryResults[d.MemoryID]
				if ok && (mr.Decision == agentobedience.StateUsedCorrectly || mr.Decision == "") {
					signal = "used_correctly"
					strength = eval.ObedienceScore
					reason = "evaluator validated correct use"
					safe = true
				} else {
					signal = "misused"
					strength = -0.5
					reason = "evaluator flagged misuse"
				}
			} else {
				signal = "misused"
				strength = -0.5
			}
		case "ignored":
			signal = "ignored_correctly"
			strength = 0.3
		case "historical_only":
			signal = "historical_only_correctly"
			strength = 0.3
		case "misused":
			signal = "misused"
			strength = -0.7
		case "unsafe":
			signal = "unsafe_use"
			strength = -1.0
		default:
			continue
		}
		out = append(out, UtilityCandidate{
			MemoryID:       d.MemoryID,
			EvaluationID:   eid,
			SignalType:     signal,
			SignalStrength: strength,
			SafeToApply:    safe,
			Reason:         reason,
		})
	}

	for _, v := range eval.Violations {
		if v == agentobedience.ViolationUnsupportedOutputClaim {
			out = append(out, UtilityCandidate{
				MemoryID:       "",
				EvaluationID:   eid,
				SignalType:     "unsupported_claim",
				SignalStrength: -0.8,
				SafeToApply:    false,
				Reason:         v,
			})
		}
	}

	if eval.ObediencePassed && len(tel.UsedMemoryIDs) > 0 {
		out = append(out, UtilityCandidate{
			MemoryID:       tel.UsedMemoryIDs[0],
			EvaluationID:   eid,
			SignalType:     "helped_output",
			SignalStrength: 0.4,
			SafeToApply:    true,
			Reason:         "obedient use linked to output",
		})
	}
	if !eval.ObediencePassed && len(eval.Violations) > 0 {
		out = append(out, UtilityCandidate{
			MemoryID:       firstMemory(tel),
			EvaluationID:   eid,
			SignalType:     "harmed_output",
			SignalStrength: -0.6,
			SafeToApply:    false,
			Reason:         "evaluation failed",
		})
	}
	return out
}

func firstMemory(tel agentobedience.MemoryUseTelemetry) string {
	if len(tel.UsedMemoryIDs) > 0 {
		return tel.UsedMemoryIDs[0]
	}
	if len(tel.RecalledMemoryIDs) > 0 {
		return tel.RecalledMemoryIDs[0]
	}
	return ""
}

func violationsToRows(evalID string, violations []string, tel agentobedience.MemoryUseTelemetry) []ViolationRow {
	eid, _ := parseUUID(evalID)
	var rows []ViolationRow
	for _, code := range violations {
		mid := ""
		for _, d := range tel.MemoryDecisions {
			for _, vc := range d.ViolationCodes {
				if vc == code {
					mid = d.MemoryID
				}
			}
		}
		if mid == "" {
			mid = firstMemory(tel)
		}
		rows = append(rows, ViolationRow{
			EvaluationID:  eid,
			MemoryID:      mid,
			ViolationCode: code,
			Severity:      "error",
			Details:       map[string]any{"source": "evaluator"},
		})
	}
	return rows
}
