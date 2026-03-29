package tooling

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFlattenDocumentSymbols(t *testing.T) {
	// LSP documentSymbol result (DocumentSymbol[]) - flatten to our Symbol type
	raw := []byte(`[
		{"name":"Foo","detail":"","kind":12,"range":{"start":{"line":10,"character":0},"end":{"line":15,"character":0}},"selectionRange":{"start":{"line":10,"character":5},"end":{"line":10,"character":8}},"children":[]},
		{"name":"Bar","detail":"struct","kind":23,"range":{"start":{"line":20,"character":0},"end":{"line":30,"character":1}},"selectionRange":{"start":{"line":20,"character":0},"end":{"line":20,"character":3}},"children":[
			{"name":"F","detail":"","kind":8,"range":{"start":{"line":21,"character":1},"end":{"line":21,"character":10}},"selectionRange":{"start":{"line":21,"character":1},"end":{"line":21,"character":2}}}
		]}
	]`)
	var rawSymbols []documentSymbolRaw
	if err := json.Unmarshal(raw, &rawSymbols); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	symbols := flattenDocumentSymbols(rawSymbols, nil)
	if len(symbols) != 2 {
		t.Fatalf("expected 2 top-level symbols, got %d", len(symbols))
	}
	if symbols[0].Name != "Foo" || symbols[0].Kind != "function" {
		t.Errorf("first symbol: name=%q kind=%q", symbols[0].Name, symbols[0].Kind)
	}
	if symbols[0].Range == nil || symbols[0].Range.StartLine != 10 || symbols[0].Range.StartCol != 5 {
		t.Errorf("first symbol range: %+v", symbols[0].Range)
	}
	if symbols[1].Name != "Bar" || symbols[1].Kind != "struct" {
		t.Errorf("second symbol: name=%q kind=%q", symbols[1].Name, symbols[1].Kind)
	}
	if len(symbols[1].Children) != 1 {
		t.Errorf("Bar children: got %d", len(symbols[1].Children))
	}
}

func TestUriToPath(t *testing.T) {
	got, err := uriToPath("file:///home/proj/foo.go")
	if err != nil {
		t.Fatalf("uriToPath: %v", err)
	}
	if filepath.ToSlash(got) != "/home/proj/foo.go" {
		t.Errorf("uriToPath(file:///home/proj/foo.go) = %q", got)
	}
}

func TestLSPSymbols_integration(t *testing.T) {
	if _, err := exec.LookPath(DefaultLSPCommand); err != nil {
		t.Skipf("%s not in PATH: %v", DefaultLSPCommand, err)
	}
	root := "."
	path := "handlers.go"
	if _, err := os.Stat(filepath.Join(root, path)); err != nil {
		t.Skipf("handlers.go not found in %s: %v", root, err)
	}
	ctx := context.Background()
	symbols, err := LSPSymbols(ctx, root, path)
	if err != nil {
		t.Fatalf("LSPSymbols: %v", err)
	}
	if len(symbols) == 0 {
		t.Error("expected at least one symbol")
	}
	// Handlers has GitDiff, Ripgrep, etc.
	var found bool
	for _, s := range symbols {
		if s.Name == "GitDiff" || s.Name == "Ripgrep" || s.Name == "LSPSymbols" {
			found = true
			break
		}
	}
	if !found {
		t.Logf("symbols: %+v", symbols)
		t.Error("expected to find at least one handler method symbol")
	}
}

func TestLSPReferences_integration(t *testing.T) {
	if _, err := exec.LookPath(DefaultLSPCommand); err != nil {
		t.Skipf("%s not in PATH: %v", DefaultLSPCommand, err)
	}
	root := "."
	path := "types.go"
	if _, err := os.Stat(filepath.Join(root, path)); err != nil {
		t.Skipf("types.go not found: %v", err)
	}
	// Symbol "Symbol" type is around line 35 (0-based ~34). Use a position inside the type name.
	ctx := context.Background()
	refs, err := LSPReferences(ctx, root, path, 34, 2)
	if err != nil {
		t.Fatalf("LSPReferences: %v", err)
	}
	// May be 0 if no references, or 1+ if the type is referenced
	t.Logf("references: %d", len(refs))
	for i, loc := range refs {
		if loc.Path == "" || loc.Range == nil {
			t.Errorf("ref[%d]: empty path or range", i)
		}
	}
}

// TestFakeLSPClient_returnsConfiguredData verifies the test double returns fixed symbols and references (Task 100).
func TestFakeLSPClient_returnsConfiguredData(t *testing.T) {
	symbols := []Symbol{{Name: "Foo", Kind: "function"}}
	refs := []Reference{{Path: "a.go", Range: &Range{StartLine: 1}}}
	fake := &FakeLSPClient{Symbols: symbols, References: refs}
	ctx := context.Background()

	gotSym, err := fake.FindSymbols(ctx, "/root", "pkg/foo.go")
	if err != nil {
		t.Fatalf("FindSymbols: %v", err)
	}
	if len(gotSym) != 1 || gotSym[0].Name != "Foo" {
		t.Errorf("FindSymbols: got %v", gotSym)
	}

	gotRefs, err := fake.FindReferences(ctx, "/root", "pkg/foo.go", 10, 5)
	if err != nil {
		t.Fatalf("FindReferences: %v", err)
	}
	if len(gotRefs) != 1 || gotRefs[0].Path != "a.go" {
		t.Errorf("FindReferences: got %v", gotRefs)
	}
}
