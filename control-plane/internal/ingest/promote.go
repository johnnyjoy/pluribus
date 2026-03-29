package ingest

import (
	"context"
	"fmt"
	"strings"

	"control-plane/internal/memory"

	"github.com/google/uuid"
)

// MemoryPromoter promotes consolidated canonical rows to durable memory (M7).
// Typically *memory.Service; nil disables all promotion paths.
type MemoryPromoter interface {
	Promote(ctx context.Context, req memory.PromoteRequest) (*memory.PromoteResponse, error)
}

func clampPromoteConfidence(c float64) float64 {
	if c < 0 {
		return 0
	}
	if c > 1 {
		return 1
	}
	return c
}

// promoteTypeFromPredicate maps normalized predicate text to memory promote contract types.
func promoteTypeFromPredicate(pred string) string {
	p := strings.ToLower(strings.TrimSpace(pred))
	switch {
	case strings.Contains(p, "constraint"), strings.Contains(p, "must not"), strings.Contains(p, "require"), strings.Contains(p, "shall"):
		return "constraint"
	case strings.Contains(p, "fail"), strings.Contains(p, "error"), strings.Contains(p, "bug"):
		return "failure"
	case strings.Contains(p, "decide"), strings.Contains(p, "choose"), strings.Contains(p, "decision"):
		return "decision"
	default:
		return "pattern"
	}
}

func canonicalRowStatement(row CanonicalFactRow) string {
	return strings.TrimSpace(row.SubjectNorm + " " + row.PredicateNorm + " " + row.ObjectNorm)
}

func rowToPromoteRequest(row CanonicalFactRow, ingestionID uuid.UUID) memory.PromoteRequest {
	conf := clampPromoteConfidence(row.Confidence)
	return memory.PromoteRequest{
		Type:       promoteTypeFromPredicate(row.PredicateNorm),
		Content:    canonicalRowStatement(row),
		Tags:       []string{"mcl", "canonical_hash:" + row.NormalizedHash},
		Source:     fmt.Sprintf("mcl-ingest:%s", ingestionID.String()),
		Confidence: conf,
	}
}

// applyPromotions runs memory.Promote for each row when server + client gates allow.
// Used after DB commit for inline ingest; CommitIngestion uses the same mapping.
func applyPromotions(ctx context.Context, p MemoryPromoter, serverEnabled bool, clientProposes bool, rows []CanonicalFactRow, ingestionID uuid.UUID) IngestPromotionDebug {
	out := IngestPromotionDebug{
		Attempted:                false,
		Reason:                   "",
		ServerAutoPromoteEnabled: serverEnabled,
		ClientProposePromotion:   clientProposes,
		Mode:                     "inline",
		MemoryIDs:                nil,
		Errors:                   nil,
	}
	if p == nil {
		out.Reason = "memory promoter not configured"
		return out
	}
	if !serverEnabled {
		out.Reason = "ingest.auto_promote is false (default); enable in config for promotion bridge"
		return out
	}
	if !clientProposes {
		out.Reason = "client did not set propose_promotion; use POST /v1/ingest/{id}/commit for operator promotion"
		return out
	}
	if len(rows) == 0 {
		out.Reason = "no canonical rows to promote"
		return out
	}
	out.Attempted = true
	for _, row := range rows {
		req := rowToPromoteRequest(row, ingestionID)
		resp, err := p.Promote(ctx, req)
		if err != nil {
			out.Errors = append(out.Errors, fmt.Sprintf("hash=%s: %v", row.NormalizedHash, err))
			continue
		}
		if resp != nil && resp.Promoted && resp.ID != "" {
			out.MemoryIDs = append(out.MemoryIDs, resp.ID)
		}
	}
	if len(out.Errors) > 0 && len(out.MemoryIDs) == 0 {
		out.Reason = "promotion failed for all rows"
	} else if len(out.Errors) > 0 {
		out.Reason = "partial promotion: some rows failed"
	} else {
		out.Reason = fmt.Sprintf("promoted %d memory object(s)", len(out.MemoryIDs))
	}
	return out
}

// applyCommitPromotions promotes all DB rows for an ingestion when the server bridge is enabled.
// Operator path: no client propose_promotion flag required.
func applyCommitPromotions(ctx context.Context, p MemoryPromoter, serverEnabled bool, rows []CanonicalFactRow, ingestionID uuid.UUID) IngestPromotionDebug {
	out := IngestPromotionDebug{
		Attempted:                false,
		Reason:                   "",
		ServerAutoPromoteEnabled: serverEnabled,
		ClientProposePromotion:   false,
		Mode:                     "commit_operator",
		MemoryIDs:                nil,
		Errors:                   nil,
	}
	if p == nil {
		out.Reason = "memory promoter not configured"
		return out
	}
	if !serverEnabled {
		out.Reason = "ingest.auto_promote is false; enable in config before POST .../commit"
		return out
	}
	if len(rows) == 0 {
		out.Reason = "no canonical rows for this ingestion"
		return out
	}
	out.Attempted = true
	for _, row := range rows {
		req := rowToPromoteRequest(row, ingestionID)
		resp, err := p.Promote(ctx, req)
		if err != nil {
			out.Errors = append(out.Errors, fmt.Sprintf("hash=%s: %v", row.NormalizedHash, err))
			continue
		}
		if resp != nil && resp.Promoted && resp.ID != "" {
			out.MemoryIDs = append(out.MemoryIDs, resp.ID)
		}
	}
	if len(out.Errors) > 0 && len(out.MemoryIDs) == 0 {
		out.Reason = "promotion failed for all rows"
	} else if len(out.Errors) > 0 {
		out.Reason = "partial promotion: some rows failed"
	} else {
		out.Reason = fmt.Sprintf("operator commit: promoted %d memory object(s)", len(out.MemoryIDs))
	}
	return out
}
