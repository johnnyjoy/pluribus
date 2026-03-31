package similarity

import "strings"

func mcpSessionFromTags(tags []string) string {
	const p = "mcp:session:"
	for _, t := range tags {
		if strings.HasPrefix(t, p) {
			return strings.TrimPrefix(t, p)
		}
	}
	return ""
}

// mcpDedupCorrelationMatch is true when the request correlation id matches the row's mcp:session tag (both empty counts as matching).
func mcpDedupCorrelationMatch(correlationID string, rowTags []string) bool {
	want := strings.TrimSpace(correlationID)
	got := mcpSessionFromTags(rowTags)
	if want == "" {
		return got == ""
	}
	return got == want
}
