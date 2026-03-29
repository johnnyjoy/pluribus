package tooling

import (
	"context"
	"sync/atomic"
)

// FakeLSPClient is a test double that returns fixed symbols and references (Task 100).
type FakeLSPClient struct {
	Symbols    []Symbol
	References []Reference // returned for every FindReferences call when RefLengths is nil
	// RefLengths optional per-call lengths for FindReferences; 0 = empty slice.
	// If more calls than entries, the last length is reused.
	RefLengths []int
	// FindSymbolsCalls counts FindSymbols invocations (tests).
	FindSymbolsCalls atomic.Int32
	// FindReferencesCalls counts FindReferences invocations (tests).
	FindReferencesCalls atomic.Int32
}

// FindSymbols returns the preconfigured symbols.
func (f *FakeLSPClient) FindSymbols(ctx context.Context, root, path string) ([]Symbol, error) {
	f.FindSymbolsCalls.Add(1)
	return f.Symbols, nil
}

// FindReferences returns the preconfigured references.
func (f *FakeLSPClient) FindReferences(ctx context.Context, root, path string, line, col int) ([]Reference, error) {
	nCalls := f.FindReferencesCalls.Add(1)
	callIdx := int(nCalls) - 1
	var n int
	if len(f.RefLengths) > 0 {
		if callIdx < len(f.RefLengths) {
			n = f.RefLengths[callIdx]
		} else {
			n = f.RefLengths[len(f.RefLengths)-1]
		}
	} else {
		n = len(f.References)
	}
	if n == 0 {
		return nil, nil
	}
	out := make([]Reference, n)
	for i := 0; i < n; i++ {
		if i < len(f.References) {
			out[i] = f.References[i]
		} else {
			out[i] = Reference{Path: path, Range: &Range{StartLine: line, StartCol: col}}
		}
	}
	return out, nil
}

// Ensure FakeLSPClient implements LSPClient.
var _ LSPClient = (*FakeLSPClient)(nil)
