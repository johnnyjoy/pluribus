package recall

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"control-plane/internal/cache"
	"control-plane/internal/memory"
	"control-plane/internal/merge"
	"control-plane/internal/runmulti"
	"control-plane/internal/signal"
	"control-plane/internal/tooling"

	"github.com/google/uuid"
)

var ErrNoCompiler = errors.New("recall is unavailable: the recall compiler is not configured on this server")
var ErrRunMultiNotConfigured = errors.New("run-multi is unavailable: the server-side runner is not configured (enable synthesis in config or use client-side run-multi)")

// RunMultiExecutor abstracts runmulti.Runner for testability.
type RunMultiExecutor interface {
	Run(ctx context.Context, input runmulti.RunMultiInput) (*runmulti.RunMultiResult, error)
}

// MemoryPromoter abstracts memory promotion endpoint logic for reuse in recall run-multi.
type MemoryPromoter interface {
	Promote(ctx context.Context, req memory.PromoteRequest) (*memory.PromoteResponse, error)
}

// MemoryUsageReinforcer reinforces authority for recalled/validated memory usage.
type MemoryUsageReinforcer interface {
	ReinforceRecallUsage(ctx context.Context, ids []uuid.UUID) error
}

func confidenceFromRuns(selected *runmulti.RunResult, runs []runmulti.RunResult) float64 {
	if selected == nil {
		return 0
	}
	best := selected.Score
	second := -1.0
	for _, r := range runs {
		if r.Variant == selected.Variant {
			continue
		}
		if second < 0 || r.Score < second {
			second = r.Score
		}
	}
	// Base confidence from best score.
	base := 1.0 / (1.0 + best)
	if second < 0 {
		return base
	}
	// Increase confidence when best and runner-up are far apart.
	delta := second - best
	if delta <= 0 {
		return base * 0.8
	}
	conf := base + (delta/(1.0+delta))*0.3
	if conf > 1 {
		return 1
	}
	return conf
}

// Service provides recall compile and preflight use cases.
type Service struct {
	Compiler         *Compiler
	Repo             *Repo
	Cache            cache.Store
	CacheTTL         time.Duration
	DefaultMaxTotal  int // RIE: 0 = no cap
	DefaultMaxTokens int // RIE: 0 = no token cap
	// SlowPath config for preflight (Task 93). When set and risk >= threshold, Preflight sets SlowPathRequired and RecommendedExpansion.
	SlowPath *SlowPathPreflightConfig
	// Pluribus Phase B: server-side run-multi orchestration dependencies.
	RunMultiRunner  RunMultiExecutor
	RunMultiBaseURL string
	// Pluribus Phase C: canonical promotion path.
	MemoryPromoter MemoryPromoter
	// UsageReinforcer updates authority from recall usage/success signals.
	UsageReinforcer MemoryUsageReinforcer
	// Pluribus phase-next: optional signal thresholds for promotion gates and debug (nil = defaults).
	RunMultiSignalConfig *signal.SignalConfig
	// optional promotion policy (nil = defaults; no extra gates).
	Promotion *PromotionPolicy
	// optional evidence scoring for min_evidence_score gate (nil = skip score gate).
	Evidence EvidencePolicyChecker
	// EvidenceInBundle enables bounded supporting_evidence on MemoryItem (nil or enabled=false = off).
	EvidenceInBundle *EvidenceInBundleConfig
	// EvidenceLister loads linked evidence for memories (typically *evidence.Service).
	EvidenceLister EvidenceLister
	// optional LSP auto-symbol fill for compile / compile-multi (when lsp.enabled in config).
	LSPEnabled       bool
	LSP              tooling.LSPClient
	LSPAutoSymbolMax int // 0 = default 64 in enrich
	// TriggerRecall optional; nil = triggered recall off unless YAML enables and router sets.
	TriggerRecall *TriggerRecallConfig
	// BehaviorValidation optional gates for run-multi overlap validation (nil = defaults).
	BehaviorValidation *BehaviorValidationConfig
}

