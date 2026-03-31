package curation

import (
	"context"
	"testing"

	"control-plane/pkg/api"

	"github.com/google/uuid"
)

type stubDupChecker struct {
	dup *uuid.UUID
	err error
}

func (s *stubDupChecker) FindActiveDuplicate(_ context.Context, _ api.MemoryKind, _ string) (*uuid.UUID, error) {
	return s.dup, s.err
}

func TestValidatePromotionCandidate_shortStatement(t *testing.T) {
	svc := &Service{Promotion: &PromotionDigestConfig{}}
	c := &CandidateEvent{SalienceScore: 0.5}
	p := &ProposalPayloadV1{Kind: api.MemoryKindFailure, Statement: "short"}
	v := svc.ValidatePromotionCandidate(context.Background(), c, p)
	if v.Allow {
		t.Fatal("expected deny")
	}
}

func TestValidatePromotionCandidate_duplicateMemory_allowsConsolidation(t *testing.T) {
	did := uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	svc := &Service{
		Promotion: &PromotionDigestConfig{},
		MemoryDup: &stubDupChecker{dup: &did},
	}
	c := &CandidateEvent{SalienceScore: 0.5}
	p := &ProposalPayloadV1{
		Kind:      api.MemoryKindFailure,
		Statement: "this statement is long enough for the minimum validation rule sixteen",
	}
	v := svc.ValidatePromotionCandidate(context.Background(), c, p)
	if !v.Allow {
		t.Fatalf("expected allow (duplicate consolidates into canonical memory): %s", v.Reason)
	}
}

func TestValidatePromotionCandidate_supersedesAllowsDuplicateKey(t *testing.T) {
	did := uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	svc := &Service{
		Promotion: &PromotionDigestConfig{},
		MemoryDup:   &stubDupChecker{dup: &did},
	}
	c := &CandidateEvent{SalienceScore: 0.5}
	p := &ProposalPayloadV1{
		Kind:               api.MemoryKindFailure,
		Statement:          "this statement is long enough for the minimum validation rule sixteen",
		SupersedesMemoryID: did.String(),
	}
	v := svc.ValidatePromotionCandidate(context.Background(), c, p)
	if !v.Allow {
		t.Fatalf("expected allow when supersedes_memory_id matches duplicate: %s", v.Reason)
	}
}

func TestValidatePromotionCandidate_wrongSupersedesDenied(t *testing.T) {
	did := uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	other := uuid.MustParse("b0000000-0000-0000-0000-000000000002")
	svc := &Service{
		Promotion: &PromotionDigestConfig{},
		MemoryDup:   &stubDupChecker{dup: &did},
	}
	c := &CandidateEvent{SalienceScore: 0.5}
	p := &ProposalPayloadV1{
		Kind:               api.MemoryKindFailure,
		Statement:          "this statement is long enough for the minimum validation rule sixteen",
		SupersedesMemoryID: other.String(),
	}
	v := svc.ValidatePromotionCandidate(context.Background(), c, p)
	if v.Allow {
		t.Fatal("expected deny when supersedes_memory_id does not match duplicate")
	}
}
