package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"control-plane/internal/memorynorm"
	"control-plane/internal/similarity"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// CanonicalConsolidationConfig gates deterministic canonical reinforcement on candidate materialize.
type CanonicalConsolidationConfig struct {
	Enabled bool `yaml:"enabled"`
	// NearDuplicateJaccardMin is minimum token Jaccard on statement_canonical (default 0.88).
	NearDuplicateJaccardMin float64 `yaml:"near_duplicate_jaccard_min"`
	// MinTagOverlapFraction is minimum fraction of incoming tags present on candidate memory (0 = require ≥1 shared tag when both have tags).
	MinTagOverlapFraction float64 `yaml:"min_tag_overlap_fraction"`
	MaxCandidatesScan     int     `yaml:"max_candidates_scan"`
	NegationGuard         bool    `yaml:"negation_guard"`
	// AuthorityDeltaPerReinforcement bounded increment per consolidation (default 1).
	AuthorityDeltaPerReinforcement int `yaml:"authority_delta_per_reinforcement"`
	// MaxSupportCount caps pluribus_consolidation.support_count (default 100).
	MaxSupportCount int `yaml:"max_support_count"`
}

// NormalizeCanonicalConsolidation fills zero values with safe defaults.
func NormalizeCanonicalConsolidation(c *CanonicalConsolidationConfig) *CanonicalConsolidationConfig {
	if c == nil {
		c = &CanonicalConsolidationConfig{}
	}
	out := *c
	if out.NearDuplicateJaccardMin <= 0 || out.NearDuplicateJaccardMin > 1 {
		out.NearDuplicateJaccardMin = 0.88
	}
	if out.MaxCandidatesScan <= 0 {
		out.MaxCandidatesScan = 50
	}
	if out.AuthorityDeltaPerReinforcement <= 0 {
		out.AuthorityDeltaPerReinforcement = 1
	}
	if out.AuthorityDeltaPerReinforcement > 10 {
		out.AuthorityDeltaPerReinforcement = 10
	}
	if out.MaxSupportCount <= 0 {
		out.MaxSupportCount = 100
	}
	return &out
}

// ConsolidationDecisionKind is a deterministic consolidation outcome for a promotion candidate.
type ConsolidationDecisionKind string

const (
	ConsolidationCreateNew     ConsolidationDecisionKind = "create_new"
	ConsolidationReinforce     ConsolidationDecisionKind = "reinforce"
	ConsolidationContradictNew ConsolidationDecisionKind = "contradict_new"
)

// ConsolidationDecision is the result of FindCanonicalConsolidationMatch (explainable, reproducible).
type ConsolidationDecision struct {
	Kind              ConsolidationDecisionKind
	TargetID          *uuid.UUID
	ConflictTargetID  *uuid.UUID
	Reason            string
	Jaccard           float64
	ExactStatementKey bool
}

// ConsolidationProposalInput is a memory-neutral view of a promotion proposal for matching.
type ConsolidationProposalInput struct {
	Kind      api.MemoryKind
	Statement string
	Tags      []string
}

func consolidationTagOverlapFraction(memTags, reqTags []string) float64 {
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

func consolidationNegationConflict(aCanon, bCanon string) bool {
	a := strings.ToLower(aCanon)
	b := strings.ToLower(bCanon)
	never := strings.Contains(a, "never ") || strings.Contains(b, "never ")
	always := strings.Contains(a, "always ") || strings.Contains(b, "always ")
	if never && always {
		return true
	}
	return false
}

func extractEntityTagNames(tags []string) []string {
	var out []string
	seen := make(map[string]struct{})
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if !strings.HasPrefix(t, "entity:") {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(t, "entity:"))
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}
	return out
}

// entityOverlapOK returns true when no entity constraint applies or at least one entity tag overlaps.
func entityOverlapOK(incomingTags, memTags []string) bool {
	ie := extractEntityTagNames(incomingTags)
	ee := extractEntityTagNames(memTags)
	if len(ie) == 0 || len(ee) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(ee))
	for _, e := range ee {
		set[strings.ToLower(e)] = struct{}{}
	}
	for _, e := range ie {
		if _, ok := set[strings.ToLower(e)]; ok {
			return true
		}
	}
	return false
}

