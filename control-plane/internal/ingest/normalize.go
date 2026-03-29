package ingest

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// NormalizePipelineVersion is bumped when normalization rules change (determinism / replay).
const NormalizePipelineVersion = "mcl_nfc_tolower_trim_v1"

// NormalizeFactToken applies deterministic normalization for subject / predicate / object:
// 1. Trim Unicode spaces (TrimSpace).
// 2. Unicode NFC (composed form).
// 3. Case fold: rune-wise unicode.ToLower for stable cross-client casing (ASCII + most Unicode).
func NormalizeFactToken(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// NFC
	s = string(norm.NFC.Bytes([]byte(s)))
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
