package curation

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ValidateDigestRequest checks bounds and required fields. limits must be non-nil with positive max proposals.
func ValidateDigestRequest(req *DigestRequest, limits *DigestLimits) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}
	if limits == nil {
		limits = defaultDigestLimits()
	}
	if len(req.WorkSummary) == 0 {
		return fmt.Errorf("work_summary is required")
	}
	if len(req.WorkSummary) > limits.WorkSummaryMaxBytes {
		return fmt.Errorf("work_summary exceeds max %d bytes", limits.WorkSummaryMaxBytes)
	}
	maxP := limits.MaxProposals
	if req.Options != nil && req.Options.MaxProposals > 0 {
		maxP = req.Options.MaxProposals
	}
	if maxP > limits.MaxProposals {
		maxP = limits.MaxProposals
	}
	if maxP <= 0 {
		maxP = 5
	}
	_ = maxP // enforced in service
	for _, id := range req.EvidenceIDs {
		if id == uuid.Nil {
			return fmt.Errorf("evidence_ids must not contain null UUID")
		}
	}
	if req.CurationAnswers != nil {
		a := req.CurationAnswers
		for _, pair := range []struct {
			name string
			val  string
		}{
			{"decision", a.Decision},
			{"constraint", a.Constraint},
			{"failure", a.Failure},
			{"pattern", a.Pattern},
			{"never_again", a.NeverAgain},
			{"what_changed", a.WhatChanged},
			{"what_learned", a.WhatLearned},
		} {
			if len(pair.val) > limits.StatementMaxBytes && len(pair.name) > 0 {
				// long fields: same cap as statement
				if len(pair.val) > limits.StatementMaxBytes*4 {
					return fmt.Errorf("curation_answers.%s exceeds max length", pair.name)
				}
			}
		}
	}
	return nil
}

func defaultDigestLimits() *DigestLimits {
	return &DigestLimits{
		MaxProposals:        5,
		WorkSummaryMaxBytes: 8192,
		StatementMaxBytes:   2048,
		ReasonMaxBytes:      1024,
	}
}

func truncateReason(s string, max int) string {
	if max <= 0 {
		return s
	}
	if len(s) <= max {
		return s
	}
	return strings.TrimSpace(s[:max])
}