// FindCanonicalConsolidationMatch scans existing canonical memories for exact key or bounded near-duplicate match.
func (s *Service) FindCanonicalConsolidationMatch(ctx context.Context, cfg *CanonicalConsolidationConfig, in *ConsolidationProposalInput) (*ConsolidationDecision, error) {
	if s == nil || s.Repo == nil || in == nil {
		return &ConsolidationDecision{Kind: ConsolidationCreateNew, Reason: "no_memory_service"}, nil
	}
	norm := NormalizeCanonicalConsolidation(cfg)
	if !norm.Enabled {
		return &ConsolidationDecision{Kind: ConsolidationCreateNew, Reason: "canonical_consolidation_disabled"}, nil
	}
	stmt := strings.TrimSpace(in.Statement)
	if stmt == "" || !validBehaviorKind(in.Kind) {
		return &ConsolidationDecision{Kind: ConsolidationCreateNew, Reason: "invalid_proposal_input"}, nil
	}
	inCanon := memorynorm.StatementCanonical(stmt)
	if inCanon == "" {
		inCanon = stmt
	}
	stmtKey := memorynorm.StatementKey(stmt)

	// 1) Exact statement_key match (active/pending)
	if stmtKey != "" {
		dupID, err := s.Repo.FindActiveDuplicate(ctx, in.Kind, stmtKey)
		if err != nil {
			return nil, err
		}
		if dupID != nil {
			target, err := s.Repo.GetByID(ctx, *dupID)
			if err != nil {
				return nil, err
			}
			if target == nil {
				return &ConsolidationDecision{Kind: ConsolidationCreateNew, Reason: "duplicate_id_missing"}, nil
			}
			prevCanon := target.StatementCanonical
			if prevCanon == "" {
				prevCanon = memorynorm.StatementCanonical(target.Statement)
			}
			j := similarity.CanonicalTokenJaccard(inCanon, prevCanon)
			if norm.NegationGuard && consolidationNegationConflict(inCanon, prevCanon) {
				id := *dupID
				return &ConsolidationDecision{
					Kind:              ConsolidationContradictNew,
					ConflictTargetID:  &id,
					Reason:            "negation_guard_exact_key",
					Jaccard:           j,
					ExactStatementKey: true,
				}, nil
			}
			return &ConsolidationDecision{
				Kind:              ConsolidationReinforce,
				TargetID:          dupID,
				Reason:            "statement_key_match",
				Jaccard:           j,
				ExactStatementKey: true,
			}, nil
		}
	}

	// 2) Near-duplicate scan (same kind)
	max := norm.MaxCandidatesScan
	cands, err := s.Repo.Search(ctx, SearchRequest{
		Kinds:  []api.MemoryKind{in.Kind},
		Tags:   in.Tags,
		Status: "active",
		Max:    max,
	})
	if err != nil {
		return nil, err
	}
	cands2, err := s.Repo.Search(ctx, SearchRequest{
		Kinds:  []api.MemoryKind{in.Kind},
		Tags:   in.Tags,
		Status: "pending",
		Max:    max,
	})
	if err != nil {
		return nil, err
	}
	seen := make(map[uuid.UUID]struct{})
	var merged []MemoryObject
	for _, o := range cands {
		if _, ok := seen[o.ID]; ok {
			continue
		}
		seen[o.ID] = struct{}{}
		merged = append(merged, o)
	}
	for _, o := range cands2 {
		if _, ok := seen[o.ID]; ok {
			continue
		}
		seen[o.ID] = struct{}{}
		merged = append(merged, o)
	}

	if len(in.Tags) == 0 {
		wide, err := s.Repo.Search(ctx, SearchRequest{
			Kinds:  []api.MemoryKind{in.Kind},
			Status: "active",
			Max:    max,
		})
		if err != nil {
			return nil, err
		}
		wide2, err := s.Repo.Search(ctx, SearchRequest{
			Kinds:  []api.MemoryKind{in.Kind},
			Status: "pending",
			Max:    max,
		})
		if err != nil {
			return nil, err
		}
		for _, o := range wide {
			if _, ok := seen[o.ID]; ok {
				continue
			}
			seen[o.ID] = struct{}{}
			merged = append(merged, o)
		}
		for _, o := range wide2 {
			if _, ok := seen[o.ID]; ok {
				continue
			}
			seen[o.ID] = struct{}{}
			merged = append(merged, o)
		}
	}

	var bestOK *MemoryObject
	var bestJOK float64
	var bestConflict *MemoryObject
	var bestJConflict float64
	for i := range merged {
		o := &merged[i]
		if o.Kind != in.Kind {
			continue
		}
		if stmtKey != "" && o.StatementKey == stmtKey {
			continue
		}
		prevCanon := o.StatementCanonical
		if prevCanon == "" {
			prevCanon = memorynorm.StatementCanonical(o.Statement)
		}
		j := similarity.CanonicalTokenJaccard(inCanon, prevCanon)
		if j < norm.NearDuplicateJaccardMin {
			continue
		}
		conflict := norm.NegationGuard && consolidationNegationConflict(inCanon, prevCanon)
		if conflict {
			if bestConflict == nil || j > bestJConflict || (j == bestJConflict && o.Authority > bestConflict.Authority) {
				cp := *o
				bestConflict = &cp
				bestJConflict = j
			}
			continue
		}
		tagsOK := consolidationTagOverlapFraction(o.Tags, in.Tags)
		if len(in.Tags) > 0 {
			if norm.MinTagOverlapFraction > 0 {
				if tagsOK < norm.MinTagOverlapFraction {
					continue
				}
			} else if tagsOK <= 0 {
				continue
			}
		}
		if !entityOverlapOK(in.Tags, o.Tags) {
			continue
		}
		if bestOK == nil || j > bestJOK || (j == bestJOK && o.Authority > bestOK.Authority) {
			cp := *o
			bestOK = &cp
			bestJOK = j
		}
	}

	if bestOK != nil {
		id := bestOK.ID
		return &ConsolidationDecision{
			Kind:              ConsolidationReinforce,
			TargetID:          &id,
			Reason:            fmt.Sprintf("near_duplicate_lexical:jaccard=%.4f", bestJOK),
			Jaccard:           bestJOK,
			ExactStatementKey: false,
		}, nil
	}
	if bestConflict != nil {
		id := bestConflict.ID
		return &ConsolidationDecision{
			Kind:             ConsolidationContradictNew,
			ConflictTargetID: &id,
			Reason:           "negation_guard_near_duplicate",
			Jaccard:          bestJConflict,
		}, nil
	}

	return &ConsolidationDecision{Kind: ConsolidationCreateNew, Reason: "no_similar_canonical_memory"}, nil
}

