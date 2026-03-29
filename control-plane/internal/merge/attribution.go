package merge

import (
	"sort"
	"strings"

	"control-plane/internal/runmulti"
)

// AttributionRole classifies a line in the merge attribution list.
type AttributionRole string

const (
	AttributionAgreement       AttributionRole = "agreement"
	AttributionUnique          AttributionRole = "unique"
	AttributionDroppedConflict AttributionRole = "dropped_conflict"
)

// AttributionLine ties synthesized bullet text to contributing variants for observability.
type AttributionLine struct {
	Text     string          `json:"text"`
	Role     AttributionRole `json:"role"`
	Variants []string        `json:"variants"` // sorted, unique contributors
}

// MergeDebug is structured observability for merge; safe to JSON-encode.
type MergeDebug struct {
	SegmentsIn       int `json:"segments_in"`
	ConflictsDropped int `json:"conflicts_dropped"`
	UniquesCapped    int `json:"uniques_capped"` // Phase 5.2: lines removed by cap/dedupe/agreement filter
	Attribution      []AttributionLine `json:"attribution,omitempty"`
}

func buildMergeDebug(segments []Segment, bad map[int]struct{}, clusters []cluster, agreements, uniques, conflictTexts []string, uniquesCapped int) MergeDebug {
	md := MergeDebug{
		SegmentsIn:       len(segments),
		ConflictsDropped: len(conflictTexts),
		UniquesCapped:    uniquesCapped,
		Attribution:      buildAttributionLines(segments, bad, clusters, agreements, uniques, conflictTexts),
	}
	return md
}

func buildAttributionLines(segments []Segment, bad map[int]struct{}, clusters []cluster, agreements, uniques, conflictTexts []string) []AttributionLine {
	var lines []AttributionLine

	agNorm := make(map[string]string)
	for _, a := range agreements {
		agNorm[Normalize(a)] = strings.TrimSpace(a)
	}
	for _, cl := range clusters {
		vs := variantSet(cl.segs)
		if len(vs) < 2 {
			continue
		}
		rep := agreementRep(cl.segs)
		n := Normalize(rep)
		if _, ok := agNorm[n]; !ok {
			continue
		}
		variants := sortedVariantKeys(vs)
		lines = append(lines, AttributionLine{
			Text:     strings.TrimSpace(rep),
			Role:     AttributionAgreement,
			Variants: variants,
		})
		delete(agNorm, n)
	}

	uniNorm := make(map[string]struct{})
	for _, u := range uniques {
		uniNorm[Normalize(u)] = struct{}{}
	}
	for _, cl := range clusters {
		if len(variantSet(cl.segs)) != 1 {
			continue
		}
		for _, s := range cl.segs {
			t := strings.TrimSpace(s.Text)
			if _, ok := uniNorm[Normalize(t)]; !ok {
				continue
			}
			lines = append(lines, AttributionLine{
				Text:     t,
				Role:     AttributionUnique,
				Variants: []string{s.Variant},
			})
			delete(uniNorm, Normalize(t))
		}
	}

	sortedCT := append([]string(nil), conflictTexts...)
	sort.Slice(sortedCT, func(i, j int) bool {
		return Normalize(sortedCT[i]) < Normalize(sortedCT[j])
	})
	seen := make(map[string]struct{})
	for _, ct := range sortedCT {
		t := strings.TrimSpace(ct)
		k := Normalize(t)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		lines = append(lines, AttributionLine{
			Text:     t,
			Role:     AttributionDroppedConflict,
			Variants: variantsForConflictText(segments, bad, t),
		})
	}

	sort.Slice(lines, func(i, j int) bool {
		if lines[i].Role != lines[j].Role {
			return lines[i].Role < lines[j].Role
		}
		return Normalize(lines[i].Text) < Normalize(lines[j].Text)
	})
	return lines
}

func agreementRep(segs []Segment) string {
	rep := segs[0].Text
	for _, s := range segs[1:] {
		if Normalize(s.Text) < Normalize(rep) {
			rep = s.Text
		}
	}
	return rep
}

func sortedVariantKeys(m map[string]struct{}) []string {
	var out []string
	for v := range m {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// MergeDebugFromRuns counts segments for single-run or empty merge paths (no multi-merge attribution).
func MergeDebugFromRuns(runs []runmulti.RunResult) MergeDebug {
	segs := ExtractSegments(runs)
	return MergeDebug{SegmentsIn: len(segs), ConflictsDropped: 0, UniquesCapped: 0}
}

func variantsForConflictText(segments []Segment, bad map[int]struct{}, text string) []string {
	want := Normalize(text)
	m := make(map[string]struct{})
	for i, seg := range segments {
		if _, ok := bad[i]; !ok {
			continue
		}
		if Normalize(strings.TrimSpace(seg.Text)) != want {
			continue
		}
		m[seg.Variant] = struct{}{}
	}
	return sortedVariantKeys(m)
}
