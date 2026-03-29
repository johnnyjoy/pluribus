package recall

import (
	"testing"
)

func TestComputePreflight(t *testing.T) {
	tests := []struct {
		name       string
		req        PreflightRequest
		wantRisk   string
		wantScore  float64
		wantActions []string
	}{
		{"low risk", PreflightRequest{ChangedFilesCount: 0}, "low", 0, nil},
		{"low risk few files", PreflightRequest{ChangedFilesCount: 3}, "low", 0, nil},
		{"medium risk", PreflightRequest{ChangedFilesCount: 5}, "medium", 0.5, []string{"drift_check"}},
		{"high risk", PreflightRequest{ChangedFilesCount: 11}, "high", 1.0, []string{"deep_recall", "drift_check"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputePreflight(tt.req)
			if got.RiskLevel != tt.wantRisk {
				t.Errorf("RiskLevel = %q, want %q", got.RiskLevel, tt.wantRisk)
			}
			if got.RiskScore != tt.wantScore {
				t.Errorf("RiskScore = %v, want %v", got.RiskScore, tt.wantScore)
			}
			if len(got.RequiredActions) != len(tt.wantActions) {
				t.Errorf("RequiredActions = %v, want %v", got.RequiredActions, tt.wantActions)
			} else {
				for i := range tt.wantActions {
					if got.RequiredActions[i] != tt.wantActions[i] {
						t.Errorf("RequiredActions[%d] = %q, want %q", i, got.RequiredActions[i], tt.wantActions[i])
					}
				}
			}
		})
	}
}
