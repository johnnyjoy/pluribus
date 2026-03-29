package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"control-plane/internal/memorynorm"
	"control-plane/internal/similarity"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// PatternGeneralizationConfig gates near-duplicate pattern reinforcement on Service.Create.
type PatternGeneralizationConfig struct {
	Enabled                    bool
	MergeJaccardMin            float64 // e.g. 0.82 — min canonical token Jaccard vs candidate
	MinTagOverlapFraction      float64 // min fraction of incoming tags present on candidate (0 = require ≥1 shared tag when both have tags)
	MaxCandidatesScan          int
	MaxSupportingStatementKeys int
	NegationGuard                bool
}

// NormalizePatternGeneralization fills zero values with safe defaults.
func NormalizePatternGeneralization(c *PatternGeneralizationConfig) *PatternGeneralizationConfig {
	if c == nil {
		return nil
	}
	out := *c
	if out.MergeJaccardMin <= 0 || out.MergeJaccardMin > 1 {
		out.MergeJaccardMin = 0.82
	}
	if out.MaxCandidatesScan <= 0 {
		out.MaxCandidatesScan = 50
	}
	if out.MaxSupportingStatementKeys <= 0 {
		out.MaxSupportingStatementKeys = 20
	}
	return &out
}

func tagOverlapFraction(memTags, reqTags []string) float64 {
	if len(reqTags) == 0 {
		return 1.0
	}
	if len(memTags) == 0 {
		return 0
	}
	set := make(map[string]struct{}, len(memTags))
	for _, t := range memTags {
		set[strings.ToLower(strings.TrimSpace(t))] = struct{}{}
	}
	var hit int
	for _, t := range reqTags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if _, ok := set[t]; ok {
			hit++
		}
	}
	return float64(hit) / float64(len(reqTags))
}

func negationConflict(aCanon, bCanon string) bool {
	a := strings.ToLower(aCanon)
	b := strings.ToLower(bCanon)
	never := strings.Contains(a, "never ") || strings.Contains(b, "never ")
	always := strings.Contains(a, "always ") || strings.Contains(b, "always ")
	if never && always {
		return true
	}
	return false
}

// tryMergeNearDuplicatePattern returns a reinforced existing pattern when incoming near-duplicates a stored pattern.
func (s *Service) tryMergeNearDuplicatePattern(ctx context.Context, req *CreateRequest, incomingKey string) (*MemoryObject, error) {
	cfg := NormalizePatternGeneralization(s.PatternGeneralization)
	if cfg == nil || !cfg.Enabled {
		return nil, nil
	}
	max := cfg.MaxCandidatesScan
	if max <= 0 {
		max = 50
	}
	cands, err := s.Repo.Search(ctx, SearchRequest{
		Tags:   req.Tags,
		Status: "active",
		Max:    max,
		Kinds:  []api.MemoryKind{api.MemoryKindPattern},
	})
	if err != nil {
		return nil, err
	}
	inCanon := req.StatementCanonical
	if inCanon == "" {
		inCanon = memorynorm.StatementCanonical(req.Statement)
	}
	var best *MemoryObject
	var bestJ float64
	for i := range cands {
		o := &cands[i]
		if o.Kind != api.MemoryKindPattern {
			continue
		}
		if o.StatementKey != "" && o.StatementKey == incomingKey {
			continue
		}
		prevCanon := o.StatementCanonical
		if prevCanon == "" {
			prevCanon = memorynorm.StatementCanonical(o.Statement)
		}
		j := similarity.CanonicalTokenJaccard(inCanon, prevCanon)
		if j < cfg.MergeJaccardMin {
			continue
		}
		if cfg.NegationGuard && negationConflict(inCanon, prevCanon) {
			continue
		}
		tagsOK := tagOverlapFraction(o.Tags, req.Tags)
		if len(req.Tags) > 0 {
			if cfg.MinTagOverlapFraction > 0 {
				if tagsOK < cfg.MinTagOverlapFraction {
					continue
				}
			} else if tagsOK <= 0 {
				continue
			}
		}
		if best == nil || j > bestJ || (j == bestJ && o.Authority > best.Authority) {
			best = o
			bestJ = j
		}
	}
	if best == nil {
		return nil, nil
	}
	newAuth := best.Authority + 2
	if newAuth > 10 {
		newAuth = 10
	}
	if err := s.Repo.UpdateAuthority(ctx, best.ID, newAuth); err != nil {
		return nil, err
	}
	newPayload, err := mergePatternPayloadForNearDup(best.Payload, incomingKey, bestJ, tagOverlapFraction(best.Tags, req.Tags), cfg.MaxSupportingStatementKeys)
	if err != nil {
		return nil, err
	}
	if len(newPayload) > 0 {
		if err := s.Repo.UpdatePayload(ctx, best.ID, newPayload); err != nil {
			return nil, err
		}
	}
	if s.Cache != nil {
		_ = s.Cache.DeleteByPrefix(ctx, "memory:tags:")
		s.invalidateRecallBundleCache(ctx)
	}
	out, err := s.Repo.GetByID(ctx, best.ID)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, fmt.Errorf("pattern %s missing after merge", best.ID)
	}
	return out, nil
}

func mergePatternPayloadForNearDup(existing []byte, incomingStatementKey string, jaccard, tagOverlap float64, maxKeys int) ([]byte, error) {
	raw := make(map[string]any)
	if len(existing) > 0 {
		_ = json.Unmarshal(existing, &raw)
	}
	gen := map[string]any{
		"reason":                 "near_duplicate_reinforce",
		"jaccard":                jaccard,
		"tag_overlap_fraction":   tagOverlap,
	}
	var keys []string
	if prev, ok := raw["generalization"].(map[string]any); ok {
		if sk, ok := prev["supporting_statement_keys"].([]any); ok {
			for _, v := range sk {
				if s, ok := v.(string); ok {
					keys = append(keys, s)
				}
			}
		}
	}
	if incomingStatementKey != "" {
		dup := false
		for _, k := range keys {
			if k == incomingStatementKey {
				dup = true
				break
			}
		}
		if !dup {
			keys = append(keys, incomingStatementKey)
		}
		if maxKeys > 0 && len(keys) > maxKeys {
			keys = keys[len(keys)-maxKeys:]
		}
	}
	if len(keys) > 0 {
		ks := make([]any, len(keys))
		for i := range keys {
			ks[i] = keys[i]
		}
		gen["supporting_statement_keys"] = ks
	}
	raw["generalization"] = gen
	return json.Marshal(raw)
}

// UUID validation helper for SupportingMemoryIDs (optional future use).
func parseUUIDString(s string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(s))
}
