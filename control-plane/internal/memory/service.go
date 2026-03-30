package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"control-plane/internal/cache"
	"control-plane/internal/memorynorm"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// WARNING: This system is memory-first.
// Do not introduce project/task/workspace/hive/scope concepts as memory partitions.

// mapPromoteType maps promote contract type values to memory kind.
func mapPromoteType(t string) (api.MemoryKind, error) {
	switch t {
	case "decision":
		return api.MemoryKindDecision, nil
	case "constraint":
		return api.MemoryKindConstraint, nil
	case "failure":
		return api.MemoryKindFailure, nil
	case "pattern":
		return api.MemoryKindPattern, nil
	case "state":
		return api.MemoryKindState, nil
	default:
		return "", fmt.Errorf("unsupported promote type %q", t)
	}
}

// LifecycleConfig holds deltas for authority adjustment (Task 74) and expiration threshold (Task 75).
type LifecycleConfig struct {
	AuthorityPositiveDelta       float64 // validation
	AuthorityNegativeDelta       float64 // contradiction, failure
	ExpirationAuthorityThreshold int     // 0-10; memories with authority < this and TTL expired get archived
}

// Service provides memory object use cases.
type Service struct {
	Repo      *Repo
	Cache     cache.Store
	CacheTTL  time.Duration
	Lifecycle *LifecycleConfig // optional; nil = no authority adjustment
	// Dedup optional; nil = exact duplicate detection on (Phase C).
	Dedup *DedupConfig
	// PatternGeneralization optional; nil or disabled = no near-duplicate pattern merge on create.
	PatternGeneralization *PatternGeneralizationConfig
	// Evidence optional linker for POST /v1/memory/promote when evidence_ids is set.
	Evidence PromotionEvidenceLinker
	// Reinforcement optional caps for recall/success authority bumps (nil = defaults).
	Reinforcement *RecallReinforcementConfig
	// Semantic optional: pgvector embeddings for semantic candidate retrieval (default off).
	Semantic *SemanticRetrievalConfig
	Embedder Embedder
	// PatternElevation optional cluster elevation into a dominant pattern row (default off).
	PatternElevation *PatternElevationConfig
}

// ReinforceRecallUsage increases authority for recalled memories (frequency/reuse signal).
// It never decreases authority and is safe to call repeatedly.
func (s *Service) ReinforceRecallUsage(ctx context.Context, ids []uuid.UUID) error {
	return s.ReinforceRecallUsageWithMeta(ctx, ids, ReinforceMeta{})
}

// ReinforceSuccess bumps authority for validated success paths (enforcement proceed, etc.).
func (s *Service) ReinforceSuccess(ctx context.Context, ids []uuid.UUID, reason string) error {
	return s.ReinforceRecallUsageWithMeta(ctx, ids, ReinforceMeta{Reason: reason})
}