// Compile builds a RecallBundle and optionally persists it. Uses cache when configured (cache is not authoritative).
func (s *Service) Compile(ctx context.Context, req CompileRequest) (*RecallBundle, error) {
	maxPerKind := req.MaxPerKind
	if maxPerKind <= 0 {
		maxPerKind = 5
	}
	if req.MaxTotal <= 0 && s.DefaultMaxTotal > 0 {
		req.MaxTotal = s.DefaultMaxTotal
	}
	if req.MaxTokens <= 0 && s.DefaultMaxTokens > 0 {
		req.MaxTokens = s.DefaultMaxTokens
	}
	s.enrichCompileSymbols(ctx, &req)
	key := cache.RecallBundleKey(req.Tags, maxPerKind, req.MaxTotal, req.MaxTokens,
		req.RetrievalQuery, req.ProposalText,
		req.Symbols, req.RepoRoot, req.LSPFocusPath, req.LSPFocusLine, req.LSPFocusColumn, s.evidenceBundleCacheKey())
	if s.Cache != nil && s.CacheTTL > 0 {
		if b, err := s.Cache.Get(ctx, key); err == nil && len(b) > 0 {
			var bundle RecallBundle
			if json.Unmarshal(b, &bundle) == nil {
				return &bundle, nil
			}
		}
	}
	if s.Compiler == nil {
		return nil, ErrNoCompiler
	}
	bundle, err := s.Compiler.Compile(ctx, req)
	if err != nil {
		return nil, err
	}
	s.reinforceRecallUsage(ctx, bundleMemoryIDs(bundle), memory.ReinforceMeta{
		Reason:         "reuse_recall",
		ContextKey:     "",
		AgentKey:       memory.AgentUsageKey(req.AgentID),
		Impact:         "low",
		SignalStrength: len(bundleMemoryIDs(bundle)),
	})
	if err := s.hydrateEvidence(ctx, bundle); err != nil {
		return nil, err
	}
	if s.Cache != nil && s.CacheTTL > 0 {
		if b, err := json.Marshal(bundle); err == nil {
			_ = s.Cache.Set(ctx, key, b, s.CacheTTL)
		}
	}
	return bundle, nil
}

func (s *Service) evidenceBundleCacheKey() cache.EvidenceBundleKey {
	if s.EvidenceInBundle == nil || !s.EvidenceInBundle.Enabled {
		return cache.EvidenceBundleKey{}
	}
	return cache.EvidenceBundleKey{
		Enabled:         s.EvidenceInBundle.Enabled,
		MaxPerMemory:    s.EvidenceInBundle.MaxPerMemory,
		MaxPerBundle:    s.EvidenceInBundle.MaxPerBundle,
		SummaryMaxChars: s.EvidenceInBundle.SummaryMaxChars,
	}
}

func (s *Service) hydrateEvidence(ctx context.Context, bundle *RecallBundle) error {
	if bundle == nil || s.EvidenceInBundle == nil || !s.EvidenceInBundle.Enabled || s.EvidenceLister == nil {
		return nil
	}
	return HydrateSupportingEvidence(ctx, s.EvidenceLister, s.EvidenceInBundle, bundle)
}

// Preflight returns risk level and required actions for the given request. Uses cache when configured.
func (s *Service) Preflight(ctx context.Context, req PreflightRequest) PreflightResult {
	key := cache.PreflightKey(req.ChangedFilesCount, req.Tags)
	if s.Cache != nil && s.CacheTTL > 0 {
		if b, err := s.Cache.Get(ctx, key); err == nil && len(b) > 0 {
			var result PreflightResult
			if json.Unmarshal(b, &result) == nil {
				return result
			}
		}
	}
	result := ComputePreflight(req)
	if s.SlowPath != nil && s.SlowPath.Enabled && result.RiskScore >= s.SlowPath.HighRiskThreshold {
		result.SlowPathRequired = true
		result.SlowPathReasons = append(result.SlowPathReasons, "risk score exceeds high_risk_threshold")
		result.RecommendedExpansion = &RecommendedExpansion{
			ConstraintsDelta: s.SlowPath.ExpandConstraintsBy,
			FailuresDelta:    s.SlowPath.ExpandFailuresBy,
			PatternsDelta:    s.SlowPath.ExpandPatternsBy,
		}
	}
	if s.Cache != nil && s.CacheTTL > 0 {
		if b, err := json.Marshal(result); err == nil {
			_ = s.Cache.Set(ctx, key, b, s.CacheTTL)
		}
	}
	return result
}

// CompileMulti produces multiple recall bundles (one per variant strategy). No cache, no persistence.
func (s *Service) CompileMulti(ctx context.Context, req CompileMultiRequest) (*CompileMultiResponse, error) {
	if s.Compiler == nil {
		return nil, ErrNoCompiler
	}
	working := req
	if working.ChangedFilesCount != nil && !working.SlowPathRequired {
		pf := s.Preflight(ctx, PreflightRequest{
			ChangedFilesCount: *working.ChangedFilesCount,
			Tags:              working.Tags,
		})
		if pf.SlowPathRequired {
			working.SlowPathRequired = true
			working.SlowPathReasons = append([]string(nil), pf.SlowPathReasons...)
			working.RecommendedExpansion = pf.RecommendedExpansion
		}
	}

	variants := working.Variants
	if variants <= 0 {
		variants = 3
	}
	if working.SlowPathRequired && s.SlowPath != nil && s.SlowPath.ExtraVariantsWhenSlow > 0 {
		variants += s.SlowPath.ExtraVariantsWhenSlow
	}
	maxStrategies := len(DefaultStrategyList())
	if variants > maxStrategies {
		variants = maxStrategies
	}
	s.enrichCompileMultiSymbols(ctx, &working)
	strategies := ResolveStrategies(variants, working.Strategy)
	maxPerKind := working.MaxPerKind
	if maxPerKind <= 0 {
		maxPerKind = 5
	}
	baseReq := CompileRequest{
		AgentID:              working.AgentID,
		RetrievalQuery:       working.RetrievalQuery,
		Tags:                 working.Tags,
		Symbols:              working.Symbols,
		RepoRoot:             working.RepoRoot,
		LSPFocusPath:         working.LSPFocusPath,
		LSPFocusLine:         working.LSPFocusLine,
		LSPFocusColumn:       working.LSPFocusColumn,
		MaxPerKind:           maxPerKind,
		MaxTotal:             working.MaxTotal,
		MaxTokens:            working.MaxTokens,
		SlowPathRequired:     working.SlowPathRequired,
		SlowPathReasons:      append([]string(nil), working.SlowPathReasons...),
		RecommendedExpansion: working.RecommendedExpansion,
	}
	out := &CompileMultiResponse{Bundles: make([]VariantBundle, 0, len(strategies))}
	for _, strategy := range strategies {
		mod := VariantModifierForStrategy(strategy, maxPerKind)
		r := baseReq
		r.VariantModifier = mod
		bundle, err := s.Compiler.Compile(ctx, r)
		if err != nil {
			return nil, err
		}
		if err := s.hydrateEvidence(ctx, bundle); err != nil {
			return nil, err
		}
		out.Bundles = append(out.Bundles, VariantBundle{Variant: strategy, Bundle: *bundle})
	}
	return out, nil
}

