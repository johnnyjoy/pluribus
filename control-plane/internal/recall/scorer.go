package recall

import (
	"encoding/json"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// DefaultSemanticSimilarityWeight is the default ranking weight for pgvector cosine similarity (hybrid recall).
const DefaultSemanticSimilarityWeight = 0.4

// ResolveSemanticSimilarityWeight maps YAML config to a concrete weight. Nil pointer => default; explicit 0 logs and disables.
func ResolveSemanticSimilarityWeight(explicit *float64) float64 {
	if explicit == nil {
		return DefaultSemanticSimilarityWeight
	}
	if *explicit == 0 {
		slog.Warn("[WARNING] semantic_similarity disabled via weight=0")
		return 0
	}
	return *explicit
}

// patternGeneralizationTier maps generalization metadata to [0,1] for pattern_generalization weighting.
func patternGeneralizationTier(g *memory.PatternGeneralizationMeta) float64 {
	if g == nil {
		return 0
	}
	if g.Reason == memory.PatternElevationReason {
		n := len(g.SupportingStatementKeys)
		if n > 8 {
			n = 8
		}
		t := 0.55 + 0.06*float64(n)
		if t > 1 {
			return 1
		}
		return t
	}
	n := len(g.SupportingStatementKeys)
	j := g.Jaccard
	if j > 1 {
		j = 1
	}
	if j < 0 {
		j = 0
	}
	t := 0.2 + 0.12*float64(min(n, 8)) + 0.35*j
	if t > 1 {
		return 1
	}
	return t
}

func patternGeneralizationScore(p *memory.PatternPayload, weights RankingWeights) float64 {
	if p == nil || p.Generalization == nil || weights.PatternGeneralization <= 0 {
		return 0
	}
	return weights.PatternGeneralization * patternGeneralizationTier(p.Generalization)
}

// PatternScoreFactor returns a multiplier for base score from pattern payload polarity/severity.
func PatternScoreFactor(baseScore float64, payload *memory.PatternPayload) float64 {
	if payload == nil {
		return baseScore
	}
	sev := strings.ToLower(strings.TrimSpace(payload.Impact.Severity))
	lessonType := strings.ToLower(strings.TrimSpace(payload.Polarity))
	var mult float64
	switch lessonType {
	case "negative":
		switch sev {
		case "high":
			mult = 2.5
		case "medium":
			mult = 1.8
		default:
			mult = 1.3
		}
	case "positive":
		switch sev {
		case "high":
			mult = 1.4
		case "medium":
			mult = 1.2
		default:
			mult = 1.1
		}
	default:
		mult = 1.0
	}
	return baseScore * mult
}

// RankingWeights holds configurable weights for weighted recall ranking.
type RankingWeights struct {
	Authority       float64 // authority score (0..1 normalized from raw authority)
	Recency         float64
	TagMatch        float64
	FailureOverlap  float64 // extra for failure-kind when request tags overlap
	SymbolOverlap   float64 // boost when task symbols overlap with memory payload symbols
	PatternPriority float64 // additive priority for strong patterns
	LexicalSimilarity float64 // additive term for statement/query token overlap (0..1)
	// PatternGeneralization scales a boost when pattern payload includes generalization metadata (merge/reinforce).
	PatternGeneralization float64
	// FailureSeverity adds keyword-based severity for failure kind only (scorer-only; default 0 = off).
	FailureSeverity float64
	// CrossContextSalience boosts memories with payload.salience.distinct_contexts (default 0 = off).
	CrossContextSalience float64
	// CrossContextSalienceK is the divisor k in min(1, log1p(distinct)/k). Zero means use 3.
	CrossContextSalienceK float64
	// CrossAgentSalience boosts memories with payload.salience.distinct_agents (default 0 = off).
	CrossAgentSalience float64
	// CrossAgentSalienceK is the divisor k for distinct_agents (zero means use 3).
	CrossAgentSalienceK float64
	// SemanticSimilarity scales per-candidate cosine similarity from pgvector (semantic candidates only).
	SemanticSimilarity float64
	// ElevationSuppression subtracts from score when a superseded pattern appears alongside its elevated replacement (default 0 = off).
	ElevationSuppression float64
}

// DefaultRankingWeights returns weights with all factors enabled (1.0 or 0.5).
func DefaultRankingWeights() RankingWeights {
	return RankingWeights{
		Authority:      1.0,
		Recency:        0.5,
		TagMatch:       1.0,
		FailureOverlap: 0.5,
		SymbolOverlap:  0.5,
		PatternPriority: 0.0,
		// Lexical overlap: kept below authority weight so situation matching cannot swamp binding strength on ties.
		LexicalSimilarity:     0.15,
		PatternGeneralization: 0.0,
		FailureSeverity:       0,
		CrossContextSalience:  0.12,
		CrossContextSalienceK: 0,
		CrossAgentSalience:    0.12,
		CrossAgentSalienceK:   0,
		SemanticSimilarity:    DefaultSemanticSimilarityWeight,
		ElevationSuppression:  0,
	}
}

func crossContextScoreTerm(distinct int, weight, k float64) float64 {
	if weight <= 0 || distinct <= 0 {
		return 0
	}
	if k <= 0 {
		k = 3
	}
	t := math.Log1p(float64(distinct)) / k
	if t > 1 {
		t = 1
	}
	return weight * t
}

// RankingWeightsFromConfig builds RankingWeights from config. Zero or omitted values use defaults so ranking is always on when config is present.
// semanticSimilarity 0 means "use DefaultSemanticSimilarityWeight" for this helper; use ResolveSemanticSimilarityWeight when YAML may set explicit 0.
func RankingWeightsFromConfig(authority, recency, tagMatch, failureOverlap, symbolOverlap, patternPriority, lexicalSimilarity, patternGeneralization, failureSeverity, crossContextSalience, crossContextSalienceK, crossAgentSalience, crossAgentSalienceK, semanticSimilarity, elevationSuppression float64) RankingWeights {
	d := DefaultRankingWeights()
	w := RankingWeights{
		Authority:             authority,
		Recency:               recency,
		TagMatch:              tagMatch,
		FailureOverlap:        failureOverlap,
		SymbolOverlap:         symbolOverlap,
		PatternPriority:       patternPriority,
		LexicalSimilarity:     lexicalSimilarity,
		PatternGeneralization: patternGeneralization,
		FailureSeverity:       failureSeverity,
		CrossContextSalience:  crossContextSalience,
		CrossContextSalienceK: crossContextSalienceK,
		CrossAgentSalience:    crossAgentSalience,
		CrossAgentSalienceK:   crossAgentSalienceK,
		SemanticSimilarity:    semanticSimilarity,
		ElevationSuppression:  elevationSuppression,
	}
	if w.Authority == 0 {
		w.Authority = d.Authority
	}
	if w.Recency == 0 {
		w.Recency = d.Recency
	}
	if w.TagMatch == 0 {
		w.TagMatch = d.TagMatch
	}
	if w.FailureOverlap == 0 {
		w.FailureOverlap = d.FailureOverlap
	}
	if w.SymbolOverlap == 0 {
		w.SymbolOverlap = d.SymbolOverlap
	}
	if w.LexicalSimilarity == 0 {
		w.LexicalSimilarity = d.LexicalSimilarity
	}
	if w.CrossContextSalience == 0 {
		w.CrossContextSalience = d.CrossContextSalience
	}
	if w.CrossAgentSalience == 0 {
		w.CrossAgentSalience = d.CrossAgentSalience
	}
	if semanticSimilarity == 0 {
		w.SemanticSimilarity = d.SemanticSimilarity
	}
	// PatternPriority, PatternGeneralization, FailureSeverity intentionally default to 0 unless explicitly enabled.
	return w
}

// semanticScoreTerm adds hybrid semantic similarity, capped so it cannot exceed the weighted authority
// contribution for this row (Authority × authNorm). Charter: semantic must not dominate authority.
func semanticScoreTerm(weights RankingWeights, authNorm, sim float64) float64 {
	if weights.SemanticSimilarity <= 0 || sim <= 0 {
		return 0
	}
	raw := weights.SemanticSimilarity * sim
	authContrib := weights.Authority * authNorm
	if raw > authContrib {
		return authContrib
	}
	return raw
}

// ScoreRequest provides the request context for scoring (tags, symbols, situation text).
type ScoreRequest struct {
	Tags     []string
	Symbols  []string // symbol names for overlap boost with memory payload symbols
	// SessionCorrelationID when set boosts rows tagged mcp:session:<id> (session continuity; does not exclude other memories).
	SessionCorrelationID string
	// SituationQuery is the derived "what is this about?" string used for lexical overlap scoring.
	// Empty => lexical similarity term is 0.
	SituationQuery string
	// SemanticSimilarity maps memory id -> cosine similarity [0,1] for semantic candidate rows (optional).
	SemanticSimilarity map[uuid.UUID]float64
	// Supersession maps superseded pattern id -> elevated pattern id (payload.superseded_by).
	Supersession map[uuid.UUID]uuid.UUID
	// CandidateSet is the set of memory ids in the current compile batch (for elevation suppression).
	CandidateSet map[uuid.UUID]struct{}
}

// lexicalSimilarity returns a bounded [0,1] Jaccard overlap between statement and query tokens.
func lexicalSimilarity(statement, query string) float64 {
	s := strings.ToLower(strings.TrimSpace(statement))
	q := strings.ToLower(strings.TrimSpace(query))
	if s == "" || q == "" {
		return 0
	}
	stop := map[string]struct{}{
		"the": {}, "a": {}, "an": {}, "and": {}, "or": {}, "to": {}, "of": {}, "in": {}, "on": {},
		"for": {}, "with": {}, "at": {}, "by": {}, "from": {}, "into": {}, "onto": {}, "via": {},
	}
	words := func(str string) map[string]struct{} {
		out := map[string]struct{}{}
		for _, tok := range strings.FieldsFunc(str, func(r rune) bool {
			return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
		}) {
			tok = strings.TrimSpace(tok)
			if len(tok) < 4 {
				continue
			}
			if _, isStop := stop[tok]; isStop {
				continue
			}
			out[tok] = struct{}{}
		}
		return out
	}
	sw := words(s)
	qw := words(q)
	if len(sw) == 0 || len(qw) == 0 {
		return 0
	}
	inter := 0
	union := len(sw)
	for tok := range qw {
		if _, ok := sw[tok]; ok {
			inter++
		} else {
			union++
		}
	}
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// scoreBase returns the weighted score without object-lesson boost (used by Score and DominantReason).
// refTime is the "as of" instant for recency; if zero, RefTimeForRanking(single-element) would apply — callers should pass RefTimeForRanking(objs) for batches.
func scoreBase(obj memory.MemoryObject, req ScoreRequest, weights RankingWeights, maxAuthority int, refTime time.Time) float64 {
	if maxAuthority <= 0 {
		maxAuthority = 10
	}
	if refTime.IsZero() {
		refTime = RefTimeForRanking([]memory.MemoryObject{obj})
	}
	authNorm := float64(obj.Authority) / float64(maxAuthority)
	if authNorm > 1 {
		authNorm = 1
	}
	score := weights.Authority * authNorm
	const year = 365 * 24 * time.Hour
	eff := memory.EffectiveRecencyTime(obj)
	age := refTime.Sub(eff)
	if age < 0 {
		age = 0
	}
	recency := 1.0 - float64(age)/float64(year)
	if recency < 0 {
		recency = 0
	}
	score += weights.Recency * recency
	tagMatch := tagMatchScore(obj.Tags, req.Tags)
	score += weights.TagMatch * tagMatch
	if req.SessionCorrelationID != "" {
		want := "mcp:session:" + strings.TrimSpace(req.SessionCorrelationID)
		for _, t := range obj.Tags {
			if t == want {
				// Fixed boost so session-tagged memories surface without filtering the global pool off the search path.
				const sessionCorrelationBoost = 0.28
				score += sessionCorrelationBoost
				break
			}
		}
	}
	if obj.Kind == api.MemoryKindFailure && tagMatch > 0 {
		score += weights.FailureOverlap
	}
	if weights.LexicalSimilarity > 0 && req.SituationQuery != "" {
		statement := obj.StatementCanonical
		if statement == "" {
			statement = obj.Statement
		}
		if sim := lexicalSimilarity(statement, req.SituationQuery); sim > 0 {
			score += weights.LexicalSimilarity * sim
		}
	}
	if obj.Kind == api.MemoryKindPattern && weights.PatternPriority > 0 {
		score += weights.PatternPriority
	}
	// FailureSeverity and CrossContextSalience are optional (default off): keyword heuristic + payload.salience.distinct_contexts.
	if weights.FailureSeverity > 0 && obj.Kind == api.MemoryKindFailure {
		score += weights.FailureSeverity * FailureSeverityScore(obj.Statement)
	}
	if weights.CrossContextSalience > 0 {
		if n := PayloadDistinctContexts(obj.Payload); n > 0 {
			score += crossContextScoreTerm(n, weights.CrossContextSalience, weights.CrossContextSalienceK)
		}
	}
	if weights.CrossAgentSalience > 0 {
		if n := PayloadDistinctAgents(obj.Payload); n > 0 {
			score += crossContextScoreTerm(n, weights.CrossAgentSalience, weights.CrossAgentSalienceK)
		}
	}
	if weights.SemanticSimilarity > 0 && req.SemanticSimilarity != nil {
		if sim, ok := req.SemanticSimilarity[obj.ID]; ok && sim > 0 {
			score += semanticScoreTerm(weights, authNorm, sim)
		}
	}
	if weights.ElevationSuppression > 0 && req.Supersession != nil && req.CandidateSet != nil {
		if elevID, ok := req.Supersession[obj.ID]; ok {
			if _, has := req.CandidateSet[elevID]; has {
				score -= weights.ElevationSuppression
			}
		}
	}
	// Non-destructive invalidation: payload.pluribus_evolution.invalidated_by deprioritizes without hiding the row.
	if evolutionInvalidated(obj.Payload) {
		const invalidationPenalty = 0.35
		score -= invalidationPenalty
	}
	return score
}

// evolutionInvalidated is true when JSON payload has non-empty pluribus_evolution.invalidated_by.
func evolutionInvalidated(payload []byte) bool {
	if len(payload) == 0 {
		return false
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(payload, &root); err != nil {
		return false
	}
	raw, ok := root["pluribus_evolution"]
	if !ok {
		return false
	}
	var evo struct {
		InvalidatedBy string `json:"invalidated_by"`
	}
	if err := json.Unmarshal(raw, &evo); err != nil {
		return false
	}
	return strings.TrimSpace(evo.InvalidatedBy) != ""
}

// Score computes a weighted score for a memory object in the context of a compile request.
// Authority is normalized to 0..1 using maxAuthority (pass the max authority in the candidate set, or 10).
// Recency is 0..1 from effective event/update time (COALESCE(occurred_at, updated_at); newer = higher; 1 year window).
// TagMatch is 0..1: fraction of request tags that appear on the memory.
// FailureOverlap is 1 if kind==failure and tag overlap > 0, else 0 (then weighted).
// Pattern payload may apply a multiplier from polarity/severity and symbol overlap boost.
func Score(obj memory.MemoryObject, req ScoreRequest, weights RankingWeights, maxAuthority int) float64 {
	return scoreAt(obj, req, weights, maxAuthority, time.Time{})
}

// scoreAt is like Score but uses refTime for recency when non-zero (single instant for whole batch).
func scoreAt(obj memory.MemoryObject, req ScoreRequest, weights RankingWeights, maxAuthority int, refTime time.Time) float64 {
	score := scoreBase(obj, req, weights, maxAuthority, refTime)
	if obj.Kind == api.MemoryKindPattern && len(obj.Payload) > 0 {
		var p memory.PatternPayload
		if json.Unmarshal(obj.Payload, &p) == nil {
			score = PatternScoreFactor(score, &p)
			score += patternGeneralizationScore(&p, weights)
			// Symbol overlap boost: at least one task symbol in common with memory symbols
			if overlap := symbolOverlapCount(req.Symbols, p.Symbols); overlap > 0 && weights.SymbolOverlap > 0 {
				score += weights.SymbolOverlap * min(1.0, float64(overlap))
			}
		}
	}
	return score
}

// symbolOverlapCount returns how many of taskSymbols appear in memorySymbols (case-sensitive).
func symbolOverlapCount(taskSymbols, memorySymbols []string) int {
	if len(taskSymbols) == 0 || len(memorySymbols) == 0 {
		return 0
	}
	set := make(map[string]struct{}, len(memorySymbols))
	for _, s := range memorySymbols {
		set[s] = struct{}{}
	}
	var n int
	for _, s := range taskSymbols {
		if _, ok := set[s]; ok {
			n++
		}
	}
	return n
}

func tagMatchScore(memTags, reqTags []string) float64 {
	if len(reqTags) == 0 {
		return 1.0 // no filter = treat as full match so we don't penalize
	}
	set := make(map[string]struct{}, len(memTags))
	for _, t := range memTags {
		set[t] = struct{}{}
	}
	var overlap int
	for _, t := range reqTags {
		if _, ok := set[t]; ok {
			overlap++
		}
	}
	return float64(overlap) / float64(len(reqTags))
}

// ScoredMemory pairs a memory object with its ranking score and dominant reason (for justification).
// When RIU scoring is used, Score is the total (ranking + RIU) and RIU holds the breakdown.
type ScoredMemory struct {
	Object memory.MemoryObject
	Score  float64
	Reason string
	RIU    *RIUScoreBreakdown `json:"riu,omitempty"`
}

// DominantReason returns the primary factor that contributed to the score (e.g. "tag_match", "authority").
func DominantReason(obj memory.MemoryObject, req ScoreRequest, weights RankingWeights, maxAuthority int) string {
	return dominantReasonAt(obj, req, weights, maxAuthority, time.Time{})
}

func dominantReasonAt(obj memory.MemoryObject, req ScoreRequest, weights RankingWeights, maxAuthority int, refTime time.Time) string {
	if maxAuthority <= 0 {
		maxAuthority = 10
	}
	if refTime.IsZero() {
		refTime = RefTimeForRanking([]memory.MemoryObject{obj})
	}
	authNorm := float64(obj.Authority) / float64(maxAuthority)
	if authNorm > 1 {
		authNorm = 1
	}
	tagMatch := tagMatchScore(obj.Tags, req.Tags)
	const year = 365 * 24 * time.Hour
	eff := memory.EffectiveRecencyTime(obj)
	age := refTime.Sub(eff)
	if age < 0 {
		age = 0
	}
	recency := 1.0 - float64(age)/float64(year)
	if recency < 0 {
		recency = 0
	}
	contrib := map[string]float64{
		"authority":              weights.Authority * authNorm,
		"tag_match":                weights.TagMatch * tagMatch,
		"lexical_similarity":       0,
		"recency":                  weights.Recency * recency,
		"failure_overlap":          0,
		"failure_severity":         0,
		"cross_context_salience":   0,
		"cross_agent_salience":     0,
		"semantic_similarity":      0,
		"pattern_boost":            0,
		"symbol_overlap":           0,
		"pattern_priority":         0,
		"pattern_generalization":   0,
	}
	if weights.LexicalSimilarity > 0 && req.SituationQuery != "" {
		statement := obj.StatementCanonical
		if statement == "" {
			statement = obj.Statement
		}
		if sim := lexicalSimilarity(statement, req.SituationQuery); sim > 0 {
			contrib["lexical_similarity"] = weights.LexicalSimilarity * sim
		}
	}
	if obj.Kind == api.MemoryKindFailure && tagMatch > 0 {
		contrib["failure_overlap"] = weights.FailureOverlap
	}
	if weights.FailureSeverity > 0 && obj.Kind == api.MemoryKindFailure {
		contrib["failure_severity"] = weights.FailureSeverity * FailureSeverityScore(obj.Statement)
	}
	if weights.CrossContextSalience > 0 {
		if n := PayloadDistinctContexts(obj.Payload); n > 0 {
			contrib["cross_context_salience"] = crossContextScoreTerm(n, weights.CrossContextSalience, weights.CrossContextSalienceK)
		}
	}
	if weights.CrossAgentSalience > 0 {
		if n := PayloadDistinctAgents(obj.Payload); n > 0 {
			contrib["cross_agent_salience"] = crossContextScoreTerm(n, weights.CrossAgentSalience, weights.CrossAgentSalienceK)
		}
	}
	if weights.SemanticSimilarity > 0 && req.SemanticSimilarity != nil {
		if sim, ok := req.SemanticSimilarity[obj.ID]; ok && sim > 0 {
			contrib["semantic_similarity"] = semanticScoreTerm(weights, authNorm, sim)
		}
	}
	if obj.Kind == api.MemoryKindPattern && len(obj.Payload) > 0 {
		contrib["pattern_priority"] = weights.PatternPriority
		var p memory.PatternPayload
		if json.Unmarshal(obj.Payload, &p) == nil {
			base := scoreBase(obj, req, weights, maxAuthority, refTime)
			contrib["pattern_boost"] = PatternScoreFactor(base, &p) - base
			contrib["pattern_generalization"] = patternGeneralizationScore(&p, weights)
			if overlap := symbolOverlapCount(req.Symbols, p.Symbols); overlap > 0 && weights.SymbolOverlap > 0 {
				contrib["symbol_overlap"] = weights.SymbolOverlap * min(1.0, float64(overlap))
			}
		}
	}
	// Fixed order for tie-break: pattern terms, cross-context, failure severity, authority, tags, lexical, recency, failure overlap.
	order := []string{"pattern_boost", "pattern_generalization", "pattern_priority", "symbol_overlap", "cross_context_salience", "cross_agent_salience", "semantic_similarity", "failure_severity", "authority", "tag_match", "lexical_similarity", "recency", "failure_overlap"}
	best := ""
	var bestVal float64
	for _, k := range order {
		v := contrib[k]
		if v > bestVal {
			bestVal = v
			best = k
		}
	}
	if best != "" {
		return best
	}
	return "authority"
}

// ScoreAndSort returns a copy of objs sorted by score descending (highest first).
// maxAuthority is computed from objs if 0.
func ScoreAndSort(objs []memory.MemoryObject, req ScoreRequest, weights RankingWeights, maxAuthority int) []memory.MemoryObject {
	scored := ScoreAndSortWithReason(objs, req, weights, maxAuthority)
	out := make([]memory.MemoryObject, len(scored))
	for i := range scored {
		out[i] = scored[i].Object
	}
	return out
}

// ScoreAndSortWithReason returns scored memories sorted by score descending, with reason for justification.
func ScoreAndSortWithReason(objs []memory.MemoryObject, req ScoreRequest, weights RankingWeights, maxAuthority int) []ScoredMemory {
	if len(objs) == 0 {
		return nil
	}
	if maxAuthority <= 0 {
		for _, o := range objs {
			if o.Authority > maxAuthority {
				maxAuthority = o.Authority
			}
		}
		if maxAuthority <= 0 {
			maxAuthority = 10
		}
	}
	refTime := RefTimeForRanking(objs)
	out := make([]ScoredMemory, len(objs))
	for i, o := range objs {
		out[i] = ScoredMemory{
			Object: o,
			Score:  scoreAt(o, req, weights, maxAuthority, refTime),
			Reason: dominantReasonAt(o, req, weights, maxAuthority, refTime),
		}
	}
	sortScoredMemoriesStable(out)
	return out
}

// sortScoredMemoriesStable orders by: authority descending (binding strength), then total score
// descending, then memory ID ascending. RC1: higher-authority memory is never ranked below
// lower-authority memory solely because of recency or soft factors.
func sortScoredMemoriesStable(out []ScoredMemory) {
	sort.SliceStable(out, func(i, j int) bool {
		ai, aj := out[i].Object.Authority, out[j].Object.Authority
		if ai != aj {
			return ai > aj
		}
		si, sj := out[i].Score, out[j].Score
		if si != sj {
			return si > sj
		}
		return out[i].Object.ID.String() < out[j].Object.ID.String()
	})
}

// BuildCandidateSet returns the set of memory ids in objs (for elevation suppression).
func BuildCandidateSet(objs []memory.MemoryObject) map[uuid.UUID]struct{} {
	m := make(map[uuid.UUID]struct{}, len(objs))
	for _, o := range objs {
		m[o.ID] = struct{}{}
	}
	return m
}

// BuildPatternSupersessionMap maps superseded pattern id -> elevated pattern id from payload.superseded_by.
func BuildPatternSupersessionMap(objs []memory.MemoryObject) map[uuid.UUID]uuid.UUID {
	out := make(map[uuid.UUID]uuid.UUID)
	for _, o := range objs {
		if o.Kind != api.MemoryKindPattern || len(o.Payload) == 0 {
			continue
		}
		var p memory.PatternPayload
		if json.Unmarshal(o.Payload, &p) != nil {
			continue
		}
		if strings.TrimSpace(p.SupersededBy) == "" {
			continue
		}
		sid, err := uuid.Parse(strings.TrimSpace(p.SupersededBy))
		if err != nil {
			continue
		}
		out[o.ID] = sid
	}
	return out
}
