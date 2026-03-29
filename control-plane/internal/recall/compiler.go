// Package recall implements situational recall over the global durable memory pool.
//
// WARNING: This system is memory-first.
// Do not introduce project/task/workspace/hive/scope concepts as memory partitions or required recall inputs.
package recall

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"control-plane/internal/memory"
	"control-plane/internal/tooling"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// mergeUniqueMemoryObjects appends objects from extra that are not already in primary (by ID). Order: primary first.
func mergeUniqueMemoryObjects(primary, extra []memory.MemoryObject) []memory.MemoryObject {
	seen := make(map[uuid.UUID]struct{}, len(primary)+len(extra))
	out := make([]memory.MemoryObject, 0, len(primary)+len(extra))
	for _, o := range primary {
		if _, ok := seen[o.ID]; ok {
			continue
		}
		seen[o.ID] = struct{}{}
		out = append(out, o)
	}
	for _, o := range extra {
		if _, ok := seen[o.ID]; ok {
			continue
		}
		seen[o.ID] = struct{}{}
		out = append(out, o)
	}
	return out
}

// MemorySearcher is satisfied by memory.Service for search.
type MemorySearcher interface {
	Search(ctx context.Context, req memory.SearchRequest) ([]memory.MemoryObject, error)
	SearchMemories(ctx context.Context, req memory.MemoriesSearchRequest) ([]memory.MemoryObject, error)
}

// ContradictionExclusionLister returns memory IDs to exclude from recall (unresolved contradictions). Task 78.
type ContradictionExclusionLister interface {
	ListMemoryIDsInUnresolved(ctx context.Context) ([]uuid.UUID, error)
}

// ExperienceLister supplies promoted JSONL experiences as synthetic memories. Nil = disabled.
type ExperienceLister interface {
	ListForCompile(ctx context.Context, limit int) ([]memory.MemoryObject, error)
}

// Compiler builds a RecallBundle from retrieval signals and global memory search.
type Compiler struct {
	Memory MemorySearcher
	// Contradiction optionally excludes memories involved in unresolved contradictions from the bundle.
	Contradiction ContradictionExclusionLister
	// Experiences optionally prepends promoted experience-derived items before ranking (config-gated).
	Experiences ExperienceLister
	// ExperiencesLimit caps how many experience lines to load (0 = default 50).
	ExperiencesLimit int
	// Ranking optionally enables weighted ranking (authority, recency, tag, failure overlap). Nil = preserve search order.
	Ranking *RankingWeights
	// RIU (Recall Intelligence Upgrade): optional structured scoring + contradiction policy. Nil or Enabled=false = ranking-only behavior.
	RIU *RIUConfig
	// optional LSP for reference counts on symbol-matched patterns.
	LSP                     tooling.LSPClient
	LSPRecallEnabled        bool
	ReferenceExpansionLimit int // from lsp.reference_expansion_limit; 0 = no cap
	// NearDupJaccardThreshold: Jaccard on statement_canonical for recall collapse (Phase F); from memory.dedup; 0 = near-dup off.
	NearDupJaccardThreshold float64
	// Semantic: pgvector candidates merged with lexical; nil or Enabled=false = off (YAML default is on via RetrievalEnabled).
	Semantic *SemanticRecallConfig
	// LogRankTopN logs top N globally ranked memories after collapse (0 = off). Tuning/debug only (YAML: recall.log_rank_top_n).
	LogRankTopN int
}

// SemanticRecallConfig gates compile-time semantic candidate merge (defaults from recall.semantic_retrieval YAML).
type SemanticRecallConfig struct {
	Enabled             bool
	MaxCandidates       int
	MinCosineSimilarity float64
	LogSemanticMatches  bool
}

