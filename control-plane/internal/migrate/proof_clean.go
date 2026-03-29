package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
)

// MaybeResetPublicSchemaForIntegrationTests drops and recreates the public schema when
// TEST_PG_RESET_SCHEMA=1. CI regression shares one Postgres across all integration
// packages; other tests may migrate first, but the proof harness requires a clean DB
// (see RequireProofHarnessCleanPostgres). Pair with go test -p 1 so no two packages
// run against the same DSN concurrently.
func MaybeResetPublicSchemaForIntegrationTests(ctx context.Context, db *sql.DB) error {
	if strings.TrimSpace(os.Getenv("TEST_PG_RESET_SCHEMA")) != "1" {
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("integration reset schema: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DROP SCHEMA IF EXISTS public CASCADE`); err != nil {
		return fmt.Errorf("integration reset schema: drop public: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `CREATE SCHEMA public`); err != nil {
		return fmt.Errorf("integration reset schema: create public: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `GRANT ALL ON SCHEMA public TO PUBLIC`); err != nil {
		return fmt.Errorf("integration reset schema: grant on public: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("integration reset schema: commit: %w", err)
	}
	return nil
}

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
