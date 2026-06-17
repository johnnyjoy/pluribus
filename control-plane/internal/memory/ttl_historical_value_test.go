package memory

import "testing"

func TestIsDurableHistoricalTag(t *testing.T) {
	cases := []struct {
		tag  string
		want bool
	}{
		{"doctrine", true},
		{"Doctrine", true},
		{"phase", true},
		{"ephemeral-noise", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isDurableHistoricalTag(c.tag); got != c.want {
			t.Errorf("isDurableHistoricalTag(%q)=%v want %v", c.tag, got, c.want)
		}
	}
}

func TestTTLHistoricalValueExclusionSQL_present(t *testing.T) {
	if ttlHistoricalValueExclusionSQL == "" {
		t.Fatal("expected exclusion SQL fragment")
	}
	for _, frag := range []string{
		"memory_utility_events",
		"memory_evidence_links",
		"memory_relationships",
		"contradiction_records",
		"superseded_by",
	} {
		if !containsSubstring(ttlHistoricalValueExclusionSQL, frag) {
			t.Errorf("exclusion SQL missing %q", frag)
		}
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexSubstring(s, sub) >= 0)
}

func indexSubstring(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
