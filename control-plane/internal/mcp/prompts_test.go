package mcp

import (
	"strings"
	"testing"
)

func TestPromptMemoryGrounding_body(t *testing.T) {
	msg, ok := PromptMessages(PromptMemoryGrounding)
	if !ok || len(msg) != 1 {
		t.Fatalf("PromptMessages: ok=%v len=%d", ok, len(msg))
	}
	c, _ := msg[0]["content"].(string)
	if !strings.Contains(c, "Pluribus memory-grounding protocol") {
		t.Fatalf("expected protocol title in embedded memory_grounding.md")
	}
	if !strings.Contains(c, "global and tag-shaped") {
		t.Fatalf("expected global/tag-shaped line in memory-grounding body")
	}
	if !strings.Contains(c, "pluribus://architecture/active-context-vs-durable-store") {
		t.Fatalf("expected architecture resource URI in body")
	}
	if !strings.Contains(c, "recall_get") {
		t.Fatalf("expected default-behavior line")
	}
}

func TestResourceText_canonVsAdvisoryAliases(t *testing.T) {
	a, ok := ResourceText("pluribus://discipline/canon-vs-advisory")
	if !ok {
		t.Fatal("canon-vs-advisory")
	}
	b, ok := ResourceText("pluribus://discipline/canon-advisory")
	if !ok {
		t.Fatal("alternate URI canon-advisory alias")
	}
	if a != b {
		t.Fatal("alias mismatch")
	}
}

func TestResourceText_architectureAliases(t *testing.T) {
	a, ok := ResourceText(URIArchitectureActiveVsDurable)
	if !ok {
		t.Fatal("architecture primary URI")
	}
	b, ok := ResourceText("pluribus://discipline/architecture-notes")
	if !ok {
		t.Fatal("alternate URI architecture-notes alias")
	}
	if a != b {
		t.Fatal("alias mismatch")
	}
}

func TestSurfaceVersion_nonEmpty(t *testing.T) {
	if strings.TrimSpace(SurfaceVersion) == "" {
		t.Fatal("SurfaceVersion")
	}
}

func TestPromptDefinitions_includeSurfaceVersion(t *testing.T) {
	for _, p := range PromptDefinitions() {
		desc, _ := p["description"].(string)
		if !strings.Contains(desc, SurfaceVersion) {
			t.Fatalf("prompt description missing SurfaceVersion: %q", desc)
		}
	}
}

func TestPromptDefinitions_memoryFirstNames(t *testing.T) {
	want := []string{PromptMemoryGrounding, PromptPreChange, PromptMemoryCuration, PromptCanonVsAdvisory}
	seen := make(map[string]struct{})
	for _, p := range PromptDefinitions() {
		n, _ := p["name"].(string)
		seen[n] = struct{}{}
	}
	for _, n := range want {
		if _, ok := seen[n]; !ok {
			t.Fatalf("missing prompt name %q in PromptDefinitions", n)
		}
	}
	if _, bad := seen["pluribus_task_start"]; bad {
		t.Fatal("legacy pluribus_task_start must not appear in definitions")
	}
	if _, bad := seen["pluribus_post_task_curation"]; bad {
		t.Fatal("legacy pluribus_post_task_curation must not appear in definitions")
	}
}

func TestResourceDefinitions_fiveCanonicalWithSurfaceVersion(t *testing.T) {
	rs := ResourceDefinitions()
	if len(rs) != 5 {
		t.Fatalf("want 5 resources, got %d", len(rs))
	}
	seen := make(map[string]struct{})
	for _, r := range rs {
		uri, _ := r["uri"].(string)
		seen[uri] = struct{}{}
		desc, _ := r["description"].(string)
		if !strings.Contains(desc, SurfaceVersion) {
			t.Fatalf("resource %s description missing SurfaceVersion", uri)
		}
	}
	want := []string{
		URIDisciplineDoctrine,
		URIDisciplineLifecycle,
		URIDisciplineCanonAdvisory,
		URIArchitectureActiveVsDurable,
		URIDisciplineHistoryNotMemory,
	}
	for _, u := range want {
		if _, ok := seen[u]; !ok {
			t.Fatalf("missing canonical URI %q", u)
		}
	}
}

