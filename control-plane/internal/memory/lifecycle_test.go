package memory

import (
	"testing"
)

func TestApplyAuthorityEvent_validation(t *testing.T) {
	// authority 5 (0.5) + 0.1*(1-0.5) = 0.55 => 6
	got := ApplyAuthorityEvent(5, "validation", 0.1, 0.2)
	if got != 6 {
		t.Errorf("ApplyAuthorityEvent(5, validation, 0.1, 0.2) = %d, want 6", got)
	}
	// high authority stays capped at 10
	got = ApplyAuthorityEvent(10, "validation", 0.1, 0.2)
	if got != 10 {
		t.Errorf("ApplyAuthorityEvent(10, validation, 0.1, 0.2) = %d, want 10 (capped)", got)
	}
}

func TestApplyAuthorityEvent_contradiction(t *testing.T) {
	// authority 5 (0.5) - 0.2*0.5 = 0.4 => 4
	got := ApplyAuthorityEvent(5, "contradiction", 0.1, 0.2)
	if got != 4 {
		t.Errorf("ApplyAuthorityEvent(5, contradiction, 0.1, 0.2) = %d, want 4", got)
	}
	// authority 1: multiplicative decay rounds back to 1, but negative events
	// guarantee at least a one-step drop (C3: demotion must always progress)
	got = ApplyAuthorityEvent(1, "contradiction", 0.1, 0.2)
	if got != 0 {
		t.Errorf("ApplyAuthorityEvent(1, contradiction, ...) = %d, want 0 (guaranteed progress)", got)
	}
	// authority 0 unchanged
	got = ApplyAuthorityEvent(0, "contradiction", 0.1, 0.2)
	if got != 0 {
		t.Errorf("ApplyAuthorityEvent(0, contradiction, ...) = %d, want 0 (capped)", got)
	}
}

func TestApplyAuthorityEvent_failureAlwaysDemotes(t *testing.T) {
	// Hostile audit C3: authority used to stick at 2 under repeated failures
	// (2 -> 1.6 -> rounds to 2). Every negative event must now drop >= 1 step.
	for start := 1; start <= 10; start++ {
		got := ApplyAuthorityEvent(start, "failure", 0.1, 0.2)
		if got >= start {
			t.Errorf("ApplyAuthorityEvent(%d, failure) = %d, want < %d", start, got, start)
		}
	}
	// Repeated failures from 2 must reach 0 (below any recall floor).
	a := 2
	for i := 0; i < 5 && a > 0; i++ {
		a = ApplyAuthorityEvent(a, "failure", 0.1, 0.2)
	}
	if a != 0 {
		t.Errorf("repeated failures from 2 ended at %d, want 0", a)
	}
}

func TestApplyAuthorityEvent_failure(t *testing.T) {
	got := ApplyAuthorityEvent(5, "failure", 0.1, 0.2)
	if got != 4 {
		t.Errorf("ApplyAuthorityEvent(5, failure, 0.1, 0.2) = %d, want 4", got)
	}
}

func TestApplyAuthorityEvent_unknownType(t *testing.T) {
	got := ApplyAuthorityEvent(5, "other", 0.1, 0.2)
	if got != 5 {
		t.Errorf("ApplyAuthorityEvent(5, other, ...) = %d, want 5 (unchanged)", got)
	}
}

func TestApplyAuthorityEvent_noCrossContamination(t *testing.T) {
	// One memory at 7: validation -> 8, validation -> 8 or 9
	a := 7
	a = ApplyAuthorityEvent(a, "validation", 0.1, 0.2)
	if a < 7 || a > 10 {
		t.Errorf("after validation: authority = %d, want 8", a)
	}
	a = ApplyAuthorityEvent(a, "contradiction", 0.1, 0.2)
	if a > 8 {
		t.Errorf("after contradiction: authority = %d, should have decreased", a)
	}
}
