package mcp

// JSON-schema helpers for MCP tool inputSchema (subset validated by validateInputSchema).

func schemaObject(props map[string]any, required []string) map[string]any {
	s := map[string]any{
		"type":                 "object",
		"properties":           props,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

func schemaEmptyObject() map[string]any {
	return schemaObject(map[string]any{}, nil)
}

func propString(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func propInt(desc string, min, max int) map[string]any {
	p := map[string]any{"type": "integer", "description": desc}
	if min > 0 {
		p["minimum"] = min
	}
	if max > 0 {
		p["maximum"] = max
	}
	return p
}

func propStringArray(desc string) map[string]any {
	return map[string]any{
		"type":        "array",
		"description": desc,
		"items":       map[string]any{"type": "string"},
	}
}

func propUUID(desc string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": desc,
		"format":      "uuid",
	}
}

func propEnumString(desc string, values ...string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": desc,
		"enum":        values,
	}
}

func propBool(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}

func propAnyObject(desc string) map[string]any {
	return map[string]any{"type": "object", "description": desc}
}

func schemaTags() map[string]any {
	return propStringArray("Optional correlation tags (shape retrieval; not a memory partition).")
}

func schemaTaskTextProps() map[string]any {
	return map[string]any{
		"task":             propString("Primary task or situation text (preferred)."),
		"task_description": propString("Alias for task — raw text describing what you are doing."),
		"query":            propString("Alias for task text."),
		"retrieval_query":  propString("Alias for task text (REST field name)."),
	}
}

func schemaRepoRootProps() map[string]any {
	return map[string]any{
		"repo_root":      propString("Optional absolute workspace path; basename boosts situational affinity (e.g. pluribus)."),
		"workspace_root": propString("Alias for repo_root."),
		"project_root":   propString("Alias for repo_root."),
	}
}
