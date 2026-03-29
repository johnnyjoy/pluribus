package recall

import (
	"context"

	"control-plane/internal/tooling"
)

// findSymbolPosition returns the first DFS symbol with the given name and a range.
func findSymbolPosition(nodes []tooling.Symbol, name string) (line, col int, ok bool) {
	for _, n := range nodes {
		if n.Name == name && n.Range != nil {
			return n.Range.StartLine, n.Range.StartCol, true
		}
		if line, col, ok := findSymbolPosition(n.Children, name); ok {
			return line, col, true
		}
	}
	return 0, 0, false
}

// maxReferenceCountForMatched runs FindReferences per unique matched name and returns the max
// effective count (len(refs) capped by refLimit when refLimit > 0).
func maxReferenceCountForMatched(ctx context.Context, lsp tooling.LSPClient, repoRoot, focusPath string, matched []string, refLimit int) int {
	if lsp == nil || repoRoot == "" || focusPath == "" || len(matched) == 0 {
		return 0
	}
	syms, err := lsp.FindSymbols(ctx, repoRoot, focusPath)
	if err != nil || len(syms) == 0 {
		return 0
	}
	seen := make(map[string]bool)
	max := 0
	for _, name := range matched {
		if seen[name] {
			continue
		}
		seen[name] = true
		line, col, ok := findSymbolPosition(syms, name)
		if !ok {
			continue
		}
		refs, err := lsp.FindReferences(ctx, repoRoot, focusPath, line, col)
		if err != nil {
			continue
		}
		n := len(refs)
		if refLimit > 0 && n > refLimit {
			n = refLimit
		}
		if n > max {
			max = n
		}
	}
	return max
}
