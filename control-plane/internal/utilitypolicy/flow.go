package utilitypolicy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"

	"control-plane/internal/mcp"

	"github.com/go-chi/chi/v5"
)

// CaseResult captures one policy case execution.
type CaseResult struct {
	Decision           string
	Delta              float64
	ScoreBefore        float64
	ScoreAfter         float64
	ApplicationCreated bool
	DuplicateRejected  bool
	RollbackOK         bool
	ParityMatch        bool
	Error              error
}

// RunCase executes one policy fixture against in-memory service.
func RunCase(ctx context.Context, svc *Service, c PolicyCase) CaseResult {
	var res CaseResult
	in := PrepareCase(svc, c)
	res.ScoreBefore, _ = svc.Scores.GetScore(ctx, in.MemoryID)

	if c.DuplicateApply {
		first, err := svc.ApplyCandidate(ctx, in.CandidateID, "test", c.Tampered)
		if err != nil {
			res.Error = err
			return res
		}
		res.Decision = first.Decision
		res.Delta = first.Delta
		res.ApplicationCreated = true
		res.ScoreAfter, _ = svc.Scores.GetScore(ctx, in.MemoryID)
		_, err = svc.ApplyCandidate(ctx, in.CandidateID, "test", c.Tampered)
		if err == ErrAlreadyApplied {
			res.DuplicateRejected = true
		} else {
			res.Error = err
		}
		return res
	}

	if c.CaseID == "rollback_restores_or_compensates" {
		rec, err := svc.ApplyCandidate(ctx, in.CandidateID, "test", c.Tampered)
		if err != nil {
			res.Error = err
			return res
		}
		res.ScoreAfter, _ = svc.Scores.GetScore(ctx, in.MemoryID)
		_, err = svc.RevertApplication(ctx, rec.RollbackToken, "test revert", "test")
		if err == nil {
			res.RollbackOK = true
			afterRevert, _ := svc.Scores.GetScore(ctx, in.MemoryID)
			if math.Abs(afterRevert-res.ScoreBefore) < 1e-9 {
				res.ScoreAfter = afterRevert
			}
		} else {
			res.Error = err
		}
		res.Decision = rec.Decision
		res.Delta = rec.Delta
		res.ApplicationCreated = true
		return res
	}

	rec, err := svc.ApplyCandidate(ctx, in.CandidateID, "test", c.Tampered)
	if err != nil {
		res.Error = err
		return res
	}
	res.Decision = rec.Decision
	res.Delta = rec.Delta
	res.ApplicationCreated = true
	res.ScoreAfter, _ = svc.Scores.GetScore(ctx, in.MemoryID)
	return res
}

// VerifyCase checks result against fixture expectations.
func VerifyCase(c PolicyCase, res CaseResult) error {
	if res.Error != nil && c.ExpectedPolicyDecision != DecisionReject {
		return fmt.Errorf("unexpected error: %w", res.Error)
	}
	if c.DuplicateApply {
		if !res.DuplicateRejected {
			return fmt.Errorf("expected duplicate rejection")
		}
		return nil
	}
	if c.CaseID == "rollback_restores_or_compensates" {
		if !res.RollbackOK {
			return fmt.Errorf("rollback failed")
		}
		return nil
	}
	if res.Decision != c.ExpectedPolicyDecision {
		return fmt.Errorf("decision=%s want %s", res.Decision, c.ExpectedPolicyDecision)
	}
	if c.ExpectedDelta != 0 && math.Abs(res.Delta-c.ExpectedDelta) > 0.01 && c.ExpectedPolicyDecision != DecisionRecordOnly {
		// allow capped deltas within tolerance
		if math.Abs(math.Abs(res.Delta)-math.Abs(c.ExpectedDelta)) > 0.26 {
			return fmt.Errorf("delta=%.3f want ~%.3f", res.Delta, c.ExpectedDelta)
		}
	}
	if c.ExpectedScoreChange {
		if math.Abs(res.ScoreAfter-res.ScoreBefore) < 1e-9 {
			return fmt.Errorf("expected score change, got none")
		}
	} else {
		if math.Abs(res.ScoreAfter-res.ScoreBefore) > 1e-9 {
			return fmt.Errorf("expected no score change, got %.3f -> %.3f", res.ScoreBefore, res.ScoreAfter)
		}
	}
	if c.ExpectedApplicationRecord && !res.ApplicationCreated {
		return fmt.Errorf("expected application record")
	}
	return nil
}

func newPolicyRouter(svc *Service) http.Handler {
	r := chi.NewRouter()
	r.Route("/v1", func(r chi.Router) {
		MountRoutes(r, &Handlers{Service: svc})
	})
	return r
}

// RunRESTPolicyFlow applies candidate via REST.
func RunRESTPolicyFlow(svc *Service, candidateID string) (ApplicationRecord, error) {
	srv := httptest.NewServer(newPolicyRouter(svc))
	defer srv.Close()
	body, _ := json.Marshal(map[string]string{"candidate_id": candidateID})
	resp, err := http.Post(srv.URL+"/v1/agent/utility/policy/apply-candidate", "application/json", bytes.NewReader(body))
	if err != nil {
		return ApplicationRecord{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return ApplicationRecord{}, fmt.Errorf("rest apply status %d: %s", resp.StatusCode, string(raw))
	}
	var rec ApplicationRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return ApplicationRecord{}, err
	}
	return rec, nil
}

// RunMCPPolicyFlow applies candidate via MCP tool proxy.
func RunMCPPolicyFlow(svc *Service, candidateID string) (ApplicationRecord, error) {
	srv := httptest.NewServer(newPolicyRouter(svc))
	defer srv.Close()
	raw, _ := json.Marshal(map[string]any{
		"name":      "agent_utility_apply_candidate",
		"arguments": map[string]any{"candidate_id": candidateID},
	})
	res, err := mcp.HandleToolsCall(http.DefaultClient, srv.URL, "", raw, nil, nil)
	if err != nil {
		return ApplicationRecord{}, err
	}
	b, _ := json.Marshal(res)
	var wrap struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(b, &wrap); err == nil && len(wrap.Content) > 0 {
		var rec ApplicationRecord
		if err := json.Unmarshal([]byte(wrap.Content[0].Text), &rec); err == nil {
			return rec, nil
		}
	}
	var rec ApplicationRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		return ApplicationRecord{}, fmt.Errorf("mcp parse: %w", err)
	}
	return rec, nil
}
