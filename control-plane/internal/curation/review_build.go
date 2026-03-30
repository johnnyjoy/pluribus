package curation

import (
	"fmt"
	"math"
	"strings"

	"control-plane/pkg/api"

	"github.com/google/uuid"
)

const (
	maxSupportingEpisodeSummaries = 3
	maxSummaryClip                = 200
	maxStatementClip              = 220
)

// groupTags splits entity:* tags from other tags; drops empty and duplicates internal noise.
func groupTags(tags []string) TagsGrouped {
	seenE := make(map[string]struct{})
	seenD := make(map[string]struct{})
	var g TagsGrouped
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "entity:") {
			name := strings.TrimSpace(strings.TrimPrefix(t, "entity:"))
			if name == "" || name == ":" {
				continue
			}
			if _, ok := seenE[name]; ok {
				continue
			}
			seenE[name] = struct{}{}
			g.Entities = append(g.Entities, name)
		} else {
			if _, ok := seenD[t]; ok {
				continue
			}
			seenD[t] = struct{}{}
			g.Domain = append(g.Domain, t)
		}
	}
	return g
}

func clipText(s string, max int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

func kindHumanPhrase(k api.MemoryKind) string {
	switch k {
	case api.MemoryKindFailure:
		return "failure"
	case api.MemoryKindPattern:
		return "pattern"
	case api.MemoryKindDecision:
		return "decision"
	case api.MemoryKindConstraint:
		return "constraint"
	case api.MemoryKindState:
		return "state"
	default:
		return string(k)
	}
}

func supportCount(p *ProposalPayloadV1) int {
	if p == nil {
		return 1
	}
	n := p.DistillSupportCount
	if n < 1 {
		n = 1
	}
	return n
}

// buildExplanation is deterministic from kind, statement, entities, support count, tags, and salience.
func buildExplanation(p *ProposalPayloadV1, rawText string, salience float64, grouped TagsGrouped) string {
	kind := "candidate"
	var stmt string
	if p != nil {
		kind = kindHumanPhrase(p.Kind)
		stmt = strings.TrimSpace(p.Statement)
	}
	if stmt == "" {
		stmt = strings.TrimSpace(rawText)
	}
	stmt = clipText(stmt, maxStatementClip)

	n := supportCount(p)
	epWord := "episode"
	if n != 1 {
		epWord = "episodes"
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "This %s candidate states: %s ", kind, stmt)
	fmt.Fprintf(&sb, "It is backed by %d supporting %s (distinct distillations or digest evidence merged into this row). ", n, epWord)

	if len(grouped.Entities) > 0 {
		fmt.Fprintf(&sb, "Named entities: %s. ", joinComma(grouped.Entities, 8))
	}
	if len(grouped.Domain) > 0 {
		fmt.Fprintf(&sb, "Context tags: %s. ", joinComma(grouped.Domain, 12))
	}
	if !math.IsNaN(salience) {
		fmt.Fprintf(&sb, "Salience from source scoring is %.2f.", salience)
	} else {
		sb.WriteString("Salience was not recorded.")
	}
	return strings.TrimSpace(sb.String())
}

func joinComma(s []string, max int) string {
	if len(s) == 0 {
		return ""
	}
	if len(s) > max {
		s = s[:max]
	}
	return strings.Join(s, ", ")
}

// computeSignalStrength returns interpretable labels (not a black-box score).
func computeSignalStrength(p *ProposalPayloadV1, salience float64) (strength, detail string) {
	s := salience
	if math.IsNaN(s) {
		s = 0
	}
	n := supportCount(p)

	var b strings.Builder
	fmt.Fprintf(&b, "Observed across %d supporting episode(s). Salience %.2f.", n, s)
	if p != nil {
		fmt.Fprintf(&b, " Kind: %s.", kindHumanPhrase(p.Kind))
	}
	detail = b.String()

	// Thresholds are explicit and documented in signal_detail (no hidden weights).
	if n >= 4 || s >= 0.85 {
		return "strong", detail
	}
	if n >= 2 || s >= 0.65 {
		return "moderate", detail
	}
	return "low", detail
}

func uniqueEpisodeIDsInOrder(ids []string, single string) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{})
	var out []uuid.UUID
	try := func(idStr string) {
		if idStr == "" || len(out) >= maxSupportingEpisodeSummaries {
			return
		}
		u, err := uuid.Parse(strings.TrimSpace(idStr))
		if err != nil {
			return
		}
		if _, ok := seen[u]; ok {
			return
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	for _, s := range ids {
		try(s)
	}
	try(single)
	return out
}

func buildPromotionPreview(p *ProposalPayloadV1, promo *PromotionDigestConfig) *PromotionPreview {
	if p == nil || p.Kind == "" {
		return nil
	}
	if !isBehaviorKind(p.Kind) {
		return nil
	}
	auth := p.ProposedAuthority
	if auth <= 0 {
		auth = defaultAuthority(p.Kind)
	}
	app := string(api.ApplicabilityAdvisory)
	if p.Kind == api.MemoryKindConstraint {
		app = string(api.ApplicabilityGoverning)
	}
	note := "Memory would be created as active."
	if promo != nil && promo.RequireReview {
		note = "Memory would be created as pending (promotion.require_review)."
	}
	return &PromotionPreview{
		Kind:              string(p.Kind),
		Statement:         p.Statement,
		Tags:              append([]string(nil), p.Tags...),
		ProposedAuthority: auth,
		Applicability:     app,
		MemoryStatusNote:  note,
	}
}
