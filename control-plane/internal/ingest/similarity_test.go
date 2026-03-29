package ingest

import "testing"

func TestTokenJaccard(t *testing.T) {
	t.Parallel()
	if g := TokenJaccard("a b c", "a b c d"); g < 0.59 || g > 0.76 {
		t.Fatalf("expected ~0.75, got %v", g)
	}
	if TokenJaccard("", "") != 1 {
		t.Fatal("empty")
	}
	if TokenJaccard("x", "") != 0 {
		t.Fatal("one empty")
	}
	if TokenJaccard("same", "same") != 1 {
		t.Fatal("identical")
	}
}

func TestSimilarForUnify(t *testing.T) {
	if !SimilarForUnify("app", "depends_on", "alpha beta gamma", "app", "depends_on", "alpha beta gamma delta", DefaultSimilarJaccardMin) {
		t.Fatal("expected similar")
	}
	if SimilarForUnify("app", "depends_on", "alpha beta gamma", "app", "needs", "alpha beta gamma delta", DefaultSimilarJaccardMin) {
		t.Fatal("different predicate should not unify")
	}
}
