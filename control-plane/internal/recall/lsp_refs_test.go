package recall

import (
	"context"
	"testing"

	"control-plane/internal/tooling"
)

func TestFindSymbolPosition_nested(t *testing.T) {
	tree := []tooling.Symbol{
		{Name: "outer", Children: []tooling.Symbol{
			{Name: "inner", Range: &tooling.Range{StartLine: 3, StartCol: 1, EndLine: 3, EndCol: 5}},
		}},
	}
	line, col, ok := findSymbolPosition(tree, "inner")
	if !ok || line != 3 || col != 1 {
		t.Fatalf("got line=%d col=%d ok=%v", line, col, ok)
	}
}

func TestMaxReferenceCountForMatched(t *testing.T) {
	lsp := &tooling.FakeLSPClient{
		Symbols: []tooling.Symbol{
			{Name: "A", Range: &tooling.Range{StartLine: 0, StartCol: 0}},
		},
		RefLengths: []int{7},
	}
	n := maxReferenceCountForMatched(context.Background(), lsp, "/r", "f.go", []string{"A"}, 0)
	if n != 7 {
		t.Fatalf("got %d", n)
	}
}