// ReinforceRecallUsageWithMeta applies bounded authority increase with optional salience merge and structured logging.
// Reuse delta (max_authority_delta_per_compile) is applied first; then optional extra authority when distinct_agents increases.
func (s *Service) ReinforceRecallUsageWithMeta(ctx context.Context, ids []uuid.UUID, meta ReinforceMeta) error {
	if s == nil || s.Repo == nil || len(ids) == 0 {
		return nil
	}
	impact := strings.ToLower(strings.TrimSpace(meta.Impact))
	if impact == "" {
		impact = "medium"
	}
	highDelta := 2
	medDelta := 1
	lowDelta := 0
	if s.Reinforcement != nil {
		if s.Reinforcement.ImpactHighDelta >= 0 {
			highDelta = s.Reinforcement.ImpactHighDelta
		}
		if s.Reinforcement.ImpactMediumDelta >= 0 {
			medDelta = s.Reinforcement.ImpactMediumDelta
		}
		if s.Reinforcement.ImpactLowDelta >= 0 {
			lowDelta = s.Reinforcement.ImpactLowDelta
		}
	}
	maxDelta := medDelta
	switch impact {
	case "high":
		maxDelta = highDelta
	case "low":
		maxDelta = lowDelta
	}
	if maxDelta > 10 {
		maxDelta = 10
	}
	if maxDelta < 0 {
		maxDelta = 0
	}
	minLowSignal := 1
	if s.Reinforcement != nil && s.Reinforcement.MinSignalStrengthForLow > 0 {
		minLowSignal = s.Reinforcement.MinSignalStrengthForLow
	}
	if impact == "low" && meta.SignalStrength < minLowSignal {
		return nil
	}
	if s.Reinforcement != nil && s.Reinforcement.MaxAuthorityDeltaPerCompile > 0 && maxDelta > s.Reinforcement.MaxAuthorityDeltaPerCompile {
		maxDelta = s.Reinforcement.MaxAuthorityDeltaPerCompile
	}
	maxAgentDelta := 1
	if s.Reinforcement != nil && s.Reinforcement.MaxAgentAuthorityDeltaPerCompile > 0 {
		maxAgentDelta = s.Reinforcement.MaxAgentAuthorityDeltaPerCompile
		if maxAgentDelta > 10 {
			maxAgentDelta = 10
		}
	}
	maxAgentForAuth := 8
	if s.Reinforcement != nil && s.Reinforcement.MaxDistinctAgentsForAuthority > 0 {
		maxAgentForAuth = s.Reinforcement.MaxDistinctAgentsForAuthority
	}
	crossCtx := true
	if s.Reinforcement != nil && s.Reinforcement.CrossContextEnabled != nil {
		crossCtx = *s.Reinforcement.CrossContextEnabled
	}
	crossAgent := true
	if s.Reinforcement != nil && s.Reinforcement.CrossAgentEnabled != nil {
		crossAgent = *s.Reinforcement.CrossAgentEnabled
	}
	ctxKey := meta.ContextKey
	if !crossCtx {
		ctxKey = ""
	}
	agKey := meta.AgentKey
	if !crossAgent {
		agKey = ""
	}
	reason := meta.Reason
	if reason == "" {
		reason = "reuse_recall"
	}
	seen := make(map[uuid.UUID]struct{}, len(ids))
	cacheDirty := false
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		obj, err := s.Repo.GetByID(ctx, id)
		if err != nil || obj == nil {
			continue
		}
		_, prevAgents := SalienceDistinctCounts(obj.Payload)
		merged := obj.Payload
		var merr error
		if ctxKey != "" || agKey != "" {
			merged, merr = mergeSaliencePayload(obj.Payload, ctxKey, agKey)
			if merr != nil {
				merged = obj.Payload
			}
		}
		newCtx, newAgents := SalienceDistinctCounts(merged)
		room := AuthorityScale - obj.Authority
		if room <= 0 {
			continue
		}
		reuseD := maxDelta
		if reuseD > room {
			reuseD = room
		}
		remaining := room - reuseD
		agentExtra := 0
		if newAgents > prevAgents && prevAgents < maxAgentForAuth {
			inc := newAgents - prevAgents
			if inc > 0 {
				agentExtra = inc
				if agentExtra > maxAgentDelta {
					agentExtra = maxAgentDelta
				}
				if agentExtra > remaining {
					agentExtra = remaining
				}
			}
		}
		totalAdd := reuseD + agentExtra
		if totalAdd <= 0 {
			continue
		}
		newAuthority := obj.Authority + totalAdd
		if newAuthority > AuthorityScale {
			newAuthority = AuthorityScale
		}
		if err := s.Repo.UpdateAuthority(ctx, id, newAuthority); err != nil {
			continue
		}
		cacheDirty = true
		logReason := reason
		if agentExtra > 0 {
			logReason = "cross_agent_usage"
		}
		if merr == nil && len(merged) > 0 && !bytes.Equal(merged, obj.Payload) {
			if err := s.Repo.UpdatePayload(ctx, id, merged); err == nil {
				cacheDirty = true
			}
		}
		slog.Info("[AUTHORITY UPDATE]", "memory_id", id.String(), "reason", logReason, "delta", totalAdd, "reuse_delta", reuseD, "agent_delta", agentExtra, "new_authority", newAuthority, "distinct_contexts", newCtx, "distinct_agents", newAgents)
		slog.Info("[LEARNING]", "memory_id", id.String(), "impact", impact, "reason", logReason, "delta", totalAdd, "signal_strength", meta.SignalStrength, "new_authority", newAuthority)
	}
	if cacheDirty && s.Cache != nil {
		_ = s.Cache.DeleteByPrefix(ctx, "memory:tags:")
		s.invalidateRecallBundleCache(ctx)
	}
	return nil
}

