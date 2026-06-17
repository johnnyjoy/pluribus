package formationquality

import (
	"strings"
)

var genericCues = map[string]struct{}{
	"project": {}, "thing": {}, "stuff": {}, "work": {}, "important": {},
	"misc": {}, "general": {}, "memory": {}, "task": {}, "note": {}, "item": {},
}

// EvaluateCues scores retrieval cue quality.
func EvaluateCues(in Input) (score float64, misleading bool, underCued bool) {
	cues := in.RetrievalCues
	if len(cues) == 0 {
		// Tags can supplement cues for probationary/advisory paths.
		for _, t := range in.Tags {
			t = strings.TrimSpace(t)
			if t != "" && !strings.HasPrefix(strings.ToLower(t), "source:") {
				cues = append(cues, t)
			}
		}
	}
	if len(cues) == 0 {
		return 0, false, true
	}
	var useful, generic int
	for _, c := range cues {
		c = strings.ToLower(strings.TrimSpace(c))
		if c == "" {
			continue
		}
		if _, bad := genericCues[c]; bad {
			generic++
			continue
		}
		if len(c) >= 3 {
			useful++
		}
	}
	total := useful + generic
	if total == 0 {
		return 0, generic > 0, true
	}
	score = float64(useful) / float64(total)
	misleading = generic > 0 && useful == 0
	underCued = useful < 2 && IsDirectLikePath(in.Path)
	if !underCued && IsProbationaryPath(in.Path) {
		underCued = useful < 1 && len(cues) > 0 && generic > 0
	}
	return score, misleading, underCued
}
