package recall

import (
	"log/slog"
	"strings"

	"control-plane/internal/memorynorm"
	"control-plane/internal/similarity"
)

// ValidationAction classifies a behavior hit for clients (reject vs revise).
type ValidationAction struct {
	Statement string `json:"statement"`
	Action    string `json:"action"` // "reject" | "revise"
	Category  string `json:"category,omitempty"`
}

// BehaviorValidation is a lightweight Recall->Validate signal used by run-multi.
type BehaviorValidation struct {
	ConstraintViolations []string           `json:"constraint_violations,omitempty"`
	RepeatedFailures     []string           `json:"repeated_failures,omitempty"`
	DecisionConflicts    []string           `json:"decision_conflicts,omitempty"`
	MovedTowardGoal      bool               `json:"moved_toward_goal"`
	Actions              []ValidationAction `json:"actions,omitempty"`
	// Blocked is true when any bucket is non-empty (same as !OK(); explicit for clients).
	Blocked bool `json:"blocked,omitempty"`
}

// BehaviorValidationConfig is loaded from YAML (recall.behavior_validation) and wired into Service.
type BehaviorValidationConfig struct {
	// SoftFailureRevise when true restores legacy behavior: overlapping failure rows map to "revise" instead of "reject"
	// when the failure statement is not severity- or imperative-guardrail flagged.
	SoftFailureRevise bool `yaml:"soft_failure_revise"`
	// OverlapThreshold in (0,1]; default 0.5. Compared to max(token overlap, canonical token Jaccard).
	OverlapThreshold float64 `yaml:"overlap_threshold"`
	// BlockOutputOnValidationFail clears selected output and merged_output in the run-multi response when validation fails.
	BlockOutputOnValidationFail bool `yaml:"block_output_on_validation_fail"`
}

// NormalizeBehaviorValidationConfig returns a copy with defaults applied (nil => zero config + defaults).
func NormalizeBehaviorValidationConfig(c *BehaviorValidationConfig) BehaviorValidationConfig {
	if c == nil {
		return BehaviorValidationConfig{OverlapThreshold: 0.5}
	}
	out := *c
	if out.OverlapThreshold <= 0 || out.OverlapThreshold > 1 {
		out.OverlapThreshold = 0.5
	}
	return out
}

func (v BehaviorValidation) OK() bool {
	return len(v.ConstraintViolations) == 0 && len(v.RepeatedFailures) == 0 && len(v.DecisionConflicts) == 0
}

func validateBehavior(bundle *RecallBundle, proposal string, cfg *BehaviorValidationConfig) BehaviorValidation {
	bc := NormalizeBehaviorValidationConfig(cfg)
	th := bc.OverlapThreshold
	p := strings.TrimSpace(strings.ToLower(proposal))
	if bundle == nil || strings.TrimSpace(proposal) == "" {
		return BehaviorValidation{}
	}
	out := BehaviorValidation{}
	for _, it := range bundle.Constraints {
		s := strings.TrimSpace(strings.ToLower(it.Statement))
		if s == "" {
			continue
		}
		score := overlapScore(proposal, it.Statement)
		if score < th {
			continue
		}
		if it.Kind == "failure" {
			action := failureOverlapAction(bc, it.Statement)
			out.RepeatedFailures = append(out.RepeatedFailures, it.Statement)
			out.Actions = append(out.Actions, ValidationAction{
				Statement: it.Statement,
				Action:    action,
				Category:  "repeated_failure",
			})
			slog.Info("[CONSTRAINT VIOLATION]", "category", "repeated_failure", "action", action, "overlap", score, "statement", it.Statement)
		} else {
			out.ConstraintViolations = append(out.ConstraintViolations, it.Statement)
			out.Actions = append(out.Actions, ValidationAction{
				Statement: it.Statement,
				Action:    "reject",
				Category:  "constraint_violation",
			})
			slog.Info("[CONSTRAINT VIOLATION]", "category", "constraint_violation", "overlap", score, "statement", it.Statement)
		}
	}
	for _, d := range bundle.Continuity {
		if d.Kind != "decision" {
			continue
		}
		s := strings.TrimSpace(strings.ToLower(d.Statement))
		if s == "" {
			continue
		}
		if decisionConflictWithProposal(p, s, proposal, d.Statement, th) {
			out.DecisionConflicts = append(out.DecisionConflicts, d.Statement)
			out.Actions = append(out.Actions, ValidationAction{
				Statement: d.Statement,
				Action:    "reject",
				Category:  "decision_conflict",
			})
			slog.Info("[CONSTRAINT VIOLATION]", "category", "decision_conflict", "statement", d.Statement)
		}
	}
	for _, it := range bundle.Continuity {
		if overlapScore(proposal, it.Statement) >= 0.25 {
			out.MovedTowardGoal = true
			break
		}
	}
	out.Blocked = !out.OK()
	return out
}

