package recall

import (
	"math"
	"testing"
)

func TestNormalizedPolicyWeights_defaults(t *testing.T) {
	wc, ws, we := NormalizedPolicyWeights(PromotionPolicy{})
	if math.Abs(wc-0.4) > 1e-9 || math.Abs(ws-0.3) > 1e-9 || math.Abs(we-0.3) > 1e-9 {
		t.Fatalf("defaults want 0.4/0.3/0.3, got %v %v %v", wc, ws, we)
	}
	if math.Abs(wc+ws+we-1.0) > 1e-9 {
		t.Fatalf("weights should sum to 1, got %v", wc+ws+we)
	}
}

func TestPolicyComposite_signalNormalizationAndClamp(t *testing.T) {
	pol := PromotionPolicy{
		WeightConfidence: 0,
		WeightSignal:     1,
		WeightEvidence:   0,
		SignalNormDivisor: 10,
	}
	// totalSignal 25 -> 2.5 -> clamp 1.0
	v := PolicyComposite(0, 25, nil, pol)
	if math.Abs(v-1.0) > 1e-9 {
		t.Fatalf("expected signal term 1.0, got %v", v)
	}
}

func TestPolicyComposite_evidenceNilIsZero(t *testing.T) {
	pol := PromotionPolicy{WeightConfidence: 1, WeightSignal: 0, WeightEvidence: 0}
	v := PolicyComposite(0.8, 0, nil, pol)
	if math.Abs(v-0.8) > 1e-9 {
		t.Fatalf("expected 0.8, got %v", v)
	}
}

func TestPolicyComposite_evidencePointer(t *testing.T) {
	ev := 0.5
	pol := PromotionPolicy{WeightConfidence: 0, WeightSignal: 0, WeightEvidence: 1}
	v := PolicyComposite(0, 0, &ev, pol)
	if math.Abs(v-0.5) > 1e-9 {
		t.Fatalf("expected 0.5, got %v", v)
	}
}

func TestPolicyCompositeUsesEvidenceTerm(t *testing.T) {
	if !PolicyCompositeUsesEvidenceTerm(PromotionPolicy{MinPolicyComposite: 0.5}) {
		t.Fatal("default weights include evidence")
	}
	if PolicyCompositeUsesEvidenceTerm(PromotionPolicy{
		MinPolicyComposite: 0.5,
		WeightConfidence:   1,
		WeightSignal:       0,
		WeightEvidence:     0,
	}) {
		t.Fatal("evidence weight 0 after normalize")
	}
}
