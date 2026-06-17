package agentcontract

import "control-plane/internal/recall"

// CollectMemoryItems flattens all bucketed MemoryItem slices from a RecallBundle.
func CollectMemoryItems(b *recall.RecallBundle) []recall.MemoryItem {
	if b == nil {
		return nil
	}
	var out []recall.MemoryItem
	for _, s := range [][]recall.MemoryItem{
		b.GoverningConstraints,
		b.Decisions,
		b.KnownFailures,
		b.ApplicablePatterns,
		b.Continuity,
		b.Constraints,
		b.Experience,
	} {
		out = append(out, s...)
	}
	return out
}

// CollectWakeupMemoryItems returns all MemoryItems from a wakeup response.
func CollectWakeupMemoryItems(w *recall.WakeupResponse) []recall.MemoryItem {
	if w == nil {
		return nil
	}
	out := make([]recall.MemoryItem, 0, len(w.Identity)+len(w.GoverningMemory))
	out = append(out, w.Identity...)
	out = append(out, w.GoverningMemory...)
	return out
}

// CollectCompileMultiMemoryItems returns MemoryItems from all variant bundles.
func CollectCompileMultiMemoryItems(resp *recall.CompileMultiResponse) []recall.MemoryItem {
	if resp == nil {
		return nil
	}
	var out []recall.MemoryItem
	for _, vb := range resp.Bundles {
		out = append(out, CollectMemoryItems(&vb.Bundle)...)
	}
	return out
}
