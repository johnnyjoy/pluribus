package signal

import (
	"strings"
	"unicode/utf8"

	"control-plane/internal/merge"
)

// IntentText is query/task context for unique-line filtering.
type IntentText struct {
	Prompt    string
	Tags      []string
	Symbols   []string
	TaskTitle string
}

func tokenSetFromString(s string) map[string]struct{} {
	s = merge.Normalize(s)
	toks := strings.Fields(s)
	m := make(map[string]struct{})
	for _, t := range toks {
		if len(t) > 1 {
			m[t] = struct{}{}
		}
	}
	return m
}

func jaccard(a, b map[string]struct{}) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1
	}
	inter := 0
	for t := range a {
		if _, ok := b[t]; ok {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func distinctTokenCount(s string) int {
	return len(tokenSetFromString(s))
}

func overlapsAnyAgreement(unique string, agreements []string, tau float64) bool {
	u := tokenSetFromString(unique)
	if len(u) == 0 {
		return false
	}
	for _, a := range agreements {
		ja := jaccard(u, tokenSetFromString(a))
		if ja >= tau {
			return true
		}
	}
	return false
}

func intentTokenSet(it IntentText) map[string]struct{} {
	var parts []string
	if it.Prompt != "" {
		parts = append(parts, it.Prompt)
	}
	if it.TaskTitle != "" {
		parts = append(parts, it.TaskTitle)
	}
	for _, t := range it.Tags {
		parts = append(parts, t)
	}
	for _, s := range it.Symbols {
		parts = append(parts, s)
	}
	return tokenSetFromString(strings.Join(parts, " "))
}

func hasConstraintKeyword(s string) bool {
	low := strings.ToLower(strings.TrimSpace(s))
	prefixes := []string{"must ", "ensure ", "shall ", "constraint", "do not ", "don't ", "never ", "always "}
	for _, p := range prefixes {
		if strings.HasPrefix(low, p) {
			return true
		}
		if strings.Contains(low, " "+p) {
			return true
		}
	}
	return false
}

// FilterUniques keeps unique lines that pass length, diversity, agreement/intent overlap, and keyword heuristics.
func FilterUniques(uniques []string, agreements []string, intent IntentText, cfg SignalConfig) []string {
	cfg = normalizeSignalConfig(cfg)
	var out []string
	intentToks := intentTokenSet(intent)
	hasIntent := len(intentToks) > 0

	for _, u := range uniques {
		u = strings.TrimSpace(u)
		if u == "" || u == "(none)" {
			continue
		}
		if utf8.RuneCountInString(u) < cfg.MinUniqueRunes {
			continue
		}
		if distinctTokenCount(u) < cfg.MinDistinctTokens {
			continue
		}
		agreeOK := overlapsAnyAgreement(u, agreements, cfg.AgreementOverlapTau)
		keyOK := hasConstraintKeyword(u)
		if !agreeOK && !keyOK {
			continue
		}
		if hasIntent {
			uTok := tokenSetFromString(u)
			if jaccard(uTok, intentToks) < cfg.IntentOverlapTau {
				continue
			}
		}
		out = append(out, u)
	}
	return out
}
