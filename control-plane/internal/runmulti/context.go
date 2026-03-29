package runmulti

import (
	"strings"
)

// BuildContext constructs the full prompt from a recall bundle, variant name, and user prompt:
// [RECALL] ... [VARIANT] ... [INSTRUCTION] ...
func BuildContext(b *RecallBundleMirror, variant string, userPrompt string) string {
	var sections []string

	// Recalled memory block: constraints, decisions, failures, patterns
	var ctxLines []string
	if b != nil {
		ctxLines = append(ctxLines, sectionLines("Constraints", b.GoverningConstraints)...)
		ctxLines = append(ctxLines, sectionLines("Decisions", b.Decisions)...)
		ctxLines = append(ctxLines, sectionLines("Known failures", b.KnownFailures)...)
		ctxLines = append(ctxLines, sectionLines("Applicable patterns", b.ApplicablePatterns)...)
	}
	if len(ctxLines) > 0 {
		sections = append(sections, "[RECALL]\n"+strings.TrimSpace(strings.Join(ctxLines, "\n\n")))
	}

	// Variant block (for traceability)
	if variant != "" {
		sections = append(sections, "[VARIANT]\n"+variant)
	}

	// Instruction block: user prompt
	sections = append(sections, "[INSTRUCTION]\n"+strings.TrimSpace(userPrompt))

	return strings.Join(sections, "\n\n")
}

func sectionLines(title string, items []MemoryItemMirror) []string {
	if len(items) == 0 {
		return nil
	}
	lines := make([]string, 0, len(items))
	for _, m := range items {
		if m.Statement != "" {
			lines = append(lines, "- "+m.Statement)
		}
	}
	if len(lines) == 0 {
		return nil
	}
	return []string{title + ":", strings.Join(lines, "\n")}
}
