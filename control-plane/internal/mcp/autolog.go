package mcp

import (
	"strings"
	"unicode"
)

// repeatedBehaviorPhrases are deterministic substrings that indicate iteration / sameness (no LLM).
var repeatedBehaviorPhrases = []string{
	" again", " again.", " again,", " repeated ", " repeated.", "repeat ",
	" second time", " third time", "same failure", "same error", "same bug",
	"still failing", "still broken", "keeps failing", "multiple times",
}

// AutoLogEpisodeIfRelevant decides whether opportunistic episode ingest should run (deterministic).
// Used by memory_log_if_relevant and aligned with low-friction mcp_episode_ingest capture.
func AutoLogEpisodeIfRelevant(text string) (shouldIngest bool, reason string) {
	s := strings.TrimSpace(text)
	if len([]rune(s)) < 8 {
		return false, "too_short"
	}
	low := strings.ToLower(s)
	for _, tok := range distillSignalTokens {
		if strings.Contains(low, tok) {
			return true, "learning_signal"
		}
	}
	for _, phrase := range repeatedBehaviorPhrases {
		if strings.Contains(low, phrase) {
			return true, "repeated_behavior"
		}
	}
	if repeatedSignificantToken(low) {
		return true, "repeated_token"
	}
	return false, "no_signal"
}

func repeatedSignificantToken(low string) bool {
	fields := strings.FieldsFunc(low, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	counts := make(map[string]int)
	for _, w := range fields {
		if len(w) < 5 {
			continue
		}
		counts[w]++
		if counts[w] >= 2 {
			return true
		}
	}
	return false
}
