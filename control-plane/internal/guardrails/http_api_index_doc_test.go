package guardrails

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestHTTPAPIIndex_listsShippedV1Routes fails when router.go adds /v1 routes not mentioned in docs/http-api-index.md
// (or when the doc is renamed). Keeps the canonical index honest vs internal/apiserver/router.go.
func TestHTTPAPIIndex_listsShippedV1Routes(t *testing.T) {
	root := recallRepositoryRoot(t)
	idxPath := filepath.Join(root, "docs", "http-api-index.md")
	body := readFile(t, idxPath)

	required := []string{
		"`/v1/memories`",
		"`/v1/memories/search`",
		"`/v1/memory`",
		"`/v1/memory/promote`",
		"`/v1/curation/digest`",
		"`/v1/curation/pending`",
		"`/v1/curation/evaluate`",
		"`/v1/recall/`",
		"`/v1/recall/preflight`",
		"`/v1/recall/compile`",
		"`/v1/recall/compile-multi`",
		"`/v1/recall/run-multi`",
		"`/v1/enforcement/evaluate`",
		"`/v1/drift/check`",
		"`/v1/evidence`",
		"`/v1/contradictions`",
		"`/v1/ingest/cognition`",
		"`/v1/advisory-episodes`",
		"`POST` | `/v1/mcp`",
		"There are no `target_id` or `context_id` JSON fields",
	}
	for _, s := range required {
		if !strings.Contains(body, s) {
			t.Errorf("docs/http-api-index.md: missing expected substring %q (update index or router)", s)
		}
	}
}

// TestCurationLoopDocs_noPhantomDigestFields ensures active curation-loop.md does not revive removed wire fields.
func TestCurationLoopDocs_noPhantomDigestFields(t *testing.T) {
	root := recallRepositoryRoot(t)
	p := filepath.Join(root, "docs", "curation-loop.md")
	body := readFile(t, p)
	if strings.Contains(body, "`target_id`") || strings.Contains(body, "`context_id`") {
		t.Errorf("%s: must not document target_id/context_id on DigestRequest (not on wire)", p)
	}
}

// TestRouterGo_containsRoutesDocumentedInIndex — reverse check: index mentions routes that exist in router source.
func TestRouterGo_containsRoutesDocumentedInIndex(t *testing.T) {
	root := recallRepositoryRoot(t)
	routerPath := filepath.Join(root, "control-plane", "internal", "apiserver", "router.go")
	routerSrc := readFile(t, routerPath)
	idx := readFile(t, filepath.Join(root, "docs", "http-api-index.md"))

	// Chi route registrations use string literals; ensure key paths still appear in router.go.
	for _, path := range []string{
		`"/memories"`,
		`"/memory"`,
		`"/curation"`,
		`"/recall"`,
		`"/drift"`,
		`"/enforcement"`,
		`"/evidence"`,
		`"/contradictions"`,
		`"/ingest"`,
		`"/advisory-episodes"`,
	} {
		if !strings.Contains(routerSrc, path) {
			t.Errorf("router.go: expected route group %s present", path)
		}
	}
	if !strings.Contains(idx, "internal/apiserver/router.go") {
		t.Error("http-api-index.md should cite router.go as implementation source")
	}
}
