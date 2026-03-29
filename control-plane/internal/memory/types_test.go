package memory

import (
	"strings"
	"testing"
)

func TestValidatePatternPayload(t *testing.T) {
	validPayload := &PatternPayload{
		Polarity:   "negative",
		Experience: "We deployed without feature flags and had to roll back.",
		Decision:   "Use feature flags for risky changes",
		Outcome:    "Safer deploys",
		Impact:     PatternImpact{Severity: "high", Surface: "deploy", Cost: "time"},
		Directive:  "Always use feature flags for deploy-time toggles.",
		Files:      []string{"main.go"},
		Symbols:    []string{"pkg.Deploy"},
	}
	if err := ValidatePatternPayload(validPayload); err != nil {
		t.Errorf("valid payload should pass: %v", err)
	}
	validPositive := &PatternPayload{
		Polarity:   "positive",
		Experience: "Documented the API contract first.",
		Decision:   "Keep OpenAPI in sync",
		Outcome:    "Fewer integration bugs",
		Impact:     PatternImpact{Severity: "medium"},
		Directive:  "Update OpenAPI on every change.",
	}
	if err := ValidatePatternPayload(validPositive); err != nil {
		t.Errorf("valid positive payload should pass: %v", err)
	}
}

func TestValidatePatternPayload_invalid(t *testing.T) {
	tests := []struct {
		name    string
		payload *PatternPayload
		wantErr string
	}{
		{"nil", nil, "payload"},
		{"empty polarity", &PatternPayload{
			Polarity:   "",
			Experience: "x", Decision: "d", Outcome: "o",
			Impact:    PatternImpact{Severity: "low"},
			Directive: "d",
		}, "polarity"},
		{"invalid polarity", &PatternPayload{
			Polarity:   "neutral",
			Experience: "x", Decision: "d", Outcome: "o",
			Impact:    PatternImpact{Severity: "low"},
			Directive: "d",
		}, "polarity"},
		{"missing experience", &PatternPayload{
			Polarity:   "negative",
			Experience: "",
			Decision:   "d", Outcome: "o",
			Impact:    PatternImpact{Severity: "low"},
			Directive: "d",
		}, "experience"},
		{"missing decision", &PatternPayload{
			Polarity:   "negative",
			Experience: "x", Decision: "", Outcome: "o",
			Impact:    PatternImpact{Severity: "low"},
			Directive: "d",
		}, "decision"},
		{"missing outcome", &PatternPayload{
			Polarity:   "negative",
			Experience: "x", Decision: "d", Outcome: "",
			Impact:    PatternImpact{Severity: "low"},
			Directive: "d",
		}, "outcome"},
		{"missing impact.severity", &PatternPayload{
			Polarity:   "negative",
			Experience: "x", Decision: "d", Outcome: "o",
			Impact:    PatternImpact{Severity: ""},
			Directive: "d",
		}, "impact.severity"},
		{"missing directive", &PatternPayload{
			Polarity:   "negative",
			Experience: "x", Decision: "d", Outcome: "o",
			Impact:    PatternImpact{Severity: "low"},
			Directive: "",
		}, "directive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePatternPayload(tt.payload)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q should contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestValidPatternPolarity(t *testing.T) {
	if !ValidPatternPolarity("positive") || !ValidPatternPolarity("negative") {
		t.Error("positive and negative should be valid")
	}
	if ValidPatternPolarity("") || ValidPatternPolarity("neutral") || ValidPatternPolarity("unknown") {
		t.Error("empty, neutral, unknown should be invalid")
	}
}