// RunMulti executes the server-side run-multi contract.
// Phase B: orchestrates runmulti and optional merge server-side.
func (s *Service) RunMulti(ctx context.Context, req RunMultiRequest) (*RunMultiResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if s.RunMultiRunner == nil {
		return nil, ErrRunMultiNotConfigured
	}

	variants := req.Variants
	if variants <= 0 {
		variants = 3
	}
	strategy := req.Strategy
	if strategy == "" {
		strategy = "default"
	}
	var rec *runmulti.RecommendedExpansionMirror
	if req.RecommendedExpansion != nil {
		rec = &runmulti.RecommendedExpansionMirror{
			ConstraintsDelta: req.RecommendedExpansion.ConstraintsDelta,
			FailuresDelta:    req.RecommendedExpansion.FailuresDelta,
			PatternsDelta:    req.RecommendedExpansion.PatternsDelta,
		}
	}
	in := runmulti.RunMultiInput{
		Variants:                 variants,
		Strategy:                 strategy,
		Prompt:                   req.Query,
		Tags:                     req.Tags,
		Symbols:                  req.Symbols,
		MaxPerKind:               req.MaxPerKind,
		MaxTotal:                 req.MaxTotal,
		MaxTokens:                req.MaxTokens,
		SlowPathRequired:         req.SlowPathRequired,
		SlowPathReasons:          append([]string(nil), req.SlowPathReasons...),
		RecommendedExpansion:     rec,
		CompileChangedFilesCount: req.ChangedFilesCount,
		RepoRoot:                 req.RepoRoot,
		LSPFocusPath:             req.LSPFocusPath,
		LSPFocusLine:             req.LSPFocusLine,
		LSPFocusColumn:           req.LSPFocusColumn,
		RetrievalQuery:           strings.TrimSpace(req.RetrievalQuery),
	}
	if req.EnableTriggeredRecall {
		if rq := s.runMultiRetrievalQuery(ctx, req); rq != "" {
			in.RetrievalQuery = rq
		}
	}
	var pendingInterventions []map[string]any
	if rq, interventions := s.autoInterventionRetrievalQuery(ctx, req, in.RetrievalQuery); rq != "" {
		in.RetrievalQuery = rq
		for _, iv := range interventions {
			pendingInterventions = append(pendingInterventions, map[string]any{
				"type":   iv.Type,
				"action": iv.Action,
				"reason": iv.Reason,
			})
			slog.Info(fmt.Sprintf("[INTERVENTION] type=%s action=%s reason=%s", iv.Type, iv.Action, iv.Reason))
		}
	}
	res, err := s.RunMultiRunner.Run(ctx, in)
	if err != nil {
		return nil, err
	}

	out := &RunMultiResponse{
		Scores:         make(map[string]float64, len(res.Runs)),
		Promoted:       false,
		MemoriesUsed:   []string{},
		Excluded:       []string{},
		Contradictions: []string{},
		Debug:          newRunMultiDebug(),
	}
	var validation BehaviorValidation
	var validationSet bool
	orch := out.Debug.Orchestration
	orch["variants_requested"] = variants
	orch["strategy"] = strategy
	orch["retrieval_query_set"] = strings.TrimSpace(in.RetrievalQuery) != ""
	orch["slow_path_explicit"] = req.SlowPathRequired
	orch["changed_files_count_set"] = req.ChangedFilesCount != nil
	if req.ChangedFilesCount != nil {
		orch["changed_files_count"] = *req.ChangedFilesCount
	}
	orch["symbols_count"] = len(req.Symbols)
	orch["enable_triggered_recall"] = req.EnableTriggeredRecall
	orch["validation_required"] = true
	// keep stable key shape in debug payload
	if len(pendingInterventions) == 0 {
		out.Debug.FilterReasons["interventions"] = []map[string]any{}
	} else {
		out.Debug.FilterReasons["interventions"] = pendingInterventions
	}
	if in.RetrievalQuery != "" {
		orch["triggered_retrieval_query_set"] = true
	}
	// LSP recall context (booleans only — no paths in debug).
	orch["lsp_recall_repo_root_set"] = req.RepoRoot != ""
	orch["lsp_recall_focus_path_set"] = req.LSPFocusPath != ""
	orch["lsp_recall_focus_position_set"] = req.LSPFocusLine != 0 || req.LSPFocusColumn != 0
	// merge drift uses same slow-path + follow-up policy when client signals high-attention path.
	mergeDriftSlowPath := req.SlowPathRequired || req.ChangedFilesCount != nil
	if req.Merge {
		orch["merge_drift_slow_path"] = mergeDriftSlowPath
	}
	for _, r := range res.Runs {
		out.Scores[r.Variant] = r.Score
	}
	if res.Selected != nil {
		out.Selected = map[string]any{
			"variant":  res.Selected.Variant,
			"score":    res.Selected.Score,
			"rejected": res.Selected.Rejected,
			"output":   res.Selected.Output,
			"drift":    res.Selected.Drift,
		}
		out.Confidence = confidenceFromRuns(res.Selected, res.Runs)
	}

	var behaviorBundle *RecallBundle
	// Phase D: expose memory context observability from authoritative recall path.
	if s.Compiler != nil {
		if bundle, err := s.Compiler.Compile(ctx, CompileRequest{
			Tags:           req.Tags,
			RetrievalQuery: firstNonEmptyStr(strings.TrimSpace(in.RetrievalQuery), strings.TrimSpace(req.Query)),
			MaxPerKind:     5,
		}); err == nil && bundle != nil {
			behaviorBundle = bundle

			used := make(map[string]struct{})
			collect := func(items []MemoryItem) {
				for _, it := range items {
					if it.ID != "" {
						used[it.ID] = struct{}{}
					}
				}
			}
			collect(bundle.GoverningConstraints)
			collect(bundle.Decisions)
			collect(bundle.KnownFailures)
			collect(bundle.ApplicablePatterns)
			out.MemoriesUsed = out.MemoriesUsed[:0]
			for id := range used {
				out.MemoriesUsed = append(out.MemoriesUsed, id)
			}
			sort.Strings(out.MemoriesUsed)

			if s.Compiler.Contradiction != nil {
				if ids, err := s.Compiler.Contradiction.ListMemoryIDsInUnresolved(ctx); err == nil {
					out.Contradictions = out.Contradictions[:0]
					usedSet := used
					for _, id := range ids {
						idStr := id.String()
						out.Contradictions = append(out.Contradictions, idStr)
						if _, ok := usedSet[idStr]; !ok {
							out.Excluded = append(out.Excluded, idStr)
						}
					}
					sort.Strings(out.Contradictions)
					sort.Strings(out.Excluded)
				}
			}
		}
	}
	if behaviorBundle == nil {
		// No recall context means we cannot validate safely; enforce revise gate.
		out.Blocked = true
		orch["execution_blocked"] = true
		orch["validation_skipped"] = true
		intervention := map[string]any{
			"type":   "missing_recall",
			"action": "recall_injected",
			"reason": "validation requires recall bundle; compiler returned no bundle",
		}
		out.Debug.FilterReasons["interventions"] = appendIntervention(out.Debug.FilterReasons["interventions"], intervention)
		slog.Info("[INTERVENTION] type=missing_recall action=recall_injected reason=validation requires recall bundle; compiler returned no bundle")
	}

	intent := signal.IntentText{Prompt: req.Query, Tags: req.Tags}
	cfg := signal.DefaultSignalConfig()
	if s.RunMultiSignalConfig != nil {
		cfg = signal.NormalizeSignalConfig(*s.RunMultiSignalConfig)
	}

	var mergeRes merge.MergeResult
	totalSignalForPromo := 0.0
	if req.Merge {
		var check merge.DriftChecker
		if s.RunMultiBaseURL != "" {
			client := &http.Client{Timeout: 2 * time.Minute}
			check = func(ctx context.Context, proposal string) (runmulti.DriftResult, error) {
				return runmulti.PostDriftCheckSlowPathOptional(ctx, s.RunMultiBaseURL, proposal, client, req.Tags, mergeDriftSlowPath)
			}
		}
		mergeRes = merge.Run(ctx, merge.EngineInput{
			Runs:       res.Runs,
			Selected:   res.Selected,
			DriftCheck: check,
			Options:    mergeOptionsFromRunMulti(req),
		})
		out.Merged = map[string]any{
			"merged_output": mergeRes.MergedOutput,
			"drift":         mergeRes.Drift,
			"used_variants": mergeRes.UsedVariants,
			"agreements":    mergeRes.Agreements,
			"unique":        mergeRes.Unique,
			"conflicts":     mergeRes.Conflicts,
			"fallback_used": mergeRes.FallbackUsed,
		}
		out.Debug.Merge = &mergeRes.Debug

		filtered := signal.FilterUniques(mergeRes.Unique, mergeRes.Agreements, intent, cfg)
		scores := signal.SegmentScores(mergeRes, filtered)
		totalSig := signal.TotalSignal(scores)
		totalSignalForPromo = totalSig
		isHigh := signal.IsHighSignal(mergeRes, intent, cfg)
		out.Debug.SignalBreakdown["is_high_signal"] = isHigh
		out.Debug.SignalBreakdown["min_agreements_met"] = len(mergeRes.Agreements) >= cfg.MinAgreements
		out.Debug.SignalBreakdown["agreements_count"] = len(mergeRes.Agreements)
		out.Debug.SignalBreakdown["total_signal"] = totalSig
		out.Debug.SignalBreakdown["min_total_score"] = cfg.MinTotalScore
		out.Debug.SignalBreakdown["fallback_used"] = mergeRes.FallbackUsed
		out.Debug.SignalBreakdown["merge_drift_violations"] = len(mergeRes.Drift.Violations)
		out.Debug.SignalBreakdown["used_variants_count"] = len(mergeRes.UsedVariants)
	}

	// Behavior validation after merge so the candidate matches merged text when merge produced output.
	if behaviorBundle != nil {
		candidateText := ""
		if req.Merge {
			if m := strings.TrimSpace(mergeRes.MergedOutput); m != "" {
				candidateText = m
			} else if res.Selected != nil {
				candidateText = strings.TrimSpace(res.Selected.Output)
			}
		} else if res.Selected != nil {
			candidateText = strings.TrimSpace(res.Selected.Output)
		}
		validation = validateBehavior(behaviorBundle, candidateText, s.BehaviorValidation)
		validationSet = true
		out.Debug.FilterReasons["behavior_validation"] = validation
		if !validation.OK() {
			out.Blocked = true
			orch["execution_blocked"] = true
		}
		bv := NormalizeBehaviorValidationConfig(s.BehaviorValidation)
		if bv.BlockOutputOnValidationFail && !validation.OK() {
			if out.Selected != nil {
				out.Selected["output"] = ""
			}
			if out.Merged != nil {
				out.Merged["merged_output"] = ""
			}
		}
		s.reinforceValidationOutcome(ctx, req, behaviorBundle, validation, candidateText)
	}

	out.Debug.FilterReasons["unresolved_contradiction_ids_excluded_count"] = len(out.Excluded)
	out.Debug.FilterReasons["compile_cache_hit"] = false
	out.Debug.FilterReasons["experiences_prepended"] = 0

	// Promotion: explicit gates (signal + merge validity + drift).
	pol := s.promotionPolicy()
	if req.Promote {
		pd := out.Debug.PromotionDecision
		pd["attempted"] = req.Merge
		if !req.Merge {
			pd["promoted"] = false
			pd["reason"] = "promotion requires merge=true"
			pd["rejection_code"] = RejectionCodeMergeRequired
			pd["gates"] = map[string]any{}
		} else {
			mergedOutput := strings.TrimSpace(mergeRes.MergedOutput)
			mergeHasOutput := mergedOutput != ""
			driftClean := len(mergeRes.Drift.Violations) == 0
			highSig := signal.IsHighSignal(mergeRes, intent, cfg)
			gates := map[string]any{
				"merge_has_output":    mergeHasOutput,
				"merge_fallback_used": mergeRes.FallbackUsed,
				"merge_drift_clean":   driftClean,
				"high_signal":         highSig,
			}
			if pol.MinPromoteConfidence > 0 {
				gates["min_promote_confidence"] = pol.MinPromoteConfidence
				gates["run_confidence"] = out.Confidence
			}
			evUnique := uniqueEvidenceIDCount(req.EvidenceIDs)
			gates["unique_evidence_ids"] = evUnique
			gates["require_evidence"] = pol.RequireEvidence
			gates["require_review"] = pol.RequireReview
			if pol.MinEvidenceLinks > 0 {
				gates["min_evidence_links"] = pol.MinEvidenceLinks
			}
			if pol.MinEvidenceScore > 0 {
				gates["min_evidence_score"] = pol.MinEvidenceScore
			}
			if pol.MinPolicyComposite > 0 {
				gates["min_policy_composite"] = pol.MinPolicyComposite
			}
			pd["gates"] = gates
			policyInputs := map[string]any{
				"run_confidence":     out.Confidence,
				"total_signal":       totalSignalForPromo,
				"evidence_avg_score": nil,
			}
			pd["policy_inputs"] = policyInputs

			var failReason string
			var code string
			switch {
			case !mergeHasOutput:
				failReason = "merged output empty"
				code = RejectionCodeMergeEmpty
			case mergeRes.FallbackUsed:
				failReason = "merge used fallback"
				code = RejectionCodeMergeFallback
			case !driftClean:
				failReason = "merge drift violations"
				code = RejectionCodeMergeDrift
			case !highSig:
				failReason = "signal below promotion threshold"
				code = RejectionCodeSignalLow
			case validationSet && !validation.OK():
				failReason = "behavior validation failed (constraint/failure/decision conflict)"
				code = RejectionCodeBehaviorValidation
			}

			if failReason != "" {
				pd["promoted"] = false
				pd["reason"] = failReason
				pd["rejection_code"] = code
			} else if pol.MinPromoteConfidence > 0 && out.Confidence < pol.MinPromoteConfidence {
				pd["promoted"] = false
				pd["reason"] = "confidence below min_promote_confidence"
				pd["rejection_code"] = RejectionCodeConfidenceBelowMinimum
			} else {
				var evFail string
				var evCode string
				var evidenceAvg *float64

				needEvidenceScore := evUnique > 0 && (pol.MinEvidenceScore > 0 || PolicyCompositeUsesEvidenceTerm(pol))
				if pol.RequireEvidence && evUnique == 0 {
					evFail = "promotion policy requires evidence_ids"
					evCode = RejectionCodeEvidenceRequired
				} else if pol.MinEvidenceLinks > 0 && evUnique < pol.MinEvidenceLinks {
					evFail = "not enough unique evidence_ids for min_evidence_links"
					evCode = RejectionCodeEvidenceRequired
				} else if pol.MinEvidenceScore > 0 && evUnique == 0 {
					evFail = "min_evidence_score requires at least one evidence_id"
					evCode = RejectionCodeEvidenceRequired
				} else if needEvidenceScore && s.Evidence == nil {
					evFail = "evidence scoring requires evidence service"
					evCode = RejectionCodeEvidenceUnavailable
				} else if needEvidenceScore {
					score, err := s.Evidence.ScoreEvidenceIDs(ctx, req.EvidenceIDs)
					if err != nil {
						evFail = "evidence validation failed: " + err.Error()
						evCode = RejectionCodeEvidenceInvalid
					} else {
						evidenceAvg = &score
						gates["evidence_avg_score"] = score
						pd["gates"] = gates
						policyInputs["evidence_avg_score"] = score
						if pol.MinEvidenceScore > 0 && score < pol.MinEvidenceScore {
							evFail = "evidence score below min_evidence_score"
							evCode = RejectionCodeEvidenceScoreLow
						}
					}
				}
				if evFail != "" {
					pd["promoted"] = false
					pd["reason"] = evFail
					pd["rejection_code"] = evCode
				} else if pol.MinPolicyComposite > 0 {
					comp := PolicyComposite(out.Confidence, totalSignalForPromo, evidenceAvg, pol)
					gates["policy_composite"] = comp
					pd["gates"] = gates
					if comp < pol.MinPolicyComposite {
						pd["promoted"] = false
						pd["reason"] = "policy composite below min_policy_composite"
						pd["rejection_code"] = RejectionCodePolicyCompositeLow
					} else if s.MemoryPromoter == nil {
						pd["promoted"] = false
						pd["reason"] = "memory promoter not configured"
						pd["rejection_code"] = RejectionCodePromoterUnconfigured
					} else {
						promoted, err := s.MemoryPromoter.Promote(ctx, runMultiPromoteRequest(mergedOutput, req, pol, out.Confidence))
						if err != nil {
							pd["promoted"] = false
							pd["reason"] = "promote failed: " + err.Error()
							pd["rejection_code"] = RejectionCodePromoteFailed
						} else if promoted == nil || !promoted.Promoted {
							pd["promoted"] = false
							pd["reason"] = "promote declined by memory service"
							pd["rejection_code"] = RejectionCodePromoteDeclined
						} else {
							out.Promoted = true
							s.reinforceRecallUsage(ctx, parseUUIDs(out.MemoriesUsed), memory.ReinforceMeta{
								Reason:         "success_runmulti_promote",
								ContextKey:     "",
								AgentKey:       memory.AgentUsageKey(req.AgentID),
								Impact:         "high",
								SignalStrength: 2,
							})
							pd["promoted"] = true
							pd["reason"] = "all promotion gates passed"
							pd["rejection_code"] = RejectionCodeOK
							if promoted.ID != "" {
								pd["memory_id"] = promoted.ID
							}
							if promoted.Status != "" {
								pd["memory_status"] = promoted.Status
							}
						}
					}
				} else if s.MemoryPromoter == nil {
					pd["promoted"] = false
					pd["reason"] = "memory promoter not configured"
					pd["rejection_code"] = RejectionCodePromoterUnconfigured
				} else {
					promoted, err := s.MemoryPromoter.Promote(ctx, runMultiPromoteRequest(mergedOutput, req, pol, out.Confidence))
					if err != nil {
						pd["promoted"] = false
						pd["reason"] = "promote failed: " + err.Error()
						pd["rejection_code"] = RejectionCodePromoteFailed
					} else if promoted == nil || !promoted.Promoted {
						pd["promoted"] = false
						pd["reason"] = "promote declined by memory service"
						pd["rejection_code"] = RejectionCodePromoteDeclined
					} else {
						out.Promoted = true
						s.reinforceRecallUsage(ctx, parseUUIDs(out.MemoriesUsed), memory.ReinforceMeta{
							Reason:         "success_runmulti_promote",
							ContextKey:     "",
							AgentKey:       memory.AgentUsageKey(req.AgentID),
							Impact:         "high",
							SignalStrength: 2,
						})
						pd["promoted"] = true
						pd["reason"] = "all promotion gates passed"
						pd["rejection_code"] = RejectionCodeOK
						if promoted.ID != "" {
							pd["memory_id"] = promoted.ID
						}
						if promoted.Status != "" {
							pd["memory_status"] = promoted.Status
						}
					}
				}
			}
		}
	} else {
		out.Debug.PromotionDecision["attempted"] = false
		out.Debug.PromotionDecision["promoted"] = false
		out.Debug.PromotionDecision["reason"] = ""
		out.Debug.PromotionDecision["rejection_code"] = RejectionCodePromoteNotAttempted
		out.Debug.PromotionDecision["gates"] = map[string]any{}
	}

	return out, nil
}

