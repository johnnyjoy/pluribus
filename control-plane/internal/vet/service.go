// Package vet implements immediate probationary memory formation at advisory ingest (no delayed path).
package vet

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"control-plane/internal/distillation"
	"control-plane/internal/formation"
	"control-plane/internal/memory"
	"control-plane/internal/similarity"
	"control-plane/pkg/api"
)

const (
	defaultMinRunes    = 12
	authorityStrongIngest = 2 // keyword / strong event signals
	authorityWeakIngest     = 1 // plausible-weak; ranking + reinforcement differentiate
	maxStatementRunes  = 2048
)

// Service wires memory creation to advisory experiences at ingest time only.
type Service struct {
	Memory     *memory.Service
	Episodes   *similarity.Repo
	Formation  *formation.Gate
}

type formationLink struct {
	memoryID string
	kind     string
}

// ProcessNewAdvisoryExperience runs inline qualification + probationary memory creation after POST /v1/advisory-episodes.
func (s *Service) ProcessNewAdvisoryExperience(ctx context.Context, rec *similarity.Record, deduplicated bool) {
	if s == nil || rec == nil {
		return
	}
	stmtKey := strings.TrimSpace(rec.SummaryText)
	if len([]rune(stmtKey)) > 120 {
		stmtKey = string([]rune(stmtKey)[:120]) + "…"
	}
	if deduplicated {
		slog.Info("[INGEST]", "outcome", "deduplicated", "advisory_experience_id", rec.ID.String(), "statement_key", stmtKey)
		return
	}
	link, skip, err := s.tryFormProbationary(ctx, rec, defaultMinRunes)
	if err != nil {
		slog.Warn("[INGEST]", "outcome", "memory_create_error", "advisory_experience_id", rec.ID.String(), "statement_key", stmtKey, "error", err.Error())
		_ = s.Episodes.SetFormationRejected(ctx, rec.ID, "memory_create_error")
		return
	}
	if skip == "no_qualifying_signal" || skip == "below_min_length" {
		reason := skip
		if err := s.Episodes.SetFormationRejected(ctx, rec.ID, reason); err != nil {
			slog.Warn("[INGEST]", "outcome", "reject_failed", "advisory_experience_id", rec.ID.String(), "error", err.Error())
			return
		}
		slog.Info("[INGEST]", "outcome", "rejected", "advisory_experience_id", rec.ID.String(), "statement_key", stmtKey, "reason", reason)
		return
	}
	if skip != "" || link == nil {
		slog.Info("[INGEST]", "outcome", "rejected", "advisory_experience_id", rec.ID.String(), "statement_key", stmtKey, "reason", skip)
		_ = s.Episodes.SetFormationRejected(ctx, rec.ID, skip)
		return
	}
	slog.Info("[INGEST]", "outcome", "accepted", "advisory_experience_id", rec.ID.String(), "memory_id", link.memoryID, "kind", link.kind, "statement_key", stmtKey, "signal", "probationary_memory")
}

func (s *Service) tryFormProbationary(ctx context.Context, ep *similarity.Record, minRunes int) (*formationLink, string, error) {
	stmt := strings.TrimSpace(ep.SummaryText)
	runes := []rune(stmt)
	if len(runes) > maxStatementRunes {
		stmt = string(runes[:maxStatementRunes])
		runes = []rune(stmt)
	}
	if len(runes) < minRunes {
		return nil, "below_min_length", nil
	}
	if s.Formation != nil {
		if reject, reason := s.Formation.RejectRecordExperienceSummary(stmt); reject {
			return nil, reason, nil
		}
	} else if formation.IsWeakRecordExperienceSummary(stmt, 4) {
		return nil, "junk_or_vague_summary", nil
	}
	kind, sigReason, ok, weak := distillation.QualifyForProbationaryMemory(stmt, ep.Tags)
	if !ok {
		return nil, "no_qualifying_signal", nil
	}
	auth := authorityStrongIngest
	if weak {
		auth = authorityWeakIngest
	}
	ingestMeta := map[string]any{
		"advisory_experience_id": ep.ID.String(),
		"advisory_episode_id":    ep.ID.String(),
		"ingest_source":          ep.Source,
		"probationary":           true,
		"signal_reason":          sigReason,
		"weak_signal":            weak,
	}
	// Provenance stamp (C2/Phase 3 attribution): record the authoring agent so
	// poisoned lessons are traceable (agent_id column, plus agent:<id> tag).
	agentID := strings.TrimSpace(ep.AgentID)
	if agentID == "" {
		for _, t := range ep.Tags {
			if rest, found := strings.CutPrefix(t, "agent:"); found && strings.TrimSpace(rest) != "" {
				agentID = strings.TrimSpace(rest)
				break
			}
		}
	}
	if agentID != "" {
		ingestMeta["agent_id"] = agentID
	}
	// Harmful-advice screen (C2): safety-negating imperative advice is stored
	// QUARANTINED — never surfaced by recall — pending review, so a single
	// poisoned record_experience cannot reach other agents.
	harmReason := formation.HarmfulAdviceReason(stmt)
	if harmReason != "" {
		ingestMeta["quarantine_reason"] = harmReason
	}
	payload := map[string]any{"pluribus_ingest": ingestMeta}
	pj, _ := json.Marshal(payload)
	tags := append([]string(nil), ep.Tags...)
	tags = append(tags, "probationary", "memory_ingest_inline")
	cr := memory.CreateRequest{
		Kind:          kind,
		Authority:     auth,
		Applicability: api.ApplicabilityAdvisory,
		Statement:     stmt,
		Tags:          uniqTags(tags),
		Payload:       (*json.RawMessage)(&pj),
		FormationPath: formation.PathProbationaryIngest,
		AgentID:       agentID,
	}
	if s.Formation != nil {
		in := &formation.CreateInput{
			Path:          formation.PathProbationaryIngest,
			Kind:          kind,
			Authority:     auth,
			Applicability: api.ApplicabilityAdvisory,
			Statement:     stmt,
			Tags:          cr.Tags,
		}
		if d, err := s.Formation.EvaluateCreateInput(in); err != nil {
			return nil, "formation_rejected", nil
		} else if d.Outcome == formation.OutcomeReject {
			return nil, d.Reason, nil
		}
		cr.Authority = in.Authority
		cr.Applicability = in.Applicability
		cr.Status = in.Status
	}
	if harmReason != "" {
		// Quarantine wins over any gate-assigned status.
		cr.Status = api.StatusQuarantined
	}
	if ep.OccurredAt != nil {
		cr.OccurredAt = ep.OccurredAt
	}
	obj, err := s.Memory.Create(ctx, cr)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "dedup") {
			slog.Info("[INGEST]", "outcome", "dedup_merge", "advisory_experience_id", ep.ID.String(), "statement_key", stmt, "error", err.Error())
		}
		return nil, "", err
	}
	if err := s.Episodes.SetRelatedMemoryID(ctx, ep.ID, obj.ID); err != nil {
		return nil, "", err
	}
	if harmReason != "" {
		slog.Warn("[INGEST]", "outcome", "quarantined", "advisory_experience_id", ep.ID.String(),
			"memory_id", obj.ID.String(), "reason", harmReason)
	}
	return &formationLink{memoryID: obj.ID.String(), kind: string(kind)}, "", nil
}

func uniqTags(in []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, t := range in {
		t = strings.TrimSpace(strings.ToLower(t))
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}
