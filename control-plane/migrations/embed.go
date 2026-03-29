// Package migrations embeds baseline SQL applied on API startup (fresh Postgres; idempotent DDL).
package migrations

import "embed"

// Files holds all *.sql migration files in this directory (lexicographic order = apply order).
//
//go:embed *.sql
var Files embed.FS
