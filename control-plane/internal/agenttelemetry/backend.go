package agenttelemetry

import (
	"context"

	"github.com/google/uuid"
)

func (s *Service) usePostgres() bool {
	return s != nil && s.Repo != nil && s.Repo.useDB()
}

func (s *Service) insertSession(ctx context.Context, sess TelemetrySession) error {
	if s.usePostgres() {
		return s.Repo.insertSession(ctx, sess)
	}
	return s.mem.insertSession(ctx, sess)
}

func (s *Service) getSession(ctx context.Context, id uuid.UUID) (*TelemetrySession, bool) {
	if s.usePostgres() {
		return s.Repo.getSession(ctx, id)
	}
	return s.mem.getSession(ctx, id)
}

func (s *Service) insertRecall(ctx context.Context, ev RecallEvent) error {
	if s.usePostgres() {
		return s.Repo.insertRecall(ctx, ev)
	}
	return s.mem.insertRecall(ctx, ev)
}

func (s *Service) getRecall(ctx context.Context, id uuid.UUID) (*RecallEvent, bool) {
	if s.usePostgres() {
		return s.Repo.getRecall(ctx, id)
	}
	return s.mem.getRecall(ctx, id)
}

func (s *Service) listRecallsBySession(ctx context.Context, sid uuid.UUID) []RecallEvent {
	if s.usePostgres() {
		return s.Repo.listRecallsBySession(ctx, sid)
	}
	return s.mem.listRecallsBySession(ctx, sid)
}

func (s *Service) insertDecisions(ctx context.Context, recallID uuid.UUID, rows []MemoryDecisionRow) error {
	if s.usePostgres() {
		return s.Repo.insertDecisions(ctx, recallID, rows)
	}
	return s.mem.insertDecisions(ctx, recallID, rows)
}

func (s *Service) listDecisionsByRecall(ctx context.Context, recallID uuid.UUID) []MemoryDecisionRow {
	if s.usePostgres() {
		return s.Repo.listDecisionsByRecall(ctx, recallID)
	}
	return s.mem.listDecisionsByRecall(ctx, recallID)
}

func (s *Service) insertOutput(ctx context.Context, o OutputEvent) error {
	if s.usePostgres() {
		return s.Repo.insertOutput(ctx, o)
	}
	return s.mem.insertOutput(ctx, o)
}

func (s *Service) getOutput(ctx context.Context, id uuid.UUID) (*OutputEvent, bool) {
	if s.usePostgres() {
		return s.Repo.getOutput(ctx, id)
	}
	return s.mem.getOutput(ctx, id)
}

func (s *Service) listOutputsBySession(ctx context.Context, sid uuid.UUID) []OutputEvent {
	if s.usePostgres() {
		return s.Repo.listOutputsBySession(ctx, sid)
	}
	return s.mem.listOutputsBySession(ctx, sid)
}

func (s *Service) insertEval(ctx context.Context, e ObedienceEvaluationRow) error {
	if s.usePostgres() {
		return s.Repo.insertEval(ctx, e)
	}
	return s.mem.insertEval(ctx, e)
}

func (s *Service) insertViolations(ctx context.Context, evalID uuid.UUID, rows []ViolationRow) error {
	if s.usePostgres() {
		return s.Repo.insertViolations(ctx, evalID, rows)
	}
	return s.mem.insertViolations(ctx, evalID, rows)
}

func (s *Service) insertCandidates(ctx context.Context, evalID uuid.UUID, rows []UtilityCandidate) error {
	if s.usePostgres() {
		return s.Repo.insertCandidates(ctx, evalID, rows)
	}
	return s.mem.insertCandidates(ctx, evalID, rows)
}

func (s *Service) listAllViolations(ctx context.Context) []ViolationRow {
	if s.usePostgres() {
		return s.Repo.listAllViolations(ctx)
	}
	return s.mem.listAllViolations(ctx)
}

func (s *Service) listAllCandidates(ctx context.Context) []UtilityCandidate {
	if s.usePostgres() {
		return s.Repo.listAllCandidates(ctx)
	}
	return s.mem.listAllCandidates(ctx)
}

func (s *Service) sessionSummary(ctx context.Context, sid uuid.UUID) *SessionSummary {
	if s.usePostgres() {
		return s.Repo.sessionSummary(ctx, sid)
	}
	return s.mem.sessionSummary(ctx, sid)
}

func (s *Service) memorySummary(ctx context.Context, memoryID string) MemorySummary {
	if s.usePostgres() {
		return s.Repo.memorySummary(ctx, memoryID)
	}
	return s.mem.memorySummary(ctx, memoryID)
}