// Compile produces a bundle from tags, retrieval_query, and the shared memory pool (RIE limits, optional ranking).
// When req.SlowPathRequired and RecommendedExpansion are set, per-kind limits are expanded and bundle metadata is set (Task 94).
func (c *Compiler) Compile(ctx context.Context, req CompileRequest) (*RecallBundle, error) {
	maxPerKind := req.MaxPerKind
	if maxPerKind <= 0 {
		maxPerKind = 5
	}
	constraintLimit := maxPerKind
	decisionLimit := maxPerKind
	failureLimit := maxPerKind
	patternLimit := maxPerKind
	if req.SlowPathRequired && req.RecommendedExpansion != nil {
		constraintLimit = maxPerKind + req.RecommendedExpansion.ConstraintsDelta
		failureLimit = maxPerKind + req.RecommendedExpansion.FailuresDelta
		patternLimit = maxPerKind + req.RecommendedExpansion.PatternsDelta
		if constraintLimit < 0 {
			constraintLimit = 0
		}
		if failureLimit < 0 {
			failureLimit = 0
		}
		if patternLimit < 0 {
			patternLimit = 0
		}
	}
	// apply variant modifier limits and ranking override
	if req.VariantModifier != nil && req.VariantModifier.Limits != nil {
		lim := req.VariantModifier.Limits
		if lim.Constraints != nil {
			if v := *lim.Constraints; v >= 0 {
				constraintLimit = v
			}
		}
		if lim.Decisions != nil {
			if v := *lim.Decisions; v >= 0 {
				decisionLimit = v
			}
		}
		if lim.Failures != nil {
			if v := *lim.Failures; v >= 0 {
				failureLimit = v
			}
		}
		if lim.Patterns != nil {
			if v := *lim.Patterns; v >= 0 {
				patternLimit = v
			}
		}
	}
	b := &RecallBundle{
		GoverningConstraints: nil,
		Decisions:            nil,
		KnownFailures:        nil,
		ApplicablePatterns:   nil,
		Constraints:          nil,
	}

	// Situation string for retrieval expansion + lexical similarity: caller-provided retrieval_query (+ optional proposal heuristic).
	situationQuery := enrichSituationQueryWithProposal(strings.TrimSpace(req.RetrievalQuery), req.ProposalText)

	semanticSim := map[uuid.UUID]float64{}

	// Memory search, optional ranking with justification, bucket by kind, apply RIE limits.
	// Recall is not project-partitioned: search is unscoped (optional tag filter from the request only).
	if c.Memory != nil {
		var semRetrievalDbg *SemanticRetrievalDebug
		searchReq := memory.SearchRequest{
			Tags:   req.Tags,
			Status: "active",
			Max:    100,
		}
		domainTags := []string{}
		objs, err := c.Memory.Search(ctx, searchReq)
		if err != nil {
			return nil, err
		}
		// Expand candidates by meaning via token-level lexical search.
		// We use SearchMemories as a lightweight "token bridge" (no embeddings required).
		if strings.TrimSpace(situationQuery) != "" {
			for _, kw := range situationKeywords(situationQuery) {
				extra, err := c.Memory.SearchMemories(ctx, memory.MemoriesSearchRequest{
					Query:  kw,
					Tags:   req.Tags,
					Status: "active",
					Max:    50,
				})
				if err != nil {
					return nil, err
				}
				objs = mergeUniqueMemoryObjects(objs, extra)
			}
		}
		// Semantic candidates (pgvector): merge with lexical; similarity map feeds hybrid ranking.
		if c.Semantic != nil && c.Semantic.Enabled && strings.TrimSpace(situationQuery) != "" {
			type semanticSearcher interface {
				EmbedQueryText(ctx context.Context, text string) ([]float32, string, error)
				SearchSimilarCandidates(ctx context.Context, query []float32, req memory.SearchRequest, limit int, minCosine float64) ([]memory.MemoryObject, map[uuid.UUID]float64, error)
			}
			qTrim := strings.TrimSpace(situationQuery)
			qLog := qTrim
			if len(qLog) > 200 {
				qLog = qLog[:200] + "…"
			}
			semRetrievalDbg = &SemanticRetrievalDebug{Attempted: true, Path: "lexical_only"}
			se, ok := c.Memory.(semanticSearcher)
			if !ok {
				semRetrievalDbg.FallbackReason = memory.SemanticFallbackBackendUnsupported
				slog.Error("[SEMANTIC ERROR] memory backend does not support semantic search, falling back to lexical")
				slog.Info(fmt.Sprintf("[SEMANTIC FALLBACK] reason=%s", semRetrievalDbg.FallbackReason))
			} else {
				qv, fbReason, qerr := se.EmbedQueryText(ctx, qTrim)
				switch {
				case qerr != nil:
					semRetrievalDbg.FallbackReason = memory.SemanticFallbackEmbeddingFailed
					slog.Error("[SEMANTIC ERROR] embedding failed, falling back to lexical", "err", qerr)
					slog.Info(fmt.Sprintf("[SEMANTIC FALLBACK] reason=%s", semRetrievalDbg.FallbackReason))
				case len(qv) == 0:
					if fbReason == "" {
						fbReason = memory.SemanticFallbackEmbeddingFailed
					}
					semRetrievalDbg.FallbackReason = fbReason
					slog.Info(fmt.Sprintf("[SEMANTIC FALLBACK] reason=%s", fbReason))
				default:
					lim := c.Semantic.MaxCandidates
					if lim <= 0 {
						lim = 50
					}
					minCos := c.Semantic.MinCosineSimilarity
					if minCos <= 0 {
						minCos = 0.35
					}
					extra, sims, serr := se.SearchSimilarCandidates(ctx, qv, searchReq, lim, minCos)
					if serr != nil {
						semRetrievalDbg.FallbackReason = memory.SemanticFallbackVectorSearchFailed
						slog.Error("[SEMANTIC ERROR] vector search failed, falling back to lexical", "err", serr)
						slog.Info(fmt.Sprintf("[SEMANTIC FALLBACK] reason=%s", semRetrievalDbg.FallbackReason))
						sims = nil
						extra = nil
					} else {
						nCand := 0
						if sims != nil {
							nCand = len(sims)
						}
						slog.Info(fmt.Sprintf("[SEMANTIC MATCH] candidates=%d query=%q", nCand, qLog))
						semRetrievalDbg.Path = "semantic_hybrid"
						semRetrievalDbg.FallbackReason = ""
						if len(extra) > 0 {
							objs = mergeUniqueMemoryObjects(objs, extra)
						}
						for id, s := range sims {
							semanticSim[id] = s
						}
						if c.Semantic.LogSemanticMatches && len(sims) > 0 {
							slog.Info("[SEMANTIC MATCH] detail", "candidates", len(sims))
						}
					}
				}
			}
		}
		// Unresolved contradictions: policy from RIU (exclude = default Task 78) or warn/bounded_pair.
		// Phase E (creative C3): when ListUnresolvedPairs is available, pick a winner per pair instead of excluding both.
		var scoringUnresolved map[uuid.UUID]bool
		policy := ContradictionPolicyExclude
		if c.RIU != nil && c.RIU.Enabled {
			policy = c.RIU.Policy
			if policy == "" {
				policy = ContradictionPolicyExclude
			}
		}
		if c.Contradiction != nil {
			raw, _ := c.Contradiction.ListMemoryIDsInUnresolved(ctx)
			unresolvedFlat := make(map[uuid.UUID]bool)
			for _, id := range raw {
				unresolvedFlat[id] = true
			}
			pairLister, hasPairLister := c.Contradiction.(interface {
				ListUnresolvedPairs(context.Context, int) ([][2]uuid.UUID, error)
			})
			var pairs [][2]uuid.UUID
			var pairErr error
			if hasPairLister {
				pairs, pairErr = pairLister.ListUnresolvedPairs(ctx, 4096)
			}
			usePairs := hasPairLister && pairErr == nil && len(pairs) > 0
			var pairLosers map[uuid.UUID]bool
			var pairResolutions []ConflictResolution
			if usePairs {
				idToMem := make(map[uuid.UUID]memory.MemoryObject, len(objs))
				for _, o := range objs {
					idToMem[o.ID] = o
				}
				pairLosers = make(map[uuid.UUID]bool)
				for _, p := range pairs {
					ma, okA := idToMem[p[0]]
					mb, okB := idToMem[p[1]]
					if !okA || !okB {
						continue
					}
					winner, loser, reason := pickContradictionWinnerPair(ma, mb)
					pairLosers[loser.ID] = true
					pairResolutions = append(pairResolutions, ConflictResolution{
						WinnerMemoryID: winner.ID.String(),
						LoserMemoryID:  loser.ID.String(),
						Reason:         reason,
					})
				}
			}
			pairApplied := len(pairLosers) > 0
			if pairApplied {
				b.ConflictResolutions = pairResolutions
				if policy == ContradictionPolicyExclude {
					objs = filterMemoryObjectsRemovingLosers(objs, pairLosers)
				} else if policy == ContradictionPolicyWarn || policy == ContradictionPolicyBoundedPair {
					scoringUnresolved = pairLosers
				}
			}
			if !pairApplied && policy == ContradictionPolicyExclude && len(unresolvedFlat) > 0 {
				filtered := objs[:0]
				for _, o := range objs {
					if !unresolvedFlat[o.ID] {
						filtered = append(filtered, o)
					}
				}
				objs = filtered
			}
			if !pairApplied && (policy == ContradictionPolicyWarn || policy == ContradictionPolicyBoundedPair) && len(unresolvedFlat) > 0 {
				scoringUnresolved = unresolvedFlat
			}
		}
		if c.Experiences != nil {
			limit := c.ExperiencesLimit
			if limit <= 0 {
				limit = 50
			}
			ex, err := c.Experiences.ListForCompile(ctx, limit)
			if err == nil && len(ex) > 0 {
				objs = append(ex, objs...)
			}
		}
		candSet := BuildCandidateSet(objs)
		superMap := BuildPatternSupersessionMap(objs)
		scoreReq := ScoreRequest{
			Tags:               req.Tags,
			Symbols:            req.Symbols,
			SituationQuery:     situationQuery,
			SemanticSimilarity: semanticSim,
			Supersession:       superMap,
			CandidateSet:       candSet,
		}
		weights := c.Ranking
		if req.VariantModifier != nil && req.VariantModifier.Ranking != nil {
			weights = req.VariantModifier.Ranking
		}
		var bucket map[api.MemoryKind][]MemoryItem
		var scored []ScoredMemory
		if weights != nil {
			if c.RIU != nil && c.RIU.Enabled {
				rw := NormalizeRIUWeights(c.RIU.Weights)
				scored = ScoreAndSortWithRIU(objs, scoreReq, *weights, rw, domainTags, req.Tags, scoringUnresolved, 0)
			} else {
				scored = ScoreAndSortWithReason(objs, scoreReq, *weights, 0)
			}
			scored = collapseScoredForRecall(scored, c.NearDupJaccardThreshold)
			if c.LogRankTopN > 0 {
				logRankTrace(scored, c.LogRankTopN)
			}
			bucket = c.bucketScored(scored, maxPerKind)
		} else {
			bucket = c.bucketUnsorted(objs, maxPerKind)
		}
		trim := func(s []MemoryItem, n int) []MemoryItem {
			if len(s) <= n {
				return s
			}
			return s[:n]
		}
		b.GoverningConstraints = trim(bucket[api.MemoryKindConstraint], constraintLimit)
		b.Decisions = trim(bucket[api.MemoryKindDecision], decisionLimit)
		b.KnownFailures = trim(bucket[api.MemoryKindFailure], failureLimit)
		b.ApplicablePatterns = trim(bucket[api.MemoryKindPattern], patternLimit)
		b.GoverningConstraints, b.Decisions, b.KnownFailures, b.ApplicablePatterns = ApplyRIELimits(
			b.GoverningConstraints, b.Decisions, b.KnownFailures, b.ApplicablePatterns,
			req.MaxTotal, req.MaxTokens,
		)
		if c.RIU != nil && c.RIU.Enabled && c.RIU.Policy == ContradictionPolicyBoundedPair && c.Contradiction != nil {
			if pl, ok := c.Contradiction.(interface {
				ListUnresolvedPairs(context.Context, int) ([][2]uuid.UUID, error)
			}); ok {
				maxPairs := c.RIU.BoundedPairMax
				if maxPairs <= 0 {
					maxPairs = 16
				}
				pairs, err := pl.ListUnresolvedPairs(ctx, maxPairs*8)
				if err == nil {
					b.ContradictionSets = buildContradictionSets(collectBundleMemoryIDs(b), pairs, maxPairs)
				}
			}
		}
		if req.SlowPathRequired && req.RecommendedExpansion != nil {
			b.SlowPathApplied = true
			b.SlowPathReasons = req.SlowPathReasons
			b.BaseLimits = &BucketLimits{
				Constraints: maxPerKind,
				Decisions:   maxPerKind,
				Failures:    maxPerKind,
				Patterns:    maxPerKind,
			}
			b.ExpandedLimits = &BucketLimits{
				Constraints: constraintLimit,
				Decisions:   decisionLimit,
				Failures:    failureLimit,
				Patterns:    patternLimit,
			}
		}
		// Task 101: symbol diagnostics from pattern payload symbols in bundle.
		if weights != nil && len(req.Symbols) > 0 && len(scored) > 0 {
			memorySymbolSet := make(map[string]bool)
			for _, s := range scored {
				if s.Object.Kind != api.MemoryKindPattern {
					continue
				}
				if len(s.Object.Payload) == 0 {
					continue
				}
				var p memory.PatternPayload
				if json.Unmarshal(s.Object.Payload, &p) != nil {
					continue
				}
				for _, sym := range p.Symbols {
					memorySymbolSet[sym] = true
				}
			}
			var matched []string
			for _, t := range req.Symbols {
				if memorySymbolSet[t] {
					matched = append(matched, t)
				}
			}
			if len(matched) > 0 {
				b.MatchedSymbols = matched
				b.SymbolRelevanceReason = symbolRelevanceReason(len(matched))
				if c.LSPRecallEnabled && c.LSP != nil && req.RepoRoot != "" && req.LSPFocusPath != "" {
					b.ReferenceCount = maxReferenceCountForMatched(ctx, c.LSP, req.RepoRoot, req.LSPFocusPath, b.MatchedSymbols, c.ReferenceExpansionLimit)
				}
			}
		}
		b.SemanticRetrieval = semRetrievalDbg
		fillGroupedViews(b, scored, objs, weights != nil, maxPerKind, req.Mode)
	}
	normalizeBundleSlices(b)
	populateAgentGrounding(b)
	return b, nil
}

