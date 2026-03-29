package ingest

import "testing"

func TestApplyReinforce(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		priorPeak float64
		incoming  float64
		want      float64
	}{
		{"prior higher", 0.8, 0.5, 0.9},
		{"incoming higher", 0.5, 0.8, 0.9},
		{"equal mid", 0.75, 0.75, 0.85},
		{"caps at one", 0.95, 0.95, 1.0},
		{"already one", 1.0, 1.0, 1.0},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ApplyReinforce(tc.priorPeak, tc.incoming)
			if got != tc.want {
				t.Fatalf("ApplyReinforce(%v,%v) = %v want %v", tc.priorPeak, tc.incoming, got, tc.want)
			}
		})
	}
}
