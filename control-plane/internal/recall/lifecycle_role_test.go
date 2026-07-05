package recall

import (
	"testing"

	"control-plane/internal/memory"
	"control-plane/pkg/api"
)

func TestApplyHistoricalScoreCap_pending(t *testing.T) {
	obj := memory.MemoryObject{Status: api.StatusPending}
	got := applyHistoricalScoreCap(RecallModeHistorical, obj, 0.95)
	if got != 0.75 {
		t.Fatalf("pending historical cap: got %v want 0.75", got)
	}
	if applyHistoricalScoreCap(RecallModeCurrent, obj, 0.95) != 0.95 {
		t.Fatal("current mode should not cap pending")
	}
}
