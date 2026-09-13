package recall

import "strings"

// AgentGrounding is a plain-text view of grouped recall for agent consumption (system-shaped output).
// Populated from Continuity / Constraints / Experience after the score floor. Empty stays empty.
type AgentGrounding struct {
	Continuity  string `json:"continuity"`
	Constraints string `json:"constraints"`
	Experience  string `json:"experience"`
	// Formatted is continuity + constraints + experience in one block (for prompts and MCP wrapping).
	Formatted string `json:"formatted"`
}

// populateAgentGrounding fills b.AgentGrounding from grouped slices only.
// Empty sections stay empty — do not refill from legacy buckets after the score floor.
func populateAgentGrounding(b *RecallBundle) {
	if b == nil {
		return
	}
	cont := b.Continuity
	cons := b.Constraints
	exp := b.Experience

	g := &AgentGrounding{
		Continuity:  formatGroundingSection("Continuity", cont),
		Constraints: formatGroundingSection("Constraints", cons),
		Experience:  formatGroundingSection("Experience", exp),
	}
	var fb strings.Builder
	fb.WriteString(g.Continuity)
	fb.WriteString("\n")
	fb.WriteString(g.Constraints)
	fb.WriteString("\n")
	fb.WriteString(g.Experience)
	g.Formatted = strings.TrimSpace(fb.String())
	b.AgentGrounding = g
}

func formatGroundingSection(title string, items []MemoryItem) string {
	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString(":\n")
	if len(items) == 0 {
		sb.WriteString("(none)\n")
		return sb.String()
	}
	for _, it := range items {
		line := strings.TrimSpace(it.Statement)
		if line == "" {
			continue
		}
		sb.WriteString("- ")
		if id := strings.TrimSpace(it.ID); id != "" {
			sb.WriteByte('[')
			sb.WriteString(id)
			sb.WriteString("] ")
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	return sb.String()
}