// normalizeBundleSlices turns nil slices into empty slices so JSON encodes [] not null
// and integration tests can rely on stable non-nil bucket shapes.
func normalizeBundleSlices(b *RecallBundle) {
	if b == nil {
		return
	}
	if b.GoverningConstraints == nil {
		b.GoverningConstraints = []MemoryItem{}
	}
	if b.Decisions == nil {
		b.Decisions = []MemoryItem{}
	}
	if b.KnownFailures == nil {
		b.KnownFailures = []MemoryItem{}
	}
	if b.ApplicablePatterns == nil {
		b.ApplicablePatterns = []MemoryItem{}
	}
	if b.Continuity == nil {
		b.Continuity = []MemoryItem{}
	}
	if b.Constraints == nil {
		b.Constraints = []MemoryItem{}
	}
	if b.Experience == nil {
		b.Experience = []MemoryItem{}
	}
}

func fillGroupedViews(b *RecallBundle, scored []ScoredMemory, raw []memory.MemoryObject, hasRanking bool, maxPerKind int, mode string) {
	if maxPerKind <= 0 {
		maxPerKind = 5
	}
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "" {
		mode = "continuity"
	}
	contLim := maxPerKind * 2
	constraintLim := maxPerKind * 2
	expLim := maxPerKind
	if mode == "thread" {
		// Thread view answers "what are we doing?" with a wider continuity slice.
		contLim = maxPerKind * 3
	}
	if !hasRanking || len(scored) == 0 {
		b.Continuity = continuityFromRaw(raw, contLim)
		b.Constraints = constraintsFromRaw(raw, constraintLim)
		b.Experience = experienceFromRaw(raw, expLim)
		return
	}
	var cont, cons, exp []MemoryItem
	for _, s := range scored {
		o := s.Object
		item := MemoryItem{
			ID:            o.ID.String(),
			Kind:          string(o.Kind),
			Statement:     o.Statement,
			Authority:     o.Authority,
			Justification: &JustificationMeta{Reason: s.Reason, Score: s.Score},
			RIU:           s.RIU,
		}
		switch o.Kind {
		case api.MemoryKindState, api.MemoryKindDecision:
			if len(cont) < contLim {
				cont = append(cont, item)
			}
		case api.MemoryKindConstraint, api.MemoryKindFailure:
			if len(cons) < constraintLim {
				cons = append(cons, item)
			}
		case api.MemoryKindPattern:
			if len(exp) < expLim {
				exp = append(exp, item)
			}
		}
	}
	b.Continuity = cont
	b.Constraints = cons
	b.Experience = exp
}

