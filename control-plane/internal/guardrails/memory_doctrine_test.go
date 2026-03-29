package guardrails

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"control-plane/internal/enforcement"
	"control-plane/internal/memory"
	"control-plane/internal/recall"
)

// Guardrails: fail CI if container ontology vocabulary or wire examples creep back into
// normative onboarding paths. Exempt docs/memory-doctrine.md and docs/anti-regression.md
// (they define the ban list). MCP *.md may contain the word "project" only inside the
// required doctrine footer ("any project, task, or container").

var bannedWireSubstrings = []string{
	`hive_id`,
	`/v1/hives`,
	`PROJECT_UUID`,
}

// Omit standalone \bscope\b: legitimate doc paths contain "-scope" (e.g. release-scope).
var bannedOntologyWords = regexp.MustCompile(`(?i)\b(workspace|hive)\b`)

var bannedProjectWord = regexp.MustCompile(`(?i)\bproject\b`)

func controlPlaneModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		b, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && strings.Contains(string(b), "module control-plane") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("control-plane go.mod not found from cwd")
		}
		dir = parent
	}
}

func recallRepositoryRoot(t *testing.T) string {
	t.Helper()
	cp := controlPlaneModuleRoot(t)
	root := filepath.Clean(filepath.Join(cp, ".."))
	st := filepath.Join(root, "docs", "memory-doctrine.md")
	if _, err := os.Stat(st); err != nil {
		t.Fatalf("expected %s (Recall repo root detection)", st)
	}
	return root
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// wireBanExemptDocs may name forbidden wire tokens when documenting the ban list (not as endorsed examples).
var wireBanExemptDocs = map[string]bool{
	"docs/memory-doctrine.md":    true,
	"docs/anti-regression.md":    true,
	"docs/rest-test-matrix.md":   true,
}

func assertNoBannedContent(t *testing.T, relPath, content string, allowProjectMention bool) {
	t.Helper()
	if !wireBanExemptDocs[relPath] {
		for _, s := range bannedWireSubstrings {
			if strings.Contains(content, s) {
				t.Errorf("%s: forbidden wire substring %q", relPath, s)
			}
		}
	}
	if bannedOntologyWords.MatchString(content) {
		t.Errorf("%s: forbidden ontology word (workspace|hive|scope)", relPath)
	}
	if !allowProjectMention && bannedProjectWord.MatchString(content) {
		t.Errorf("%s: forbidden word project", relPath)
	}
}

func TestMemoryDoctrine_onboardingMarkdownAndREADME(t *testing.T) {
	root := recallRepositoryRoot(t)
	paths := []string{
		filepath.Join(root, "README.md"),
		filepath.Join(root, "CONTRIBUTING.md"),
		filepath.Join(root, "docs", "architecture.md"),
		filepath.Join(root, "docs", "pluribus-quickstart.md"),
		filepath.Join(root, "docs", "mcp-usage.md"),
		filepath.Join(root, "docs", "http-api-index.md"),
		filepath.Join(root, "docs", "rest-test-matrix.md"),
		filepath.Join(root, "docs", "curation-loop.md"),
		filepath.Join(root, "docs", "control-plane-design-and-starter.md"),
		filepath.Join(root, "control-plane", "README.md"),
	}
	for _, p := range paths {
		rel, _ := filepath.Rel(root, p)
		assertNoBannedContent(t, rel, readFile(t, p), false)
	}
}

func TestMemoryDoctrine_mcpPromptMarkdown(t *testing.T) {
	cp := controlPlaneModuleRoot(t)
	dir := filepath.Join(cp, "internal", "mcp")
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	needles := []string{
		"## Pluribus doctrine (MCP)",
		"Do not assume any project, task, or container.",
		"This system is memory-first.",
	}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		body := readFile(t, p)
		for _, n := range needles {
			if !strings.Contains(body, n) {
				t.Errorf("%s: missing required doctrine fragment %q", e.Name(), n)
			}
		}
		// Footer is allowed to say "project"; still forbid wire + workspace/hive/scope outside doctrine.
		assertNoBannedContent(t, "internal/mcp/"+e.Name(), body, true)
	}
}

func TestMemoryDoctrine_promptsGoToolDescriptions(t *testing.T) {
	// prompts.go is vetted by compilation; scan source for accidental wire strings.
	cp := controlPlaneModuleRoot(t)
	src := readFile(t, filepath.Join(cp, "internal", "mcp", "prompts.go"))
	assertNoBannedContent(t, "internal/mcp/prompts.go", src, false)
}

func TestMemoryDoctrine_toolsGoDescriptions(t *testing.T) {
	cp := controlPlaneModuleRoot(t)
	src := readFile(t, filepath.Join(cp, "internal", "mcp", "tools.go"))
	assertNoBannedContent(t, "internal/mcp/tools.go", src, false)
}

func TestMemoryDoctrine_resourcesGoEmbeddedMarkdown(t *testing.T) {
	cp := controlPlaneModuleRoot(t)
	src := readFile(t, filepath.Join(cp, "internal", "mcp", "resources.go"))
	assertNoBannedContent(t, "internal/mcp/resources.go", src, false)
}

func TestMemoryDoctrine_quickstartsRecallExampleShape(t *testing.T) {
	root := recallRepositoryRoot(t)
	for _, doc := range []string{"docs/pluribus-quickstart.md"} {
		p := filepath.Join(root, doc)
		body := readFile(t, p)
		if !strings.Contains(body, "retrieval_query") || !strings.Contains(body, `"tags"`) {
			t.Errorf("%s: expected memory-first compile example with retrieval_query and tags", doc)
		}
	}
}

var forbiddenJSONNameParts = []string{
	"hive_id", "project_id", "workspace_id", "task_id",
}

func collectJSONFieldNames(t *testing.T, typ reflect.Type, seen map[reflect.Type]struct{}, out *[]string) {
	t.Helper()
	if typ == nil {
		return
	}
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return
	}
	if _, ok := seen[typ]; ok {
		return
	}
	seen[typ] = struct{}{}
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		tag := f.Tag.Get("json")
		if tag != "" && tag != "-" {
			name := strings.Split(tag, ",")[0]
			if name != "" {
				*out = append(*out, name)
			}
		}
		collectJSONFieldNames(t, f.Type, seen, out)
	}
}

func TestMemoryDoctrine_coreRequestJSONTags(t *testing.T) {
	types := []reflect.Type{
		reflect.TypeOf(recall.CompileRequest{}),
		reflect.TypeOf(recall.RunMultiRequest{}),
		reflect.TypeOf(enforcement.EvaluateRequest{}),
		reflect.TypeOf(memory.MemoriesCreateRequest{}),
		reflect.TypeOf(memory.PromoteRequest{}),
	}
	for _, typ := range types {
		var names []string
		collectJSONFieldNames(t, typ, map[reflect.Type]struct{}{}, &names)
		for _, n := range names {
			for _, b := range forbiddenJSONNameParts {
				if n == b || strings.Contains(n, b) {
					t.Errorf("%s: json field %q must not contain %q", typ.String(), n, b)
				}
			}
		}
	}
}
