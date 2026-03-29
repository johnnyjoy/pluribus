package recall

import (
	"testing"

	"github.com/google/uuid"
)

func TestUniqueEvidenceIDCount(t *testing.T) {
	a := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	b := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	if n := uniqueEvidenceIDCount([]uuid.UUID{a, b, a}); n != 2 {
		t.Fatalf("got %d", n)
	}
	if n := uniqueEvidenceIDCount(nil); n != 0 {
		t.Fatalf("got %d", n)
	}
}