func continuityFromRaw(objs []memory.MemoryObject, limit int) []MemoryItem {
	var out []MemoryItem
	for _, o := range objs {
		if o.Kind != api.MemoryKindState && o.Kind != api.MemoryKindDecision {
			continue
		}
		if len(out) >= limit {
			break
		}
		out = append(out, MemoryItem{
			ID: o.ID.String(), Kind: string(o.Kind), Statement: o.Statement, Authority: o.Authority,
		})
	}
	return out
}

func experienceFromRaw(objs []memory.MemoryObject, limit int) []MemoryItem {
	var out []MemoryItem
	for _, o := range objs {
		if o.Kind != api.MemoryKindPattern {
			continue
		}
		if len(out) >= limit {
			return out
		}
		out = append(out, MemoryItem{
			ID: o.ID.String(), Kind: string(o.Kind), Statement: o.Statement, Authority: o.Authority,
		})
	}
	return out
}

func constraintsFromRaw(objs []memory.MemoryObject, limit int) []MemoryItem {
	var out []MemoryItem
	for _, o := range objs {
		if o.Kind != api.MemoryKindConstraint && o.Kind != api.MemoryKindFailure {
			continue
		}
		if len(out) >= limit {
			return out
		}
		out = append(out, MemoryItem{
			ID: o.ID.String(), Kind: string(o.Kind), Statement: o.Statement, Authority: o.Authority,
		})
	}
	return out
}

