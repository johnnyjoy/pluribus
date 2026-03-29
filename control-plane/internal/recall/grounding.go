package recall

import "strings"

// AgentGrounding is a plain-text view of grouped recall for agent consumption (system-shaped output).
// Populated from Continuity / Constraints / Experience slices when non-empty; otherwise derived from legacy buckets.
type AgentGrounding struct {
	Continuity  string `json:"continuity"`
	Constraints string `json:"constraints"`
	Experience  string `json:"experience"`
	// Formatted is continuity + constraints + experience in one block (for prompts and MCP wrapping).
	Formatted string `json:"formatted"`
}

var (
	kindsContinuity  = []string{"state", "decision"}
	kindsConstraints = []string{"constraint", "failure"}
	kindsExperience    = []string{"pattern"}
)

// populateAgentGrounding fills b.AgentGrounding from grouped slices, with fallback to bucket lists.
func populateAgentGrounding(b *RecallBundle) {
	if b == nil {
		return
	}
	cont := b.Continuity
	if len(cont) == 0 {
		cont = itemsFromBucketsByKinds(b, kindsContinuity)
	}
	cons := b.Constraints
	if len(cons) == 0 {
		cons = itemsFromBucketsByKinds(b, kindsConstraints)
	}
	exp := b.Experience
	if len(exp) == 0 {
		exp = itemsFromBucketsByKinds(b, kindsExperience)
	}

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
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	return sb.String()
}

func itemsFromBucketsByKinds(b *RecallBundle, kinds []string) []MemoryItem {
	want := make(map[string]struct{}, len(kinds))
	for _, k := range kinds {
		want[k] = struct{}{}
	}
	var out []MemoryItem
	take := func(items []MemoryItem) {
		for _, it := range items {
			if _, ok := want[it.Kind]; ok {
				out = append(out, it)
			}
		}
	}
	take(b.GoverningConstraints)
	take(b.Decisions)
	take(b.KnownFailures)
	take(b.ApplicablePatterns)
	return out
}
