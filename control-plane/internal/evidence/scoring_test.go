package evidence

import "testing"

func TestBaseScore(t *testing.T) {
	tests := []struct {
		kind string
		want float64
	}{
		{KindTest, 1.0},
		{KindBenchmark, 0.8},
		{KindLog, 0.5},
		{KindObservation, 0.2},
		{"unknown", 0.2},
		{"", 0.2},
	}
	for _, tt := range tests {
		got := BaseScore(tt.kind)
		if got != tt.want {
			t.Errorf("BaseScore(%q) = %v, want %v", tt.kind, got, tt.want)
		}
	}
}
