package recall

// newRunMultiDebug returns a debug block with non-nil JSON maps (always serializes object keys).
func newRunMultiDebug() RunMultiDebug {
	return RunMultiDebug{
		SignalBreakdown:   make(map[string]any),
		FilterReasons:     make(map[string]any),
		PromotionDecision: make(map[string]any),
		Orchestration:     make(map[string]any),
	}
}
