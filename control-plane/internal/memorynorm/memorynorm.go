// Package memorynorm provides deterministic normalization for durable memory statements
// (Phase A, post-RC1 functional quality). Aligned in spirit with ingest.NormalizeFactToken
// but operates on full sentences: whitespace is collapsed across the string, not per-token only.
package memorynorm

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// PipelineVersion is incremented when normalization rules change (replay / dedup compatibility).
const PipelineVersion = "memorynorm_v1"

// StatementCanonical returns a deterministic normalized form for equality and hashing.
// Steps:
//  1. Trim outer Unicode space (strings.TrimSpace).
//  2. Unicode NFC.
//  3. Case fold: rune-wise unicode.ToLower (ASCII + most Unicode), same family as ingest token norm.
//  4. Collapse internal whitespace: split on Unicode space (strings.Fields), join with single ASCII space.
//
// Unlike ingest.NormalizeFactToken, this applies to multi-word statements; Fields collapses all runs
// of Unicode whitespace to single separators.
//
// Empty input yields "". Callers that require non-empty content must reject "" when the raw input was non-empty.
func StatementCanonical(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = string(norm.NFC.Bytes([]byte(s)))
	s = foldLower(s)
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, " ")
}

func foldLower(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// StatementKey is a fixed-length hex string suitable for dedup indexes.
// It is SHA-256 over the UTF-8 encoding of StatementCanonical(s). Collision probability is
// negligible for adversarially chosen distinct logical statements; identical keys imply identical canonical text.
func StatementKey(s string) string {
	c := StatementCanonical(s)
	if c == "" {
		return ""
	}
	h := sha256.Sum256([]byte(c))
	return hex.EncodeToString(h[:])
}

