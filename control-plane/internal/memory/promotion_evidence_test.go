package memory

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestService_Promote_evidenceIDsWithoutLinker(t *testing.T) {
	svc := &Service{
		Repo: &Repo{}, // non-nil; Promote will fail on DB if we got past evidence check — we should not
	}
	eid := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	_, err := svc.Promote(context.Background(), PromoteRequest{
		Type:        "decision",
		Content:     "statement",
		Confidence:  0.5,
		EvidenceIDs: []uuid.UUID{eid},
	})
	if err == nil || !strings.Contains(err.Error(), "evidence linker") {
		t.Fatalf("expected evidence linker error, got %v", err)
	}
}
