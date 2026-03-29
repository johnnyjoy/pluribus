package recall

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"unicode/utf8"

	"control-plane/internal/evidence"

	"github.com/google/uuid"
)

// EvidenceLister lists evidence records linked to a memory (bounded hydration for recall bundles).
type EvidenceLister interface {
	ListEvidenceForMemory(ctx context.Context, memoryID uuid.UUID) ([]evidence.Record, error)
}

// ApplyEvidenceInBundleDefaults sets bounds when enabled; no-op for nil or disabled.
func ApplyEvidenceInBundleDefaults(c *EvidenceInBundleConfig) {
	if c == nil || !c.Enabled {
		return
	}
	if c.MaxPerMemory <= 0 {
		c.MaxPerMemory = 2
	}
	if c.MaxPerBundle <= 0 {
		c.MaxPerBundle = 12
	}
	if c.SummaryMaxChars <= 0 {
		c.SummaryMaxChars = 256
	}
}

// HydrateSupportingEvidence attaches compact evidence refs to each MemoryItem in order (bucket order, then slice order).
// cfg must be non-nil with Enabled true; callers should apply ApplyEvidenceInBundleDefaults first.
func HydrateSupportingEvidence(ctx context.Context, lister EvidenceLister, cfg *EvidenceInBundleConfig, bundle *RecallBundle) error {
	if bundle == nil || cfg == nil || !cfg.Enabled || lister == nil {
		return nil
	}
	remaining := cfg.MaxPerBundle
	budgetHit := false

	hydrateSlice := func(items *[]MemoryItem) error {
		for i := range *items {
			if remaining <= 0 {
				budgetHit = true
				continue
			}
			id, err := uuid.Parse((*items)[i].ID)
			if err != nil {
				continue
			}
			records, err := lister.ListEvidenceForMemory(ctx, id)
			if err != nil {
				return err
			}
			if len(records) == 0 {
				continue
			}
			sortEvidenceRecords(records)
			take := minInt(minInt(cfg.MaxPerMemory, remaining), len(records))
			if take <= 0 {
				continue
			}
			refs := make([]EvidenceRef, 0, take)
			for _, rec := range records[:take] {
				refs = append(refs, recordToEvidenceRef(rec, cfg.SummaryMaxChars))
			}
			(*items)[i].SupportingEvidence = refs
			remaining -= take
		}
		return nil
	}

	if err := hydrateSlice(&bundle.GoverningConstraints); err != nil {
		return err
	}
	if err := hydrateSlice(&bundle.Decisions); err != nil {
		return err
	}
	if err := hydrateSlice(&bundle.KnownFailures); err != nil {
		return err
	}
	if err := hydrateSlice(&bundle.ApplicablePatterns); err != nil {
		return err
	}
	if budgetHit {
		bundle.EvidenceBudgetApplied = true
	}
	return nil
}

func sortEvidenceRecords(records []evidence.Record) {
	sort.SliceStable(records, func(i, j int) bool {
		si := evidence.BaseScore(records[i].Kind)
		sj := evidence.BaseScore(records[j].Kind)
		if si != sj {
			return si > sj
		}
		return records[i].CreatedAt.After(records[j].CreatedAt)
	})
}

func recordToEvidenceRef(rec evidence.Record, summaryMax int) EvidenceRef {
	title := rec.Kind
	if title == "" {
		title = "evidence"
	}
	base := filepath.Base(rec.Path)
	if base == "." || base == "" {
		base = rec.Path
	}
	dig := rec.Digest
	if len(dig) > 12 {
		dig = dig[:12]
	}
	summary := fmt.Sprintf("%s · %s", title, base)
	if dig != "" {
		summary += " · " + dig
	}
	summary = truncateString(summary, summaryMax)
	return EvidenceRef{
		ID:      rec.ID.String(),
		Kind:    rec.Kind,
		Title:   title,
		Summary: summary,
		Ref:     rec.Path,
	}
}

func truncateString(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	rs := []rune(s)
	if len(rs) <= max {
		return s
	}
	return string(rs[:max-1]) + "…"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