// ConsolidatePromotionRequest merges an incoming promotion into an existing canonical row (non-destructive).
type ConsolidatePromotionRequest struct {
	TargetID           uuid.UUID
	CandidateID        uuid.UUID
	IncomingTags       []string
	AuthorityDelta     int
	Jaccard            float64
	Reason             string
	ExactKeyMatch      bool
	MaterializePayload []byte // JSON from buildMaterializePayload (pluribus_promotion, etc.)
}

// ConsolidatePromotion strengthens an existing memory: authority (bounded), tag union, payload merge.
func (s *Service) ConsolidatePromotion(ctx context.Context, req ConsolidatePromotionRequest) (*MemoryObject, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("memory service not configured")
	}
	if req.AuthorityDelta <= 0 {
		req.AuthorityDelta = 1
	}
	obj, err := s.Repo.GetByID(ctx, req.TargetID)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, fmt.Errorf("target memory %s not found", req.TargetID)
	}
	newAuth := obj.Authority + req.AuthorityDelta
	if newAuth > AuthorityScale {
		newAuth = AuthorityScale
	}
	if err := s.Repo.UpdateAuthority(ctx, req.TargetID, newAuth); err != nil {
		return nil, err
	}
	if err := s.Repo.MergeTagsIntoMemory(ctx, req.TargetID, req.IncomingTags); err != nil {
		return nil, err
	}
	mergedPayload, err := mergeConsolidationPayload(obj.Payload, req)
	if err != nil {
		return nil, err
	}
	if len(mergedPayload) > 0 {
		if err := s.Repo.UpdatePayload(ctx, req.TargetID, mergedPayload); err != nil {
			return nil, err
		}
	}
	if s.Cache != nil {
		_ = s.Cache.DeleteByPrefix(ctx, "memory:tags:")
		s.invalidateRecallBundleCache(ctx)
	}
	out, err := s.Repo.GetByID(ctx, req.TargetID)
	if err != nil {
		return nil, err
	}
	if out != nil {
		out.UpdatedAt = time.Now()
	}
	return out, nil
}

