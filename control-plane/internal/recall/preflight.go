package recall

// ComputePreflight derives risk level and required actions from changed-files count and tags.
// High risk when many files changed; optional required_actions for deep_recall, drift_check.
// RiskScore is 0 (low), 0.5 (medium), or 1.0 (high) for comparison with slow_path.high_risk_threshold (Task 93).
func ComputePreflight(req PreflightRequest) PreflightResult {
	risk := "low"
	score := 0.0
	if req.ChangedFilesCount > 10 {
		risk = "high"
		score = 1.0
	} else if req.ChangedFilesCount > 3 {
		risk = "medium"
		score = 0.5
	}
	actions := []string{}
	if risk == "high" {
		actions = append(actions, "deep_recall", "drift_check")
	} else if risk == "medium" {
		actions = append(actions, "drift_check")
	}
	return PreflightResult{RiskLevel: risk, RequiredActions: actions, RiskScore: score}
}
