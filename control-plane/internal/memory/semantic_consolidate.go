package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"control-plane/pkg/api"
)

func (s *Service) semanticConsolidateThreshold() float64 {
	if s == nil || s.Dedup == nil {
		return 0
	}
	return s.Dedup.SemanticConsolidateThreshold
}

// tryMergeSemanticNearDuplicate reinforces an existing active row when the incoming
// embedding is semantically near-duplicative (same kind). No-op when threshold is 0,
// semantic retrieval is off, or no embedding was computed on create.
func (s *Service) tryMergeSemanticNearDuplicate(ctx context.Context, req *CreateRequest) (*MemoryObject, error) {
	threshold := s.semanticConsolidateThreshold()
	if threshold <= 0 || req == nil || len(req.Embedding) == 0 {
		return nil, nil
	}
	if s.Semantic == nil || !s.Semantic.RetrievalEnabled() {
		return nil, nil
	}
	cands, sims, err := s.SearchSimilarCandidates(ctx, req.Embedding, SearchRequest{
		Tags:     req.Tags,
		Statuses: []string{string(api.StatusActive)},
		Kinds:    []api.MemoryKind{req.Kind},
	}, 10, threshold)
	if err != nil {
		return nil, err
	}
	if len(cands) == 0 {
		return nil, nil
	}
	var best *MemoryObject
	var bestSim float64
	for i := range cands {
		o := &cands[i]
		if o.StatementKey != "" && o.StatementKey == req.StatementKey {
			continue
		}
		sim := sims[o.ID]
		if sim < threshold {
			continue
		}
		if best == nil || sim > bestSim || (sim == bestSim && o.Authority > best.Authority) {
			best = o
			bestSim = sim
		}
	}
	if best == nil {
		return nil, nil
	}
	full, err := s.Repo.GetByID(ctx, best.ID)
	if err != nil {
		return nil, err
	}
	if full == nil {
		return nil, fmt.Errorf("memory %s missing before semantic consolidate", best.ID)
	}
	best = full
	sameAuthor := stringsTrimEqual(req.AgentID, best.AgentID)
	newPayload, err := mergeSemanticConsolidatePayload(best.Payload, req.StatementKey, bestSim, req.AgentID)
	if err != nil {
		return nil, err
	}
	if len(newPayload) > 0 {
		if err := s.Repo.UpdatePayload(ctx, best.ID, newPayload); err != nil {
			return nil, err
		}
	}
	if !sameAuthor {
		newAuth := best.Authority + 1
		if newAuth > 10 {
			newAuth = 10
		}
		if err := s.Repo.UpdateAuthority(ctx, best.ID, newAuth); err != nil {
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
		return nil, fmt.Errorf("memory %s missing after semantic consolidate", best.ID)
	}
	out.Consolidated = true
	slog.Info("[SEMANTIC CONSOLIDATE]", "memory_id", out.ID.String(), "kind", out.Kind,
		"cosine", bestSim, "incoming_key", req.StatementKey, "same_author", sameAuthor)
	return out, nil
}

func mergeSemanticConsolidatePayload(existing []byte, incomingStatementKey string, cosine float64, agentID string) ([]byte, error) {
	merged, err := mergeSaliencePayload(existing, incomingStatementKey, AgentUsageKey(agentID))
	if err != nil {
		return existing, err
	}
	var root map[string]any
	if err := json.Unmarshal(merged, &root); err != nil {
		return merged, err
	}
	if root == nil {
		root = map[string]any{}
	}
	cons := map[string]any{
		"reason": "semantic_near_duplicate_reinforce",
		"cosine": cosine,
	}
	if incomingStatementKey != "" {
		cons["incoming_statement_key"] = incomingStatementKey
	}
	if strings.TrimSpace(agentID) != "" {
		cons["incoming_agent_id"] = strings.TrimSpace(agentID)
	}
	root["semantic_consolidation"] = cons
	return json.Marshal(root)
}

func stringsTrimEqual(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	return a != "" && b != "" && a == b
}
