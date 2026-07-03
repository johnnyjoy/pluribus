package mcp

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSchemaQuality_noBareObjectSchemas(t *testing.T) {
	if issues := SchemaQualityIssues(); len(issues) > 0 {
		t.Fatalf("schema quality issues:\n- %s", strings.Join(issues, "\n- "))
	}
}

func TestSchemaQuality_allToolsRegistered(t *testing.T) {
	defs := ToolDefinitions()
	if len(defs) != len(toolRegistry()) {
		t.Fatalf("ToolDefinitions count %d != registry %d", len(defs), len(toolRegistry()))
	}
	seen := map[string]bool{}
	for _, d := range defs {
		name, _ := d["name"].(string)
		if name == "" {
			t.Fatal("tool missing name")
		}
		if seen[name] {
			t.Fatalf("duplicate tool name %s", name)
		}
		seen[name] = true
		desc, _ := d["description"].(string)
		if strings.TrimSpace(desc) == "" {
			t.Fatalf("tool %s missing description", name)
		}
		schema, ok := d["inputSchema"].(map[string]any)
		if !ok || schema == nil {
			t.Fatalf("tool %s missing inputSchema", name)
		}
	}
}

func TestValidateToolArguments_recallRequiresTask(t *testing.T) {
	err := ValidateToolArguments("recall_context", []byte(`{"tags":["a"]}`))
	if err == nil || !strings.Contains(err.Error(), "task") {
		t.Fatalf("expected task required error, got %v", err)
	}
}

func TestValidateToolArguments_recordRequiresSummary(t *testing.T) {
	err := ValidateToolArguments("record_experience", []byte(`{}`))
	if err == nil || !strings.Contains(err.Error(), "summary") {
		t.Fatalf("expected summary required, got %v", err)
	}
}

func TestValidateToolArguments_toleratesExtraProperties(t *testing.T) {
	// Unknown arguments are tolerated (dropped at forwarding), never rejected:
	// clients attach metadata like agent_id/repo_root to every call (H2).
	if err := ValidateToolArguments("health", []byte(`{"unexpected":true}`)); err != nil {
		t.Fatalf("extra properties must be tolerated, got %v", err)
	}
	if err := ValidateToolArguments("record_experience", []byte(`{"summary":"Fixed the flaky retry loop in ingest by bounding backoff and adding jitter; retries now converge.","agent_id":"cursor:fable","some_future_field":1}`)); err != nil {
		t.Fatalf("metadata arguments must be tolerated, got %v", err)
	}
}

func TestFilterArgumentsToSchema_dropsUndeclaredKeys(t *testing.T) {
	out := FilterArgumentsToSchema("memory_create", []byte(`{"kind":"decision","statement":"Use pgvector for ANN.","agent_id":"a1","stray":true}`))
	s := string(out)
	if strings.Contains(s, "stray") {
		t.Fatalf("undeclared keys not dropped: %s", s)
	}
	// agent_id is declared on memory_create (Phase 3 attribution) and must survive.
	if !strings.Contains(s, "statement") || !strings.Contains(s, "kind") || !strings.Contains(s, "agent_id") {
		t.Fatalf("declared keys lost: %s", s)
	}
}

func TestValidateToolArguments_enforcementRequiresProposal(t *testing.T) {
	err := ValidateToolArguments("enforcement_evaluate", []byte(`{"intent":"change"}`))
	if err == nil || !strings.Contains(err.Error(), "proposal_text") {
		t.Fatalf("expected proposal_text required, got %v", err)
	}
}

func TestValidateToolArguments_unknownTool(t *testing.T) {
	err := ValidateToolArguments("not_a_real_tool", []byte(`{}`))
	if err == nil || !strings.Contains(err.Error(), "unknown tool") {
		t.Fatalf("expected unknown tool, got %v", err)
	}
}

func TestSchemaQuality_allToolsHaveCallCoverage(t *testing.T) {
	if issues := CallCoverageIssues(); len(issues) > 0 {
		t.Fatalf("call coverage issues:\n- %s", strings.Join(issues, "\n- "))
	}
}

func TestMCPToolsDocMatchesRegistry(t *testing.T) {
	repoRoot := findRepoRoot(t)
	docPath := filepath.Join(repoRoot, "docs", "mcp-tools.md")
	generated := GenerateMCPToolsMarkdown()
	if os.Getenv("UPDATE_MCP_TOOLS_DOC") == "1" {
		if err := os.WriteFile(docPath, []byte(generated), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Log("updated", docPath)
		return
	}
	onDisk, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read %s: %v (run UPDATE_MCP_TOOLS_DOC=1 go test ./internal/mcp/ -run TestMCPToolsDocMatchesRegistry)", docPath, err)
	}
	if string(onDisk) != generated {
		t.Fatalf("docs/mcp-tools.md drift: run UPDATE_MCP_TOOLS_DOC=1 go test ./internal/mcp/ -run TestMCPToolsDocMatchesRegistry -v")
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err2 := os.Stat(filepath.Join(dir, "..", "docs")); err2 == nil {
				return filepath.Join(dir, "..")
			}
		}
		if _, err := os.Stat(filepath.Join(dir, "control-plane", "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not find repo root")
	return ""
}
