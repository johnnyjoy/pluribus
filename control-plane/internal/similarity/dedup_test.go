package similarity

import "testing"

func Test_mcpDedupCorrelationMatch(t *testing.T) {
	t.Parallel()
	cases := []struct {
		corr string
		tags []string
		want bool
	}{
		{"", nil, true},
		{"", []string{"foo"}, true},
		{"", []string{"mcp:session:"}, true},
		{"x", []string{"mcp:session:x"}, true},
		{"x", []string{"mcp:session:y"}, false},
		{"x", []string{"other"}, false},
		{"", []string{"mcp:session:z"}, false},
	}
	for _, tc := range cases {
		if got := mcpDedupCorrelationMatch(tc.corr, tc.tags); got != tc.want {
			t.Errorf("corr=%q tags=%v got %v want %v", tc.corr, tc.tags, got, tc.want)
		}
	}
}
