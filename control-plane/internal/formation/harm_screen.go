package formation

import (
	"fmt"
	"strings"
)

// Harmful-advice screen (hostile audit 2026-07, finding C2).
//
// record_experience is intentionally low-friction, which makes it an injection
// vector: a single agent can plant advice like "always skip verification" that
// then surfaces to every other agent. This deterministic screen flags
// safety-negating imperative advice at ingest so it is stored QUARANTINED
// (never recalled) pending review, instead of entering the shared pool.
//
// Heuristic v1 (no LLM): flag when
//  1. a safety-negating verb appears within a short window before a safety
//     target ("skip ... verification", "disable ... auth"), and
//  2. the text is advice-shaped (imperative/normative marker present), and
//  3. the verb is not itself negated ("never skip tests" is good advice).
//
// False positives land in quarantine (reviewable), never in active recall,
// so the screen errs toward safety without destroying anything.

var harmNegatingVerbs = []string{
	"skip", "bypass", "disable", "ignore", "suppress", "circumvent",
	"turn off", "switch off", "remove", "omit", "silence",
}

var harmSafetyTargets = []string{
	"verification", "verify", "verifying", "validation", "validate", "validating",
	"test", "tests", "testing", "review", "reviews", "reviewing",
	"auth", "authentication", "authorization", "safety", "security",
	"sanitiz", "escap", "lint", "linting", "ci", "check", "checks", "checking",
	"backup", "backups", "error handling", "errors", "warnings", "guardrail",
	"guardrails", "enforcement", "audit", "logging",
}

var harmAdviceMarkers = []string{
	"always", "never", "should", "must", "just ", "remember to", "make sure",
	"best practice", "recommended", "no need", "don't bother", "do not bother",
	"you can safely", "it is safe to", "it's safe to", "critical:", "important:",
	"rule:", "tip:", "pro tip", "going forward", "from now on", "in general",
	"faster", "saves time", "to save time", "trust",
}

var harmVerbNegators = []string{
	"never", "not", "n't", "avoid", "stop", "without", "no ",
}

// standalone phrases that are harmful advice regardless of verb/target pairing.
var harmStandalonePhrases = []string{
	"without verifying", "without testing", "without checking", "without review",
	"no need to test", "no need to verify", "no need to check", "no need to review",
	"trust all input", "trust user input", "always trust", "assume it works",
	"delete the backups", "delete backups", "force push to main", "force push to master",
	"--no-verify",
}

const harmProximityWindow = 48 // max chars between verb end and target start

// HarmfulAdviceReason returns a non-empty reason when text reads as
// safety-negating imperative advice; empty string when the text is clean.
func HarmfulAdviceReason(text string) string {
	low := strings.ToLower(text)
	if strings.TrimSpace(low) == "" {
		return ""
	}

	for _, p := range harmStandalonePhrases {
		if strings.Contains(low, p) {
			return fmt.Sprintf("harmful_advice_phrase:%q", p)
		}
	}

	if !hasAdviceMarker(low) {
		return ""
	}
	for _, verb := range harmNegatingVerbs {
		start := 0
		for {
			i := indexWord(low[start:], verb)
			if i < 0 {
				break
			}
			vi := start + i
			if verbNegated(low, vi) {
				start = vi + len(verb)
				continue
			}
			windowEnd := vi + len(verb) + harmProximityWindow
			if windowEnd > len(low) {
				windowEnd = len(low)
			}
			window := low[vi+len(verb) : windowEnd]
			for _, target := range harmSafetyTargets {
				if strings.Contains(window, target) {
					return fmt.Sprintf("harmful_advice:%q near %q", verb, target)
				}
			}
			start = vi + len(verb)
		}
	}
	return ""
}

func hasAdviceMarker(low string) bool {
	for _, m := range harmAdviceMarkers {
		if strings.Contains(low, m) {
			return true
		}
	}
	// Imperative opener: text starts directly with a negating verb.
	trimmed := strings.TrimLeft(low, " \t\"'*-")
	for _, v := range harmNegatingVerbs {
		if strings.HasPrefix(trimmed, v+" ") {
			return true
		}
	}
	return false
}

// indexWord finds verb at a word boundary (avoids matching "skip" in "skipper"
// only loosely; suffixes like "skipping" still match by design).
func indexWord(s, verb string) int {
	from := 0
	for {
		i := strings.Index(s[from:], verb)
		if i < 0 {
			return -1
		}
		abs := from + i
		if abs == 0 || !isWordChar(s[abs-1]) {
			return abs
		}
		from = abs + len(verb)
		if from >= len(s) {
			return -1
		}
	}
}

func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
}

// verbNegated reports whether a negator immediately precedes the verb
// ("never skip", "don't ignore", "avoid bypassing").
func verbNegated(low string, verbIdx int) bool {
	windowStart := verbIdx - 16
	if windowStart < 0 {
		windowStart = 0
	}
	prefix := low[windowStart:verbIdx]
	for _, n := range harmVerbNegators {
		if strings.Contains(prefix, n) {
			return true
		}
	}
	return false
}
