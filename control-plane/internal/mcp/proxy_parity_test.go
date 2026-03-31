package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildEpisodeSimilarBody(t *testing.T) {
	b, err := buildEpisodeSimilarBody(json.RawMessage(`{"summary_text":"hello","tags":["a"]}`))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["query"] != "hello" {
		t.Fatalf("query: %v", m["query"])
	}
}

func TestBuildRecallAdvancedBody_modes(t *testing.T) {
	for _, tc := range []struct {
		mode string
		want string
	}{
		{"continuity", "continuity"},
		{"episodic", "thread"},
	} {
		raw := json.RawMessage(`{"query":"x","mode":"` + tc.mode + `"}`)
		b, err := buildRecallAdvancedBody(raw)
		if err != nil {
			t.Fatalf("%s: %v", tc.mode, err)
		}
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		if m["mode"] != tc.want {
			t.Fatalf("%s: mode=%v want %s", tc.mode, m["mode"], tc.want)
		}
	}
	b, err := buildRecallAdvancedBody(json.RawMessage(`{"query":"q","mode":"constraint"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "variant_modifier") {
		t.Fatalf("constraint should set variant_modifier: %s", string(b))
	}
}

func TestBuildContradictionDetectBody(t *testing.T) {
	_, err := buildContradictionDetectBody(json.RawMessage(`{"memory_id":"not-uuid","conflict_with_id":"11111111-1111-1111-1111-111111111111"}`))
	if err == nil {
		t.Fatal("expected error")
	}
	b, err := buildContradictionDetectBody(json.RawMessage(`{"memory_id":"11111111-1111-1111-1111-111111111111","conflict_with_id":"22222222-2222-2222-2222-222222222222"}`))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
}
