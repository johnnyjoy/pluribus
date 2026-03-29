package recall

import (
	"context"
	"log/slog"
	"strings"
)

// CompileTriggered runs heuristic triggers, merges RetrievalQuery, then calls Compile once.
func (s *Service) CompileTriggered(ctx context.Context, req CompileRequest) (*RecallBundle, *TriggerMetadata, error) {
	cfg := NormalizeTriggerRecall(s.TriggerRecall)
	meta := &TriggerMetadata{}
	if cfg == nil || !cfg.Enabled {
		meta.SkippedReason = "triggered_recall_disabled"
		b, err := s.Compile(ctx, req)
		if b != nil {
			b.TriggerMetadata = meta
		}
		return b, meta, err
	}

	tin := s.buildTriggerInput(ctx, req)
	raw := DetectTriggers(tin, cfg.MinContextTokens)
	triggers := filterTriggersByConfig(raw, cfg)
	var skipCap string
	triggers, skipCap = capTriggers(triggers, cfg.MaxTriggersPerRequest)
	if skipCap != "" {
		meta.SkippedReason = skipCap
	}

	meta.Triggers = triggers
	if len(triggers) == 0 {
		meta.RetrievalQueryEffective = strings.TrimSpace(req.RetrievalQuery)
		b, err := s.Compile(ctx, req)
		if b != nil {
			b.TriggerMetadata = meta
		}
		return b, meta, err
	}

	effective := mergeRetrievalQuery(req.RetrievalQuery, triggers)
	meta.RetrievalQueryEffective = effective

	logTriggerBlock(triggers, effective, skipCap != "")

	req2 := req
	req2.RetrievalQuery = effective

	b, err := s.Compile(ctx, req2)
	if err != nil {
		return nil, meta, err
	}
	logRecallResultBlock(b)
	if b != nil {
		b.TriggerMetadata = meta
	}
	return b, meta, err
}

func (s *Service) buildTriggerInput(ctx context.Context, req CompileRequest) TriggerInput {
	in := TriggerInput{
		ProposalText:  req.ProposalText,
		ExistingQuery: req.RetrievalQuery,
		Tags:          req.Tags,
	}
	_ = ctx
	_ = s
	return in
}

func mergeRetrievalQuery(base string, triggers []TriggerDecision) string {
	var parts []string
	if b := strings.TrimSpace(base); b != "" {
		parts = append(parts, b)
	}
	for _, t := range triggers {
		if f := TriggerFragment(t); f != "" {
			parts = append(parts, f)
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func logTriggerBlock(triggers []TriggerDecision, effectiveQuery string, dedupeSkipped bool) {
	q := effectiveQuery
	if len(q) > 200 {
		q = q[:200] + "…"
	}
	var kinds, reasons []string
	for _, t := range triggers {
		kinds = append(kinds, string(t.Kind))
		reasons = append(reasons, t.Reason)
	}
	slog.Info("[TRIGGER]", "types", kinds, "reasons", reasons, "effective_query", q, "dedupe_skipped", dedupeSkipped)
}

func logRecallResultBlock(b *RecallBundle) {
	if b == nil {
		return
	}
	top := func(items []MemoryItem, n int) []string {
		var s []string
		for i := range items {
			if i >= n {
				break
			}
			st := strings.TrimSpace(items[i].Statement)
			if len(st) > 80 {
				st = st[:80] + "…"
			}
			s = append(s, items[i].ID+":"+st)
		}
		return s
	}
	slog.Info("[RECALL_RESULT]",
		"constraints", top(b.GoverningConstraints, 3),
		"failures", top(b.KnownFailures, 3),
		"patterns", top(b.ApplicablePatterns, 3),
	)
}