func collectBundleMemoryIDs(b *RecallBundle) map[string]bool {
	m := make(map[string]bool)
	for _, it := range b.GoverningConstraints {
		m[it.ID] = true
	}
	for _, it := range b.Decisions {
		m[it.ID] = true
	}
	for _, it := range b.KnownFailures {
		m[it.ID] = true
	}
	for _, it := range b.ApplicablePatterns {
		m[it.ID] = true
	}
	return m
}

func buildContradictionSets(ids map[string]bool, pairs [][2]uuid.UUID, max int) []ContradictionSet {
	if max <= 0 || len(pairs) == 0 {
		return nil
	}
	var out []ContradictionSet
	for _, p := range pairs {
		if len(out) >= max {
			break
		}
		a, b := p[0].String(), p[1].String()
		if ids[a] && ids[b] {
			out = append(out, ContradictionSet{
				PairID: a + "_" + b,
				Items:  []string{a, b},
				Reason: "unresolved_contradiction",
			})
		}
	}
	return out
}

// symbolRelevanceReason returns a short explanation for symbol overlap (Task 101).
func symbolRelevanceReason(matchedCount int) string {
	if matchedCount <= 0 {
		return ""
	}
	if matchedCount == 1 {
		return "1 task symbol matched pattern symbols"
	}
	return "multiple task symbols matched pattern symbols"
}

