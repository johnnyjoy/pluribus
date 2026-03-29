package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"control-plane/internal/cache"
	"control-plane/internal/httpx"
	"control-plane/internal/migrate"
	sqlmigrations "control-plane/migrations"

	_ "github.com/lib/pq"
)

// Container holds wired dependencies (DB, EvidenceRoot, optional API key, optional Cache).
type Container struct {
	DB           *sql.DB
	EvidenceRoot string
	// APIKey is the configured Pluribus API key (from PLURIBUS_API_KEY), or nil when auth is off.
	APIKey []byte
	Cache cache.Store
}

// Boot opens the database, ensures evidence root exists, and loads PLURIBUS_API_KEY when set.
func Boot(cfg *Config) (*Container, error) {
	db, err := sql.Open("postgres", cfg.Postgres.DSN)
	if err != nil {
		return nil, err
	}
	if err := waitForDB(context.Background(), db, cfg.Startup.DBWaitTimeoutSeconds, cfg.Startup.DBWaitIntervalMillis); err != nil {
		db.Close()
		return nil, err
	}
	// Baseline SQL replay (idempotent DDL). Not a versioned upgrade: fresh/disposable DBs only until releases define one.
	if err := migrate.Apply(context.Background(), db, sqlmigrations.Files, log.Printf); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply baseline schema: %w", err)
	}
	ok, err := migrate.CoreSchemaReady(context.Background(), db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("check core schema: %w", err)
	}
	if !ok {
		db.Close()
		return nil, fmt.Errorf("baseline schema did not create required tables (expected memories)")
	}
	root := cfg.Evidence.RootPath
	if root == "" {
		root = "var/evidence"
	}
	if err := os.MkdirAll(filepath.Clean(root), 0o755); err != nil {
		db.Close()
		return nil, err
	}
	apiKey := httpx.LoadPluribusAPIKey()
	var c cache.Store
	if cfg.Redis.Enabled && cfg.Redis.Addr != "" {
		ttl := time.Duration(cfg.Redis.DefaultTTLSec) * time.Second
		if ttl <= 0 {
			ttl = 5 * time.Minute
		}
		redisStore, err := cache.NewRedis(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, ttl)
		if err != nil {
			db.Close()
			return nil, err
		}
		c = redisStore
	}
	return &Container{
		DB:           db,
		EvidenceRoot: root,
		APIKey:       apiKey,
		Cache:        c,
	}, nil
}

func waitForDB(ctx context.Context, db *sql.DB, timeoutSec, intervalMS int) error {
	timeout := time.Duration(timeoutSec) * time.Second
	interval := time.Duration(intervalMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	if interval <= 0 {
		interval = 1 * time.Second
	}

	deadlineCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastErr error
	for {
		pingCtx, pingCancel := context.WithTimeout(deadlineCtx, 3*time.Second)
		err := db.PingContext(pingCtx)
		pingCancel()
		if err == nil {
			log.Printf("startup: database reachable")
			return nil
		}
		lastErr = err
		log.Printf("startup: waiting for database: %v", err)
		select {
		case <-deadlineCtx.Done():
			if errors.Is(deadlineCtx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("database wait timeout after %s: %w", timeout, lastErr)
			}
			return fmt.Errorf("database wait canceled: %w", deadlineCtx.Err())
		case <-ticker.C:
		}
	}
}
