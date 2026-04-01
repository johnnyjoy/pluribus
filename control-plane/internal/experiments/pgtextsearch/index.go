package pgtextsearch

import (
	"context"
	"database/sql"
	"fmt"
)

// EnsureExtension creates pg_textsearch if missing. Fails if shared_preload_libraries is wrong.
func EnsureExtension(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `CREATE EXTENSION IF NOT EXISTS pg_textsearch`)
	return err
}

// IndexExists reports whether the BM25 index exists on the projection table.
func IndexExists(ctx context.Context, db *sql.DB, projectionTable string) (bool, error) {
	var n int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND c.relkind = 'i' AND c.relname = $1
	`, bm25IndexName).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// DropBM25Index removes the BM25 index if present (e.g. before bulk reload).
func DropBM25Index(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `DROP INDEX IF EXISTS `+bm25IndexName)
	return err
}

// CreateBM25Index builds the BM25 index on doc_text (english text config).
func CreateBM25Index(ctx context.Context, db *sql.DB, projectionTable string) error {
	if projectionTable != DefaultProjectionTable {
		return fmt.Errorf("index layer expects table %q for fixed index name", DefaultProjectionTable)
	}
	_, err := db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS `+bm25IndexName+`
		ON `+DefaultProjectionTable+`
		USING bm25 (doc_text) WITH (text_config = 'english')
	`)
	return err
}

// BM25ForceMerge runs post-bulk maintenance if available (ignored on error for older builds).
func BM25ForceMerge(ctx context.Context, db *sql.DB) {
	_, _ = db.ExecContext(ctx, `SELECT bm25_force_merge($1::regclass)`, bm25IndexName)
}