// situationKeywords extracts a small deterministic keyword set for token-level retrieval.
// This is a "semantic bridge" without embeddings: it reduces paraphrase sensitivity by searching per-keyword.
func situationKeywords(query string) []string {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	stop := map[string]struct{}{
		"the": {}, "a": {}, "an": {}, "and": {}, "or": {}, "to": {}, "of": {}, "in": {}, "on": {},
		"for": {}, "with": {}, "at": {}, "by": {}, "from": {}, "into": {}, "onto": {}, "via": {},
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, 6)
	for _, tok := range strings.FieldsFunc(q, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	}) {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		if len(tok) < 4 {
			continue
		}
		if _, isStop := stop[tok]; isStop {
			continue
		}
		if _, ok := seen[tok]; ok {
			continue
		}
		seen[tok] = struct{}{}
		out = append(out, tok)
		if len(out) >= 6 {
			break
		}
	}
	return out
}

func (c *Compiler) bucketScored(scored []ScoredMemory, maxPerKind int) map[api.MemoryKind][]MemoryItem {
	bucket := map[api.MemoryKind][]MemoryItem{
		api.MemoryKindConstraint: nil,
		api.MemoryKindDecision:   nil,
		api.MemoryKindFailure:    nil,
		api.MemoryKindPattern:    nil,
	}
	for _, s := range scored {
		o := s.Object
		item := MemoryItem{
			ID:            o.ID.String(),
			Kind:          string(o.Kind),
			Statement:     o.Statement,
			Authority:     o.Authority,
			Justification: &JustificationMeta{Reason: s.Reason, Score: s.Score},
			RIU:           s.RIU,
		}
		switch o.Kind {
		case api.MemoryKindConstraint:
			bucket[api.MemoryKindConstraint] = append(bucket[api.MemoryKindConstraint], item)
		case api.MemoryKindDecision:
			bucket[api.MemoryKindDecision] = append(bucket[api.MemoryKindDecision], item)
		case api.MemoryKindFailure:
			bucket[api.MemoryKindFailure] = append(bucket[api.MemoryKindFailure], item)
		case api.MemoryKindPattern:
			bucket[api.MemoryKindPattern] = append(bucket[api.MemoryKindPattern], item)
		}
	}
	return bucket
}

