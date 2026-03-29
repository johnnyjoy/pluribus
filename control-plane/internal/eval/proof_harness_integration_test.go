//go:build integration
// +build integration

package eval

import (
	"context"
	"database/sql"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"control-plane/internal/apiserver"
	"control-plane/internal/app"
	"control-plane/internal/migrate"

	_ "github.com/lib/pq"
)

func proofIntegrationConfigPath(t *testing.T) string {
	t.Helper()
	if p := strings.TrimSpace(os.Getenv("CONFIG")); p != "" {
		return p
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "configs", "config.example.yaml")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found for proof integration config")
		}
		dir = parent
	}
}

// TestProofHarnessREST_Postgres runs all proof-*.json scenarios against apiserver.NewRouter (REST only).
func TestProofHarnessREST_Postgres(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	pgdb, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open postgres for proof clean check: %v", err)
	}
	if err := pgdb.PingContext(context.Background()); err != nil {
		_ = pgdb.Close()
		t.Fatalf("ping postgres: %v", err)
	}
	if err := migrate.RequireProofHarnessCleanPostgres(context.Background(), pgdb); err != nil {
		_ = pgdb.Close()
		t.Fatal(err)
	}
	_ = pgdb.Close()

	cfg, err := app.LoadConfig(proofIntegrationConfigPath(t))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Postgres.DSN = dsn
	container, err := app.Boot(cfg)
	if err != nil {
		t.Fatalf("boot: %v", err)
	}
	defer container.DB.Close()

	rtr, err := apiserver.NewRouter(cfg, container)
	if err != nil {
		t.Fatalf("router: %v", err)
	}
	srv := httptest.NewServer(rtr)
	defer srv.Close()

	ctx := context.Background()
	// Two full REST passes in-process (same server, accumulating DB); pass/fail signature must match.
	rep, err := RunProofHarnessRESTDeterminism(ctx, srv.URL, nil)
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if !rep.AllPassed {
		for _, sc := range rep.Scenarios {
			if sc.AllPassed {
				continue
			}
			t.Logf("FAIL scenario %s suite=%s", sc.ScenarioID, sc.Suite)
			for _, st := range sc.Steps {
				if !st.Pass {
					t.Logf("  step %s %s: %s", st.StepID, st.Path, st.Detail)
				}
			}
			for _, iv := range sc.Invariants {
				if !iv.Pass {
					t.Logf("  invariant %s: %s", iv.Name, iv.Detail)
				}
			}
		}
		t.Fatal("proof harness: one or more scenarios failed (see [PROOF] logs)")
	}
	if !rep.DeterminismPass {
		t.Fatalf("determinism: %s", rep.DeterminismNote)
	}
}
