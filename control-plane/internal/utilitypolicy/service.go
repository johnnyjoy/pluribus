package utilitypolicy

import (
	"context"
	"strings"
	"time"

	"control-plane/internal/agenttelemetry"
	"control-plane/internal/utility"

	"github.com/google/uuid"
)

// TelemetryReader loads persisted utility candidates and evaluations.
type TelemetryReader interface {
	GetCandidate(ctx context.Context, id uuid.UUID) (*agenttelemetry.UtilityCandidate, error)
	GetEvaluation(ctx context.Context, id uuid.UUID) (*agenttelemetry.ObedienceEvaluationRow, error)
}

// ScoreStore applies bounded score changes for policy decisions.
type ScoreStore interface {
	GetScore(ctx context.Context, memoryID string) (float64, error)
	SetScore(ctx context.Context, memoryID string, score float64) error
}

// ApplicationBackend persists application audit rows.
type ApplicationBackend interface {
	HasApplication(ctx context.Context, candidateID uuid.UUID) (bool, error)
	CountPositiveBySession(ctx context.Context, sessionID uuid.UUID, memoryID string) (int, error)
	CountPositiveByAgent(ctx context.Context, agentID, memoryID string) (int, error)
	InsertApplication(ctx context.Context, rec ApplicationRecord) error
	GetByRollbackToken(ctx context.Context, token string) (*ApplicationRecord, error)
	MarkReverted(ctx context.Context, applicationID uuid.UUID, reason string) error
	ListByMemory(ctx context.Context, memoryID string, limit int) ([]ApplicationRecord, error)
	ListAll(ctx context.Context, limit int) ([]ApplicationRecord, error)
	Summary(ctx context.Context) (PolicySummary, error)
}

// Service orchestrates guarded utility policy evaluation and application.
type Service struct {
	Telemetry TelemetryReader
	Scores    ScoreStore
	Apps      ApplicationBackend
	Mem       *memStore
	Config    PolicyConfig
	Utility   *utility.Service
}

// NewTestService returns an in-memory policy service for hostile fixtures.
func NewTestService() *Service {
	mem := newMemStore()
	return &Service{
		Mem:    mem,
		Scores: mem,
		Apps:   &memAppBackend{mem},
		Config: DefaultPolicyConfig(),
	}
}

type memAppBackend struct{ m *memStore }

func (b *memAppBackend) HasApplication(ctx context.Context, id uuid.UUID) (bool, error) {
	return b.m.hasApplication(ctx, id), nil
}
func (b *memAppBackend) CountPositiveBySession(ctx context.Context, sid uuid.UUID, mid string) (int, error) {
	return b.m.sessionPositiveCount(ctx, sid, mid), nil
}
func (b *memAppBackend) CountPositiveByAgent(ctx context.Context, aid, mid string) (int, error) {
	return b.m.agentPositiveCount(ctx, aid, mid), nil
}
func (b *memAppBackend) InsertApplication(ctx context.Context, rec ApplicationRecord) error {
	return b.m.insertApplication(ctx, rec)
}
func (b *memAppBackend) GetByRollbackToken(ctx context.Context, token string) (*ApplicationRecord, error) {
	rec, ok := b.m.getApplicationByRollback(ctx, token)
	if !ok {
		return nil, ErrApplicationNotFound
	}
	return &rec, nil
}
func (b *memAppBackend) MarkReverted(ctx context.Context, id uuid.UUID, reason string) error {
	for cid, rec := range b.m.applications {
		if rec.ApplicationID == id {
			now := time.Now().UTC()
			rec.RevertedAt = &now
			rec.RevertReason = reason
			b.m.applications[cid] = rec
			return nil
		}
	}
	return ErrApplicationNotFound
}
func (b *memAppBackend) ListByMemory(ctx context.Context, memoryID string, limit int) ([]ApplicationRecord, error) {
	return b.m.listApplications(ctx, memoryID), nil
}
func (b *memAppBackend) ListAll(ctx context.Context, limit int) ([]ApplicationRecord, error) {
	return b.m.listApplications(ctx, ""), nil
}
func (b *memAppBackend) Summary(ctx context.Context) (PolicySummary, error) {
	return b.m.summary(ctx), nil
}

