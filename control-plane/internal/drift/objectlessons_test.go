package drift

import (
	"encoding/json"
	"strings"
	"testing"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

func TestNegativePatternMatches_positiveLessonIgnored(t *testing.T) {
	payload := memory.PatternPayload{
		Polarity:   "positive",
		Experience: "We added tests first",
		Decision:   "TDD",
		Outcome:    "Fewer bugs",
		Impact:     memory.PatternImpact{Severity: "high"},
		Directive:  "Always add tests",
	}
	raw, _ := json.Marshal(payload)
	objs := []memory.MemoryObject{{
		ID:        uuid.New(),
		Kind:      api.MemoryKindPattern,
		Statement: "Always add tests",
		Payload:   raw,
	}}
	got := NegativePatternMatches("we should always add tests", objs)
	if len(got) != 0 {
		t.Errorf("positive lesson should be ignored: got %d issues", len(got))
	}
}

func TestNegativePatternMatches_directiveMatch(t *testing.T) {
	payload := memory.PatternPayload{
		Polarity:   "negative",
		Experience: "Broke prod",
		Decision:   "Skipped review",
		Outcome:    "Outage",
		Impact:     memory.PatternImpact{Severity: "high"},
		Directive:  "add alternate path",
	}
	raw, _ := json.Marshal(payload)
	objs := []memory.MemoryObject{{
		ID:        uuid.New(),
		Kind:      api.MemoryKindPattern,
		Statement: "Do not add alternate path",
		Payload:   raw,
	}}
	got := NegativePatternMatches("I will add alternate path for obsolete code", objs)
	if len(got) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(got))
	}
	if got[0].Code != "negative_pattern" {
		t.Errorf("Code = %q, want negative_pattern", got[0].Code)
	}
	if got[0].Score != 4.0 {
		t.Errorf("Score (high severity) = %v, want 4", got[0].Score)
	}
	if !strings.Contains(got[0].Statement, "alternate path") {
		t.Errorf("Statement should mention directive: %q", got[0].Statement)
	}
}

func TestNegativePatternMatches_experienceMatch(t *testing.T) {
	payload := memory.PatternPayload{
		Polarity:   "negative",
		Experience: "fluent API",
		Decision:   "Changed terminator",
		Outcome:    "Regression",
		Impact:     memory.PatternImpact{Severity: "medium"},
		Directive:  "Do not break fluent chain",
	}
	raw, _ := json.Marshal(payload)
	objs := []memory.MemoryObject{{
		ID:      uuid.New(),
		Kind:    api.MemoryKindPattern,
		Payload: raw,
	}}
	got := NegativePatternMatches("refactor the fluent API for clarity", objs)
	if len(got) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(got))
	}
	if got[0].Code != "negative_pattern" {
		t.Errorf("Code = %q", got[0].Code)
	}
	if got[0].Score != 2.5 {
		t.Errorf("Score (medium) = %v, want 2.5", got[0].Score)
	}
}

func TestNegativePatternMatches_tagMatch(t *testing.T) {
	payload := memory.PatternPayload{
		Polarity:   "negative",
		Experience: "Fluent misuse",
		Decision:   "Avoid",
		Outcome:    "Bugs",
		Impact:     memory.PatternImpact{Severity: "low"},
		Directive:  "Document fluent usage",
	}
	raw, _ := json.Marshal(payload)
	objs := []memory.MemoryObject{{
		ID:        uuid.New(),
		Kind:      api.MemoryKindPattern,
		Tags:      []string{"fluent", "api"},
		Payload:   raw,
	}}
	got := NegativePatternMatches("the fluent builder is used here", objs)
	if len(got) != 1 {
		t.Fatalf("expected 1 issue (tag match), got %d", len(got))
	}
	if got[0].Score != 1.0 {
		t.Errorf("Score (low) = %v, want 1", got[0].Score)
	}
}

func TestNegativePatternMatches_noMatch(t *testing.T) {
	payload := memory.PatternPayload{
		Polarity:   "negative",
		Experience: "We broke X",
		Decision:   "Avoid X",
		Outcome:    "Bad",
		Impact:     memory.PatternImpact{Severity: "high"},
		Directive:  "Do not touch X",
	}
	raw, _ := json.Marshal(payload)
	objs := []memory.MemoryObject{{
		ID:      uuid.New(),
		Kind:    api.MemoryKindPattern,
		Payload: raw,
	}}
	got := NegativePatternMatches("we will improve Y and Z only", objs)
	if len(got) != 0 {
		t.Errorf("no overlap: expected 0 issues, got %d", len(got))
	}
}

