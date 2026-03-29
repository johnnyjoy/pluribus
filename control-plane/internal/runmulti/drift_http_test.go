package runmulti

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostDriftCheckSlowPathOptional_followup(t *testing.T) {
	var bodies [][]byte
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/drift/check", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, b)
		var req DriftCheckRequest
		_ = json.Unmarshal(b, &req)
		if req.SlowPathRequired && !req.IsFollowupCheck {
			_ = json.NewEncoder(w).Encode(DriftResult{Passed: true, RequiresFollowupCheck: true, FollowupReason: "x"})
			return
		}
		_ = json.NewEncoder(w).Encode(DriftResult{Passed: true, RiskLevel: "low"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	base := strings.TrimSuffix(srv.URL, "/")

	d, err := PostDriftCheckSlowPathOptional(context.Background(), base, "hello", http.DefaultClient, []string{"a", "b"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Passed {
		t.Fatalf("expected passed, got %+v", d)
	}
	if len(bodies) != 2 {
		t.Fatalf("want 2 drift calls (follow-up), got %d", len(bodies))
	}
	var second DriftCheckRequest
	if err := json.Unmarshal(bodies[1], &second); err != nil {
		t.Fatal(err)
	}
	if !second.IsFollowupCheck {
		t.Errorf("second call want is_followup_check: %+v", second)
	}
}

func TestPostDriftCheckSlowPathOptional_noSlowPathNoFollowup(t *testing.T) {
	var n int
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/drift/check", func(w http.ResponseWriter, r *http.Request) {
		n++
		_ = json.NewEncoder(w).Encode(DriftResult{Passed: true})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	base := strings.TrimSuffix(srv.URL, "/")

	_, err := PostDriftCheckSlowPathOptional(context.Background(), base, "x", http.DefaultClient, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("drift calls = %d, want 1", n)
	}
}