// SeedCandidate registers a candidate for test/policy flows.
func (s *Service) SeedCandidate(c CandidateInput) CandidateInput {
	if s.Mem == nil {
		return c
	}
	if c.CandidateID == uuid.Nil {
		c.CandidateID = uuid.New()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	s.Mem.putCandidate(c)
	return c
}

// SetInitialScore seeds score for test memory keys.
func (s *Service) SetInitialScore(memoryID string, score float64) {
	if s.Scores != nil {
		_ = s.Scores.SetScore(context.Background(), memoryID, score)
	}
}

func (s *Service) cfg() PolicyConfig {
	if s.Config.MaxPositiveDelta == 0 {
		return DefaultPolicyConfig()
	}
	return s.Config
}

func (s *Service) loadCandidate(ctx context.Context, id uuid.UUID) (CandidateInput, error) {
	if s.Mem != nil {
		if c, ok := s.Mem.getCandidate(ctx, id); ok {
			return c, nil
		}
	}
	if s.Telemetry != nil {
		raw, err := s.Telemetry.GetCandidate(ctx, id)
		if err != nil {
			return CandidateInput{}, err
		}
		if raw == nil {
			return CandidateInput{}, ErrCandidateNotFound
		}
		evalPassed := false
		var sessionID uuid.UUID
		var agentID string
		if raw.EvaluationID != uuid.Nil && s.Telemetry != nil {
			if ev, err := s.Telemetry.GetEvaluation(ctx, raw.EvaluationID); err == nil && ev != nil {
				evalPassed = ev.ObediencePassed
				sessionID = ev.SessionID
			}
		}
		return CandidateInput{
			CandidateID:     raw.ID,
			MemoryID:        raw.MemoryID,
			EvaluationID:    raw.EvaluationID,
			SignalType:      raw.SignalType,
			SignalStrength:  raw.SignalStrength,
			SafeToApply:     raw.SafeToApply,
			Reason:          raw.Reason,
			EvaluatorPassed: evalPassed,
			SessionID:       sessionID,
			AgentID:         agentID,
			CreatedAt:       raw.CreatedAt,
		}, nil
	}
	return CandidateInput{}, ErrCandidateNotFound
}

func (s *Service) policyContext(ctx context.Context, c CandidateInput, tampered bool) (PolicyContext, error) {
	pctx := PolicyContext{Tampered: tampered}
	if s.Apps != nil {
		applied, err := s.Apps.HasApplication(ctx, c.CandidateID)
		if err != nil {
			return pctx, err
		}
		pctx.AlreadyApplied = applied
		if c.SessionID != uuid.Nil && c.MemoryID != "" {
			n, err := s.Apps.CountPositiveBySession(ctx, c.SessionID, c.MemoryID)
			if err != nil {
				return pctx, err
			}
			pctx.SessionPositiveCount = n
		}
		if c.AgentID != "" && c.MemoryID != "" {
			n, err := s.Apps.CountPositiveByAgent(ctx, c.AgentID, c.MemoryID)
			if err != nil {
				return pctx, err
			}
			pctx.AgentPositiveCount = n
		}
	}
	return pctx, nil
}

// EvaluateCandidate returns policy decision without mutation.
func (s *Service) EvaluateCandidate(ctx context.Context, candidateID uuid.UUID, tampered bool) (PolicyDecision, error) {
	if s == nil {
		return PolicyDecision{}, ErrNoService
	}
	c, err := s.loadCandidate(ctx, candidateID)
	if err != nil {
		return PolicyDecision{}, err
	}
	pctx, err := s.policyContext(ctx, c, tampered)
	if err != nil {
		return PolicyDecision{}, err
	}
	return EvaluatePolicy(c, s.cfg(), pctx), nil
}

// ApplyCandidate evaluates and persists application; mutates score only for apply_positive/apply_negative.
func (s *Service) ApplyCandidate(ctx context.Context, candidateID uuid.UUID, appliedBy string, tampered bool) (ApplicationRecord, error) {
	if s == nil {
		return ApplicationRecord{}, ErrNoService
	}
	c, err := s.loadCandidate(ctx, candidateID)
	if err != nil {
		return ApplicationRecord{}, err
	}
	pctx, err := s.policyContext(ctx, c, tampered)
	if err != nil {
		return ApplicationRecord{}, err
	}
	if pctx.AlreadyApplied {
		return ApplicationRecord{}, ErrAlreadyApplied
	}
	dec := EvaluatePolicy(c, s.cfg(), pctx)

	prev := 0.0
	if s.Scores != nil && c.MemoryID != "" {
		prev, _ = s.Scores.GetScore(ctx, c.MemoryID)
	}
	next := prev
	if dec.Decision == DecisionApplyPositive || dec.Decision == DecisionApplyNegative {
		next = ComputeNewScore(prev, dec.Delta)
	}

	rec := ApplicationRecord{
		ApplicationID:        uuid.New(),
		CandidateID:          c.CandidateID,
		MemoryID:             c.MemoryID,
		EvaluationID:         c.EvaluationID,
		Decision:             dec.Decision,
		Delta:                dec.Delta,
		PreviousUtilityScore: prev,
		NewUtilityScore:      next,
		PolicyVersion:        dec.PolicyVersion,
		Reason:               dec.Reason,
		Evidence:             dec.Evidence,
		RollbackToken:        dec.RollbackToken,
		AppliedBy:            strings.TrimSpace(appliedBy),
		SessionID:            c.SessionID,
		AgentID:              c.AgentID,
		CreatedAt:            time.Now().UTC(),
	}
	if rec.AppliedBy == "" {
		rec.AppliedBy = "system"
	}

	if s.Apps == nil {
		return ApplicationRecord{}, ErrNoService
	}
	if err := s.Apps.InsertApplication(ctx, rec); err != nil {
		return ApplicationRecord{}, err
	}

	if dec.Decision == DecisionApplyPositive || dec.Decision == DecisionApplyNegative {
		if err := s.applyScoreMutation(ctx, c, dec, prev, next); err != nil {
			return ApplicationRecord{}, err
		}
	}
	return rec, nil
}

func (s *Service) applyScoreMutation(ctx context.Context, c CandidateInput, dec PolicyDecision, prev, next float64) error {
	if s.Scores != nil {
		return s.Scores.SetScore(ctx, c.MemoryID, next)
	}
	return nil
}

// ApplyBatch applies multiple candidates respecting caps sequentially.
func (s *Service) ApplyBatch(ctx context.Context, ids []uuid.UUID, appliedBy string) ([]ApplicationRecord, error) {
	var out []ApplicationRecord
	for _, id := range ids {
		rec, err := s.ApplyCandidate(ctx, id, appliedBy, false)
		if err != nil && err != ErrAlreadyApplied && err != ErrSessionCapExceeded && err != ErrAgentCapExceeded {
			return out, err
		}
		if err == nil {
			out = append(out, rec)
		}
	}
	return out, nil
}

// RevertApplication restores prior score using rollback token.
func (s *Service) RevertApplication(ctx context.Context, token, reason, appliedBy string) (ApplicationRecord, error) {
	if s == nil || s.Apps == nil {
		return ApplicationRecord{}, ErrNoService
	}
	rec, err := s.Apps.GetByRollbackToken(ctx, token)
	if err != nil {
		return ApplicationRecord{}, err
	}
	if rec.RevertedAt != nil {
		return ApplicationRecord{}, ErrAlreadyReverted
	}
	if rec.Decision != DecisionApplyPositive && rec.Decision != DecisionApplyNegative {
		return ApplicationRecord{}, ErrScoreMutationDenied
	}
	if s.Scores != nil && rec.MemoryID != "" {
		_ = s.Scores.SetScore(ctx, rec.MemoryID, rec.PreviousUtilityScore)
	}
	now := time.Now().UTC()
	rec.RevertedAt = &now
	rec.RevertReason = strings.TrimSpace(reason)
	if rec.RevertReason == "" {
		rec.RevertReason = "policy revert"
	}
	if err := s.Apps.MarkReverted(ctx, rec.ApplicationID, rec.RevertReason); err != nil {
		return ApplicationRecord{}, err
	}
	return *rec, nil
}

// GetCandidateDecision returns latest application for candidate if any.
func (s *Service) GetCandidateApplication(ctx context.Context, candidateID uuid.UUID) (*ApplicationRecord, error) {
	if s.Apps == nil {
		return nil, ErrNoService
	}
	all, err := s.Apps.ListAll(ctx, 500)
	if err != nil {
		return nil, err
	}
	for _, rec := range all {
		if rec.CandidateID == candidateID {
			r := rec
			return &r, nil
		}
	}
	return nil, ErrApplicationNotFound
}

// ListMemoryApplications returns history for memory.
func (s *Service) ListMemoryApplications(ctx context.Context, memoryID string) ([]ApplicationRecord, error) {
	if s.Apps == nil {
		return nil, ErrNoService
	}
	rows, err := s.Apps.ListByMemory(ctx, memoryID, 200)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []ApplicationRecord{}
	}
	return rows, nil
}

