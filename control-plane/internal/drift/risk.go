package drift

// Risk levels for structural escalation (Task 77).
const (
	RiskLow    = "low"
	RiskMedium = "medium"
	RiskHigh   = "high"
)

// AssessRisk maps structural signals to risk level and recommended action (Task 77).
// low: ≤1 change, no boundary violation.
// medium: 2-3 changes or one boundary violation.
// high: ≥4 changes or multiple boundary violations → block execution.
// Returns riskLevel, blockExecution (true only for high), and warnings to append.
func AssessRisk(signals StructuralSignals) (riskLevel string, blockExecution bool, warnings []string) {
	if signals.ChangeCount >= 4 || signals.BoundaryViolationCount >= 2 {
		riskLevel = RiskHigh
		blockExecution = true
		warnings = append(warnings, "High structural risk: block execution and review changes.")
		return
	}
	if signals.ChangeCount >= 2 || signals.BoundaryViolationCount >= 1 {
		riskLevel = RiskMedium
		warnings = append(warnings, "Medium structural risk: consider reviewing before proceeding.")
		return
	}
	riskLevel = RiskLow
	return
}
