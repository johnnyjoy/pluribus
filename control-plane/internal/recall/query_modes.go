package recall

import (
	"strings"
	"unicode"

	"control-plane/internal/memory"
	"control-plane/pkg/api"
)

var conversationalQueryTerms = map[string]struct{}{
	"help": {}, "please": {}, "properly": {}, "easily": {}, "simply": {},
	"want": {}, "need": {}, "trying": {}, "try": {}, "how": {},
}

func isConversationalQueryTerm(tok string) bool {
	_, ok := conversationalQueryTerms[strings.ToLower(tok)]
	return ok
}

func conversationalVagueQuery(query string, reqTags []string) bool {
	if len(reqTags) > 0 {
		return false
	}
	toks := orderedTokens(query)
	hasConversational := false
	for _, tok := range toks {
		if isConversationalQueryTerm(tok) {
			hasConversational = true
			break
		}
	}
	if !hasConversational {
		return false
	}
	anchors := queryStrongDomainAnchors(query, reqTags)
	return len(anchors) == 0
}

func technicalGenericIntegrationCluster(query string, reqTags []string) bool {
	if len(reqTags) > 0 {
		return false
	}
	if conversationalVagueQuery(query, reqTags) {
		return false
	}
	toks := orderedTokens(query)
	if len(toks) < 4 {
		return false
	}
	cluster := map[string]struct{}{
		"agent": {}, "interface": {}, "integration": {}, "binding": {}, "recall": {},
	}
	hits := 0
	for _, tok := range toks {
		if _, ok := cluster[tok]; ok {
			hits++
		}
	}
	genericHits := 0
	for _, tok := range toks {
		if isGenericRecallTerm(tok) {
			genericHits++
		}
	}
	return hits >= 3 && genericHits >= len(toks)-1
}

func tagsAreAllGeneric(tags []string) bool {
	if len(tags) == 0 {
		return true
	}
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" || isBenchmarkInfraTag(tag) {
			continue
		}
		seg := tag
		if i := strings.LastIndexAny(tag, "/:"); i >= 0 && i+1 < len(tag) {
			seg = tag[i+1:]
		}
		if !isGenericRecallTerm(seg) && len(seg) >= 5 {
			return false
		}
	}
	return true
}

var commonDistinctiveTokens = map[string]struct{}{
	"external": {}, "standard": {}, "systems": {}, "deployment": {}, "checklists": {},
	"verified": {}, "releases": {}, "interface": {}, "integration": {}, "constraint": {},
	"compliance": {}, "teams": {}, "services": {}, "agents": {}, "layers": {},
	"workflows": {}, "workflow": {}, "experience": {}, "afterward": {}, "multiword": {},
	"completion": {}, "outcomes": {}, "improve": {}, "prior": {}, "action": {},
}

func memoryHasProductAnchor(statement string, tags []string) bool {
	_ = tags
	for _, tok := range strings.FieldsFunc(statement, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r))
	}) {
		tok = strings.Trim(strings.TrimSpace(tok), ".,;:")
		if len(tok) >= 5 && unicode.IsUpper(rune(tok[0])) {
			lower := strings.ToLower(tok)
			if isGenericRecallTerm(lower) || isCommonDistinctive(lower) {
				continue
			}
			if len(lower) >= 6 {
				return true
			}
		}
	}
	for tok := range distinctiveTokenSet(statement, nil) {
		if len(tok) >= 8 && !isGenericRecallTerm(tok) && !isCommonDistinctive(tok) {
			return true
		}
	}
	return false
}

func isCommonDistinctive(tok string) bool {
	_, ok := commonDistinctiveTokens[strings.ToLower(tok)]
	return ok
}

func isLowAuthorityGenericPattern(obj memory.MemoryObject, scoringTags []string) bool {
	if obj.Authority > 4 {
		return false
	}
	if obj.Kind != api.MemoryKindPattern && obj.Kind != api.MemoryKindState && obj.Kind != api.MemoryKindConstraint {
		return false
	}
	return !memoryHasProductAnchor(obj.Statement, scoringTags) && tagsAreAllGeneric(scoringTags)
}

func queryAnchorsIncludingRepo(query string, reqTags []string, repoRoot string) map[string]struct{} {
	out := queryStrongDomainAnchors(query, reqTags)
	if repoRoot != "" {
		base := strings.ToLower(strings.TrimSpace(repoRoot))
		if i := strings.LastIndex(base, "/"); i >= 0 {
			base = base[i+1:]
		}
		if len(base) >= 3 && !isGenericRecallTerm(base) {
			out[base] = struct{}{}
		}
	}
	return out
}

func hyphenatedTagQueryAffinity(query string, reqTags, memTags []string) float64 {
	qToks := orderedTokens(query)
	qSet := map[string]bool{}
	for _, t := range qToks {
		qSet[t] = true
	}
	for _, tag := range reqTags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" || isBenchmarkInfraTag(tag) {
			continue
		}
		for _, p := range strings.FieldsFunc(tag, func(r rune) bool { return r == '-' || r == '_' || r == '/' }) {
			if len(p) >= 3 {
				qSet[p] = true
			}
		}
	}
	var hits int
	for _, tag := range memTags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" || isBenchmarkInfraTag(tag) {
			continue
		}
		parts := strings.FieldsFunc(tag, func(r rune) bool { return r == '-' || r == '_' || r == '/' })
		if len(parts) < 2 {
			continue
		}
		ok := true
		seen := 0
		for _, p := range parts {
			if len(p) < 3 {
				continue
			}
			seen++
			if !qSet[p] {
				ok = false
				break
			}
		}
		if ok && seen >= 2 {
			hits++
		}
	}
	if hits == 0 {
		return 0
	}
	if hits > 2 {
		hits = 2
	}
	return float64(hits) * 0.5
}

func strongCompoundPhraseHit(query, statement string, tags []string) bool {
	qBi := bigramsFromText(query)
	if len(qBi) == 0 {
		return false
	}
	memText := statement + " " + strings.Join(tags, " ")
	mBi := bigramsFromText(memText)
	for bg := range qBi {
		if _, ok := mBi[bg]; ok {
			return true
		}
	}
	return false
}

func matchedTagsAreAllGeneric(reqTags, memTags []string) bool {
	if len(reqTags) == 0 {
		return tagsAreAllGeneric(memTags)
	}
	for _, want := range reqTags {
		want = strings.ToLower(strings.TrimSpace(want))
		if want == "" || isBenchmarkInfraTag(want) {
			continue
		}
		if !isGenericRecallTerm(want) {
			return false
		}
	}
	return true
}

func memoryTagContains(memTags []string, needle string) bool {
	needle = strings.ToLower(strings.TrimSpace(needle))
	for _, tag := range memTags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == needle || strings.Contains(tag, needle) {
			return true
		}
	}
	return false
}

func isGenericNoiseMemory(obj memory.MemoryObject, scoringTags []string) bool {
	if obj.Authority > 4 {
		return false
	}
	return !memoryHasProductAnchor(obj.Statement, scoringTags)
}

func queryNamedProduct(query string) bool {
	for _, tok := range orderedTokens(query) {
		if len(tok) >= 8 && !isGenericRecallTerm(tok) {
			return true
		}
	}
	for _, tok := range strings.FieldsFunc(query, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r))
	}) {
		if len(tok) > 0 && unicode.IsUpper(rune(tok[0])) && len(strings.ToLower(tok)) >= 5 {
			return true
		}
	}
	return false
}
