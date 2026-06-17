package utility

import (
	"testing"
)

func TestEventWeight(t *testing.T) {
	if EventWeight(EventHelpful) != 1.0 {
		t.Fatalf("helpful weight")
	}
	if EventWeight(EventWrong) != -3.0 {
		t.Fatalf("wrong weight")
	}
	if EventWeight(EventRefuted) != -4.0 {
		t.Fatalf("refuted weight")
	}
}

func TestClampUtilityScore(t *testing.T) {
	if ClampUtilityScore(15) != MaxUtilityScore {
		t.Fatalf("cap high")
	}
	if ClampUtilityScore(-15) != MinUtilityScore {
		t.Fatalf("cap low")
	}
}

func TestRankingTerm(t *testing.T) {
	if RankingTerm(10, 0.12) <= 0 {
		t.Fatalf("positive utility should boost")
	}
	if RankingTerm(-10, 0.12) >= 0 {
		t.Fatalf("negative utility should penalize")
	}
	if RankingTerm(0, 0.12) != 0 {
		t.Fatalf("neutral utility")
	}
}

func TestDiminishingWeight(t *testing.T) {
	w0 := DiminishingWeight(1.0, 0)
	w3 := DiminishingWeight(1.0, 3)
	w5 := DiminishingWeight(1.0, 4)
	w9 := DiminishingWeight(1.0, 8)
	if w0 != 1.0 || w3 != 1.0 {
		t.Fatalf("first four events full weight: w0=%v w3=%v", w0, w3)
	}
	if w5 >= 1.0 || w9 >= w5 {
		t.Fatalf("diminishing expected: w5=%v w9=%v", w5, w9)
	}
}

func TestValidateFeedbackRequest(t *testing.T) {
	if err := ValidateFeedbackRequest(FeedbackRequest{EventType: "helpful"}); err != nil {
		t.Fatalf("helpful ok: %v", err)
	}
	if err := ValidateFeedbackRequest(FeedbackRequest{EventType: "wrong"}); err != ErrReasonRequired {
		t.Fatalf("wrong needs reason")
	}
	if err := ValidateFeedbackRequest(FeedbackRequest{EventType: "bogus"}); err != ErrInvalidEventType {
		t.Fatalf("invalid type")
	}
}

func TestApplyScoreDelta(t *testing.T) {
	got := ApplyScoreDelta(0, -3)
	if got != -3 {
		t.Fatalf("delta apply: %v", got)
	}
	got = ApplyScoreDelta(9, 5)
	if got != MaxUtilityScore {
		t.Fatalf("bounded high: %v", got)
	}
}
