package formation

import (
	"regexp"
	"strings"
)

// junkExactPhrases are normalized summaries/statements rejected as low-value memory.
var junkExactPhrases = []string{
	"worked on project.",
	"worked on project",
	"fixed typo.",
	"fixed typo",
	"the thing is done.",
	"the thing is done",
	"remember this maybe.",
	"remember this maybe",
	"important: everything is critical.",
	"important: everything is critical",
	"do better next time.",
	"do better next time",
	"made progress.",
	"made progress",
	"had issue.",
	"had issue",
	"learned important thing.",
	"learned important thing",
}

var junkPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^fixed\s+(a\s+)?typo\.?$`),
	regexp.MustCompile(`(?i)^worked\s+on\s+(the\s+)?project\.?$`),
	regexp.MustCompile(`(?i)^made\s+progress\.?$`),
	regexp.MustCompile(`(?i)^had\s+(an?\s+)?issue\.?$`),
	regexp.MustCompile(`(?i)^learned\s+(an?\s+)?important\s+(thing|stuff)\.?$`),
	regexp.MustCompile(`(?i)^remember\s+this\s+maybe\.?$`),
	regexp.MustCompile(`(?i)^do\s+better\s+next\s+time\.?$`),
}

// keywordSpamPhrases match vague summaries that only trigger weak keyword gates.
var keywordSpamPhrases = []string{
	"learned important thing",
	"important: everything is critical",
	"worked on project",
	"made progress",
	"fixed typo",
}

// IsJunkStatement returns true when text is too vague/trivial for durable memory.
func IsJunkStatement(statement string) bool {
	s := strings.TrimSpace(strings.ToLower(statement))
	if s == "" {
		return true
	}
	for _, p := range junkExactPhrases {
		if s == p {
			return true
		}
	}
	for _, re := range junkPatterns {
		if re.MatchString(s) {
			return true
		}
	}
	if isKeywordSpamOnly(s) {
		return true
	}
	return false
}

func isKeywordSpamOnly(lower string) bool {
	for _, p := range keywordSpamPhrases {
		if lower == p || strings.HasPrefix(lower, p+".") {
			return true
		}
	}
	// "Worked on project." style with trailing context that's still junk
	if strings.Contains(lower, "worked on project") && wordCount(lower) < 8 {
		return true
	}
	if strings.Contains(lower, "made progress") && wordCount(lower) < 7 {
		return true
	}
	return false
}

// IsWeakRecordExperienceSummary rejects vague advisory summaries before probationary formation.
func IsWeakRecordExperienceSummary(summary string, minActionableWords int) bool {
	if IsJunkStatement(summary) {
		return true
	}
	s := strings.TrimSpace(summary)
	if minActionableWords <= 0 {
		minActionableWords = 4
	}
	words := meaningfulWords(s)
	if len(words) < minActionableWords {
		return true
	}
	// Generic single-clause without concrete object/action
	lower := strings.ToLower(s)
	if wordCount(lower) <= 5 && containsAny(lower, []string{"worked", "progress", "important", "learned", "fixed", "issue"}) {
		return true
	}
	return false
}

func wordCount(s string) int {
	return len(strings.Fields(s))
}

func meaningfulWords(s string) []string {
	var out []string
	for _, w := range strings.Fields(s) {
		w = strings.Trim(strings.ToLower(w), ".,;:!?\"'()[]{}")
		if len(w) < 2 {
			continue
		}
		if isStopWord(w) {
			continue
		}
		out = append(out, w)
	}
	return out
}

func isStopWord(w string) bool {
	switch w {
	case "the", "a", "an", "on", "in", "at", "to", "for", "of", "and", "or", "is", "was", "were", "be", "this", "that", "it", "had", "has", "have":
		return true
	default:
		return false
	}
}

func containsAny(s string, hints []string) bool {
	for _, h := range hints {
		if strings.Contains(s, h) {
			return true
		}
	}
	return false
}
