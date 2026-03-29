package signal

import (
	"sort"
	"strings"
	"unicode/utf8"

	"control-plane/internal/merge"
)

// SegmentScore describes one scored line (agreement or passing unique).
type SegmentScore struct {
	Frequency int     // synthetic frequency from merge metadata
	Length    int     // rune count
	Position  int     // ordinal in scored list (0 = first)
	Score     float64 // composite
}

// LengthWeight rewards medium-length lines; penalizes very short or very long.
func LengthWeight(runeLen int) float64 {
	if runeLen < 20 {
		return 0.25
	}
	if runeLen <= 800 {
		return 2.0 + float64(runeLen)/2000.0
	}
	return 2.4 - float64(runeLen-800)/5000.0 // gentle decay for huge lines
}

// PositionPenalty increases with ordinal (earlier segments score higher).
func PositionPenalty(ordinal int) float64 {
	return float64(ordinal) * 0.08
}

// agreementFrequency returns synthetic frequency for agreement lines.
func agreementFrequency(usedVariants int) int {
	if usedVariants >= 2 {
		if usedVariants > 2 {
			return usedVariants
		}
		return 2
	}
	return 1
}

// ScoreSegment computes composite score for one segment.
func ScoreSegment(frequency, runeLen, position int) float64 {
	return float64(frequency)*2.0 + LengthWeight(runeLen) - PositionPenalty(position)
}

func sortedLines(in []string) []string {
	out := append([]string(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		return merge.Normalize(out[i]) < merge.Normalize(out[j])
	})
	return out
}

// SegmentScores builds scores for sorted agreements then filtered uniques (stable ordering).
func SegmentScores(m merge.MergeResult, filteredUniques []string) []SegmentScore {
	agreements := sortedLines(m.Agreements)
	uniques := sortedLines(filteredUniques)
	freq := agreementFrequency(len(m.UsedVariants))

	var scores []SegmentScore
	pos := 0
	for _, line := range agreements {
		line = strings.TrimSpace(line)
		if line == "" || line == "(none)" {
			continue
		}
		n := utf8.RuneCountInString(line)
		s := ScoreSegment(freq, n, pos)
		scores = append(scores, SegmentScore{Frequency: freq, Length: n, Position: pos, Score: s})
		pos++
	}
	for _, line := range uniques {
		line = strings.TrimSpace(line)
		if line == "" || line == "(none)" {
			continue
		}
		n := utf8.RuneCountInString(line)
		s := ScoreSegment(1, n, pos)
		scores = append(scores, SegmentScore{Frequency: 1, Length: n, Position: pos, Score: s})
		pos++
	}
	return scores
}

// TotalSignal sums segment scores.
func TotalSignal(scores []SegmentScore) float64 {
	var t float64
	for _, s := range scores {
		t += s.Score
	}
	return t
}
