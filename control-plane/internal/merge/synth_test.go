package merge

import (
	"strings"
	"testing"
)

func TestSynthesize_sortedAgreements(t *testing.T) {
	out := Synthesize([]string{"zebra line", "alpha line"}, []string{"m middle", "a first"}, []string{"v1", "v2"})
	if !strings.Contains(out, "alpha line") || !strings.Contains(out, "zebra line") {
		t.Fatal(out)
	}
	// order: sorted by normalized — alpha before zebra
	iAlpha := strings.Index(out, "alpha line")
	iZebra := strings.Index(out, "zebra line")
	if iAlpha > iZebra {
		t.Fatal("agreements not sorted", out)
	}
}

func TestSynthesize_dedupe(t *testing.T) {
	// dedupe happens in AgreementsUniquesFromClusters; synth receives already deduped
	out := Synthesize([]string{"same", "other"}, nil, nil)
	c := strings.Count(out, "same")
	if c > 1 {
		t.Fatalf("unexpected duplicate: %d", c)
	}
}