// ListApplications returns recent applications.
func (s *Service) ListApplications(ctx context.Context) ([]ApplicationRecord, error) {
	if s.Apps == nil {
		return nil, ErrNoService
	}
	rows, err := s.Apps.ListAll(ctx, 200)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []ApplicationRecord{}
	}
	return rows, nil
}

// Summary returns policy summary.
func (s *Service) Summary(ctx context.Context) (PolicySummary, error) {
	if s.Apps == nil {
		return PolicySummary{}, ErrNoService
	}
	return s.Apps.Summary(ctx)
}

// NewServiceWithDeps wires production dependencies.
func NewServiceWithDeps(telemetry TelemetryReader, utilitySvc *utility.Service, repo *Repo) *Service {
	var scores ScoreStore
	if utilitySvc != nil && utilitySvc.Repo != nil {
		scores = &utilityScoreStore{svc: utilitySvc}
	}
	var apps ApplicationBackend
	if repo != nil && repo.useDB() {
		apps = repo
	} else {
		apps = &memAppBackend{newMemStore()}
	}
	return &Service{
		Telemetry: telemetry,
		Scores:    scores,
		Apps:      apps,
		Utility:   utilitySvc,
		Config:    DefaultPolicyConfig(),
	}
}

type utilityScoreStore struct{ svc *utility.Service }

func (u *utilityScoreStore) GetScore(ctx context.Context, memoryID string) (float64, error) {
	if strings.HasPrefix(memoryID, "test:") {
		return 0, nil
	}
	id, err := uuid.Parse(memoryID)
	if err != nil {
		return 0, nil
	}
	sc, err := u.svc.GetUtilityScore(ctx, id)
	if err != nil {
		return 0, nil
	}
	return sc.UtilityScore, nil
}

func (u *utilityScoreStore) SetScore(ctx context.Context, memoryID string, score float64) error {
	if strings.HasPrefix(memoryID, "test:") {
		return nil
	}
	id, err := uuid.Parse(memoryID)
	if err != nil {
		return err
	}
	sc, err := u.svc.Repo.GetScore(ctx, id)
	if err != nil {
		return err
	}
	if sc == nil {
		sc = &utility.Score{MemoryID: id}
	}
	sc.UtilityScore = ApplyScoreBounds(score)
	_, err = u.svc.Repo.UpsertScore(ctx, *sc)
	return err
}
