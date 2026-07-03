package recall

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"
)

// genericRecallTerms are high-frequency tokens that must not dominate relevance alone.
var genericRecallTerms = map[string]struct{}{
	"interface": {}, "integration": {}, "system": {}, "memory": {}, "agent": {},
	"tool": {}, "tools": {}, "service": {}, "project": {}, "platform": {},
	"configuration": {}, "config": {}, "proof": {}, "test": {}, "tests": {},
	"binding": {}, "credential": {}, "recall": {}, "constraint": {}, "constraints": {},
	"decision": {}, "pattern": {}, "failure": {}, "context": {}, "session": {},
	"server": {}, "client": {}, "database": {}, "deploy": {}, "deployment": {},
	"production": {}, "staging": {}, "reverse": {}, "proxy": {}, "marketing": {},
	"workflow": {}, "compliance": {}, "telemetry": {}, "benchmark": {}, "schema": {},
	"schemas": {}, "docker": {}, "stdio": {}, "http": {}, "rest": {}, "api": {},
	"integrate": {}, "properly": {}, "help": {}, "situation": {},
}

func isGenericRecallTerm(tok string) bool {
	_, ok := genericRecallTerms[strings.ToLower(tok)]
	return ok
}

// productAnchorQueryTerms are product/deployment names that trigger the
// named-product scoring branches. Deployment-specific: override via
// recall.ranking.product_anchor_terms (M2 de-overfit: no hardcoded literals
// buried in scoring logic).
var productAnchorQueryTerms = []string{"pluribus"}

// RankingTermConfig carries deployment-tunable term lists for relevance scoring.
type RankingTermConfig struct {
	// ProductAnchorTerms replaces the default product-name list when non-empty.
	ProductAnchorTerms []string
	// ExtraGenericTerms extends genericRecallTerms (tokens that must not
	// dominate relevance alone) for corpus-specific noise.
	ExtraGenericTerms []string
	// ExtraCommonDistinctiveTokens extends commonDistinctiveTokens (tokens too
	// common to count as product anchors).
	ExtraCommonDistinctiveTokens []string
}

// ApplyRankingTermConfig installs deployment-specific term lists at wiring time
// (before any recall traffic; not safe for concurrent mutation afterwards).
func ApplyRankingTermConfig(cfg RankingTermConfig) {
	if len(cfg.ProductAnchorTerms) > 0 {
		terms := make([]string, 0, len(cfg.ProductAnchorTerms))
		for _, t := range cfg.ProductAnchorTerms {
			if t = strings.ToLower(strings.TrimSpace(t)); t != "" {
				terms = append(terms, t)
			}
		}
		if len(terms) > 0 {
			productAnchorQueryTerms = terms
		}
	}
	for _, t := range cfg.ExtraGenericTerms {
		if t = strings.ToLower(strings.TrimSpace(t)); t != "" {
			genericRecallTerms[t] = struct{}{}
		}
	}
	for _, t := range cfg.ExtraCommonDistinctiveTokens {
		if t = strings.ToLower(strings.TrimSpace(t)); t != "" {
			commonDistinctiveTokens[t] = struct{}{}
		}
	}
}

// textMentionsProductAnchorTerm reports whether text names a configured product anchor.
func textMentionsProductAnchorTerm(text string) bool {
	low := strings.ToLower(text)
	for _, t := range productAnchorQueryTerms {
		if strings.Contains(low, t) {
			return true
		}
	}
	return false
}

func isBenchmarkInfraTag(tag string) bool {
	tag = strings.ToLower(strings.TrimSpace(tag))
	return strings.HasPrefix(tag, "bench:") || strings.HasPrefix(tag, "domain:")
}

