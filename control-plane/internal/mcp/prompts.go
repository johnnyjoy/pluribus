package mcp

import _ "embed"

//go:embed memory_grounding.md
var promptMemoryGroundingBody string

//go:embed pre_change.md
var promptPreChangeBody string

//go:embed curation_after_work.md
var promptCurationAfterWorkBody string

//go:embed canon_vs_advisory.md
var promptCanonVsAdvisoryBody string

// Prompt list names (MCP prompts/get uses name).
// Memory-first lifecycle: grounding + curation prompts use execution/situation language, not backlog-shaped ontology.
const (
	PromptMemoryGrounding = "pluribus_memory_grounding"
	PromptPreChange       = "pluribus_pre_change_enforcement"
	PromptMemoryCuration  = "pluribus_memory_curation"
	PromptCanonVsAdvisory = "pluribus_canon_vs_advisory"
)

// PromptDefinitions returns prompts/list entries (operational, phase-specific).
func PromptDefinitions() []map[string]any {
	return []map[string]any{
		{
			"name": PromptMemoryGrounding,
			"description": "Memory grounding: recall before substantive work; act from governing memory (constraints/decisions/failures/patterns); never guess missing context. " +
				"Surface " + SurfaceVersion + ". Doctrine: Recall repo docs/memory-doctrine.md; memory-first; no container partition assumptions.",
		},
		{
			"name": PromptPreChange,
			"description": "Pre-change: enforcement_evaluate for risky proposals (binding memory vs proposal); skip gate for trivial edits. Not recall-only or digest. Surface " + SurfaceVersion + ". Doctrine: docs/memory-doctrine.md; no container assumptions.",
		},
		{
			"name": PromptMemoryCuration,
			"description": "Memory curation: curation_digest only after meaningful work; candidates are not canon; materialize validated rows only. Surface " + SurfaceVersion + ". Doctrine: docs/memory-doctrine.md; no container assumptions.",
		},
		{
			"name": PromptCanonVsAdvisory,
			"description": "Authority classes: governing vs advisory vs digest candidates vs evidence vs transcript. Surface " + SurfaceVersion + ". Doctrine: docs/memory-doctrine.md; no container assumptions.",
		},
	}
}

// PromptMessages returns MCP-style messages for prompts/get (single user message with steps).
func PromptMessages(name string) ([]map[string]any, bool) {
	switch name {
	case PromptMemoryGrounding:
		return []map[string]any{
			{"role": "user", "content": promptMemoryGroundingBody},
		}, true
	case PromptPreChange:
		return []map[string]any{
			{"role": "user", "content": promptPreChangeBody},
		}, true
	case PromptMemoryCuration:
		return []map[string]any{
			{"role": "user", "content": promptCurationAfterWorkBody},
		}, true
	case PromptCanonVsAdvisory:
		return []map[string]any{
			{"role": "user", "content": promptCanonVsAdvisoryBody},
		}, true
	default:
		return nil, false
	}
}
