package pgtextsearch

import (
	"context"
	"database/sql"
	"fmt"

	"control-plane/internal/memorynorm"
)

// CountSeeded returns how many memories carry the eval tag.
func CountSeeded(ctx context.Context, db *sql.DB) (int, error) {
	var n int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT memory_id) FROM memories_tags WHERE tag = $1
	`, EvalTag).Scan(&n)
	return n, err
}

// DeleteSeeded removes memories tagged with EvalTag (projection rows cascade via memory delete).
func DeleteSeeded(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		DELETE FROM memories WHERE id IN (
			SELECT memory_id FROM memories_tags WHERE tag = $1
		)
	`, EvalTag)
	return err
}

// Seed inserts canonical eval memories (idempotent: skips if already seeded unless replace is true).
func Seed(ctx context.Context, db *sql.DB, replace bool) (inserted int, err error) {
	if replace {
		if err := DeleteSeeded(ctx, db); err != nil {
			return 0, fmt.Errorf("delete seeded: %w", err)
		}
	} else {
		n, err := CountSeeded(ctx, db)
		if err != nil {
			return 0, err
		}
		if n > 0 {
			return 0, fmt.Errorf("eval seed already present (%d rows with tag %q); use -replace-seed or run delete first", n, EvalTag)
		}
	}
	rows := BuildSeedRows()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	for _, r := range rows {
		canon := memorynorm.StatementCanonical(r.Statement)
		stmtKey := memorynorm.StatementKey(r.Statement)
		if stmtKey == "" {
			return 0, fmt.Errorf("empty statement_key for row kind=%s dedup=%s", r.Kind, r.DedupKey)
		}
		var id string
		err := tx.QueryRowContext(ctx, `
			INSERT INTO memories (kind, statement, statement_canonical, statement_key, dedup_key, authority, applicability, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 'active')
			RETURNING id::text
		`, r.Kind, r.Statement, canon, stmtKey, r.DedupKey, r.Authority, r.Applicability).Scan(&id)
		if err != nil {
			return inserted, fmt.Errorf("insert memory dedup=%s: %w", r.DedupKey, err)
		}
		for _, t := range r.Tags {
			if _, err := tx.ExecContext(ctx, `INSERT INTO memories_tags (memory_id, tag) VALUES ($1::uuid, $2)`, id, t); err != nil {
				return inserted, fmt.Errorf("insert tag %q: %w", t, err)
			}
		}
		inserted++
	}
	if err := tx.Commit(); err != nil {
		return inserted, err
	}
	return inserted, nil
}
