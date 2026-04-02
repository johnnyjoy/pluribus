package pgtextsearch

import (
	"context"
	"database/sql"
	"fmt"
)

// Backfill upserts lexical_memory_projection from canonical memories (all active rows).
func Backfill(ctx context.Context, db *sql.DB) (int64, error) {
	_, err := db.ExecContext(ctx, `
		INSERT INTO lexical_memory_projection (memory_id, doc_text)
		SELECT m.id,
			trim(both ' ' FROM (
				m.kind || ' ' || m.statement || ' ' ||
				COALESCE(string_agg(mt.tag, ' ' ORDER BY mt.tag), '')
			))
		FROM memories m
		LEFT JOIN memories_tags mt ON mt.memory_id = m.id
		WHERE m.status = 'active'
		GROUP BY m.id, m.kind, m.statement
		ON CONFLICT (memory_id) DO UPDATE SET doc_text = EXCLUDED.doc_text
	`)
	if err != nil {
		return 0, err
	}
	var n int64
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM lexical_memory_projection`).Scan(&n)
	return n, err
}

// TruncateProjection removes all projection rows (canonical memories untouched).
func TruncateProjection(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `TRUNCATE lexical_memory_projection`)
	return err
}

// Reindex truncates projection, backfills, recreates BM25 index, optional merge.
func Reindex(ctx context.Context, db *sql.DB) error {
	if err := DropBM25Index(ctx, db); err != nil {
		return fmt.Errorf("drop bm25: %w", err)
	}
	if err := TruncateProjection(ctx, db); err != nil {
		return fmt.Errorf("truncate projection: %w", err)
	}
	if _, err := Backfill(ctx, db); err != nil {
		return fmt.Errorf("backfill: %w", err)
	}
	if err := CreateBM25Index(ctx, db, DefaultProjectionTable); err != nil {
		return fmt.Errorf("create bm25: %w", err)
	}
	BM25ForceMerge(ctx, db)
	return nil
}

// VerifyReport holds invariant checks.
type VerifyReport struct {
	MemoriesActive     int64
	MemoriesWithEvalTag int64
	ProjectionRows     int64
	MismatchIDs        []string
	OK                 bool
	Message            string
}

// Verify checks row counts; seeded eval expects projection to include all active memories after backfill.
func Verify(ctx context.Context, db *sql.DB) (*VerifyReport, error) {
	r := &VerifyReport{}
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memories WHERE status = 'active'`).Scan(&r.MemoriesActive)
	if err != nil {
		return nil, err
	}
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT memory_id) FROM memories_tags WHERE tag = $1
	`, EvalTag).Scan(&r.MemoriesWithEvalTag)
	if err != nil {
		return nil, err
	}
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM lexical_memory_projection`).Scan(&r.ProjectionRows)
	if err != nil {
		return nil, err
	}
	// Every active memory should appear in projection after backfill.
	var missing int64
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM memories m
		WHERE m.status = 'active'
		  AND NOT EXISTS (SELECT 1 FROM lexical_memory_projection p WHERE p.memory_id = m.id)
	`).Scan(&missing)
	if err != nil {
		return nil, err
	}
	if r.MemoriesActive != r.ProjectionRows || missing > 0 {
		r.OK = false
		r.Message = fmt.Sprintf("projection count %d != active memories %d (missing links: %d)", r.ProjectionRows, r.MemoriesActive, missing)
	} else {
		r.OK = true
		r.Message = "projection row count matches active memories"
	}
	return r, nil
}