func mergeConsolidationPayload(existing []byte, req ConsolidatePromotionRequest) ([]byte, error) {
	raw := make(map[string]any)
	if len(existing) > 0 {
		_ = json.Unmarshal(existing, &raw)
	}
	var cons map[string]any
	if v, ok := raw["pluribus_consolidation"].(map[string]any); ok {
		cons = v
	} else {
		cons = map[string]any{"v": 1}
	}
	var sc float64
	if v, ok := cons["support_count"].(float64); ok {
		sc = v
	} else if v, ok := cons["support_count"].(int); ok {
		sc = float64(v)
	}
	sc++
	maxSC := float64(NormalizeCanonicalConsolidation(&CanonicalConsolidationConfig{}).MaxSupportCount)
	if sc > maxSC {
		sc = maxSC
	}
	cons["support_count"] = sc
	cons["last_reason"] = req.Reason
	cons["last_jaccard"] = req.Jaccard
	cons["last_candidate_id"] = req.CandidateID.String()
	cons["last_exact_key_match"] = req.ExactKeyMatch
	cons["last_at"] = time.Now().UTC().Format(time.RFC3339Nano)

	var reinforce []any
	if v, ok := cons["reinforcing_candidates"].([]any); ok {
		reinforce = v
	}
	entry := map[string]any{
		"candidate_id": req.CandidateID.String(),
		"reason":       req.Reason,
		"jaccard":      req.Jaccard,
		"at":           time.Now().UTC().Format(time.RFC3339Nano),
	}
	reinforce = append(reinforce, entry)
	if len(reinforce) > 48 {
		reinforce = reinforce[len(reinforce)-48:]
	}
	cons["reinforcing_candidates"] = reinforce
	raw["pluribus_consolidation"] = cons

	if len(req.MaterializePayload) > 0 {
		var inc map[string]any
		if err := json.Unmarshal(req.MaterializePayload, &inc); err == nil {
			for k, v := range inc {
				if k == "pluribus_consolidation" {
					continue
				}
				// Merge promotion trace: union supporting episode ids from pluribus_promotion if present
				if k == "pluribus_promotion" {
					mergePluribusPromotionInto(raw, v)
					continue
				}
				raw[k] = v
			}
		}
	}
	return json.Marshal(raw)
}

func mergePluribusPromotionInto(root map[string]any, v any) {
	pmNew, ok := v.(map[string]any)
	if !ok {
		return
	}
	existing, _ := root["pluribus_promotion"].(map[string]any)
	if existing == nil {
		root["pluribus_promotion"] = pmNew
		return
	}
	// Union supporting_episode_ids
	var a, b []any
	if x, ok := existing["supporting_episode_ids"].([]any); ok {
		a = x
	}
	if x, ok := pmNew["supporting_episode_ids"].([]any); ok {
		b = x
	}
	seen := make(map[string]struct{})
	var union []any
	add := func(arr []any) {
		for _, e := range arr {
			s := fmt.Sprint(e)
			if s == "" {
				continue
			}
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			union = append(union, e)
		}
	}
	add(a)
	add(b)
	if len(union) > 0 {
		existing["supporting_episode_ids"] = union
	}
	if v, ok := pmNew["distill_support_count_at_promotion"]; ok {
		existing["distill_support_count_at_promotion"] = v
	}
	root["pluribus_promotion"] = existing
}
