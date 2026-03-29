package merge

import "testing"

func TestApplyUniquesPipeline_nil(t *testing.T) {
	u := []string{"a", "b"}
	out, rm := applyUniquesPipeline(u, []string{"x"}, nil)
	if rm != 0 || len(out) != 2 {
		t.Fatalf("got %v %d", out, rm)
	}
}

func TestApplyUniquesPipeline_maxCap(t *testing.T) {
	u := []string{"zebra line here with enough", "alpha line here with enough", "beta line here with enough"}
	opts := &MergeOptions{MaxUniqueBullets: 2}
	out, rm := applyUniquesPipeline(u, nil, opts)
	if rm != 1 || len(out) != 2 {
		t.Fatalf("got out=%v removed=%d", out, rm)
	}
	// sorted by Normalize: alpha, beta
	if Normalize(out[0]) != Normalize("alpha line here with enough") {
		t.Fatalf("first kept: %q", out[0])
	}
}

func TestApplyUniquesPipeline_dedupeSimilar(t *testing.T) {
	a := "first unique bullet with sufficient length here"
	b := "first unique bullet with sufficient length here plus extra words at end for similarity"
	opts := &MergeOptions{DedupeSimilarUniques: true}
	out, rm := applyUniquesPipeline([]string{a, b}, nil, opts)
	if len(out) != 1 || rm != 1 {
		t.Fatalf("expected 1 kept 1 removed, got %v %d", out, rm)
	}
}

func TestApplyUniquesPipeline_dropSimilarToAgreement(t *testing.T) {
	ag := "shared agreement text with enough characters in it for testing purposes here"
	u := "shared agreement text with enough characters in it for testing purposes here and more"
	opts := &MergeOptions{DropUniqueIfSimilarToAgreement: 0.9}
	out, rm := applyUniquesPipeline([]string{u, "totally different unique line with enough chars"}, []string{ag}, opts)
	if rm != 1 || len(out) != 1 {
		t.Fatalf("got %v removed=%d", out, rm)
	}
	if Normalize(out[0]) != Normalize("totally different unique line with enough chars") {
		t.Fatalf("wrong survivor: %q", out[0])
	}
}
