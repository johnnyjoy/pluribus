package recall

import (
	"context"

	"control-plane/internal/tooling"
)

// enrichCompileSymbols fills req.Symbols from LSP documentSymbol when the server is configured
// for recall LSP, the client omitted symbols, and repo_root + lsp_focus_path are set.
func (s *Service) enrichCompileSymbols(ctx context.Context, req *CompileRequest) {
	if req == nil || !s.lspRecallActive() || len(req.Symbols) > 0 {
		return
	}
	if req.RepoRoot == "" || req.LSPFocusPath == "" {
		return
	}
	max := s.LSPAutoSymbolMax
	if max <= 0 {
		max = 64
	}
	syms, err := s.LSP.FindSymbols(ctx, req.RepoRoot, req.LSPFocusPath)
	if err != nil || len(syms) == 0 {
		return
	}
	names := flattenLSPNames(syms, max)
	if len(names) > 0 {
		req.Symbols = names
	}
}

func (s *Service) lspRecallActive() bool {
	return s.LSPEnabled && s.LSP != nil
}

// flattenLSPNames walks document symbols depth-first, collects unique names, caps at max.
func flattenLSPNames(syms []tooling.Symbol, max int) []string {
	if max <= 0 {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	var walk func([]tooling.Symbol)
	walk = func(nodes []tooling.Symbol) {
		for _, n := range nodes {
			if len(out) >= max {
				return
			}
			if n.Name != "" {
				if _, ok := seen[n.Name]; !ok {
					seen[n.Name] = struct{}{}
					out = append(out, n.Name)
				}
			}
			if len(n.Children) > 0 {
				walk(n.Children)
			}
			if len(out) >= max {
				return
			}
		}
	}
	walk(syms)
	return out
}

// enrichCompileMultiSymbols applies the same auto-symbol rules to compile-multi (mutates working).
func (s *Service) enrichCompileMultiSymbols(ctx context.Context, working *CompileMultiRequest) {
	if working == nil || !s.lspRecallActive() || len(working.Symbols) > 0 {
		return
	}
	if working.RepoRoot == "" || working.LSPFocusPath == "" {
		return
	}
	max := s.LSPAutoSymbolMax
	if max <= 0 {
		max = 64
	}
	syms, err := s.LSP.FindSymbols(ctx, working.RepoRoot, working.LSPFocusPath)
	if err != nil || len(syms) == 0 {
		return
	}
	names := flattenLSPNames(syms, max)
	if len(names) > 0 {
		working.Symbols = names
	}
}
