package promotion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"control-plane/internal/merge"
	"control-plane/internal/signal"
)

const maxExperienceRunes = 60000 // ~64KiB of text in runes, stay under JSON size issues

// PromoteFromMerge writes an experience record when merge outcome passes all gates and IsHighSignal.
// Returns promoted=true when a new line was appended to the JSONL store.
func PromoteFromMerge(ctx context.Context, m merge.MergeResult, in PromotionInput, sigCfg signal.SignalConfig) (promoted bool, err error) {
	_ = ctx
	if !signal.IsHighSignal(m, in.Intent, sigCfg) {
		return false, nil
	}
	content := strings.TrimSpace(m.MergedOutput)
	if content == "" {
		return false, nil
	}
	if len([]rune(content)) > maxExperienceRunes {
		r := []rune(content)
		content = string(r[:maxExperienceRunes]) + "\n...[truncated]"
	}

	filtered := signal.FilterUniques(m.Unique, m.Agreements, in.Intent, sigCfg)
	scores := signal.SegmentScores(m, filtered)
	total := signal.TotalSignal(scores)

	rec := ExperienceRecord{
		Type:           experienceType,
		Timestamp:      time.Now().UTC(),
		Tags:           nil,
		Content:        content,
		SourceVariants: append([]string(nil), m.UsedVariants...),
		Score:          total,
	}
	rec.Tags = BuildTags(rec, m)

	path := in.StorePath
	if path == "" {
		path = DefaultExperiencesPath
	}
	if err := AppendRecord(path, rec); err != nil {
		return false, err
	}
	return true, nil
}

// ContentFingerprint returns a stable hex digest for logging/testing.
func ContentFingerprint(content string, ts time.Time) string {
	h := sha256.Sum256([]byte(content + "\n" + ts.UTC().Format(time.RFC3339Nano)))
	return hex.EncodeToString(h[:16])
}
