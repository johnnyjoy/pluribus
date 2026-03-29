package guardrails

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestRESTTestMatrix_docListsCoreRoutes keeps docs/rest-test-matrix.md aligned with the REST-first hardening story.
func TestRESTTestMatrix_docListsCoreRoutes(t *testing.T) {
	root := recallRepositoryRoot(t)
	p := filepath.Join(root, "docs", "rest-test-matrix.md")
	body := readFile(t, p)

	required := []string{
		"POST /v1/memory",
		"POST /v1/memories",
		"POST /v1/recall/compile",
		"GET /v1/recall/",
		"POST /v1/recall/preflight",
		"POST /v1/recall/compile-multi",
		"POST /v1/enforcement/evaluate",
		"POST /v1/curation/digest",
		"TEST_PG_DSN",
		"TestREST_memoryCreate_rejectsContainerOntologyJSON",
		"project_id",
		"hive_id",
		"DisallowUnknownFields",
	}
	for _, s := range required {
		if !strings.Contains(body, s) {
			t.Errorf("docs/rest-test-matrix.md: missing expected substring %q", s)
		}
	}
}
