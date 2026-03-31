package curation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"control-plane/pkg/api"
)

// AutoPromoteBatch materializes all pending candidates that pass auto-promote thresholds and validation.
// Requires Promotion.AutoPromote true in configuration.
func (s *Service) AutoPromoteBatch(ctx context.Context) (*AutoPromoteResponse, error) {
	if s.Promotion == nil || !s.Promotion.AutoPromote {
		return nil, fmt.Errorf("auto_promote is disabled in configuration")
	}
	if s.Memory == nil {
		return nil, fmt.Errorf("memory service required for auto-promote")
	}
	pending, err := s.Repo.ListPending(ctx)
	if err != nil {
		return nil, err
	}
	var out []AutoPromoteResultRow
	for i := range pending {
		c := pending[i]
		if len(c.ProposalJSON) == 0 {
			out = append(out, AutoPromoteResultRow{CandidateID: c.ID.String(), Status: "skipped", Detail: "no proposal_json"})
			continue
		}
		var p ProposalPayloadV1
		if err := json.Unmarshal(c.ProposalJSON, &p); err != nil {
			out = append(out, AutoPromoteResultRow{CandidateID: c.ID.String(), Status: "skipped", Detail: "invalid proposal_json"})
			continue
		}
		if !s.autoPromoteEligible(&p, c.SalienceScore) {
			out = append(out, AutoPromoteResultRow{CandidateID: c.ID.String(), Status: "skipped", Detail: "not eligible for auto-promote thresholds"})
			continue
		}
		val := s.ValidatePromotionCandidate(ctx, &c, &p)
		if !val.Allow {
			out = append(out, AutoPromoteResultRow{CandidateID: c.ID.String(), Status: "skipped", Detail: val.Reason})
			continue
		}
		mat, err := s.materializeInternal(ctx, c.ID, true)
		if err != nil {
			out = append(out, AutoPromoteResultRow{CandidateID: c.ID.String(), Status: "error", Detail: err.Error()})
			continue
		}
		if mat == nil || mat.Memory == nil {
			out = append(out, AutoPromoteResultRow{CandidateID: c.ID.String(), Status: "error", Detail: "materialize returned empty"})
			continue
		}
		out = append(out, AutoPromoteResultRow{CandidateID: c.ID.String(), MemoryID: mat.Memory.ID.String(), Status: "promoted", Detail: "ok"})
	}
	return &AutoPromoteResponse{Results: out}, nil
}

func (s *Service) autoPromoteEligible(p *ProposalPayloadV1, salience float64) bool {
	cfg := s.Promotion
	if cfg == nil {
		return false
	}
	kinds := cfg.AutoAllowedKinds
	if len(kinds) == 0 {
		kinds = []string{string(api.MemoryKindFailure), string(api.MemoryKindPattern)}
	}
	ok := false
	want := string(p.Kind)
	for _, k := range kinds {
		if strings.TrimSpace(k) == want {
			ok = true
			break
		}
	}
	if !ok {
		return false
	}
	if supportCount(p) < cfg.AutoMinSupportCount {
		return false
	}
	if salience < cfg.AutoMinSalience {
		return false
	}
	return true
}
