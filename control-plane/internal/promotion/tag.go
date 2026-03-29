package promotion

import (
	"strings"

	"control-plane/internal/merge"
)

// BuildTags merges structural tags with merge metadata and light content keywords.
func BuildTags(rec ExperienceRecord, m merge.MergeResult) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(strings.ToLower(s))
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	add("experience")
	add("merge")
	if len(m.Drift.Violations) == 0 {
		add("drift-safe")
	}
	for _, t := range rec.Tags {
		add(t)
	}
	for _, v := range m.UsedVariants {
		add(v)
	}
	contentKeywords(rec.Content, add)
	return out
}

func contentKeywords(content string, add func(string)) {
	low := strings.ToLower(content)
	keywords := []string{"api", "error", "test", "security", "performance", "database", "cache", "auth"}
	for _, k := range keywords {
		if strings.Contains(low, k) {
			add(k)
		}
	}
}
