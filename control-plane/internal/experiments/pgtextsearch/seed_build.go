package pgtextsearch

import (
	"fmt"
	"math/rand"
	"strings"
)

// SeedRow is one canonical memory to insert.
type SeedRow struct {
	Kind       string
	Statement  string
	Tags       []string
	DedupKey   string
	Authority  int
	Applicability string
}

// BuildSeedRows returns deterministic eval memories (≥50 rows, typically 120+).
func BuildSeedRows() []SeedRow {
	rnd := rand.New(rand.NewSource(42))
	var out []SeedRow
	seq := 0
	for _, kind := range []string{"constraint", "failure", "pattern", "decision", "state"} {
		stmts := seedStatements[kind]
		for i, s := range stmts {
			seq++
			dk := fmt.Sprintf("eval:pt:v1:%s:%04d", kind, i+1)
			tags := buildTags(kind, i, rnd)
			out = append(out, SeedRow{
				Kind:          kind,
				Statement:     s,
				Tags:          tags,
				DedupKey:      dk,
				Authority:     5 + rnd.Intn(5),
				Applicability: pickApplicability(rnd),
			})
		}
	}
	// Near-duplicate / paraphrase pairs for lexical overlap tests.
	pairs := []struct {
		kind string
		a, b string
	}{
		{"constraint", "SQLite must not back durable writes in production.", "Production durable writes cannot use SQLite as the source of truth."},
		{"failure", "Downstream sync duplicated rows when retries lacked idempotency keys.", "Duplicate records appeared in downstream sync after non-idempotent retries."},
		{"pattern", "Stream-parse huge JSON payloads to avoid OOM during imports.", "Use streaming JSON parsing on large imports to prevent out-of-memory failures."},
		{"decision", "Hybrid recall will fuse lexical and vector candidates before authority weighting.", "We will merge lexical BM25 and embedding candidates prior to applying authority."},
		{"state", "Warehouse lag can exceed five minutes during peaks.", "Peak traffic may delay warehouse freshness by more than five minutes."},
	}
	for i, p := range pairs {
		seq++
		dk := fmt.Sprintf("eval:pt:v1:dup:%04d", i+1)
		tags := append(buildTags(p.kind, 100+i, rnd), "eval:near-dup")
		out = append(out, SeedRow{
			Kind:          p.kind,
			Statement:     p.a,
			Tags:          tags,
			DedupKey:      dk,
			Authority:     6,
			Applicability: "governing",
		})
		seq++
		dk2 := fmt.Sprintf("eval:pt:v1:dupb:%04d", i+1)
		out = append(out, SeedRow{
			Kind:          p.kind,
			Statement:     p.b,
			Tags:          append(tags, "eval:paraphrase"),
			DedupKey:      dk2,
			Authority:     6,
			Applicability: "governing",
		})
	}
	// Short ugly operator-style phrasing (same eval tag).
	for i := 0; i < 25; i++ {
		seq++
		s := uglyOperatorLine(rnd, i)
		out = append(out, SeedRow{
			Kind:          pickKind(rnd, i),
			Statement:     s,
			Tags:          []string{EvalTag, "eval:ugly", fmt.Sprintf("eval:seq:%d", seq)},
			DedupKey:      fmt.Sprintf("eval:pt:v1:ugly:%04d", i+1),
			Authority:     4,
			Applicability: "governing",
		})
	}
	_ = seq
	return out
}

func pickApplicability(r *rand.Rand) string {
	if r.Intn(10) < 8 {
		return "governing"
	}
	return "advisory"
}

func pickKind(r *rand.Rand, i int) string {
	kinds := []string{"constraint", "failure", "pattern", "decision", "state"}
	return kinds[i%len(kinds)]
}

func buildTags(kind string, idx int, r *rand.Rand) []string {
	tags := []string{EvalTag, "eval:" + kind}
	tags = append(tags, fmt.Sprintf("domain:%s", kind))
	entities := []string{"entity:postgres", "entity:warehouse", "entity:sync", "entity:ingest", "entity:oncall"}
	tags = append(tags, entities[r.Intn(len(entities))])
	if r.Intn(3) == 0 {
		tags = append(tags, "tag:ops")
	}
	if r.Intn(4) == 0 {
		tags = append(tags, fmt.Sprintf("team:%s", []string{"platform", "data", "core"}[r.Intn(3)]))
	}
	tags = append(tags, fmt.Sprintf("idx:%d", idx))
	return uniqStrings(tags)
}

func uniqStrings(s []string) []string {
	seen := make(map[string]struct{}, len(s))
	var out []string
	for _, x := range s {
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

func uglyOperatorLine(r *rand.Rand, i int) string {
	// Terse, operator-style strings (not lorem ipsum).
	opts := []string{
		"sqlite prod writes — blocked",
		"json import oom — use stream decode",
		"warehouse lag badge race — known",
		"webhook retry dupes — need idempotency",
		"tls1.0 endpoint — kill",
		"migration double-run — check advisory lock",
		"pool fd leak — restart mitigates",
		"offset pagination @50k — bad",
		"canary p95 ok errors bad — check SLO",
		"rrf fusion later — lexical first",
	}
	s := opts[i%len(opts)]
	// Suffix keeps statement_key unique (canonical hash) across repeated templates.
	s = fmt.Sprintf("%s [ugly-%d]", s, i)
	if r.Intn(2) == 0 {
		s = strings.ToUpper(s[:1]) + s[1:]
	}
	return s
}
