package similarity

import "testing"

func TestCanonicalTokenJaccard_exportMatchesInternal(t *testing.T) {
	a, b := "payment webhook timeout", "payment webhook retry"
	if g1, g2 := CanonicalTokenJaccard(a, b), tokenJaccard(a, b); g1 != g2 {
		t.Fatalf("export %v vs internal %v", g1, g2)
	}
}

func TestTokenJaccard_identical(t *testing.T) {
	if g := tokenJaccard("payment webhook timeout", "payment webhook timeout"); g < 0.99 {
		t.Fatalf("got %v", g)
	}
}

func TestTokenJaccard_partialOverlap(t *testing.T) {
	g := tokenJaccard("payment webhook timeout", "payment webhook retry")
	if g < 0.2 || g > 0.95 {
		t.Fatalf("got %v", g)
	}
}

func TestTokenJaccard_disjoint(t *testing.T) {
	if g := tokenJaccard("database migration", "unrelated pizza"); g > 0.15 {
		t.Fatalf("got %v want low", g)
	}
}

func TestTagJaccard(t *testing.T) {
	if g := tagJaccard([]string{"a", "b"}, []string{"b", "c"}); g < 0.2 || g > 0.4 {
		t.Fatalf("got %v", g)
	}
}
