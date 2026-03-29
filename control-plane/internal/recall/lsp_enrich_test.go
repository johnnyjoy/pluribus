package recall

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/internal/tooling"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

func TestFlattenLSPNames_uniqueAndCap(t *testing.T) {
	syms := []tooling.Symbol{
		{Name: "A", Children: []tooling.Symbol{{Name: "B"}, {Name: "A"}}},
		{Name: "C"},
	}
	got := flattenLSPNames(syms, 2)
	if len(got) != 2 || got[0] != "A" || got[1] != "B" {
		t.Fatalf("got %v", got)
	}
}

func TestService_enrichCompileSymbols_fillsFromLSP(t *testing.T) {
	fake := &tooling.FakeLSPClient{
		Symbols: []tooling.Symbol{{Name: "Alpha"}, {Name: "Beta"}},
	}
	svc := &Service{LSP: fake, LSPEnabled: true, LSPAutoSymbolMax: 10}
	req := &CompileRequest{RepoRoot: "/tmp", LSPFocusPath: "x.go"}
	svc.enrichCompileSymbols(context.Background(), req)
	if len(req.Symbols) != 2 || req.Symbols[0] != "Alpha" || req.Symbols[1] != "Beta" {
		t.Fatalf("symbols %+v", req.Symbols)
	}
	if fake.FindSymbolsCalls.Load() != 1 {
		t.Fatalf("FindSymbols calls = %d", fake.FindSymbolsCalls.Load())
	}
}

func TestService_enrichCompileSymbols_skipsWhenClientSentSymbols(t *testing.T) {
	fake := &tooling.FakeLSPClient{Symbols: []tooling.Symbol{{Name: "X"}}}
	svc := &Service{LSP: fake, LSPEnabled: true}
	req := &CompileRequest{RepoRoot: "/tmp", LSPFocusPath: "x.go", Symbols: []string{"Client"}}
	svc.enrichCompileSymbols(context.Background(), req)
	if len(req.Symbols) != 1 || req.Symbols[0] != "Client" {
		t.Fatalf("symbols %+v", req.Symbols)
	}
	if fake.FindSymbolsCalls.Load() != 0 {
		t.Fatal("FindSymbols should not run when client sent symbols")
	}
}

func TestService_enrichCompileSymbols_skipsWhenLSPDisabled(t *testing.T) {
	fake := &tooling.FakeLSPClient{Symbols: []tooling.Symbol{{Name: "X"}}}
	svc := &Service{LSP: fake, LSPEnabled: false}
	req := &CompileRequest{RepoRoot: "/tmp", LSPFocusPath: "x.go"}
	svc.enrichCompileSymbols(context.Background(), req)
	if len(req.Symbols) != 0 {
		t.Fatalf("symbols %+v", req.Symbols)
	}
	if fake.FindSymbolsCalls.Load() != 0 {
		t.Fatal("FindSymbols should not run when LSP disabled")
	}
}

func TestService_CompileMulti_lspAutoSymbolsMatchedInBundle(t *testing.T) {
	payload := memory.PatternPayload{
		Polarity:   "negative",
		Symbols:    []string{"pkg.Foo"},
	}
	payloadBytes, _ := json.Marshal(payload)
	fakeMemory := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "ol", Authority: 5, UpdatedAt: time.Now(), Payload: payloadBytes},
		},
	}
	weights := DefaultRankingWeights()
	compiler := &Compiler{Memory: fakeMemory, Ranking: &weights}
	lsp := &tooling.FakeLSPClient{
		Symbols: []tooling.Symbol{{Name: "pkg.Foo"}, {Name: "Other"}},
	}
	svc := &Service{
		Compiler:         compiler,
		LSP:              lsp,
		LSPEnabled:       true,
		LSPAutoSymbolMax: 10,
	}
	req := CompileMultiRequest{
		RetrievalQuery: "test situation",
		Variants:       1,
		Strategy:       "default",
		MaxPerKind:     5,
		RepoRoot:       "/repo",
		LSPFocusPath:   "pkg/foo.go",
	}
	resp, err := svc.CompileMulti(context.Background(), req)
	if err != nil {
		t.Fatalf("CompileMulti: %v", err)
	}
	if lsp.FindSymbolsCalls.Load() != 1 {
		t.Fatalf("FindSymbols calls = %d", lsp.FindSymbolsCalls.Load())
	}
	if len(resp.Bundles) != 1 {
		t.Fatalf("bundles %d", len(resp.Bundles))
	}
	b := resp.Bundles[0].Bundle
	if len(b.MatchedSymbols) == 0 {
		t.Fatal("expected MatchedSymbols after LSP auto-fill")
	}
	found := false
	for _, s := range b.MatchedSymbols {
		if s == "pkg.Foo" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("MatchedSymbols = %v", b.MatchedSymbols)
	}
}
