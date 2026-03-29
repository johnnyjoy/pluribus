package enforcement

import (
	"context"
	"testing"

	"control-plane/internal/app"
)

func TestService_Evaluate_disabled(t *testing.T) {
	off := false
	svc := &Service{
		Config: &app.EnforcementConfig{Enabled: &off},
	}
	_, err := svc.Evaluate(context.Background(), EvaluateRequest{})
	if err != ErrDisabled {
		t.Fatalf("got %v", err)
	}
}

