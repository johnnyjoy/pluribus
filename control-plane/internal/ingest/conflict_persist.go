package ingest

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// persistCanonicalContradictions stores in-batch and cross-batch contradictions after canonical rows are finalized.
// Idempotent: ON CONFLICT DO NOTHING. Returns RowsAffected sum from inserts.
func persistCanonicalContradictions(ctx context.Context, tx *sql.Tx, ingestionID uuid.UUID, rows []CanonicalFactRow, conflictObjectMaxJaccard float64) (int64, error) {
	maxJ := conflictObjectMaxJaccard
	if maxJ <= 0 {
		maxJ = DefaultConflictObjectMaxJaccard
	}
	seen := make(map[string]struct{})
	var inserted int64

	tryInsert := func(subject string, predA, predB, hashA, hashB, typ string) error {
		ha, hb := hashA, hashB
		pa, pb := predA, predB
		if ha > hb {
			ha, hb = hb, ha
			pa, pb = pb, pa
		}
		key := fmt.Sprintf("%s|%s|%s|%s|%s", ha, hb, typ, subject, pa)
		if _, ok := seen[key]; ok {
			return nil
		}
		seen[key] = struct{}{}
		res, err := tx.ExecContext(ctx,
			`INSERT INTO canonical_fact_contradictions (ingestion_id, subject_norm, predicate_norm_a, predicate_norm_b, fact_hash_a, fact_hash_b, contradiction_type, confidence_delta)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, NULL)
			 ON CONFLICT (fact_hash_a, fact_hash_b, contradiction_type) DO NOTHING`,
			ingestionID, subject, pa, pb, ha, hb, typ,
		)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		inserted += n
		return nil
	}

	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[i].SubjectNorm != rows[j].SubjectNorm {
				continue
			}
			a, b := rows[i], rows[j]
			if predicatesOpposite(a.PredicateNorm, b.PredicateNorm) {
				if err := tryInsert(a.SubjectNorm, a.PredicateNorm, b.PredicateNorm, a.NormalizedHash, b.NormalizedHash, "opposite_predicate"); err != nil {
					return 0, err
				}
				continue
			}
			if a.PredicateNorm == b.PredicateNorm {
				if a.ObjectNorm == b.ObjectNorm {
					continue
				}
				jacc := TokenJaccard(a.ObjectNorm, b.ObjectNorm)
				if jacc < maxJ {
					if err := tryInsert(a.SubjectNorm, a.PredicateNorm, b.PredicateNorm, a.NormalizedHash, b.NormalizedHash, "same_predicate_divergent_object"); err != nil {
						return 0, err
					}
				}
			}
		}
	}

	for i := range rows {
		row := &rows[i]
		opp, ok := OppositePredicateNorm(row.PredicateNorm)
		if !ok {
			continue
		}
		dbRows, err := tx.QueryContext(ctx,
			`SELECT normalized_hash, predicate_norm FROM canonical_fact_extractions
			 WHERE subject_norm = $1 AND predicate_norm = $2`,
			row.SubjectNorm, opp,
		)
		if err != nil {
			return 0, err
		}
		for dbRows.Next() {
			var dbHash, dbPred string
			if err := dbRows.Scan(&dbHash, &dbPred); err != nil {
				_ = dbRows.Close()
				return 0, err
			}
			if dbHash == row.NormalizedHash {
				continue
			}
			if err := tryInsert(row.SubjectNorm, row.PredicateNorm, dbPred, row.NormalizedHash, dbHash, "opposite_predicate"); err != nil {
				_ = dbRows.Close()
				return 0, err
			}
		}
		if err := dbRows.Err(); err != nil {
			_ = dbRows.Close()
			return 0, err
		}
		_ = dbRows.Close()
	}

	return inserted, nil
}
