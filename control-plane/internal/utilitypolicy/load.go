package utilitypolicy

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// PolicyCase is one hostile guarded utility policy fixture.
type PolicyCase struct {
	CaseID                        string         `json:"case_id"`
	CandidateInput                CandidateInput `json:"candidate_input"`
	ExistingMemoryUtility         float64        `json:"existing_memory_utility"`
	PolicyConfig                  *PolicyConfig  `json:"policy_config,omitempty"`
	ExpectedPolicyDecision        string         `json:"expected_policy_decision"`
	ExpectedDelta                 float64        `json:"expected_delta"`
	ExpectedScoreChange           bool           `json:"expected_score_change"`
	ExpectedApplicationRecord     bool           `json:"expected_application_record"`
	ExpectedUtilityEvent          bool           `json:"expected_utility_event,omitempty"`
	ExpectedRejectionOrReviewReason string       `json:"expected_rejection_or_review_reason,omitempty"`
	ExpectedMCPRESTParity         bool           `json:"expected_mcp_rest_parity,omitempty"`
	ExpectedPostgresBehavior      bool           `json:"expected_postgres_behavior,omitempty"`
	Tampered                      bool           `json:"tampered,omitempty"`
	DuplicateApply                bool           `json:"duplicate_apply,omitempty"`
	PreSeedPositiveCount          int            `json:"pre_seed_positive_count,omitempty"`
	Interface                     string         `json:"interface,omitempty"`
}

type policyCasesFile struct {
	Cases []PolicyCase `json:"cases"`
}

// LoadPolicyCases loads hostile utility policy fixtures.
func LoadPolicyCases(path string) ([]PolicyCase, error) {
	if path == "" {
		path = filepath.Join(repoRoot(), "control-plane", "testdata", "guarded_utility_policy", "cases.json")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f policyCasesFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, err
	}
	rebaseCaseTimestamps(f.Cases)
	return f.Cases, nil
}

// rebaseCaseTimestamps shifts every fixture created_at so the newest one is
// "now", preserving relative ages. The fixture stores absolute timestamps from
// generation day; without rebasing, the stale-candidate window (7d) eventually
// marks EVERY case stale and the whole suite silently degrades into
// review_required (time-bomb found 2026-07).
func rebaseCaseTimestamps(cases []PolicyCase) {
	var newest time.Time
	for _, c := range cases {
		if ts := c.CandidateInput.CreatedAt; !ts.IsZero() && ts.After(newest) {
			newest = ts
		}
	}
	if newest.IsZero() {
		return
	}
	shift := time.Now().UTC().Sub(newest)
	for i := range cases {
		if ts := cases[i].CandidateInput.CreatedAt; !ts.IsZero() {
			cases[i].CandidateInput.CreatedAt = ts.Add(shift)
		}
	}
}

func repoRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "control-plane", "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}

// PrepareCase seeds service state for one fixture case.
func PrepareCase(svc *Service, c PolicyCase) CandidateInput {
	cfg := DefaultPolicyConfig()
	if c.PolicyConfig != nil {
		over := *c.PolicyConfig
		if over.MaxPositiveDelta > 0 {
			cfg.MaxPositiveDelta = over.MaxPositiveDelta
		}
		if over.MaxNegativeDelta > 0 {
			cfg.MaxNegativeDelta = over.MaxNegativeDelta
		}
		if over.MaxPositivePerSessionPerMemory > 0 {
			cfg.MaxPositivePerSessionPerMemory = over.MaxPositivePerSessionPerMemory
		}
		if over.MaxPositivePerAgentPerMemory > 0 {
			cfg.MaxPositivePerAgentPerMemory = over.MaxPositivePerAgentPerMemory
		}
		if over.StaleCandidateWindow > 0 {
			cfg.StaleCandidateWindow = over.StaleCandidateWindow
		}
		cfg.AllowHistoricalPositive = over.AllowHistoricalPositive
	}
	svc.Config = cfg

	in := c.CandidateInput
	if in.CandidateID == uuid.Nil {
		in.CandidateID = uuid.New()
	}
	if in.EvaluationID == uuid.Nil && c.CaseID != "missing_evaluation_rejected" {
		in.EvaluationID = uuid.New()
	}
	if in.SessionID == uuid.Nil {
		in.SessionID = uuid.New()
	}
	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now().UTC()
	}
	if in.MemoryID == "" {
		in.MemoryID = "test:mem-" + c.CaseID
	}
	in = svc.SeedCandidate(in)
	if c.ExistingMemoryUtility != 0 {
		svc.SetInitialScore(in.MemoryID, c.ExistingMemoryUtility)
	}
	for i := 0; i < c.PreSeedPositiveCount; i++ {
		seed := in
		seed.CandidateID = uuid.New()
		seed = svc.SeedCandidate(seed)
		_, _ = svc.ApplyCandidate(context.Background(), seed.CandidateID, "preseed", false)
	}
	return in
}