// ScoreComponentBreakdown exposes ranking components for diagnostics and benchmark reports.
type ScoreComponentBreakdown struct {
	FinalScore           float64  `json:"final_score"`
	RelevanceScore       float64  `json:"relevance_score"`
	AuthorityMultiplier  float64  `json:"authority_multiplier"`
	LexicalScore         float64  `json:"lexical_score"`
	SituationalScore     float64  `json:"situational_score"`
	TagMatchScore        float64  `json:"tag_match_score"`
	RecencyScore         float64  `json:"recency_score"`
	WrongDomainPenalty   float64  `json:"wrong_domain_penalty"`
	GenericTermPenalty   float64  `json:"generic_term_penalty"`
	VagueQueryDampening  float64  `json:"vague_query_dampening"`
	UtilityScore         float64  `json:"utility_score,omitempty"`
	MatchedTerms         []string `json:"matched_terms,omitempty"`
}

func distinctiveTokenSet(text string, tags []string) map[string]struct{} {
	tokens := domainTokensFromText(text)
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" || isBenchmarkInfraTag(tag) {
			continue
		}
		if len(tag) >= 3 {
			tokens[tag] = struct{}{}
		}
		if i := strings.LastIndexAny(tag, "/:"); i >= 0 && i+1 < len(tag) {
			seg := tag[i+1:]
			if len(seg) >= 3 && !isBenchmarkInfraTag(seg) {
				tokens[seg] = struct{}{}
			}
		}
	}
	dist := make(map[string]struct{}, len(tokens))
	for tok := range tokens {
		if !isGenericRecallTerm(tok) {
			dist[tok] = struct{}{}
		}
	}
	return dist
}

func queryDistinctiveTokens(in SituationAffinityInput) map[string]struct{} {
	qTokens := querySituationTokens(in)
	dist := make(map[string]struct{}, len(qTokens))
	for tok := range qTokens {
		if !isGenericRecallTerm(tok) {
			dist[tok] = struct{}{}
		}
	}
	return dist
}

func weightedDistinctiveOverlap(qTokens, mTokens map[string]struct{}) (overlap float64, matched []string) {
	if len(qTokens) == 0 || len(mTokens) == 0 {
		return 0, nil
	}
	var matchedW, totalW float64
	for tok := range qTokens {
		w := tokenWeight(tok)
		totalW += w
		if _, ok := mTokens[tok]; ok {
			matchedW += w
			matched = append(matched, tok)
		}
	}
	if totalW == 0 {
		return 0, matched
	}
	sort.Strings(matched)
	return matchedW / totalW, matched
}

func distinctiveLexicalSimilarity(statement, query string, memTags []string) float64 {
	qDist := distinctiveTokenSet(query, nil)
	mDist := distinctiveTokenSet(statement, memTags)
	if len(qDist) == 0 {
		return 0
	}
	overlap, _ := weightedDistinctiveOverlap(qDist, mDist)
	return overlap
}

func wrongDomainPenalty(query string, reqTags []string, repoRoot, statement string, memTags []string, authNorm, tagMatchFrac float64) float64 {
	if conversationalVagueQuery(query, reqTags) {
		if memoryHasProductAnchor(statement, memTags) && !queryNamedProduct(query) {
			p := 0.55 * (0.35 + 0.65*authNorm)
			if p > 0.95 {
				return 0.95
			}
			return p
		}
		return 0
	}
	qDist := queryDistinctiveTokens(SituationAffinityInput{
		SituationQuery: query,
		RequestTags:    reqTags,
	})
	if len(qDist) < 2 {
		return 0
	}
	mDist := distinctiveTokenSet(statement, memTags)
	if len(mDist) == 0 {
		return 0
	}
	overlap, _ := weightedDistinctiveOverlap(qDist, mDist)
	var penalty float64
	if tagMatchFrac < 0.5 {
		penalty = missingAnchorPenalty(qDist, mDist, authNorm, reqTags)
	}
	if overlap >= 0.38 && penalty == 0 {
		return repoMismatchPenalty(repoRoot, statement, memTags, authNorm)
	}
	fullLex := lexicalSimilarity(statement, query)
	domainPenalty := 0.0
	if overlap < 0.38 {
		domainPenalty = (0.55 - overlap) * (0.35 + 0.65*authNorm)
		if domainPenalty < 0 {
			domainPenalty = 0
		}
		if fullLex >= 0.08 && overlap < 0.25 {
			domainPenalty *= 1.65
		}
	}
	penalty += domainPenalty
	penalty += repoMismatchPenalty(repoRoot, statement, memTags, authNorm)
	if tagMatchFrac >= 0.66 && !matchedTagsAreAllGeneric(reqTags, memTags) {
		penalty *= 0.15
	} else if tagMatchFrac >= 0.5 && matchedTagsAreAllGeneric(reqTags, memTags) {
		penalty *= 0.55
	} else if tagMatchFrac >= 0.5 {
		penalty *= 0.45
	}
	if penalty > 0.95 {
		return 0.95
	}
	return penalty
}

