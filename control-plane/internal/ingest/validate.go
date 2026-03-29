package ingest

import (
	"fmt"
	"strings"
)

// Limits caps request size (deterministic rejects).
type Limits struct {
	MaxTempContributorIDLen int
	MaxQueryLen             int
	MinQueryLen             int
	MaxReasoningSteps       int
	MaxReasoningStepLen     int
	MaxFacts                int
	MaxSourceRefs           int
	MaxSourceRefLen         int
}

// DefaultLimits returns production-style defaults.
func DefaultLimits() Limits {
	return Limits{
		MaxTempContributorIDLen: 256,
		MaxQueryLen:             8192,
		MinQueryLen:             1,
		MaxReasoningSteps:       32,
		MaxReasoningStepLen:     8192,
		MaxFacts:                256,
		MaxSourceRefs:           64,
		MaxSourceRefLen:         2048,
	}
}

// Validate returns a deterministic rejected_reason message, or "" if ok.
func Validate(req CognitionRequest, lim Limits, requireContextHash bool) string {
	tcid := strings.TrimSpace(req.TempContributorID)
	if tcid == "" {
		return "temp_contributor_id: required"
	}
	if len(tcid) > lim.MaxTempContributorIDLen {
		return fmt.Sprintf("temp_contributor_id: exceeds max length %d", lim.MaxTempContributorIDLen)
	}
	q := strings.TrimSpace(req.Query)
	if len(q) < lim.MinQueryLen {
		return "query: required"
	}
	if len(q) > lim.MaxQueryLen {
		return fmt.Sprintf("query: exceeds max length %d", lim.MaxQueryLen)
	}
	if len(req.ReasoningTrace) < 1 {
		return "reasoning_trace: at least one step required"
	}
	if len(req.ReasoningTrace) > lim.MaxReasoningSteps {
		return fmt.Sprintf("reasoning_trace: exceeds max steps %d", lim.MaxReasoningSteps)
	}
	for i, step := range req.ReasoningTrace {
		s := strings.TrimSpace(step)
		if s == "" {
			return fmt.Sprintf("reasoning_trace[%d]: empty step", i)
		}
		if len(s) > lim.MaxReasoningStepLen {
			return fmt.Sprintf("reasoning_trace[%d]: exceeds max length %d", i, lim.MaxReasoningStepLen)
		}
	}
	if len(req.ExtractedFacts) < 1 {
		return "extracted_facts: at least one fact required"
	}
	if len(req.ExtractedFacts) > lim.MaxFacts {
		return fmt.Sprintf("extracted_facts: exceeds max %d", lim.MaxFacts)
	}
	for i, f := range req.ExtractedFacts {
		if strings.TrimSpace(f.Subject) == "" {
			return fmt.Sprintf("extracted_facts[%d].subject: required", i)
		}
		if strings.TrimSpace(f.Predicate) == "" {
			return fmt.Sprintf("extracted_facts[%d].predicate: required", i)
		}
		if strings.TrimSpace(f.Object) == "" {
			return fmt.Sprintf("extracted_facts[%d].object: required", i)
		}
		if f.Confidence != nil {
			c := *f.Confidence
			if c < 0 || c > 1 {
				return fmt.Sprintf("extracted_facts[%d].confidence: must be between 0 and 1", i)
			}
		}
	}
	if req.Confidence < 0 || req.Confidence > 1 {
		return "confidence: must be between 0 and 1"
	}
	if requireContextHash && strings.TrimSpace(req.ContextWindowHash) == "" {
		return "context_window_hash: required"
	}
	if len(req.SourceRefs) > lim.MaxSourceRefs {
		return fmt.Sprintf("source_refs: exceeds max %d", lim.MaxSourceRefs)
	}
	for i, ref := range req.SourceRefs {
		r := strings.TrimSpace(ref)
		if r == "" {
			return fmt.Sprintf("source_refs[%d]: empty entry", i)
		}
		if len(r) > lim.MaxSourceRefLen {
			return fmt.Sprintf("source_refs[%d]: exceeds max length %d", i, lim.MaxSourceRefLen)
		}
	}
	return ""
}