func failureOverlapAction(bc BehaviorValidationConfig, stmt string) string {
	if severitySignal(stmt) || imperativeGuardrailLanguage(stmt) {
		return "reject"
	}
	if bc.SoftFailureRevise {
		return "revise"
	}
	return "reject"
}

// imperativeGuardrailLanguage and severitySignal mirror curation/promotion_rules (avoid import cycle).
func imperativeGuardrailLanguage(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}
	ls := strings.ToLower(strings.TrimSpace(s))
	if strings.Contains(ls, "forbidden") {
		return true
	}
	t := " " + ls + " "
	return strings.Contains(t, " never ") ||
		strings.Contains(t, "must not") ||
		strings.Contains(t, "mustn't") ||
		strings.Contains(t, " do not ") ||
		strings.Contains(t, " don't ") ||
		strings.Contains(t, " always ") ||
		strings.Contains(t, " shall not ") ||
		strings.Contains(t, " will not ")
}

func severitySignal(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}
	ld := strings.ToLower(s)
	return strings.Contains(ld, "production") ||
		strings.Contains(ld, "release") ||
		strings.Contains(ld, "corruption") ||
		strings.Contains(ld, "data loss") ||
		strings.Contains(ld, "duplicate charge") ||
		strings.Contains(ld, "outage") ||
		strings.Contains(ld, "p0") ||
		strings.Contains(ld, "p1") ||
		strings.Contains(ld, "security breach") ||
		strings.Contains(ld, "incident")
}

func overlapScore(proposal, memStatement string) float64 {
	p := strings.TrimSpace(proposal)
	m := strings.TrimSpace(memStatement)
	if p == "" || m == "" {
		return 0
	}
	pl := strings.ToLower(p)
	ml := strings.ToLower(m)
	to := tokenOverlap(pl, ml)
	cj := similarity.CanonicalTokenJaccard(memorynorm.StatementCanonical(p), memorynorm.StatementCanonical(m))
	if cj > to {
		return cj
	}
	return to
}

func decisionConflictWithProposal(proposalLower string, decisionLower string, proposalOrig, decisionOrig string, th float64) bool {
	if overlapScore(proposalOrig, decisionOrig) < th {
		return false
	}
	if strings.Contains(proposalLower, "instead") || strings.Contains(proposalLower, "replace") || strings.Contains(proposalLower, "revert") {
		return true
	}
	// Strong negative guardrail: trusted durable-store decision vs SQLite in proposal.
	if (strings.Contains(decisionLower, "postgres") || strings.Contains(decisionLower, "postgresql")) &&
		(strings.Contains(decisionLower, "only") || strings.Contains(decisionLower, "must") || strings.Contains(decisionLower, "required") || strings.Contains(decisionLower, "shall")) &&
		strings.Contains(proposalLower, "sqlite") {
		return true
	}
	return false
}

func tokenOverlap(a, b string) float64 {
	as := tokenSet(a)
	bs := tokenSet(b)
	if len(as) == 0 || len(bs) == 0 {
		return 0
	}
	var inter int
	for t := range as {
		if _, ok := bs[t]; ok {
			inter++
		}
	}
	den := len(as)
	if len(bs) < den {
		den = len(bs)
	}
	return float64(inter) / float64(den)
}

func tokenSet(s string) map[string]struct{} {
	s = strings.ToLower(s)
	r := strings.NewReplacer(",", " ", ".", " ", ";", " ", ":", " ", "(", " ", ")", " ", "-", " ", "_", " ", "/", " ")
	parts := strings.Fields(r.Replace(s))
	out := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		if len(p) < 3 {
			continue
		}
		out[p] = struct{}{}
	}
	return out
}
