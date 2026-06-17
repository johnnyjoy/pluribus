package utility

import (
	"context"
	"fmt"
	"strings"
	"time"

	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// MemoryReader checks memory existence and optional status updates.
type MemoryReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (exists bool, kind api.MemoryKind, applicability api.Applicability, status api.Status, err error)
	SetStatusPending(ctx context.Context, id uuid.UUID) error
}

// ComplianceRecorder optionally logs feedback to agent loop telemetry.
type ComplianceRecorder interface {
	RecordMemoryFeedback(ctx context.Context, memoryID uuid.UUID, eventType, correlationID string)
}

// Service records utility feedback and maintains scores.
type Service struct {
	Repo       *Repo
	Memory     MemoryReader
	Compliance ComplianceRecorder
	Config     *Config
}

// ValidateFeedbackRequest checks REST/MCP shared rules.
func ValidateFeedbackRequest(req FeedbackRequest) error {
	et := strings.ToLower(strings.TrimSpace(req.EventType))
	if !ExposedEventTypes[et] {
		return ErrInvalidEventType
	}
	if NegativeEventTypes[et] && strings.TrimSpace(req.Reason) == "" {
		return ErrReasonRequired
	}
	return nil
}

// RecordFeedback appends event and updates bounded utility score.
func (s *Service) RecordFeedback(ctx context.Context, memoryID uuid.UUID, req FeedbackRequest) (*FeedbackResponse, error) {
	return s.recordFeedback(ctx, memoryID, req, true)
}

// recordSystemEvent records internal events (contradiction refuted) bypassing exposed-type check.
func (s *Service) recordSystemEvent(ctx context.Context, memoryID uuid.UUID, req FeedbackRequest) (*FeedbackResponse, error) {
	return s.recordFeedback(ctx, memoryID, req, false)
}

func (s *Service) recordFeedback(ctx context.Context, memoryID uuid.UUID, req FeedbackRequest, requireExposed bool) (*FeedbackResponse, error) {
	if s == nil || s.Repo == nil {
		return nil, ErrNoRepo
	}
	et := strings.ToLower(strings.TrimSpace(req.EventType))
	if requireExposed {
		if err := ValidateFeedbackRequest(req); err != nil {
			return nil, err
		}
	} else {
		if NegativeEventTypes[et] && strings.TrimSpace(req.Reason) == "" {
			return nil, ErrReasonRequired
		}
		if EventWeight(et) == 0 && et != EventRefuted && et != EventDuplicateSeen {
			return nil, ErrInvalidEventType
		}
	}
	exists, kind, applicability, status, err := s.memoryExists(ctx, memoryID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrMemoryNotFound
	}
	_ = kind
	_ = applicability
	_ = status

	baseW := EventWeight(et)
	sameCount, err := s.Repo.CountEventsByType(ctx, memoryID, et)
	if err != nil {
		return nil, err
	}
	weight := DiminishingWeight(baseW, sameCount)

	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "agent"
	}

	ev, err := s.Repo.InsertEvent(ctx, Event{
		MemoryID:        memoryID,
		EventType:       et,
		EventWeight:     weight,
		Source:          source,
		SourceTool:      strings.TrimSpace(req.SourceTool),
		SourceSessionID: strings.TrimSpace(req.SourceSessionID),
		CorrelationID:   strings.TrimSpace(req.CorrelationID),
		RecallBundleID:  strings.TrimSpace(req.RecallBundleID),
		Reason:          strings.TrimSpace(req.Reason),
		EvidenceID:      req.EvidenceID,
		Payload:         req.Payload,
	})
	if err != nil {
		return nil, err
	}

	score, err := s.Repo.GetScore(ctx, memoryID)
	if err != nil {
		return nil, err
	}
	if score == nil {
		score = &Score{MemoryID: memoryID}
	}
	score.UtilityScore = ApplyScoreDelta(score.UtilityScore, weight)
	ApplyCountField(score, et)
	now := time.Now().UTC()
	if IsPositiveEvent(et) {
		score.LastPositiveAt = &now
	}
	if IsNegativeEvent(et) {
		score.LastNegativeAt = &now
	}
	updated, err := s.Repo.UpsertScore(ctx, *score)
	if err != nil {
		return nil, err
	}

	if s.Compliance != nil {
		s.Compliance.RecordMemoryFeedback(ctx, memoryID, et, req.CorrelationID)
	}

	return &FeedbackResponse{
		MemoryID:        memoryID,
		EventID:         ev.ID,
		EventType:       et,
		NewUtilityScore: updated.UtilityScore,
		Counts:          *updated,
	}, nil
}

// GetUtilityScore returns aggregate score for memory.
func (s *Service) GetUtilityScore(ctx context.Context, memoryID uuid.UUID) (*Score, error) {
	if s == nil || s.Repo == nil {
		return nil, ErrNoRepo
	}
	exists, _, _, _, err := s.memoryExists(ctx, memoryID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrMemoryNotFound
	}
	return s.Repo.GetScore(ctx, memoryID)
}

// ListUtilityEvents lists feedback events for memory.
func (s *Service) ListUtilityEvents(ctx context.Context, memoryID uuid.UUID, limit int) ([]Event, error) {
	if s == nil || s.Repo == nil {
		return nil, ErrNoRepo
	}
	exists, _, _, _, err := s.memoryExists(ctx, memoryID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrMemoryNotFound
	}
	list, err := s.Repo.ListEvents(ctx, memoryID, limit)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []Event{}
	}
	return list, nil
}

// GetScoresForMemories implements recall.UtilityScoreProvider.
func (s *Service) GetScoresForMemories(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]float64, error) {
	if s == nil || s.Repo == nil {
		return map[uuid.UUID]float64{}, nil
	}
	return s.Repo.GetScoresForMemories(ctx, ids)
}

// GetUtilitySummaries implements recall.UtilitySummaryProvider.
func (s *Service) GetUtilitySummaries(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Score, error) {
	if s == nil || s.Repo == nil {
		return map[uuid.UUID]Score{}, nil
	}
	return s.Repo.GetSummariesForMemories(ctx, ids)
}

// RecordContradictionDemotion records refuted events for both memories in a contradiction pair.
func (s *Service) RecordContradictionDemotion(ctx context.Context, memoryID, conflictWithID uuid.UUID) error {
	reason := fmt.Sprintf("contradiction_record: conflict_with=%s", conflictWithID)
	for _, id := range []uuid.UUID{memoryID, conflictWithID} {
		if id == uuid.Nil {
			continue
		}
		_, err := s.recordSystemEvent(ctx, id, FeedbackRequest{
			EventType: EventRefuted,
			Reason:    reason,
			Source:    "system",
			SourceTool: "contradiction",
		})
		if err != nil && err != ErrMemoryNotFound {
			return err
		}
	}
	// Optional pending for governing constraint on primary memory
	if s.Memory != nil {
		exists, kind, app, st, err := s.memoryExists(ctx, memoryID)
		if err == nil && exists && kind == api.MemoryKindConstraint &&
			app == api.ApplicabilityGoverning && st == api.StatusActive {
			_ = s.Memory.SetStatusPending(ctx, memoryID)
		}
	}
	return nil
}

func (s *Service) memoryExists(ctx context.Context, id uuid.UUID) (bool, api.MemoryKind, api.Applicability, api.Status, error) {
	if s.Memory != nil {
		return s.Memory.GetByID(ctx, id)
	}
	ok, err := s.Repo.MemoryExists(ctx, id)
	return ok, "", "", "", err
}
