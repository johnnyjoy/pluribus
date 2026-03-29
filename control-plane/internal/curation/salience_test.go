package curation

import (
	"testing"
)

func TestScoreText_alwaysOrNever_increasesScore(t *testing.T) {
	cfg := &SalienceConfig{CandidateThreshold: 0.5, ReviewThreshold: 0.7, PromoteThreshold: 0.85}
	low := ScoreText("some random note", cfg)
	always := ScoreText("we must always use POST for creates", cfg)
	never := ScoreText("never use global state", cfg)
	if always <= low {
		t.Errorf("expected 'always' to score higher than generic: low=%f always=%f", low, always)
	}
	if never <= low {
		t.Errorf("expected 'never' to score higher than generic: low=%f never=%f", low, never)
	}
}

func TestScoreText_speculation_decreasesScore(t *testing.T) {
	cfg := &SalienceConfig{}
	directive := ScoreText("must use HTTPS", cfg)
	speculative := ScoreText("we might perhaps use HTTPS maybe", cfg)
	if speculative >= directive {
		t.Errorf("expected speculative to score lower: directive=%f speculative=%f", directive, speculative)
	}
}
