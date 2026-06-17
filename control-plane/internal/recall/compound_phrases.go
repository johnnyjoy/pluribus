package recall

import (
	"strings"
	"unicode"
)

// compoundPhraseScore returns [0,1] overlap of normalized bigrams between query and memory text+tags.
func compoundPhraseScore(query, statement string, tags []string) float64 {
	qBi := bigramsFromText(query)
	if len(qBi) == 0 {
		return 0
	}
	memText := statement + " " + strings.Join(tags, " ")
	mBi := bigramsFromText(memText)
	if len(mBi) == 0 {
		return 0
	}
	var hit, total float64
	for bg := range qBi {
		total++
		if _, ok := mBi[bg]; ok {
			hit++
		}
	}
	if total == 0 {
		return 0
	}
	return hit / total
}

func bigramsFromText(text string) map[string]struct{} {
	toks := orderedTokens(text)
	out := map[string]struct{}{}
	for i := 0; i+1 < len(toks); i++ {
		a, b := toks[i], toks[i+1]
		if a == "" || b == "" {
			continue
		}
		if isGenericRecallTerm(a) && isGenericRecallTerm(b) {
			continue
		}
		out[a+" "+b] = struct{}{}
	}
	return out
}

func orderedTokens(text string) []string {
	text = strings.ToLower(text)
	var toks []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() >= 3 {
			toks = append(toks, cur.String())
		}
		cur.Reset()
	}
	for _, r := range text {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			cur.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return toks
}

// prefixTokenOverlap adds partial credit when distinctive tokens share a meaningful prefix.
func prefixTokenOverlap(qDist, mDist map[string]struct{}) float64 {
	if len(qDist) == 0 || len(mDist) == 0 {
		return 0
	}
	var hits, total float64
	for qt := range qDist {
		if isGenericRecallTerm(qt) {
			continue
		}
		total++
		for mt := range mDist {
			if tokensPrefixMatch(qt, mt) {
				hits++
				break
			}
		}
	}
	if total == 0 {
		return 0
	}
	return hits / total
}

func tokensPrefixMatch(a, b string) bool {
	if a == b {
		return true
	}
	if len(a) < 5 || len(b) < 5 {
		return false
	}
	if strings.HasPrefix(b, a) || strings.HasPrefix(a, b) {
		return true
	}
	return false
}

func interfaceIntegrationAffinity(query string, memTags []string) float64 {
	qToks := orderedTokens(query)
	var hits, total float64
	for _, want := range []string{"interface", "integration", "integrate"} {
		for _, qt := range qToks {
			if qt == want || tokensPrefixMatch(qt, want) || tokensPrefixMatch(want, qt) {
				total++
				for _, tag := range memTags {
					tag = strings.ToLower(strings.TrimSpace(tag))
					if tag == "" || isBenchmarkInfraTag(tag) {
						continue
					}
					seg := tag
					if i := strings.LastIndexAny(tag, "/:"); i >= 0 && i+1 < len(tag) {
						seg = tag[i+1:]
					}
					if seg == "interface" || seg == "integration" || tokensPrefixMatch(qt, seg) {
						hits++
						break
					}
				}
				break
			}
		}
	}
	if total == 0 {
		return 0
	}
	return hits / total
}

// queryHasStrongDomainAnchor is true when the query names a project/product (capitalized token or long anchor).
func queryStrongDomainAnchors(query string, reqTags []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, tok := range strings.FieldsFunc(query, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r))
	}) {
		tok = strings.TrimSpace(tok)
		if len(tok) < 4 {
			continue
		}
		lower := strings.ToLower(tok)
		if isGenericRecallTerm(lower) {
			continue
		}
		if len(tok) >= 8 || (len(tok) > 0 && unicode.IsUpper(rune(tok[0]))) {
			out[lower] = struct{}{}
		}
	}
	for _, tag := range reqTags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag != "" && !isGenericRecallTerm(tag) {
			out[tag] = struct{}{}
		}
	}
	return out
}

func genericSaturatedQuery(query string, reqTags []string) bool {
	if len(queryDistinctiveTokens(SituationAffinityInput{
		SituationQuery: query,
		RequestTags:    reqTags,
	})) > 0 {
		return false
	}
	genericHits := 0
	for _, tok := range orderedTokens(query) {
		if isGenericRecallTerm(tok) {
			genericHits++
		}
	}
	return genericHits >= 3
}