func (c *Compiler) bucketUnsorted(objs []memory.MemoryObject, maxPerKind int) map[api.MemoryKind][]MemoryItem {
	bucket := map[api.MemoryKind][]MemoryItem{
		api.MemoryKindConstraint: nil,
		api.MemoryKindDecision:   nil,
		api.MemoryKindFailure:    nil,
		api.MemoryKindPattern:    nil,
	}
	for _, o := range objs {
		item := MemoryItem{
			ID:        o.ID.String(),
			Kind:      string(o.Kind),
			Statement: o.Statement,
			Authority: o.Authority,
		}
		switch o.Kind {
		case api.MemoryKindConstraint:
			bucket[api.MemoryKindConstraint] = append(bucket[api.MemoryKindConstraint], item)
		case api.MemoryKindDecision:
			bucket[api.MemoryKindDecision] = append(bucket[api.MemoryKindDecision], item)
		case api.MemoryKindFailure:
			bucket[api.MemoryKindFailure] = append(bucket[api.MemoryKindFailure], item)
		case api.MemoryKindPattern:
			bucket[api.MemoryKindPattern] = append(bucket[api.MemoryKindPattern], item)
		}
	}
	return bucket
}

func logRankTrace(scored []ScoredMemory, n int) {
	if n <= 0 || len(scored) == 0 {
		return
	}
	for i := 0; i < len(scored) && i < n; i++ {
		s := scored[i]
		o := s.Object
		slog.Info("[RANK]",
			"memory_id", o.ID.String(),
			"kind", string(o.Kind),
			"authority", o.Authority,
			"score", s.Score,
			"reason", s.Reason,
			"distinct_contexts", PayloadDistinctContexts(o.Payload),
			"distinct_agents", PayloadDistinctAgents(o.Payload),
		)
	}
}
