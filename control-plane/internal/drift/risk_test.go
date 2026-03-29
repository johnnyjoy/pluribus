package drift

import (
	"testing"
)

func TestAssessRisk_low(t *testing.T) {
	level, block, _ := AssessRisk(StructuralSignals{ChangeCount: 0, BoundaryViolationCount: 0})
	if level != RiskLow || block {
		t.Errorf("AssessRisk(0,0) = %q, block=%v; want low, false", level, block)
	}
	level, block, _ = AssessRisk(StructuralSignals{ChangeCount: 1, BoundaryViolationCount: 0})
	if level != RiskLow || block {
		t.Errorf("AssessRisk(1,0) = %q, block=%v; want low, false", level, block)
	}
}

func TestAssessRisk_medium(t *testing.T) {
	level, block, warnings := AssessRisk(StructuralSignals{ChangeCount: 2, BoundaryViolationCount: 0})
	if level != RiskMedium || block {
		t.Errorf("AssessRisk(2,0) = %q, block=%v; want medium, false", level, block)
	}
	if len(warnings) == 0 {
		t.Error("expected at least one warning for medium risk")
	}
	level, block, _ = AssessRisk(StructuralSignals{ChangeCount: 3, BoundaryViolationCount: 0})
	if level != RiskMedium || block {
		t.Errorf("AssessRisk(3,0) = %q, block=%v; want medium, false", level, block)
	}
	level, block, _ = AssessRisk(StructuralSignals{ChangeCount: 1, BoundaryViolationCount: 1})
	if level != RiskMedium || block {
		t.Errorf("AssessRisk(1,1) = %q, block=%v; want medium, false", level, block)
	}
}

func TestAssessRisk_high(t *testing.T) {
	level, block, warnings := AssessRisk(StructuralSignals{ChangeCount: 4, BoundaryViolationCount: 0})
	if level != RiskHigh || !block {
		t.Errorf("AssessRisk(4,0) = %q, block=%v; want high, true", level, block)
	}
	if len(warnings) == 0 {
		t.Error("expected at least one warning for high risk")
	}
	level, block, _ = AssessRisk(StructuralSignals{ChangeCount: 0, BoundaryViolationCount: 2})
	if level != RiskHigh || !block {
		t.Errorf("AssessRisk(0,2) = %q, block=%v; want high, true", level, block)
	}
	level, block, _ = AssessRisk(StructuralSignals{ChangeCount: 5, BoundaryViolationCount: 1})
	if level != RiskHigh || !block {
		t.Errorf("AssessRisk(5,1) = %q, block=%v; want high, true", level, block)
	}
}
