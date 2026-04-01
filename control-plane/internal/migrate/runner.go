// Package migrate applies embedded baseline SQL on startup. There is no schema version
// table, no ledger, and no supported upgrade path from older installs: use a fresh Postgres
// (or an empty public schema you treat as disposable). Replaying SQL on boot is bootstrap
// idempotency for greenfield deployments — not a versioned migration engine.
package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// Apply runs every *.sql file from files in lexicographic order. Each file runs in one transaction.
// Baseline DDL must be idempotent (CREATE IF NOT EXISTS, CREATE INDEX IF NOT EXISTS, etc.).
// Callers must not use this to “upgrade” arbitrary legacy databases; there is no compatibility matrix.
func Apply(ctx context.Context, db *sql.DB, files fs.FS, logf func(string, ...any)) error {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	names, err := listSQLFiles(files)
	if err != nil {
		return err
	}
	sort.Strings(names)
	for _, name := range names {
		body, err := fs.ReadFile(files, name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if err := applySQLFile(ctx, db, name, string(body)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
		logf("schema: applied %s", name)
	}
	logf("schema: done (%d file(s))", len(names))
	return nil
}

// CoreSchemaReady is true when the primary memories table exists (post-apply sanity check).
func CoreSchemaReady(ctx context.Context, db *sql.DB) (bool, error) {
	var ok bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'memories'
		)
	`).Scan(&ok)
	return ok, err
}

func listSQLFiles(files fs.FS) ([]string, error) {
	ents, err := fs.ReadDir(files, ".")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasSuffix(n, ".sql") {
			names = append(names, n)
		}
	}
	return names, nil
}

// applySQLFile runs SQL in a transaction. Multi-statement files are split on semicolons
// after stripping `--` comments (including inline), so semicolons inside comments do not split.
func applySQLFile(ctx context.Context, db *sql.DB, label, sqlText string) error {
	stmts := splitStatements(stripSQLLineComments(sqlText))
	if len(stmts) == 0 {
		return fmt.Errorf("empty SQL file %s", label)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for i, s := range stmts {
		if _, err := tx.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("statement %d in %s: %w", i+1, label, err)
		}
	}
	return tx.Commit()
}

// stripSQLLineComments removes `-- ...` to end of line (Postgres line comments).
func stripSQLLineComments(sqlText string) string {
	sqlText = strings.ReplaceAll(sqlText, "\r\n", "\n")
	var b strings.Builder
	for _, line := range strings.Split(sqlText, "\n") {
		if i := strings.Index(line, "--"); i >= 0 {
			line = line[:i]
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// splitStatements splits on semicolons outside single-quoted strings and dollar-quoted
// bodies (e.g. DO $$ ... $$;). Naive strings.Split would break PL/pgSQL blocks.
func splitStatements(sqlText string) []string {
	text := strings.TrimSpace(sqlText)
	var out []string
	var b strings.Builder
	b.Grow(len(text))
	inSingle := false
	for i := 0; i < len(text); {
		c := text[i]
		if inSingle {
			b.WriteByte(c)
			if c == '\'' {
				if i+1 < len(text) && text[i+1] == '\'' {
					b.WriteByte(text[i+1])
					i += 2
					continue
				}
				inSingle = false
			}
			i++
			continue
		}
		if c == '\'' {
			inSingle = true
			b.WriteByte(c)
			i++
			continue
		}
		if c == '$' {
			tag, openEnd, ok := parseDollarQuoteOpen(text, i)
			if ok {
				closeDelim := "$" + tag + "$"
				rest := text[openEnd:]
				closeIdx := strings.Index(rest, closeDelim)
				if closeIdx < 0 {
					b.WriteString(text[i:])
					break
				}
				end := openEnd + closeIdx + len(closeDelim)
				b.WriteString(text[i:end])
				i = end
				continue
			}
		}
		if c == ';' {
			s := strings.TrimSpace(b.String())
			if s != "" {
				out = append(out, s)
			}
			b.Reset()
			i++
			continue
		}
		b.WriteByte(c)
		i++
	}
	if s := strings.TrimSpace(b.String()); s != "" {
		out = append(out, s)
	}
	return out
}

// parseDollarQuoteOpen parses a PostgreSQL dollar-quote opener at text[i].
// Returns the tag (possibly empty for $$), the index after the closing $ of the opener, and true on success.
func parseDollarQuoteOpen(text string, i int) (tag string, openEnd int, ok bool) {
	if i >= len(text) || text[i] != '$' {
		return "", i, false
	}
	j := i + 1
	for j < len(text) && text[j] != '$' {
		j++
	}
	if j >= len(text) {
		return "", i, false
	}
	return text[i+1 : j], j + 1, true
}
