package migrate

import (
	"context"
	"database/sql"
	"fmt"
)

// RequireProofHarnessCleanPostgres fails if public.memories already exists.
// The proof harness applies baseline migrations to an empty database; reusing a DB that
// already has the schema (or data) leads to confusing failures or flaky scenarios.
func RequireProofHarnessCleanPostgres(ctx context.Context, db *sql.DB) error {
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'memories'
		)`).Scan(&exists)
	if err != nil {
		return fmt.Errorf("proof harness: could not inspect database: %w", err)
	}
	if exists {
		return fmt.Errorf("proof harness requires a clean database: public.memories already exists (drop and recreate the database, or use a new database name). See docs/evaluation.md")
	}
	return nil
}
