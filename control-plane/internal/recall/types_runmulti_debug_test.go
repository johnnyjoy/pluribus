package recall

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRunMultiResponse_JSON_alwaysIncludesDebugMaps(t *testing.T) {
	resp := &RunMultiResponse{
		Debug: newRunMultiDebug(),
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, key := range []string{`"debug"`, `"signal_breakdown"`, `"filter_reasons"`, `"promotion_decision"`, `"orchestration"`} {
		if !strings.Contains(s, key) {
			t.Fatalf("expected JSON to contain %s, got %s", key, s)
		}
	}
}
