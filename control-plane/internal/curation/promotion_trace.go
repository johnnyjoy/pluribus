package curation

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
)

func traceEpisodeIDs(p *ProposalPayloadV1) []string {
	if p == nil {
		return nil
	}
	var out []string
	seen := make(map[string]struct{})
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, s := range p.SourceAdvisoryEpisodeIDs {
		add(s)
	}
	add(p.SourceAdvisoryEpisodeID)
	return out
}

// buildMaterializePayload returns JSON for materialize: pluribus_promotion + optional pluribus_evolution (additive).
func buildMaterializePayload(candidateID uuid.UUID, p *ProposalPayloadV1) *json.RawMessage {
	root := map[string]any{
		"pluribus_promotion": map[string]any{
			"v":                                  1,
			"candidate_id":                       candidateID.String(),
			"supporting_episode_ids":             traceEpisodeIDs(p),
			"distill_support_count_at_promotion": supportCount(p),
		},
	}
	if p != nil && p.PluribusEvolution != nil {
		evo := map[string]any{}
		if s := strings.TrimSpace(p.PluribusEvolution.SupersededBy); s != "" {
			evo["superseded_by"] = s
		}
		if s := strings.TrimSpace(p.PluribusEvolution.InvalidatedBy); s != "" {
			evo["invalidated_by"] = s
		}
		if len(p.PluribusEvolution.Contradicts) > 0 {
			cp := append([]string(nil), p.PluribusEvolution.Contradicts...)
			evo["contradicts"] = cp
		}
		if len(evo) > 0 {
			root["pluribus_evolution"] = evo
		}
	}
	b, err := json.Marshal(root)
	if err != nil {
		return nil
	}
	raw := json.RawMessage(b)
	return &raw
}