// invalidateRecallBundleCache drops cached recall compile / GET bundle results so recall matches DB after memory changes.
// Keys: recall:bundle:* (see internal/cache.RecallBundleKey).
func (s *Service) invalidateRecallBundleCache(ctx context.Context) {
	if s.Cache == nil {
		return
	}
	_ = s.Cache.DeleteByPrefix(ctx, "recall:bundle:")
}

// Create creates a memory object with tags. Invalidates tag-index cache for the project.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*MemoryObject, error) {
	if req.Statement == "" {
		return nil, fmt.Errorf("statement is required")
	}
	if !validBehaviorKind(req.Kind) {
		return nil, fmt.Errorf("invalid kind %q", req.Kind)
	}
	if req.Status != "" && !validCreateStatus(req.Status) {
		return nil, fmt.Errorf("invalid status %q for create (use active or pending)", req.Status)
	}
	if req.Payload != nil && len(*req.Payload) > 0 {
		var any map[string]any
		if err := json.Unmarshal(*req.Payload, &any); err != nil {
			return nil, fmt.Errorf("payload: %w", err)
		}
	}
	canon := memorynorm.StatementCanonical(req.Statement)
	if strings.TrimSpace(req.Statement) != "" && canon == "" {
		return nil, fmt.Errorf("statement normalizes to empty")
	}
	req.StatementCanonical = canon
	req.StatementKey = memorynorm.StatementKey(req.Statement)
	if s.dedupEnabled() && req.StatementKey != "" {
		dupID, err := s.Repo.FindActiveDuplicate(ctx, req.Kind, req.StatementKey)
		if err != nil {
			return nil, err
		}
		if dupID != nil {
			existing, gerr := s.Repo.GetByID(ctx, *dupID)
			if gerr != nil || existing == nil {
				return nil, &ErrDuplicateMemory{ExistingID: *dupID}
			}
			newAuth := existing.Authority + 1
			if newAuth > 10 {
				newAuth = 10
			}
			if err := s.Repo.UpdateAuthority(ctx, *dupID, newAuth); err != nil {
				return nil, err
			}
			if s.Cache != nil {
				_ = s.Cache.DeleteByPrefix(ctx, "memory:tags:")
				s.invalidateRecallBundleCache(ctx)
			}
			existing.Authority = newAuth
			existing.UpdatedAt = time.Now()
			return existing, nil
		}
	}
	if req.Kind == api.MemoryKindPattern && !req.SkipPatternNearMerge {
		merged, err := s.tryMergeNearDuplicatePattern(ctx, &req, req.StatementKey)
		if err != nil {
			return nil, err
		}
		if merged != nil {
			return merged, nil
		}
	}
	s.maybeEmbedOnCreate(ctx, &req)
	obj, err := s.Repo.Create(ctx, req)
	if err != nil {
		return nil, err
	}
	if req.SupersedesID != nil && *req.SupersedesID != uuid.Nil {
		old, _ := s.Repo.GetByID(ctx, *req.SupersedesID)
		if old != nil && old.Status == api.StatusActive {
			_ = s.Repo.MarkSuperseded(ctx, *req.SupersedesID, time.Now())
			if s.Cache != nil {
				_ = s.Cache.DeleteByPrefix(ctx, "memory:tags:")
			}
		}
	}
	if s.Cache != nil {
		_ = s.Cache.DeleteByPrefix(ctx, "memory:tags:")
		s.invalidateRecallBundleCache(ctx)
	}
	return obj, nil
}

