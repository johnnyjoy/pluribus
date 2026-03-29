package merge

import "testing"

func TestSimilarity_identical(t *testing.T) {
	if Similarity("Hello World foo", "hello world foo") < 0.99 {
		t.Fatalf("expected ~1, got %v", Similarity("Hello World foo", "hello world foo"))
	}
}

func TestSimilarity_highOverlap(t *testing.T) {
	a := "use the repository pattern for data access layer implementation"
	b := "we should use the repository pattern for our data access layer"
	if Similarity(a, b) < AgreementSimilarityThreshold {
		t.Fatalf("expected high similarity, got %v", Similarity(a, b))
	}
}

func TestSimilarity_low(t *testing.T) {
	a := "completely different topic about bananas and fruit"
	b := "quantum mechanics and wave function collapse details"
	if Similarity(a, b) > 0.4 {
		t.Fatalf("expected low similarity, got %v", Similarity(a, b))
	}
}

func TestNormalize(t *testing.T) {
	if Normalize("  Foo   Bar  ") != "foo bar" {
		t.Fatalf("got %q", Normalize("  Foo   Bar  "))
	}
}