func TestResourceText_doctrineStatesGlobalMemory(t *testing.T) {
	body, ok := ResourceText(URIDisciplineDoctrine)
	if !ok {
		t.Fatal("doctrine resource")
	}
	for _, needle := range []string{
		"global memory system",
		"retrieval_query",
		"memory-doctrine.md",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("doctrine body missing %q", needle)
		}
	}
}

func TestResourceText_primaryBodiesNonEmpty(t *testing.T) {
	uris := []string{
		URIDisciplineDoctrine,
		URIDisciplineLifecycle,
		URIDisciplineCanonAdvisory,
		URIDisciplineHistoryNotMemory,
		URIArchitectureActiveVsDurable,
	}
	for _, u := range uris {
		body, ok := ResourceText(u)
		if !ok || len(strings.TrimSpace(body)) < 20 {
			t.Fatalf("ResourceText(%q): ok=%v len=%d", u, ok, len(body))
		}
	}
}

func TestPromptMemoryGrounding_lifecycleMnemonicAndResources(t *testing.T) {
	msg, ok := PromptMessages(PromptMemoryGrounding)
	if !ok || len(msg) != 1 {
		t.Fatalf("PromptMessages memory_grounding")
	}
	c, _ := msg[0]["content"].(string)
	if !strings.Contains(c, "Recall → Act → Validate → Update → Repeat") {
		t.Fatal("expected locked lifecycle mnemonic substring")
	}
	if !strings.Contains(c, "pluribus://discipline/history-not-memory") {
		t.Fatal("expected history-not-memory URI in resource list")
	}
}

func TestPromptPreChange_enforcementGate(t *testing.T) {
	msg, ok := PromptMessages(PromptPreChange)
	if !ok || len(msg) != 1 {
		t.Fatal("pre_change")
	}
	c, _ := msg[0]["content"].(string)
	if !strings.Contains(c, "enforcement_evaluate") {
		t.Fatal("expected enforcement_evaluate in pre_change body")
	}
}

func TestPromptMemoryCuration_candidatesNotCanon(t *testing.T) {
	msg, ok := PromptMessages(PromptMemoryCuration)
	if !ok || len(msg) != 1 {
		t.Fatal("memory_curation")
	}
	c, _ := msg[0]["content"].(string)
	if !strings.Contains(c, "Candidates are not canon") {
		t.Fatal("expected candidate vs canon line")
	}
}

func TestPromptCanonVsAdvisory_authorityOnly(t *testing.T) {
	msg, ok := PromptMessages(PromptCanonVsAdvisory)
	if !ok || len(msg) != 1 {
		t.Fatal("canon_vs_advisory")
	}
	c, _ := msg[0]["content"].(string)
	if !strings.Contains(c, "pluribus://discipline/canon-vs-advisory") {
		t.Fatal("expected resource pointer")
	}
}

func TestAllEmbeddedPrompts_includeMCPDoctrineFooter(t *testing.T) {
	needles := []string{
		"## Pluribus doctrine (MCP)",
		"Do not assume any project, task, or container.",
		"This system is memory-first.",
	}
	for _, name := range []string{PromptMemoryGrounding, PromptPreChange, PromptMemoryCuration, PromptCanonVsAdvisory} {
		msg, ok := PromptMessages(name)
		if !ok || len(msg) != 1 {
			t.Fatalf("prompt %q", name)
		}
		c, _ := msg[0]["content"].(string)
		for _, n := range needles {
			if !strings.Contains(c, n) {
				t.Fatalf("prompt %q missing %q", name, n)
			}
		}
	}
}