// Search returns memory objects matching the request, ordered by authority descending. Uses cache when configured.
func (s *Service) Search(ctx context.Context, req SearchRequest) ([]MemoryObject, error) {
	if s.Repo == nil {
		return nil, fmt.Errorf("memory repo not configured")
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	max := req.Max
	if max <= 0 {
		max = 20
	}
	key := cache.MemorySearchKey(req.Tags, status, max, kindStringsForSearchCache(req.Kinds))
	if s.Cache != nil && s.CacheTTL > 0 {
		if b, err := s.Cache.Get(ctx, key); err == nil && len(b) > 0 {
			var list []MemoryObject
			if json.Unmarshal(b, &list) == nil {
				return list, nil
			}
		}
	}
	list, err := s.Repo.Search(ctx, req)
	if err != nil {
		return nil, err
	}
	if s.Cache != nil && s.CacheTTL > 0 {
		if b, err := json.Marshal(list); err == nil {
			_ = s.Cache.Set(ctx, key, b, s.CacheTTL)
		}
	}
	return list, nil
}

// SearchMemories runs POST /v1/memories/search (tag/query filters without project-scoped SQL).
func (s *Service) SearchMemories(ctx context.Context, req MemoriesSearchRequest) ([]MemoryObject, error) {
	return s.Repo.SearchTagOnly(ctx, req.Query, req.Tags, req.Status, req.Max)
}

func validKind(k api.MemoryKind) bool {
	switch k {
	case api.MemoryKindDecision, api.MemoryKindConstraint, api.MemoryKindFailure, api.MemoryKindPattern, api.MemoryKindState:
		return true
	default:
		return false
	}
}

func validBehaviorKind(k api.MemoryKind) bool {
	switch k {
	case api.MemoryKindState, api.MemoryKindDecision, api.MemoryKindFailure, api.MemoryKindPattern, api.MemoryKindConstraint:
		return true
	default:
		return false
	}
}

func validCreateStatus(s api.Status) bool {
	switch s {
	case api.StatusActive, api.StatusPending:
		return true
	default:
		return false
	}
}

func (s *Service) dedupEnabled() bool {
	if s == nil || s.Dedup == nil {
		return true
	}
	return s.Dedup.IsEnabled()
}

// ApplyAuthorityEvent updates a memory's authority based on a validation or contradiction/failure event.
// Returns the updated memory or an error if not found or lifecycle not configured.
// Contradiction/failure events apply uniformly to memories.
func (s *Service) ApplyAuthorityEvent(ctx context.Context, memoryID uuid.UUID, eventType string) (*MemoryObject, error) {
	if s.Lifecycle == nil {
		return nil, fmt.Errorf("memory lifecycle not configured")
	}
	obj, err := s.Repo.GetByID(ctx, memoryID)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, fmt.Errorf("memory not found")
	}
	deltaPos := s.Lifecycle.AuthorityPositiveDelta
	if deltaPos <= 0 {
		deltaPos = 0.1
	}
	deltaNeg := s.Lifecycle.AuthorityNegativeDelta
	if deltaNeg <= 0 {
		deltaNeg = 0.2
	}
	newAuthority := ApplyAuthorityEvent(obj.Authority, eventType, deltaPos, deltaNeg)
	if newAuthority == obj.Authority {
		return obj, nil
	}
	if err := s.Repo.UpdateAuthority(ctx, memoryID, newAuthority); err != nil {
		return nil, err
	}
	if s.Cache != nil {
		_ = s.Cache.DeleteByPrefix(ctx, "memory:tags:")
		s.invalidateRecallBundleCache(ctx)
	}
	obj.Authority = newAuthority
	return obj, nil
}