func repoMismatchPenalty(repoRoot, statement string, tags []string, authNorm float64) float64 {
	if strings.TrimSpace(repoRoot) == "" {
		return 0
	}
	if repoRootAffinityBoost(repoRoot, statement, tags) > 0 {
		return 0
	}
	base := strings.ToLower(filepath.Base(filepath.Clean(strings.TrimSpace(repoRoot))))
	if base == "" || len(base) < 3 {
		return 0
	}
	return 0.55 * (0.35 + 0.65*authNorm)
}

// missingAnchorPenalty applies when high-weight anchor tokens from the query are absent from the memory.
func missingAnchorPenalty(qDist, mDist map[string]struct{}, authNorm float64, reqTags []string) float64 {
	if len(qDist) == 0 {
		return 0
	}
	var missingW, totalW float64
	for tok := range qDist {
		if isConversationalQueryTerm(tok) {
			continue
		}
		if !isAnchorToken(tok, reqTags) {
			continue
		}
		w := tokenWeight(tok)
		totalW += w
		if _, ok := mDist[tok]; !ok {
			missingW += w
		}
	}
	if totalW == 0 || missingW == 0 {
		return 0
	}
	ratio := missingW / totalW
	if ratio < 0.55 {
		return 0
	}
	p := ratio * 0.5 * (0.35 + 0.65*authNorm)
	if p > 0.65 {
		return 0.65
	}
	return p
}

func isAnchorToken(tok string, reqTags []string) bool {
	tok = strings.ToLower(tok)
	for _, tag := range reqTags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == tok {
			return true
		}
		if i := strings.LastIndexAny(tag, "/:"); i >= 0 && i+1 < len(tag) && tag[i+1:] == tok {
			return true
		}
	}
	return len(tok) >= 8
}

func genericOnlyLexicalPenalty(statement, query string, memTags []string) float64 {
	fullLex := lexicalSimilarity(statement, query)
	if fullLex < 0.08 {
		return 0
	}
	qDist := queryDistinctiveTokens(SituationAffinityInput{SituationQuery: query})
	mDist := distinctiveTokenSet(statement, memTags)
	if len(qDist) < 2 {
		return 0
	}
	distOverlap, _ := weightedDistinctiveOverlap(qDist, mDist)
	if distOverlap >= 0.25 {
		return 0
	}
	return 0.45 * fullLex * (1.0 - distOverlap)
}

func vagueQueryDampening(query string, reqTags []string) float64 {
	qDist := queryDistinctiveTokens(SituationAffinityInput{
		SituationQuery: query,
		RequestTags:    reqTags,
	})
	switch {
	case len(qDist) >= 2:
		return 1.0
	case len(qDist) == 1:
		return 0.7
	default:
		return 0.5
	}
}

func memoryTagsForScoring(tags []string) []string {
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		if !isBenchmarkInfraTag(t) {
			out = append(out, t)
		}
	}
	return out
}

func authorityMultiplier(authNorm float64) float64 {
	if authNorm > 1 {
		authNorm = 1
	}
	return 0.45 + 0.55*authNorm
}

