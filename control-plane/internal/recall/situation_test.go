package recall

import "testing"

func TestEnrichSituationQueryWithProposal(t *testing.T) {
	got := enrichSituationQueryWithProposal("ship feature", "use idempotency keys")
	want := "ship feature use idempotency keys"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if enrichSituationQueryWithProposal("only", "") != "only" {
		t.Fatal("empty proposal")
	}
}
