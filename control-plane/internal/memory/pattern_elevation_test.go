package memory

import (
	"context"
	"testing"
)

func TestReuseScore(t *testing.T) {
	s := ReuseScore(4, 2, 1)
	if s < 5 {
		t.Fatalf("reuse score too low: %v", s)
	}
}

func TestLongestCommonPrefixWords(t *testing.T) {
	got := longestCommonPrefixWords([]string{"use idempotency keys for webhooks", "use idempotency keys for retries"})
	if got == "" || len(got) < 3 {
		t.Fatalf("expected common prefix, got %q", got)
	}
}

func TestNormalizePatternElevation_defaults(t *testing.T) {
	cfg := NormalizePatternElevation(&PatternElevationConfig{Enabled: true})
	if cfg.MinAuthority != 4 {
		t.Fatalf("MinAuthority: %d", cfg.MinAuthority)
	}
	if cfg.MergeJaccardMin != 0.82 {
		t.Fatalf("MergeJaccardMin: %v", cfg.MergeJaccardMin)
	}
}

func TestTryElevatePatterns_disabled(t *testing.T) {
	s := &Service{PatternElevation: &PatternElevationConfig{Enabled: false}}
	got, err := s.TryElevatePatterns(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("want nil when disabled, got %d", len(got))
	}
}