type interventionEvent struct {
	Type   string
	Action string
	Reason string
}

func (s *Service) autoInterventionRetrievalQuery(ctx context.Context, req RunMultiRequest, current string) (string, []interventionEvent) {
	if strings.TrimSpace(current) != "" {
		return "", nil
	}
	cfg := NormalizeTriggerRecall(s.TriggerRecall)
	if cfg == nil || !cfg.Enabled {
		return "", nil
	}
	tin := TriggerInput{
		ProposalText: req.Query,
		Tags:         req.Tags,
	}
	raw := DetectTriggers(tin, cfg.MinContextTokens)
	triggers := filterTriggersByConfig(raw, cfg)
	if len(triggers) == 0 {
		return "", nil
	}
	needsIntervention := false
	for _, t := range triggers {
		if t.Kind == TriggerKindRisk || t.Kind == TriggerKindDecision {
			needsIntervention = true
			break
		}
	}
	if !needsIntervention {
		return "", nil
	}
	triggers, _ = capTriggers(triggers, cfg.MaxTriggersPerRequest)
	effective := mergeRetrievalQuery(strings.TrimSpace(req.Query), triggers)
	if strings.TrimSpace(effective) == "" {
		return "", nil
	}
	return effective, []interventionEvent{{
		Type:   "missing_recall",
		Action: "recall_injected",
		Reason: "risk or decision boundary detected without explicit triggered recall",
	}, {
		Type:   "risk_detected",
		Action: "recall_injected",
		Reason: "proposal matched heuristic risk/decision trigger set",
	}}
}

