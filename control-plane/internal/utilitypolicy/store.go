package utilitypolicy

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// memStore is an in-memory application and score store for tests.
type memStore struct {
	mu           sync.Mutex
	candidates   map[uuid.UUID]CandidateInput
	applications map[uuid.UUID]ApplicationRecord
	byRollback   map[string]uuid.UUID
	scores       map[string]float64
	posSession   map[string]int // session|memory -> count
	posAgent     map[string]int // agent|memory -> count
}

func newMemStore() *memStore {
	return &memStore{
		candidates:   map[uuid.UUID]CandidateInput{},
		applications: map[uuid.UUID]ApplicationRecord{},
		byRollback:   map[string]uuid.UUID{},
		scores:       map[string]float64{},
		posSession:   map[string]int{},
		posAgent:     map[string]int{},
	}
}

func (m *memStore) putCandidate(c CandidateInput) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c.CandidateID == uuid.Nil {
		c.CandidateID = uuid.New()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	m.candidates[c.CandidateID] = c
}

func (m *memStore) getCandidate(_ context.Context, id uuid.UUID) (CandidateInput, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.candidates[id]
	return c, ok
}

func (m *memStore) hasApplication(_ context.Context, candidateID uuid.UUID) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.applications[candidateID]
	return ok
}

func (m *memStore) insertApplication(_ context.Context, rec ApplicationRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, dup := m.applications[rec.CandidateID]; dup {
		return ErrAlreadyApplied
	}
	m.applications[rec.CandidateID] = rec
	if rec.RollbackToken != "" {
		m.byRollback[rec.RollbackToken] = rec.ApplicationID
	}
	if rec.Decision == DecisionApplyPositive {
		sk := rec.SessionID.String() + "|" + rec.MemoryID
		m.posSession[sk]++
		ak := rec.AgentID + "|" + rec.MemoryID
		m.posAgent[ak]++
	}
	return nil
}

func (m *memStore) getApplicationByRollback(_ context.Context, token string) (ApplicationRecord, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	appID, ok := m.byRollback[token]
	if !ok {
		return ApplicationRecord{}, false
	}
	for _, rec := range m.applications {
		if rec.ApplicationID == appID {
			return rec, true
		}
	}
	return ApplicationRecord{}, false
}

func (m *memStore) updateApplication(_ context.Context, rec ApplicationRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.applications[rec.CandidateID] = rec
	return nil
}

func (m *memStore) listApplications(_ context.Context, memoryID string) []ApplicationRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []ApplicationRecord
	for _, rec := range m.applications {
		if memoryID == "" || rec.MemoryID == memoryID {
			out = append(out, rec)
		}
	}
	return out
}

func (m *memStore) GetScore(_ context.Context, memoryID string) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.scores[memoryID], nil
}

func (m *memStore) SetScore(_ context.Context, memoryID string, score float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scores[memoryID] = ApplyScoreBounds(score)
	return nil
}

func (m *memStore) sessionPositiveCount(_ context.Context, sessionID uuid.UUID, memoryID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.posSession[sessionID.String()+"|"+memoryID]
}

func (m *memStore) agentPositiveCount(_ context.Context, agentID, memoryID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.posAgent[agentID+"|"+memoryID]
}

func (m *memStore) summary(_ context.Context) PolicySummary {
	m.mu.Lock()
	defer m.mu.Unlock()
	var s PolicySummary
	s.PolicyVersion = PolicyVersion
	for _, rec := range m.applications {
		s.TotalApplications++
		switch rec.Decision {
		case DecisionApplyPositive:
			s.PositiveApplications++
		case DecisionApplyNegative:
			s.NegativeApplications++
		case DecisionRecordOnly:
			s.RecordOnlyCount++
		case DecisionReviewRequired:
			s.ReviewRequiredCount++
		case DecisionReject:
			s.RejectedCount++
		}
		if rec.RevertedAt != nil {
			s.RevertedCount++
		}
	}
	return s
}
