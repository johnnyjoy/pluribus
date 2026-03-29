package mcp

// InitializeResult builds the MCP initialize response (protocol + capabilities).
func InitializeResult(serverName, version string) map[string]any {
	return map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"tools":     map[string]any{},
			"prompts":   map[string]any{},
			"resources": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    serverName,
			"version": version,
		},
	}
}