func computeScoreComponents(
	obj memory.MemoryObject,
	req ScoreRequest,
	weights RankingWeights,
	maxAuthority int,
	refTime time.Time,
) ScoreComponentBreakdown {
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
	statement := obj.StatementCanonical
	if statement == "" {
		statement = obj.Statement
	}
	scoringTags := memoryTagsForScoring(obj.Tags)

	var bd ScoreComponentBreakdown
	bd.AuthorityMultiplier = authorityMultiplier(authNorm)

	if weights.LexicalSimilarity > 0 && req.SituationQuery != "" {
		bd.LexicalScore = weights.LexicalSimilarity * distinctiveLexicalSimilarity(statement, req.SituationQuery, scoringTags)
	}
	if weights.SituationalAffinity > 0 && req.SituationQuery != "" {
		aff := SituationAffinityScore(statement, scoringTags, SituationAffinityInput{
			SituationQuery: req.SituationQuery,
			RepoRoot:       req.RepoRoot,
			RequestTags:    req.Tags,
		})
		if aff > 0 {
			bd.SituationalScore = weights.SituationalAffinity * aff
		}
	}
	if weights.SituationalAffinity > 0 && req.SituationQuery != "" {
		if phr := compoundPhraseScore(req.SituationQuery, statement, scoringTags); phr > 0 {
			bd.SituationalScore += weights.SituationalAffinity * 0.45 * phr
		}
		qDist := queryDistinctiveTokens(SituationAffinityInput{
			SituationQuery: req.SituationQuery,
			RequestTags:    req.Tags,
		})
		mDist := distinctiveTokenSet(statement, scoringTags)
		if pref := prefixTokenOverlap(qDist, mDist); pref > 0 {
			bd.LexicalScore += weights.LexicalSimilarity * 0.35 * pref
		}
	}
	if req.RepoRoot != "" {
		if boost := repoRootAffinityBoost(req.RepoRoot, statement, scoringTags); boost > 0 {
			bd.SituationalScore += weights.SituationalAffinity * boost
		}
	}
	tagMatch := inferredTagMatchScore(scoringTags, req.SituationQuery, req.Tags)
	tagFrac := tagMatch
	if len(req.Tags) > 0 {
		tagFrac = tagMatchScore(scoringTags, req.Tags)
	}
	bd.TagMatchScore = weights.TagMatch * tagMatch

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
	bd.RecencyScore = weights.Recency * recency * 0.35

	relevance := bd.LexicalScore + bd.SituationalScore + bd.TagMatchScore
	var technicalGenericPenalty float64
	if req.SessionCorrelationID != "" {
		want := "mcp:session:" + strings.TrimSpace(req.SessionCorrelationID)
		for _, t := range obj.Tags {
			if t == want {
				relevance += 0.28
				break
			}
		}
	}
	if obj.Kind == api.MemoryKindFailure && len(req.Tags) > 0 && tagMatch > 0 {
		relevance += weights.FailureOverlap
	}
	if conversationalVagueQuery(req.SituationQuery, req.Tags) {
		if aff := interfaceIntegrationAffinity(req.SituationQuery, scoringTags); aff > 0 {
			relevance += weights.TagMatch * 0.65 * aff
			if obj.Kind == api.MemoryKindPattern && isLowAuthorityGenericPattern(obj, scoringTags) {
				relevance += 0.12 * aff
			}
		}
		if isGenericNoiseMemory(obj, scoringTags) {
			if phr := compoundPhraseScore(req.SituationQuery, statement, scoringTags); phr > 0 {
				relevance += weights.SituationalAffinity * 0.35 * phr
			}
		}
	}
	if technicalGenericIntegrationCluster(req.SituationQuery, req.Tags) {
		if obj.Kind == api.MemoryKindConstraint && obj.Applicability == api.ApplicabilityGoverning && memoryHasProductAnchor(statement, scoringTags) {
			relevance += 0.50
		}
		if memoryTagContains(scoringTags, "mcp") && obj.Kind == api.MemoryKindConstraint {
			relevance += 0.18
		}
		if memoryTagContains(scoringTags, "recall") && memoryTagContains(scoringTags, "agent") {
			relevance += 0.24
		}
		if isLowAuthorityGenericPattern(obj, scoringTags) || isGenericNoiseMemory(obj, scoringTags) {
			technicalGenericPenalty = 0.85
		}
	}
	if obj.Kind == api.MemoryKindDecision && req.SituationQuery != "" {
		if phr := compoundPhraseScore(req.SituationQuery, statement, scoringTags); phr >= 0.12 {
			bd.SituationalScore += weights.SituationalAffinity * 1.1 * phr
			bd.TagMatchScore += weights.TagMatch * 0.35 * phr
			relevance = bd.LexicalScore + bd.SituationalScore + bd.TagMatchScore
		}
		if strongCompoundPhraseHit(req.SituationQuery, statement, scoringTags) {
			relevance += 0.95
		}
	}
	if textMentionsProductAnchorTerm(req.SituationQuery) &&
		obj.Kind == api.MemoryKindConstraint &&
		memoryTagContains(scoringTags, "enforcement") &&
		tagMatchScore(scoringTags, req.Tags) > 0 {
		relevance += 0.38
	}
	if len(req.Symbols) > 0 && obj.Kind == api.MemoryKindPattern && len(obj.Payload) > 0 {
		var p memory.PatternPayload
		if json.Unmarshal(obj.Payload, &p) == nil {
			if overlap := symbolOverlapCount(req.Symbols, p.Symbols); overlap > 0 && weights.SymbolOverlap > 0 {
				relevance += weights.SymbolOverlap * min(1.0, float64(overlap))
			}
		}
	}

	if req.SituationQuery != "" {
		bd.WrongDomainPenalty = wrongDomainPenalty(req.SituationQuery, req.Tags, req.RepoRoot, statement, scoringTags, authNorm, tagFrac)
		bd.GenericTermPenalty = genericOnlyLexicalPenalty(statement, req.SituationQuery, scoringTags)
		bd.GenericTermPenalty += symbolQueryEchoPenalty(req, obj, statement)
		bd.GenericTermPenalty += genericSaturationPenalty(req.SituationQuery, req.Tags, statement, scoringTags, tagFrac)
		bd.GenericTermPenalty += technicalGenericPenalty
		bd.WrongDomainPenalty += strongAnchorMismatchPenalty(req.SituationQuery, req.Tags, req.RepoRoot, statement, scoringTags, authNorm)
		if bd.WrongDomainPenalty > 0.95 {
			bd.WrongDomainPenalty = 0.95
		}
		if phr := compoundPhraseScore(req.SituationQuery, statement, scoringTags); phr >= 0.12 {
			vagueProductLeak := conversationalVagueQuery(req.SituationQuery, req.Tags) &&
				memoryHasProductAnchor(statement, scoringTags) && !queryNamedProduct(req.SituationQuery)
			if !vagueProductLeak {
				bd.WrongDomainPenalty *= 1.0 - 0.75*min(1.0, phr)
			}
		}
		if hyp := hyphenatedTagQueryAffinity(req.SituationQuery, req.Tags, scoringTags); hyp > 0 {
			bd.WrongDomainPenalty *= 1.0 - 0.55*hyp
		}
		if strongCompoundPhraseHit(req.SituationQuery, statement, scoringTags) && obj.Kind == api.MemoryKindDecision {
			bd.WrongDomainPenalty *= 0.12
		}
		if textMentionsProductAnchorTerm(req.SituationQuery) &&
			obj.Kind == api.MemoryKindConstraint &&
			memoryTagContains(scoringTags, "enforcement") &&
			tagMatchScore(scoringTags, req.Tags) > 0 {
			bd.WrongDomainPenalty *= 0.15
		}
		qDist := queryDistinctiveTokens(SituationAffinityInput{
			SituationQuery: req.SituationQuery,
			RequestTags:    req.Tags,
		})
		mDist := distinctiveTokenSet(statement, scoringTags)
		if sharesProductAnchor(qDist, mDist) {
			bd.WrongDomainPenalty *= 0.45
		}
	}
	bd.VagueQueryDampening = vagueQueryDampening(req.SituationQuery, req.Tags)

	relevance -= bd.WrongDomainPenalty + bd.GenericTermPenalty
	if obj.Status == api.StatusSuperseded && req.RecallMode != RecallModeHistorical && req.SituationQuery != "" && !includeSupersededCandidates(req.SituationQuery) {
		relevance *= 0.12
	}
	if relevance < 0 {
		relevance = 0
	}
	relevance *= bd.VagueQueryDampening
	bd.RelevanceScore = relevance

	score := relevance * bd.AuthorityMultiplier
	if req.SituationQuery == "" {
		score += bd.RecencyScore
	} else if relevance >= 0.05 {
		score += bd.RecencyScore
	} else {
		bd.RecencyScore = 0
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
	if evolutionInvalidated(obj.Payload) {
		score -= 0.35
	}

	// Tie-breaker when relevance is near-zero among equally vague rows (same project context only).
	if relevance < 0.02 && bd.WrongDomainPenalty < 0.25 {
		score += 0.008 * weights.Authority * authNorm
	}
	if bd.WrongDomainPenalty >= 0.35 && tagFrac < 0.5 && !conversationalVagueQuery(req.SituationQuery, req.Tags) {
		score = min(score, 0.04)
	} else if relevance < 0.05 && bd.WrongDomainPenalty > 0 && tagFrac < 0.5 && !conversationalVagueQuery(req.SituationQuery, req.Tags) {
		score = min(score, 0.06)
	}
	if technicalGenericIntegrationCluster(req.SituationQuery, req.Tags) &&
		obj.Authority <= 4 &&
		(obj.Kind == api.MemoryKindPattern || obj.Kind == api.MemoryKindState) &&
		!memoryHasProductAnchor(statement, scoringTags) {
		score = min(score, 0.04)
	}
	if conversationalVagueQuery(req.SituationQuery, req.Tags) && memoryHasProductAnchor(statement, scoringTags) {
		score = min(score, 0.28)
	}
	if queryNamedProduct(req.SituationQuery) && memoryTagContains(scoringTags, "checklist") &&
		!textMentionsProductAnchorTerm(statement) {
		score = min(score, 0.05)
	}
	if req.RepoRoot != "" && repoRootAffinityBoost(req.RepoRoot, statement, scoringTags) == 0 {
		anchors := queryAnchorsIncludingRepo(req.SituationQuery, req.Tags, req.RepoRoot)
		if len(anchors) > 0 {
			mDist := distinctiveTokenSet(statement, scoringTags)
			var miss, total int
			for a := range anchors {
				total++
				if _, ok := mDist[a]; !ok {
					miss++
				}
			}
			if total > 0 && float64(miss)/float64(total) >= 0.75 {
				score = min(score, 0.10)
			}
		}
	}

	if req.UtilityScores != nil && req.UtilityWeight > 0 {
		if u, ok := req.UtilityScores[obj.ID]; ok && u != 0 {
			bd.UtilityScore = utilityRankingTerm(u, req.UtilityWeight)
			score += bd.UtilityScore
		}
	}

	score = applyHistoricalScoreCap(req.RecallMode, obj, score)
	bd.FinalScore = score
	if req.SituationQuery != "" {
		qDist := queryDistinctiveTokens(SituationAffinityInput{
			SituationQuery: req.SituationQuery,
			RequestTags:    req.Tags,
		})
		mDist := distinctiveTokenSet(statement, scoringTags)
		_, bd.MatchedTerms = weightedDistinctiveOverlap(qDist, mDist)
	}
	return bd
}

func sharesProductAnchor(qDist, mDist map[string]struct{}) bool {
	for tok := range qDist {
		if len(tok) < 8 || isGenericRecallTerm(tok) {
			continue
		}
		if _, ok := mDist[tok]; ok {
			return true
		}
	}
	return false
}

func repoRootAffinityBoost(repoRoot, statement string, tags []string) float64 {
	base := strings.ToLower(filepath.Base(filepath.Clean(strings.TrimSpace(repoRoot))))
	if base == "" || base == "." || len(base) < 3 {
		return 0
	}
	stmt := strings.ToLower(statement)
	if strings.Contains(stmt, base) {
		return 0.55
	}
	for _, t := range tags {
		if strings.Contains(strings.ToLower(t), base) {
			return 0.4
		}
	}
	return 0
}

func genericSaturationPenalty(query string, reqTags []string, statement string, memTags []string, tagMatchFrac float64) float64 {
	if !genericSaturatedQuery(query, reqTags) {
		return 0
	}
	qDist := queryDistinctiveTokens(SituationAffinityInput{
		SituationQuery: query,
		RequestTags:    reqTags,
	})
	if len(qDist) > 0 {
		return 0
	}
	mDist := distinctiveTokenSet(statement, memTags)
	if len(mDist) == 0 {
		return 0.35
	}
	overlap, _ := weightedDistinctiveOverlap(qDist, mDist)
	if overlap > 0 {
		return 0
	}
	fullLex := lexicalSimilarity(statement, query)
	if fullLex < 0.12 {
		return 0
	}
	p := 0.35 * fullLex
	if tagMatchFrac < 0.34 {
		p += 0.15
	}
	if p > 0.55 {
		return 0.55
	}
	return p
}

func strongAnchorMismatchPenalty(query string, reqTags []string, repoRoot, statement string, memTags []string, authNorm float64) float64 {
	if conversationalVagueQuery(query, reqTags) {
		return 0
	}
	anchors := queryAnchorsIncludingRepo(query, reqTags, repoRoot)
	if len(anchors) == 0 {
		return 0
	}
	mDist := distinctiveTokenSet(statement, memTags)
	var missing int
	for a := range anchors {
		if _, ok := mDist[a]; !ok {
			missing++
		}
	}
	if missing == 0 {
		return 0
	}
	ratio := float64(missing) / float64(len(anchors))
	if ratio < 0.5 {
		return 0
	}
	p := ratio * 0.45 * (0.35 + 0.65*authNorm)
	if p > 0.55 {
		return 0.55
	}
	return p
}

func inferredTagMatchScore(memTags []string, situationQuery string, reqTags []string) float64 {
	if len(reqTags) > 0 {
		return tagMatchScore(memTags, reqTags)
	}
	qTokens := querySituationTokens(SituationAffinityInput{SituationQuery: situationQuery})
	if len(qTokens) == 0 {
		return 0
	}
	var matched int
	var memTagCount int
	for _, tag := range memTags {
		if isBenchmarkInfraTag(tag) {
			continue
		}
		memTagCount++
		tagLower := strings.ToLower(strings.TrimSpace(tag))
		if _, ok := qTokens[tagLower]; ok {
			matched++
			continue
		}
		if i := strings.LastIndexAny(tagLower, "/:"); i >= 0 && i+1 < len(tagLower) {
			if _, ok := qTokens[tagLower[i+1:]]; ok {
				matched++
			}
		}
	}
	if matched == 0 || memTagCount == 0 {
		return 0
	}
	score := float64(matched) / float64(min(memTagCount, len(qTokens)))
	if score > 1 {
		return 1
	}
	return score
}

// symbolQueryEchoPenalty downweights pattern memories that echo the query lexically but lack payload symbol overlap when the task names explicit symbols.
func symbolQueryEchoPenalty(req ScoreRequest, obj memory.MemoryObject, statement string) float64 {
	if len(req.Symbols) == 0 || req.SituationQuery == "" || obj.Kind != api.MemoryKindPattern {
		return 0
	}
	var p memory.PatternPayload
	if len(obj.Payload) == 0 || json.Unmarshal(obj.Payload, &p) != nil {
		return 0
	}
	if symbolOverlapCount(req.Symbols, p.Symbols) > 0 {
		return 0
	}
	fullLex := lexicalSimilarity(statement, req.SituationQuery)
	if fullLex < 0.30 {
		return 0
	}
	penalty := 0.55 * fullLex
	if penalty > 0.75 {
		return 0.75
	}
	return penalty
}
