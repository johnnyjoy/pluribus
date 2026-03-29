package ingest

import (
	"fmt"
	"strings"
)

// DefaultMinConfidenceTrustProduct: reject if any fact has confidence × trust_weight below this (after structural validation).
const DefaultMinConfidenceTrustProduct = 0.15

// DefaultMinTraceTotalChars: reject if sum of trimmed reasoning_trace step lengths is below this (0 disables).
const DefaultMinTraceTotalChars = 40

// NoiseRejectReason returns a deterministic rejection message for M5 gates, or "" if ok.
// trust is the client's trust_weight (typically 1.0 when no profile row exists).
func NoiseRejectReason(req CognitionRequest, trust float64, minProduct float64, minTraceChars int) string {
	if minProduct <= 0 {
		minProduct = DefaultMinConfidenceTrustProduct
	}
	for i, f := range req.ExtractedFacts {
		c := req.Confidence
		if f.Confidence != nil {
			c = *f.Confidence
		}
		eff := c * trust
		if eff < minProduct {
			return fmt.Sprintf("noise: extracted_facts[%d] confidence*trust_weight=%.6g below minimum %.6g", i, eff, minProduct)
		}
	}
	if minTraceChars > 0 {
		total := 0
		for _, step := range req.ReasoningTrace {
			total += len(strings.TrimSpace(step))
		}
		if total < minTraceChars {
			return fmt.Sprintf("noise: reasoning_trace total characters %d below minimum %d", total, minTraceChars)
		}
	}
	return ""
}
