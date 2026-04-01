package pgtextsearch

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"control-plane/internal/lexical"
)

// QueryCase is one benchmark query with optional expectations.
type QueryCase struct {
	Category    string `json:"category"`
	Query       string `json:"query"`
	ExpectTerms []string `json:"expect_terms,omitempty"` // if any appear in top-1 statement, Plausible=true
}

// DefaultQuerySuite covers exact, failure, pattern, ugly, mixed-style retrieval.
func DefaultQuerySuite() []QueryCase {
	return []QueryCase{
		{Category: "exact_constraint", Query: "sqlite production writes", ExpectTerms: []string{"sqlite", "production"}},
		{Category: "exact_constraint", Query: "idempotency retry duplicates", ExpectTerms: []string{"idempotency", "duplicate"}},
		{Category: "failure_recall", Query: "downstream sync duplicate", ExpectTerms: []string{"downstream", "sync"}},
		{Category: "failure_recall", Query: "badge update race warehouse", ExpectTerms: []string{"badge", "warehouse"}},
		{Category: "pattern_recall", Query: "streaming decode large json", ExpectTerms: []string{"json", "stream"}},
		{Category: "pattern_recall", Query: "reduce peak memory import", ExpectTerms: []string{"memory", "import"}},
		{Category: "ugly_operator", Query: "sqlite dupes", ExpectTerms: []string{"sqlite"}},
		{Category: "ugly_operator", Query: "json import oom", ExpectTerms: []string{"json"}},
		{Category: "ugly_operator", Query: "warehouse lag badge", ExpectTerms: []string{"warehouse"}},
		{Category: "mixed", Query: "postgres idempotency webhook", ExpectTerms: []string{"idempotency"}},
		{Category: "mixed", Query: "entity:warehouse replication lag", ExpectTerms: []string{"warehouse"}},
	}
}

// SuiteResult is one query outcome.
type SuiteResult struct {
	QueryCase
	LatencyMS      float64  `json:"latency_ms"`
	Hits           []lexical.Hit `json:"hits"`
	TopStatements  []string `json:"top_statements,omitempty"`
	Plausible      bool     `json:"plausible"`
	BaselineIDs    []string `json:"baseline_ilike_ids,omitempty"`
	Notes          string   `json:"notes,omitempty"`
}

// RunQuerySuite runs BM25 search + a naive ILIKE baseline for sanity.
func RunQuerySuite(ctx context.Context, db *sql.DB, projectionTable string, limit int) ([]SuiteResult, error) {
	cases := DefaultQuerySuite()
	out := make([]SuiteResult, 0, len(cases))
	for _, qc := range cases {
		sr := SuiteResult{QueryCase: qc}
		t0 := time.Now()
		hits, err := lexical.Search(ctx, db, projectionTable, qc.Query, limit)
		sr.LatencyMS = float64(time.Since(t0).Microseconds()) / 1000.0
		if err != nil {
			sr.Notes = err.Error()
			out = append(out, sr)
			continue
		}
		sr.Hits = hits
		// Resolve statements for top hits
		for _, h := range hits {
			if len(sr.TopStatements) >= 3 {
				break
			}
			var stmt string
			_ = db.QueryRowContext(ctx, `SELECT statement FROM memories WHERE id = $1::uuid`, h.MemoryID).Scan(&stmt)
			if stmt != "" {
				sr.TopStatements = append(sr.TopStatements, stmt)
			}
		}
		sr.Plausible = scorePlausible(sr.TopStatements, qc.ExpectTerms)
		// Naive baseline: ILIKE '%term%' using first expect term or first query token
		baseTerm := pickBaselineTerm(qc)
		if baseTerm != "" {
			rows, err := db.QueryContext(ctx, `
				SELECT id::text FROM memories WHERE status = 'active' AND statement ILIKE $1 LIMIT 5
			`, "%"+baseTerm+"%")
			if err == nil {
				for rows.Next() {
					var id string
					if rows.Scan(&id) == nil {
						sr.BaselineIDs = append(sr.BaselineIDs, id)
					}
				}
				_ = rows.Close()
			}
		}
		out = append(out, sr)
	}
	return out, nil
}

func pickBaselineTerm(qc QueryCase) string {
	if len(qc.ExpectTerms) > 0 {
		return qc.ExpectTerms[0]
	}
	parts := strings.Fields(qc.Query)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func scorePlausible(stmts []string, terms []string) bool {
	if len(stmts) == 0 || len(terms) == 0 {
		return false
	}
	top := strings.ToLower(stmts[0])
	matched := 0
	for _, t := range terms {
		if strings.Contains(top, strings.ToLower(t)) {
			matched++
		}
	}
	return matched >= 1
}

// FailIfSuiteEmpty returns error if every query returned zero lexical hits.
func FailIfSuiteEmpty(results []SuiteResult) error {
	any := false
	for _, r := range results {
		if len(r.Hits) > 0 {
			any = true
			break
		}
	}
	if !any {
		return fmt.Errorf("lexical query suite returned zero hits for all queries (extension/index/corpus check failed)")
	}
	return nil
}
