package curation

import (
	"sort"
	"strings"

	"control-plane/pkg/api"
)

// draft is an internal classifier row before limits and UUID assignment.
type draft struct {
	kind      api.MemoryKind
	statement string
	reason    string
	tags      []string
}

// Classify extracts structured memory drafts from a digest request (max caps slice length).
// It does not query durable memory (no repeated-failure promotion); use Digest for the full pipeline.
func Classify(req DigestRequest, max int) []draft {
	if max <= 0 {
		max = 5
	}
	drafts := buildDrafts(req)
	drafts = dedupeDrafts(drafts)
	drafts = prioritizeTruncateDrafts(drafts, max)
	return drafts
}

func buildDrafts(req DigestRequest) []draft {
	var out []draft
	ans := req.CurationAnswers

	add := func(kind api.MemoryKind, statement, reason string) {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			return
		}
		out = append(out, draft{
			kind:      kind,
			statement: truncateRunes(statement, 2048),
			reason:    truncateRunes(reason, 1024),
			tags:      tagMerge(req.Signals, string(kind)),
		})
	}

	if ans != nil {
		add(api.MemoryKindDecision, ans.Decision, "Structured field: decision")
		if c, ok := constraintFromDecision(strings.TrimSpace(ans.Decision)); ok {
			add(api.MemoryKindConstraint, c, "promoted: decision forbids alternative")
		}
		add(api.MemoryKindConstraint, ans.Constraint, "Structured field: constraint")
		if f := strings.TrimSpace(ans.Failure); f != "" {
			add(api.MemoryKindFailure, f, "Structured field: failure")
			shaped := shapeConstraintStatement(f)
			if hasImperativeGuardrailLanguage(f) && !hasConstraintStatement(out, shaped) {
				add(api.MemoryKindConstraint, shaped, "imperative language")
			} else if hasSeveritySignal(f) && !hasConstraintStatement(out, shaped) {
				add(api.MemoryKindConstraint, shaped, "promoted: severe failure")
			}
		}
		add(api.MemoryKindPattern, ans.Pattern, "Structured field: pattern")
		if strings.TrimSpace(ans.NeverAgain) != "" {
			na := strings.TrimSpace(ans.NeverAgain)
			add(api.MemoryKindFailure, na, "Captured as failure (never again)")
			add(api.MemoryKindConstraint, shapeConstraintStatement(na), "promoted: never again")
		}
		wc := strings.TrimSpace(ans.WhatChanged)
		if wc != "" && (hasImperativeGuardrailLanguage(wc) || hasSeveritySignal(wc)) {
			if len(out) == 0 {
				add(api.MemoryKindConstraint, shapeConstraintStatement(wc), "From what_changed (imperative or severe)")
			} else {
				add(api.MemoryKindConstraint, shapeConstraintStatement(wc), "From what_changed (imperative or severe)")
			}
		}
		if strings.TrimSpace(ans.WhatLearned) != "" {
			if !hasKind(out, api.MemoryKindPattern) {
				add(api.MemoryKindPattern, ans.WhatLearned, "From what_learned")
			}
		}
	}

	if len(out) == 0 {
		ws := strings.TrimSpace(req.WorkSummary)
		if len(ws) >= 20 {
			out = append(out, draft{
				kind:      api.MemoryKindState,
				statement: truncateRunes(ws, 2048),
				reason:    "Digest fallback: no structured curation_answers; capture state for continuity",
				tags:      tagMerge(req.Signals, "digest_fallback"),
			})
		}
	}

	return out
}

func hasConstraintStatement(ds []draft, stmt string) bool {
	t := strings.ToLower(strings.TrimSpace(stmt))
	if t == "" {
		return false
	}
	for _, d := range ds {
		if d.kind != api.MemoryKindConstraint {
			continue
		}
		if strings.ToLower(strings.TrimSpace(d.statement)) == t {
			return true
		}
	}
	return false
}

func dedupeDrafts(ds []draft) []draft {
	seen := make(map[string]struct{})
	var out []draft
	for _, d := range ds {
		key := string(d.kind) + "\x00" + strings.ToLower(strings.TrimSpace(d.statement))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, d)
	}
	return out
}

func prioritizeTruncateDrafts(ds []draft, max int) []draft {
	if len(ds) <= max {
		return ds
	}
	sort.SliceStable(ds, func(i, j int) bool {
		pi, pj := draftPriority(ds[i].kind), draftPriority(ds[j].kind)
		if pi != pj {
			return pi < pj
		}
		return i < j
	})
	return ds[:max]
}

func hasKind(ds []draft, k api.MemoryKind) bool {
	for _, d := range ds {
		if d.kind == k {
			return true
		}
	}
	return false
}

func tagMerge(signals []string, extra string) []string {
	seen := make(map[string]struct{})
	var tags []string
	for _, s := range signals {
		s = strings.TrimSpace(strings.ToLower(s))
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		tags = append(tags, s)
	}
	if extra != "" {
		e := strings.ToLower(extra)
		if _, ok := seen[e]; !ok {
			tags = append(tags, e)
		}
	}
	return tags
}

func truncateRunes(s string, maxBytes int) string {
	if maxBytes <= 0 || len(s) <= maxBytes {
		return s
	}
	return s[:maxBytes]
}
