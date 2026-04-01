// Command pg-textsearch-eval automates pg_textsearch seed, ETL, index, and eval (branch experiments).
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"

	"control-plane/internal/experiments/pgtextsearch"
	"control-plane/internal/migrate"
	sqlmigrations "control-plane/migrations"
)

func main() {
	log.SetPrefix("[LEXICAL] ")
	log.SetFlags(0)

	dsn := flag.String("dsn", firstNonEmpty(os.Getenv("PG_TEXTSEARCH_EVAL_DSN"), os.Getenv("DATABASE_URL"), defaultDSN()), "Postgres DSN")
	replaceSeed := flag.Bool("replace-seed", true, "delete prior eval-tagged memories before seed (recommended for eval)")
	skipMigrate := flag.Bool("skip-migrate", false, "skip embedded SQL migrations (if schema already applied)")
	artifactDir := flag.String("artifact-dir", "", "write eval.json here (default: <repo>/artifacts/pg-textsearch when running from repo)")
	mdPath := flag.String("markdown", "", "write human summary markdown (default: <repo>/docs/experiments/pg-textsearch-eval-latest.md)")
	queryLimit := flag.Int("query-limit", 8, "top-k per query in suite")
	flag.Parse()

	args := flag.Args()
	sub := "eval"
	if len(args) > 0 {
		sub = args[0]
	}

	db, err := sql.Open("postgres", *dsn)
	if err != nil {
		log.Fatalf("sql open: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(4)
	db.SetConnMaxLifetime(time.Minute * 5)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	var lastPing error
	for i := 0; i < 60; i++ {
		lastPing = db.PingContext(ctx)
		if lastPing == nil {
			break
		}
		if i == 59 {
			log.Fatalf("postgres not reachable after 60 attempts: %v", lastPing)
		}
		time.Sleep(500 * time.Millisecond)
	}

	repoRoot := detectRepoRoot()
	ad := *artifactDir
	if ad == "" {
		ad = filepath.Join(repoRoot, "artifacts", "pg-textsearch")
	}
	mp := *mdPath
	if mp == "" {
		mp = filepath.Join(repoRoot, "docs", "experiments", "pg-textsearch-eval-latest.md")
	}

	switch sub {
	case "seed":
		log.Printf("seeding canonical memory (replace=%v)\n", *replaceSeed)
		n, err := pgtextsearch.Seed(ctx, db, *replaceSeed)
		if err != nil {
			log.Fatalf("seed: %v", err)
		}
		log.Printf("inserted %d memories\n", n)
	case "backfill":
		log.Printf("populating projection\n")
		n, err := pgtextsearch.Backfill(ctx, db)
		if err != nil {
			log.Fatalf("backfill: %v", err)
		}
		log.Printf("projection rows: %d\n", n)
	case "reindex":
		log.Printf("reindex (truncate + backfill + bm25)\n")
		if err := pgtextsearch.EnsureExtension(ctx, db); err != nil {
			log.Fatalf("extension: %v", err)
		}
		t0 := time.Now()
		if err := pgtextsearch.Reindex(ctx, db); err != nil {
			log.Fatalf("reindex: %v", err)
		}
		log.Printf("reindex done in %s\n", time.Since(t0).Truncate(time.Millisecond))
	case "verify":
		v, err := pgtextsearch.Verify(ctx, db)
		if err != nil {
			log.Fatalf("verify: %v", err)
		}
		if !v.OK {
			log.Fatalf("VERIFY FAIL: %s", v.Message)
		}
		log.Printf("verify ok: %s\n", v.Message)
	case "eval":
		log.Printf("running full eval pipeline\n")
		rep, err := pgtextsearch.RunEval(ctx, db, pgtextsearch.EvalOptions{
			DSN:             *dsn,
			ProjectionTable: pgtextsearch.DefaultProjectionTable,
			ReplaceSeed:     *replaceSeed,
			SkipMigrate:     *skipMigrate,
			ArtifactDir:     ad,
			MarkdownPath:    mp,
			QueryLimit:      *queryLimit,
		})
		if err != nil {
			log.Fatalf("eval: %v", err)
		}
		log.Printf("verify=%v projection_rows=%d plausible_ratio=%.2f recommendation=%s\n", rep.VerifyOK, rep.ProjectionRows, rep.PlausibleRatio, rep.Recommendation)
		log.Printf("wrote %s and %s\n", filepath.Join(ad, "eval.json"), mp)
		if len(rep.Errors) > 0 {
			for _, e := range rep.Errors {
				log.Printf("ERROR: %s\n", e)
			}
			os.Exit(1)
		}
		if !rep.VerifyOK {
			os.Exit(1)
		}
	case "migrate":
		if err := migrate.Apply(ctx, db, sqlmigrations.Files, log.Printf); err != nil {
			log.Fatalf("migrate: %v", err)
		}
		log.Printf("migrations applied\n")
	default:
		fmt.Fprintf(os.Stderr, "usage: pg-textsearch-eval [-dsn=...] <seed|backfill|reindex|verify|eval|migrate>\n")
		os.Exit(2)
	}
}

func defaultDSN() string {
	return "postgres://controlplane:controlplane@127.0.0.1:5432/controlplane?sslmode=disable"
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

func detectRepoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	dir := wd
	for i := 0; i < 10; i++ {
		if fileExists(filepath.Join(dir, "control-plane", "go.mod")) {
			return dir
		}
		if fileExists(filepath.Join(dir, "go.mod")) && filepath.Base(dir) == "control-plane" {
			return filepath.Dir(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return wd
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
