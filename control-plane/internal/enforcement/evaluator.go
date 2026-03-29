package enforcement

import (
	"strings"
	"unicode"

	"control-plane/internal/drift"
	"control-plane/internal/memory"
	"control-plane/internal/memorynorm"
	"control-plane/pkg/api"
)

// evalHit is an internal evaluation hit before evidence attachment.
type evalHit struct {
	Memory     memory.MemoryObject
	Decision   EnforcementDecision
	ReasonCode string
	Detail     string
}

func evaluateAll(objs []memory.MemoryObject, proposal string, intent string, failureThreshold float64, patternBlockScore float64) []evalHit {
	propCanon := memorynorm.StatementCanonical(proposal)
	intentCanon := normalizeIntent(intent)
	var out []evalHit
	for _, m := range objs {
		for _, t := range evaluateOne(m, propCanon, intentCanon, failureThreshold, patternBlockScore) {
			out = append(out, t)
		}
	}
	return out
}

// memoryStatementCanonical returns canonical durable text for evaluation (Phase D).
func memoryStatementCanonical(m memory.MemoryObject) string {
	if m.StatementCanonical != "" {
		return m.StatementCanonical
	}
	return memorynorm.StatementCanonical(m.Statement)
}

// normalizeIntent trims and lowercases intent labels (e.g. datastore) for stable matching.
func normalizeIntent(intent string) string {
	s := strings.TrimSpace(intent)
	if s == "" {
		return ""
	}
	return strings.ToLower(s)
}

func evaluateOne(m memory.MemoryObject, proposalCanon string, intentCanon string, failureThreshold float64, patternBlockScore float64) []evalHit {
	switch m.Kind {
	case api.MemoryKindConstraint, api.MemoryKindDecision:
		if ok, detail := normativeConflict(m, proposalCanon, intentCanon); ok {
			return []evalHit{{
				Memory:     m,
				Decision:   DecisionBlock,
				ReasonCode: "normative_conflict",
				Detail:     detail,
			}}
		}
	case api.MemoryKindFailure:
		memCanon := memoryStatementCanonical(m)
		if wordOverlapRatio(wordSet(proposalCanon), wordSet(memCanon)) >= failureThreshold {
			return []evalHit{{
				Memory:     m,
				Decision:   DecisionRequireReview,
				ReasonCode: "anti_pattern_overlap",
				Detail:     "Proposal overlaps a recorded failure / anti-pattern.",
			}}
		}
	case api.MemoryKindPattern:
		issues := drift.NegativePatternMatches(proposalCanon, []memory.MemoryObject{m})
		if len(issues) == 0 {
			return nil
		}
		iss := issues[0]
		if iss.Score >= patternBlockScore {
			return []evalHit{{
				Memory:     m,
				Decision:   DecisionBlockOverrideable,
				ReasonCode: "negative_pattern",
				Detail:     iss.Statement,
			}}
		}
		return []evalHit{{
			Memory:     m,
			Decision:   DecisionRequireReview,
			ReasonCode: "negative_pattern",
			Detail:     iss.Statement,
		}}
	}
	return nil
}

// normativeConflict detects high-confidence contradictions (v1 heuristics; no embeddings).
// proposalCanon and memory lines use memorynorm.StatementCanonical (Phase D) so paraphrases match consistently.
func normativeConflict(m memory.MemoryObject, proposalCanon string, intentCanon string) (bool, string) {
	lm := memoryStatementCanonical(m)
	lp := proposalCanon

	// Trusted decision/constraint requires Postgres; proposal pushes SQLite.
	if (strings.Contains(lm, "postgres") || strings.Contains(lm, "postgresql")) &&
		(strings.Contains(lm, "must") || strings.Contains(lm, "only") || strings.Contains(lm, "required") || strings.Contains(lm, "shall")) {
		if strings.Contains(lp, "sqlite") {
			return true, "Proposal introduces SQLite while trusted memory requires Postgres for durable data."
		}
	}

	// Explicit prohibition on SQLite in memory; proposal still uses SQLite.
	if strings.Contains(lm, "sqlite") &&
		(strings.Contains(lm, "never") || strings.Contains(lm, "must not") || strings.Contains(lm, "do not") || strings.Contains(lm, "forbidden")) {
		if strings.Contains(lp, "sqlite") {
			return true, "Proposal conflicts with an explicit prohibition on SQLite."
		}
	}

	// Datastore intent: reinforce Postgres vs SQLite conflict.
	if intentCanon == "datastore" || intentCanon == "" {
		if strings.Contains(lm, "postgres") && strings.Contains(lm, "only") && strings.Contains(lp, "sqlite") {
			return true, "Datastore intent conflicts with project requirement to use Postgres only."
		}
	}

	return false, ""
}

func wordSet(text string) map[string]struct{} {
	set := make(map[string]struct{})
	f := func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) }
	for _, w := range strings.FieldsFunc(strings.ToLower(text), f) {
		if len(w) > 0 {
			set[w] = struct{}{}
		}
	}
	return set
}

func wordOverlapRatio(proposalWords map[string]struct{}, statementWords map[string]struct{}) float64 {
	if len(statementWords) == 0 {
		return 0
	}
	var overlap int
	for w := range statementWords {
		if _, ok := proposalWords[w]; ok {
			overlap++
		}
	}
	return float64(overlap) / float64(len(statementWords))
}

func worstDecision(ds []EnforcementDecision) EnforcementDecision {
	w := DecisionAllow
	rank := map[EnforcementDecision]int{
		DecisionAllow:             0,
		DecisionRequireReview:     1,
		DecisionBlockOverrideable: 2,
		DecisionBlock:             3,
	}
	for _, d := range ds {
		if rank[d] > rank[w] {
			w = d
		}
	}
	return w
}
