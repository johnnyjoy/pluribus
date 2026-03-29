package ingest

import (
	"encoding/json"
	"testing"
)

func TestCognitionRequest_JSON_roundTrip(t *testing.T) {
	raw := `{"temp_contributor_id":"ingest-a","query":"q","reasoning_trace":["x"],"extracted_facts":[{"subject":"a","predicate":"b","object":"c"}],"confidence":0.5,"context_window_hash":"h"}`
	var r CognitionRequest
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	if r.TempContributorID != "ingest-a" {
		t.Fatalf("temp_contributor_id: got %q", r.TempContributorID)
	}
}
