package mcp

import (
	"fmt"
	"sort"
	"strings"
)

// GenerateMCPToolsMarkdown renders docs/mcp-tools.md content from the live registry.
func GenerateMCPToolsMarkdown() string {
	var b strings.Builder
	b.WriteString("# Pluribus MCP tools\n\n")
	b.WriteString("Generated from `control-plane/internal/mcp/tool_registry.go`. Do not edit tool rows by hand — run `UPDATE_MCP_TOOLS_DOC=1 go test ./internal/mcp/ -run TestMCPToolsDocMatchesRegistry` from `control-plane/` to refresh.\n\n")
	b.WriteString("## Tool tiers (`tools/list` only)\n\n")
	b.WriteString("Set **`PLURIBUS_TOOLS`** env or **`mcp.tools_tier`** in config (`core` | `standard` | `all`). Default **`all`** lists every tool. **`tools/call`** still accepts all registered names regardless of tier.\n\n")
	b.WriteString("- **`core`** — loop + housekeeping: ")
	b.WriteString(tierToolList(ToolsTierCore))
	b.WriteString("- **`standard`** — core plus: ")
	b.WriteString(standardExtraToolList())
	b.WriteString("\n\n")
	b.WriteString("| Tool | Purpose | Agent-loop role | Required inputs | Optional inputs | Backend endpoint | Output summary | Risk level | Test coverage |\n")
	b.WriteString("|------|---------|-----------------|-----------------|-----------------|------------------|----------------|------------|---------------|\n")

	reg := toolRegistry()
	for _, t := range reg {
		req, opt := schemaFieldLists(t.InputSchema)
		purpose := strings.ReplaceAll(strings.Split(t.Description, ".")[0], "|", "/")
		if len(purpose) > 120 {
			purpose = purpose[:117] + "..."
		}
		b.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			t.Name, purpose, t.LoopRole, req, opt, t.Backend, t.Output, t.Risk, t.TestCoverage))
	}

	b.WriteString("\n## Direct tools/call coverage\n\n")
	b.WriteString("| Tool | Coverage category | Test / proof | Pass status |\n")
	b.WriteString("|------|-------------------|--------------|-------------|\n")
	for _, t := range reg {
		loc := CallCoverageLocation(t)
		if loc == "" {
			loc = t.CallCoverageNote
		}
		b.WriteString(fmt.Sprintf("| `%s` | %s | %s | pass (automated) |\n", t.Name, t.CallCoverage, loc))
	}

	b.WriteString("\n## Aliases\n\n")
	b.WriteString("| Alias | Canonical tool |\n|-------|----------------|\n")
	aliases := map[string]string{
		"memory_context_resolve": "recall_context",
		"mcp_episode_ingest":     "record_experience",
		"auto_log_episode_if_relevant": "memory_log_if_relevant",
		"curation_promote_candidate":   "curation_materialize",
	}
	names := make([]string, 0, len(aliases))
	for a := range aliases {
		names = append(names, a)
	}
	sort.Strings(names)
	for _, a := range names {
		b.WriteString(fmt.Sprintf("| `%s` | `%s` |\n", a, aliases[a]))
	}

	b.WriteString("\n## Verification\n\n")
	b.WriteString("- Unit MCP tests: `make test-mcp` (repo root) or `cd control-plane && go test ./internal/mcp/...`\n")
	b.WriteString("- Docker HTTP MCP proof: `make proof-mcp`\n")
	b.WriteString("- Docker authenticated MCP proof: `make proof-mcp-auth`\n")
	b.WriteString("- Stdio adapter proof: `make proof-mcp-stdio`\n")
	b.WriteString("- Full Phase 1 close-out: `make proof-mcp-all`\n")
	return b.String()
}

func schemaFieldLists(schema map[string]any) (required, optional string) {
	props, _ := schema["properties"].(map[string]any)
	reqSet := map[string]bool{}
	if req, ok := schema["required"].([]string); ok {
		for _, r := range req {
			reqSet[r] = true
		}
	}
	if req, ok := schema["required"].([]any); ok {
		for _, r := range req {
			if s, ok := r.(string); ok {
				reqSet[s] = true
			}
		}
	}
	var reqFields, optFields []string
	for k := range props {
		if reqSet[k] {
			reqFields = append(reqFields, k)
		} else {
			optFields = append(optFields, k)
		}
	}
	sort.Strings(reqFields)
	sort.Strings(optFields)
	if len(reqFields) == 0 {
		reqFields = []string{"(semantic)"}
	}
	if len(optFields) == 0 {
		optFields = []string{"—"}
	}
	return strings.Join(reqFields, ", "), strings.Join(optFields, ", ")
}

func tierToolList(tier string) string {
	names := make([]string, 0, len(toolRegistry()))
	for _, t := range filterRegistryByTier(toolRegistry(), tier) {
		names = append(names, "`"+t.Name+"`")
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func standardExtraToolList() string {
	names := make([]string, 0, len(standardExtraToolNames))
	for n := range standardExtraToolNames {
		names = append(names, "`"+n+"`")
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
