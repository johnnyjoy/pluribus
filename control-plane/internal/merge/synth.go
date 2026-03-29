package merge

import (
	"fmt"
	"sort"
	"strings"
)

// Synthesize builds the merged document from agreement and unique lines (conflicts excluded).
func Synthesize(agreements, uniques []string, usedVariants []string) string {
	agreements = sortedCopy(agreements)
	uniques = sortedCopy(uniques)

	var b strings.Builder
	b.WriteString("[CORE AGREEMENTS]\n")
	if len(agreements) == 0 {
		b.WriteString("(none)\n")
	} else {
		for _, a := range agreements {
			b.WriteString("- ")
			b.WriteString(strings.TrimSpace(a))
			b.WriteByte('\n')
		}
	}
	b.WriteByte('\n')
	b.WriteString("[VALID UNIQUE ADDITIONS]\n")
	if len(uniques) == 0 {
		b.WriteString("(none)\n")
	} else {
		for _, u := range uniques {
			b.WriteString("- ")
			b.WriteString(strings.TrimSpace(u))
			b.WriteByte('\n')
		}
	}
	b.WriteByte('\n')
	b.WriteString("[REFINED STRUCTURE]\n")
	b.WriteString(refinedSummary(len(agreements), len(uniques), usedVariants))
	return strings.TrimRight(b.String(), "\n")
}

func refinedSummary(nAgree, nUnique int, variants []string) string {
	vstr := strings.Join(variants, ", ")
	if vstr == "" {
		vstr = "none"
	}
	return fmt.Sprintf("Merged %d agreements and %d unique points from variants: %s.", nAgree, nUnique, vstr)
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		return Normalize(out[i]) < Normalize(out[j])
	})
	return out
}