func appendIntervention(existing any, event map[string]any) []map[string]any {
	switch v := existing.(type) {
	case []map[string]any:
		return append(v, event)
	default:
		return []map[string]any{event}
	}
}

func firstNonEmptyStr(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func (s *Service) reinforceRecallUsage(ctx context.Context, ids []uuid.UUID, meta memory.ReinforceMeta) {
	if s == nil || s.UsageReinforcer == nil || len(ids) == 0 {
		return
	}
	type metaReinforcer interface {
		ReinforceRecallUsageWithMeta(context.Context, []uuid.UUID, memory.ReinforceMeta) error
	}
	if m, ok := s.UsageReinforcer.(metaReinforcer); ok {
		_ = m.ReinforceRecallUsageWithMeta(ctx, ids, meta)
		return
	}
	_ = s.UsageReinforcer.ReinforceRecallUsage(ctx, ids)
}

func bundleMemoryIDs(b *RecallBundle) []uuid.UUID {
	if b == nil {
		return nil
	}
	seen := map[uuid.UUID]struct{}{}
	add := func(items []MemoryItem) {
		for _, it := range items {
			id, err := uuid.Parse(strings.TrimSpace(it.ID))
			if err != nil || id == uuid.Nil {
				continue
			}
			seen[id] = struct{}{}
		}
	}
	add(b.GoverningConstraints)
	add(b.Decisions)
	add(b.KnownFailures)
	add(b.ApplicablePatterns)
	add(b.Continuity)
	add(b.Experience)
	out := make([]uuid.UUID, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	return out
}

func parseUUIDs(ids []string) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(ids))
	seen := map[uuid.UUID]struct{}{}
	for _, raw := range ids {
		id, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil || id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func runMultiPromoteRequest(mergedOutput string, req RunMultiRequest, pol PromotionPolicy, confidence float64) memory.PromoteRequest {
	return memory.PromoteRequest{
		Type:          "decision",
		Content:       mergedOutput,
		Tags:          req.Tags,
		Source:        "recall.run-multi",
		Confidence:    confidence,
		EvidenceIDs:   req.EvidenceIDs,
		RequireReview: pol.RequireReview,
	}
}

func mergeOptionsFromRunMulti(req RunMultiRequest) *merge.MergeOptions {
	if req.MergeMaxUniqueBullets <= 0 && !req.MergeDedupeSimilarUniques && req.MergeDropUniqueSimilarToAgreement <= 0 && !req.MergeStrictConflicts {
		return nil
	}
	return &merge.MergeOptions{
		StrictConflicts:                req.MergeStrictConflicts,
		MaxUniqueBullets:               req.MergeMaxUniqueBullets,
		DedupeSimilarUniques:           req.MergeDedupeSimilarUniques,
		DropUniqueIfSimilarToAgreement: req.MergeDropUniqueSimilarToAgreement,
	}
}

func (s *Service) reinforceValidationOutcome(ctx context.Context, req RunMultiRequest, bundle *RecallBundle, validation BehaviorValidation, candidateText string) {
	if bundle == nil {
		return
	}
	contextKey := ""
	agentKey := memory.AgentUsageKey(req.AgentID)
	if len(validation.ConstraintViolations) > 0 {
		ids := matchMemoryIDsByStatements(bundle.GoverningConstraints, validation.ConstraintViolations)
		s.reinforceRecallUsage(ctx, ids, memory.ReinforceMeta{
			Reason:         "constraint_block",
			ContextKey:     contextKey,
			AgentKey:       agentKey,
			Impact:         "high",
			SignalStrength: len(validation.ConstraintViolations),
		})
	}
	if len(validation.DecisionConflicts) > 0 {
		ids := matchMemoryIDsByStatements(bundle.Decisions, validation.DecisionConflicts)
		s.reinforceRecallUsage(ctx, ids, memory.ReinforceMeta{
			Reason:         "decision_conflict_block",
			ContextKey:     contextKey,
			AgentKey:       agentKey,
			Impact:         "high",
			SignalStrength: len(validation.DecisionConflicts),
		})
	}
	if len(validation.RepeatedFailures) > 0 {
		ids := matchMemoryIDsByStatements(bundle.KnownFailures, validation.RepeatedFailures)
		s.reinforceRecallUsage(ctx, ids, memory.ReinforceMeta{
			Reason:         "failure_prevented",
			ContextKey:     contextKey,
			AgentKey:       agentKey,
			Impact:         "high",
			SignalStrength: len(validation.RepeatedFailures),
		})
	}
	// On successful validation, reinforce memories that directly influenced candidate output.
	if validation.OK() {
		patternIDs := matchMemoryIDsByOutput(bundle.ApplicablePatterns, candidateText)
		s.reinforceRecallUsage(ctx, patternIDs, memory.ReinforceMeta{
			Reason:         "pattern_success",
			ContextKey:     contextKey,
			AgentKey:       agentKey,
			Impact:         "high",
			SignalStrength: len(patternIDs),
		})
		decisionIDs := matchMemoryIDsByOutput(bundle.Decisions, candidateText)
		s.reinforceRecallUsage(ctx, decisionIDs, memory.ReinforceMeta{
			Reason:         "decision_alignment",
			ContextKey:     contextKey,
			AgentKey:       agentKey,
			Impact:         "medium",
			SignalStrength: len(decisionIDs),
		})
	}
}

func matchMemoryIDsByStatements(items []MemoryItem, statements []string) []uuid.UUID {
	if len(items) == 0 || len(statements) == 0 {
		return nil
	}
	want := map[string]struct{}{}
	for _, s := range statements {
		n := strings.ToLower(strings.TrimSpace(s))
		if n != "" {
			want[n] = struct{}{}
		}
	}
	if len(want) == 0 {
		return nil
	}
	out := []uuid.UUID{}
	seen := map[uuid.UUID]struct{}{}
	for _, it := range items {
		stmt := strings.ToLower(strings.TrimSpace(it.Statement))
		if _, ok := want[stmt]; !ok {
			continue
		}
		id, err := uuid.Parse(strings.TrimSpace(it.ID))
		if err != nil || id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func matchMemoryIDsByOutput(items []MemoryItem, output string) []uuid.UUID {
	content := strings.ToLower(strings.TrimSpace(output))
	if len(items) == 0 || content == "" {
		return nil
	}
	out := []uuid.UUID{}
	seen := map[uuid.UUID]struct{}{}
	for _, it := range items {
		stmt := strings.ToLower(strings.TrimSpace(it.Statement))
		if stmt == "" || !strings.Contains(content, stmt) {
			continue
		}
		id, err := uuid.Parse(strings.TrimSpace(it.ID))
		if err != nil || id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func (s *Service) runMultiRetrievalQuery(ctx context.Context, req RunMultiRequest) string {
	cfg := NormalizeTriggerRecall(s.TriggerRecall)
	if cfg == nil || !cfg.Enabled || !req.EnableTriggeredRecall {
		return ""
	}
	tin := TriggerInput{
		ProposalText: req.Query,
		Tags:         req.Tags,
	}
	raw := DetectTriggers(tin, cfg.MinContextTokens)
	triggers := filterTriggersByConfig(raw, cfg)
	triggers, _ = capTriggers(triggers, cfg.MaxTriggersPerRequest)
	if len(triggers) == 0 {
		return ""
	}
	return mergeRetrievalQuery(strings.TrimSpace(req.RetrievalQuery), triggers)
}
