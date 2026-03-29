package merge

import (
	"regexp"
	"strings"

	"control-plane/internal/runmulti"
)

// MinSegmentLen drops very short fragments (noise).
const MinSegmentLen = 12

var paraBreak = regexp.MustCompile(`\n\s*\n`)
var bulletLine = regexp.MustCompile(`^\s*([-*•]|\d+\.)\s+`)

// ExtractSegments splits run outputs into paragraphs or bullet lines (deterministic run order).
func ExtractSegments(runs []runmulti.RunResult) []Segment {
	var out []Segment
	for _, r := range runs {
		raw := strings.TrimSpace(r.Output)
		if raw == "" {
			continue
		}
		paras := paraBreak.Split(raw, -1)
		for _, p := range paras {
			p = strings.TrimSpace(p)
			if len(p) < MinSegmentLen {
				continue
			}
			if isAllBulletLines(p) {
				for _, line := range strings.Split(p, "\n") {
					line = strings.TrimSpace(line)
					if line == "" || len(line) < MinSegmentLen {
						continue
					}
					out = append(out, Segment{Text: line, Variant: r.Variant})
				}
			} else {
				out = append(out, Segment{Text: p, Variant: r.Variant})
			}
		}
	}
	return out
}

func isAllBulletLines(para string) bool {
	lines := strings.Split(para, "\n")
	hasAny := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		hasAny = true
		if !bulletLine.MatchString(line) {
			return false
		}
	}
	return hasAny
}