// ExpireMemories archives active memories that have TTL set and have expired, and authority below threshold (Task 75).
// Threshold is from Lifecycle.ExpirationAuthorityThreshold (0-10 scale; default 2). Returns count archived.
func (s *Service) ExpireMemories(ctx context.Context, asOf time.Time) (int, error) {
	threshold := 2
	if s.Lifecycle != nil && s.Lifecycle.ExpirationAuthorityThreshold > 0 {
		threshold = s.Lifecycle.ExpirationAuthorityThreshold
	}
	list, err := s.Repo.ListExpiredCandidates(ctx, threshold, asOf)
	if err != nil {
		return 0, err
	}
	archived := 0
	for _, m := range list {
		if err := s.Repo.UpdateStatus(ctx, m.ID, api.StatusArchived); err != nil {
			continue
		}
		archived++
	}
	if s.Cache != nil {
		_ = s.Cache.DeleteByPrefix(ctx, "memory:tags:")
		s.invalidateRecallBundleCache(ctx)
	}
	return archived, nil
}

// SetAttributes replaces all attributes for a memory (Task 78: constraint attributes for conflict detection).
func (s *Service) SetAttributes(ctx context.Context, memoryID uuid.UUID, attrs map[string]string) error {
	obj, err := s.Repo.GetByID(ctx, memoryID)
	if err != nil || obj == nil {
		return fmt.Errorf("memory not found")
	}
	if err := s.Repo.ReplaceAttributes(ctx, memoryID, attrs); err != nil {
		return err
	}
	if s.Cache != nil {
		_ = s.Cache.DeleteByPrefix(ctx, "memory:tags:")
		s.invalidateRecallBundleCache(ctx)
	}
	return nil
}

// Promote validates the promote contract.
// Phase C centralizes promotion into durable DB-backed memory.
func (s *Service) Promote(ctx context.Context, req PromoteRequest) (*PromoteResponse, error) {
	if s.Repo == nil {
		return nil, fmt.Errorf("memory repo not configured")
	}
	if req.Type == "" {
		return nil, fmt.Errorf("type is required")
	}
	if req.Content == "" {
		return nil, fmt.Errorf("content is required")
	}
	if req.Confidence < 0 || req.Confidence > 1 {
		return nil, fmt.Errorf("confidence must be between 0 and 1")
	}
	kind, err := mapPromoteType(req.Type)
	if err != nil {
		return nil, err
	}
	tags := append([]string{}, req.Tags...)
	if req.Source != "" {
		tags = append(tags, "source:"+req.Source)
	}
	tags = append(tags, "promoted")
	authority := int(req.Confidence * 10)
	if authority < 1 {
		authority = 1
	}
	if authority > 10 {
		authority = 10
	}
	if len(req.EvidenceIDs) > 0 && s.Evidence == nil {
		return nil, fmt.Errorf("evidence_ids requires evidence linker (server misconfiguration)")
	}
	cr := CreateRequest{
		Kind:          kind,
		Authority:     authority,
		Applicability: api.ApplicabilityAdvisory,
		Statement:     req.Content,
		Tags:          tags,
		OccurredAt:    req.OccurredAt,
	}
	if kind == api.MemoryKindConstraint {
		cr.Applicability = api.ApplicabilityGoverning
	}
	if req.RequireReview {
		cr.Status = api.StatusPending
	}
	obj, err := s.Create(ctx, cr)
	if err != nil {
		return nil, err
	}
	if len(req.EvidenceIDs) > 0 {
		if err := s.Evidence.LinkPromotedEvidence(ctx, obj.ID, req.EvidenceIDs); err != nil {
			return nil, err
		}
	}
	return &PromoteResponse{
		Promoted: true,
		ID:       obj.ID.String(),
		Reason:   "persisted via memory promote endpoint",
		Status:   string(obj.Status),
	}, nil
}

func kindStringsForSearchCache(kinds []api.MemoryKind) []string {
	if len(kinds) == 0 {
		return nil
	}
	out := make([]string, 0, len(kinds))
	for _, k := range kinds {
		if k != "" {
			out = append(out, string(k))
		}
	}
	sort.Strings(out)
	return out
}
