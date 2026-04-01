package distillation

import (
	"strings"

	"control-plane/pkg/api"
)

// draft is one proposed candidate kind + rule tag for the reason field.
type draft struct {
	kind   api.MemoryKind
	reason string
}

var (
	constraintHints = []string{"must not", "never ", "never.", "shall not", "prohibited", "required that", "do not ", "don't ", "cannot "}
	failureHints    = []string{"error", "rollback", "rolled back", "failed", "rejection", "rejected", "timeout", "outage", "exception", "blocked", "incident"}
	decisionHints   = []string{" chose ", " decided ", " picked ", " selected ", " instead of ", " went with ", " opted for "}
	patternHints    = []string{"worked", "this worked", " best practice", " pattern", "always use", "always "}
	// plausibleWeakHints — broad intake: insight / experiment language without requiring strict distillation keywords.
	plausibleWeakHints = []string{
		"learned", "observed", "hypothesis", "experiment", "benchmark", "trial", "measured",
		"compared", "validated", "metrics", "latency", "throughput", "outcome", "outcomes",
		"finding", "takeaway", "lesson", "insight", "baseline", "candidate", "variant",
		"reproduce", "repro", "a/b", "ab test", "sample size", "p95", "p99",
	}
)

func extractDrafts(lower string) []draft {
	var out []draft
	if containsAny(lower, constraintHints) {
		out = append(out, draft{kind: api.MemoryKindConstraint, reason: "distilled:constraint_keywords"})
	}
	if containsAny(lower, failureHints) {
		out = append(out, draft{kind: api.MemoryKindFailure, reason: "distilled:failure_keywords"})
	}
	if containsAny(lower, decisionHints) {
		out = append(out, draft{kind: api.MemoryKindDecision, reason: "distilled:decision_keywords"})
	}
	if containsAny(lower, patternHints) {
		out = append(out, draft{kind: api.MemoryKindPattern, reason: "distilled:pattern_keywords"})
	}
	return out
}

func containsAny(s string, hints []string) bool {
	for _, h := range hints {
		if strings.Contains(s, strings.TrimSpace(h)) {
			return true
		}
	}
	return false
}

func normalizeLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func clipStatement(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// KindHintFromSummary picks one memory kind from keyword signals (same priority as
// QualifyForProbationaryMemory). When no plausibly useful signal matches, returns ("", false).
func KindHintFromSummary(summary string) (api.MemoryKind, bool) {
	kind, _, ok, _ := QualifyForProbationaryMemory(summary, nil)
	return kind, ok
}

// QualifyForProbationaryMemory decides ingest-time probationary memory formation.
// Strong keyword / event signals set weakSignal=false (default authority 2 at ingest).
// Plausible-weak paths set weakSignal=true (authority 1): ranking must separate quality over time.
// Returns ok=false only for clear garbage or empty signal after checks.
func QualifyForProbationaryMemory(summary string, tags []string) (kind api.MemoryKind, signalReason string, ok bool, weakSignal bool) {
	lower := normalizeLower(summary)
	if strings.TrimSpace(lower) == "" {
		return "", "", false, false
	}
	// Event tags first so short MCP summaries still qualify (broad capture).
	for _, t := range tags {
		if k, r, okTag, weak := eventTagMemorySignal(t); okTag {
			return k, r, true, weak
		}
	}
	if isIngestGarbage(lower) {
		return "", "", false, false
	}
	drafts := extractDrafts(lower)
	if len(drafts) > 0 {
		order := []api.MemoryKind{
			api.MemoryKindConstraint,
			api.MemoryKindFailure,
			api.MemoryKindDecision,
			api.MemoryKindPattern,
		}
		for _, want := range order {
			for _, d := range drafts {
				if d.kind == want {
					return want, d.reason, true, false
				}
			}
		}
		return drafts[0].kind, drafts[0].reason, true, false
	}
	if k, r, okPl := qualifyPlausibleWeak(lower, tags); okPl {
		return k, r, true, true
	}
	return "", "", false, false
}

func isIngestGarbage(lower string) bool {
	s := strings.TrimSpace(lower)
	if s == "" {
		return true
	}
	r := []rune(s)
	if len(r) < 8 {
		return true
	}
	if isSingleRuneRepeat(r) {
		return true
	}
	// Tiny vocabulary spam (e.g. "lol lol lol lol")
	words := strings.Fields(s)
	if len(words) <= 2 {
		w0 := words[0]
		if len(words) == 1 && (w0 == "test" || w0 == "asdf" || w0 == "hello" || w0 == "hi" || w0 == "lol") {
			return true
		}
	}
	return false
}

func isSingleRuneRepeat(r []rune) bool {
	if len(r) < 8 {
		return false
	}
	first := r[0]
	for _, c := range r[1:] {
		if c != first {
			return false
		}
	}
	return true
}

// qualifyPlausibleWeak admits longer, context-rich or experiment-tagged text without strict keyword lists.
func qualifyPlausibleWeak(lower string, tags []string) (api.MemoryKind, string, bool) {
	for _, t := range tags {
		tl := strings.ToLower(strings.TrimSpace(t))
		if tl == "" {
			continue
		}
		if strings.Contains(tl, "experiment") || strings.Contains(tl, "benchmark") ||
			strings.Contains(tl, "trial") || strings.HasPrefix(tl, "bench:") {
			return api.MemoryKindPattern, "plausible:experiment_context_tag", true
		}
	}
	words := strings.Fields(lower)
	if len(words) < 5 {
		return "", "", false
	}
	if containsAny(lower, plausibleWeakHints) {
		return api.MemoryKindPattern, "plausible:insight_keywords", true
	}
	if len(words) >= 12 && wordDiversity(words) >= 6 {
		return api.MemoryKindPattern, "plausible:long_context", true
	}
	return "", "", false
}

func wordDiversity(words []string) int {
	seen := make(map[string]struct{})
	for _, w := range words {
		w = strings.Trim(strings.ToLower(w), ".,;:!?\"'()[]{}")
		if len(w) < 2 {
			continue
		}
		seen[w] = struct{}{}
	}
	return len(seen)
}

// eventTagMemorySignal maps mcp:event:* tags (MCP record_experience) to a memory kind.
// weak=true means use lower ingest authority (ranking differentiates over time).
func eventTagMemorySignal(tag string) (kind api.MemoryKind, reason string, ok bool, weak bool) {
	const p = "mcp:event:"
	k := normalizeLower(strings.TrimSpace(tag))
	if !strings.HasPrefix(k, p) {
		return "", "", false, false
	}
	suf := strings.TrimSpace(k[len(p):])
	if suf == "" || suf == "unspecified" {
		return "", "", false, false
	}
	switch suf {
	case "failure", "error", "incident", "outage", "timeout", "regression", "rollback", "blocked":
		return api.MemoryKindFailure, "mcp:event:" + suf, true, false
	case "decision", "learning", "fix":
		return api.MemoryKindDecision, "mcp:event:" + suf, true, false
	case "constraint", "violation":
		return api.MemoryKindConstraint, "mcp:event:" + suf, true, false
	case "experiment", "benchmark", "trial", "bench", "abtest", "ab-test":
		return api.MemoryKindPattern, "mcp:event:" + suf, true, true
	default:
		// Unknown event kinds: still capture (broad intake); treat as weak pattern.
		return api.MemoryKindPattern, "mcp:event:" + suf, true, true
	}
}
