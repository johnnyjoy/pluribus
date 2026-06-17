package agenttelemetry

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// memStore is in-memory persistence for tests and nil-DB deployments.
type memStore struct {
	mu          sync.RWMutex
	sessions    map[uuid.UUID]TelemetrySession
	recalls     map[uuid.UUID]RecallEvent
	decisions   map[uuid.UUID][]MemoryDecisionRow
	outputs     map[uuid.UUID]OutputEvent
	evals       map[uuid.UUID]ObedienceEvaluationRow
	violations  map[uuid.UUID][]ViolationRow
	candidates  map[uuid.UUID][]UtilityCandidate
}

func newMemStore() *memStore {
	return &memStore{
		sessions:   map[uuid.UUID]TelemetrySession{},
		recalls:    map[uuid.UUID]RecallEvent{},
		decisions:  map[uuid.UUID][]MemoryDecisionRow{},
		outputs:    map[uuid.UUID]OutputEvent{},
		evals:      map[uuid.UUID]ObedienceEvaluationRow{},
		violations: map[uuid.UUID][]ViolationRow{},
		candidates: map[uuid.UUID][]UtilityCandidate{},
	}
}

func (m *memStore) insertSession(_ context.Context, s TelemetrySession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.ID] = s
	return nil
}

func (m *memStore) getSession(_ context.Context, id uuid.UUID) (*TelemetrySession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, false
	}
	return &s, true
}

func (m *memStore) insertRecall(_ context.Context, r RecallEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recalls[r.ID] = r
	return nil
}

func (m *memStore) getRecall(_ context.Context, id uuid.UUID) (*RecallEvent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.recalls[id]
	if !ok {
		return nil, false
	}
	return &r, true
}

func (m *memStore) listRecallsBySession(_ context.Context, sid uuid.UUID) []RecallEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []RecallEvent
	for _, r := range m.recalls {
		if r.SessionID == sid {
			out = append(out, r)
		}
	}
	return out
}

func (m *memStore) insertDecisions(_ context.Context, recallID uuid.UUID, rows []MemoryDecisionRow) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.decisions[recallID] = append(m.decisions[recallID], rows...)
	return nil
}

func (m *memStore) listDecisionsByRecall(_ context.Context, recallID uuid.UUID) []MemoryDecisionRow {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]MemoryDecisionRow(nil), m.decisions[recallID]...)
}

func (m *memStore) insertOutput(_ context.Context, o OutputEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outputs[o.ID] = o
	return nil
}

func (m *memStore) getOutput(_ context.Context, id uuid.UUID) (*OutputEvent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	o, ok := m.outputs[id]
	if !ok {
		return nil, false
	}
	return &o, true
}

func (m *memStore) listOutputsBySession(_ context.Context, sid uuid.UUID) []OutputEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []OutputEvent
	for _, o := range m.outputs {
		if o.SessionID == sid {
			out = append(out, o)
		}
	}
	return out
}

func (m *memStore) insertEval(_ context.Context, e ObedienceEvaluationRow) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.evals[e.ID] = e
	return nil
}

func (m *memStore) listEvalsBySession(_ context.Context, sid uuid.UUID) []ObedienceEvaluationRow {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []ObedienceEvaluationRow
	for _, e := range m.evals {
		if e.SessionID == sid {
			out = append(out, e)
		}
	}
	return out
}

func (m *memStore) insertViolations(_ context.Context, evalID uuid.UUID, rows []ViolationRow) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.violations[evalID] = append(m.violations[evalID], rows...)
	return nil
}

func (m *memStore) listAllViolations(_ context.Context) []ViolationRow {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []ViolationRow
	for _, vs := range m.violations {
		out = append(out, vs...)
	}
	return out
}

func (m *memStore) insertCandidates(_ context.Context, evalID uuid.UUID, rows []UtilityCandidate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.candidates[evalID] = append(m.candidates[evalID], rows...)
	return nil
}

func (m *memStore) listAllCandidates(_ context.Context) []UtilityCandidate {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []UtilityCandidate
	for _, cs := range m.candidates {
		out = append(out, cs...)
	}
	return out
}

func (m *memStore) sessionSummary(_ context.Context, sid uuid.UUID) *SessionSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sess, ok := m.sessions[sid]
	if !ok {
		return nil
	}
	sum := &SessionSummary{Session: sess}
	for _, r := range m.recalls {
		if r.SessionID == sid {
			sum.RecallEvents = append(sum.RecallEvents, r)
			sum.Decisions = append(sum.Decisions, m.decisions[r.ID]...)
		}
	}
	for _, o := range m.outputs {
		if o.SessionID == sid {
			sum.Outputs = append(sum.Outputs, o)
		}
	}
	for _, e := range m.evals {
		if e.SessionID == sid {
			sum.Evaluations = append(sum.Evaluations, e)
			sum.Violations = append(sum.Violations, m.violations[e.ID]...)
			sum.Candidates = append(sum.Candidates, m.candidates[e.ID]...)
		}
	}
	return sum
}

func (m *memStore) memorySummary(_ context.Context, memoryID string) MemorySummary {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sum := MemorySummary{MemoryID: memoryID}
	var pass, total int
	for _, r := range m.recalls {
		for _, mid := range r.RecalledMemoryIDs {
			if mid == memoryID {
				sum.RecallCount++
			}
		}
	}
	for _, rows := range m.decisions {
		for _, d := range rows {
			if d.MemoryID != memoryID {
				continue
			}
			switch d.Decision {
			case "used":
				sum.UsedCount++
			case "ignored", "historical_only":
				sum.IgnoredCount++
			}
		}
	}
	for _, vs := range m.violations {
		for _, v := range vs {
			if v.MemoryID == memoryID {
				sum.ViolationCount++
			}
		}
	}
	for _, e := range m.evals {
		total++
		if e.ObediencePassed {
			pass++
		}
	}
	if total > 0 {
		sum.ObediencePassRate = float64(pass) / float64(total)
	}
	return sum
}
