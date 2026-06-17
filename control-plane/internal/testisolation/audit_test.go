package testisolation_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"control-plane/internal/testisolation"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	dir := filepath.Dir(file)
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "control-plane", "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("repo root not found")
	return ""
}

func TestCodebaseIsolationAudit(t *testing.T) {
	root := repoRoot(t)
	report, err := testisolation.AuditRepo(root, testisolation.DefaultAllowlist())
	if err != nil {
		t.Fatal(err)
	}
	if report.Metrics.LocalCodebaseTestPathsCount == 0 {
		t.Fatal("expected local codebase test patterns")
	}
}

func TestCodebaseIsolationGate(t *testing.T) {
	if os.Getenv("CODEBASE_ISOLATION_GATE") != "1" && os.Getenv("PROOF_CODEBASE_ISOLATION") != "1" {
		t.Skip("set CODEBASE_ISOLATION_GATE=1")
	}
	root := repoRoot(t)
	report, err := testisolation.AuditRepo(root, testisolation.DefaultAllowlist())
	if err != nil {
		t.Fatal(err)
	}
	artifact := isolationArtifactPath(root)
	if err := testisolation.WriteReport(&report, artifact); err != nil {
		t.Fatal(err)
	}
	if !report.GatePassed {
		for _, f := range report.Findings {
			if !f.Allowed {
				t.Logf("violation: %s:%d %s %s", f.File, f.Line, f.Rule, f.Snippet)
			}
		}
		t.Fatalf("isolation gate failed: %+v", report.Metrics)
	}
}

func TestNoCursorMCPConfigInTests(t *testing.T) {
	root := repoRoot(t)
	report, err := testisolation.AuditRepo(root, testisolation.DefaultAllowlist())
	if err != nil {
		t.Fatal(err)
	}
	if report.Metrics.CursorMCPConfigDependencyCount != 0 {
		t.Fatalf("cursor mcp config deps: %d", report.Metrics.CursorMCPConfigDependencyCount)
	}
}

func TestNoGlobalPluribusMCPBinaryLookup(t *testing.T) {
	root := repoRoot(t)
	report, err := testisolation.AuditRepo(root, testisolation.DefaultAllowlist())
	if err != nil {
		t.Fatal(err)
	}
	if report.Metrics.GlobalBinaryDependencyCount != 0 {
		t.Fatalf("global binary deps: %d", report.Metrics.GlobalBinaryDependencyCount)
	}
}

func isolationArtifactPath(root string) string {
	if p := os.Getenv("CODEBASE_TEST_ISOLATION_ARTIFACT"); p != "" {
		return p
	}
	return filepath.Join(root, "artifacts", "codebase-test-isolation.json")
}
