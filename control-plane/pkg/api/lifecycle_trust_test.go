package api_test

import (
	"testing"

	"control-plane/pkg/api"
)

func TestEffectiveBindingAuthority_pendingDampens(t *testing.T) {
	got := api.EffectiveBindingAuthority(10, api.StatusPending)
	want := 10 * api.PendingTrustDampener
	if got != want {
		t.Fatalf("got %v want %v", got, want)
	}
	if api.EffectiveBindingAuthority(10, api.StatusActive) != 10 {
		t.Fatal("active should not dampen")
	}
}
