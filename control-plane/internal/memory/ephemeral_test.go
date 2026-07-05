package memory

import (
	"testing"

	"control-plane/pkg/api"
)

func TestApplyEphemeralDefaults_setsTTLWhenUnset(t *testing.T) {
	req := &CreateRequest{Tags: []string{api.TagEphemeral, api.TagProofScenario}}
	ApplyEphemeralDefaults(req)
	if req.TTLSeconds != DefaultEphemeralTTLSec {
		t.Fatalf("TTLSeconds=%d want %d", req.TTLSeconds, DefaultEphemeralTTLSec)
	}
}

func TestApplyEphemeralDefaults_respectsExplicitTTL(t *testing.T) {
	req := &CreateRequest{Tags: []string{api.TagEphemeral}, TTLSeconds: 3600}
	ApplyEphemeralDefaults(req)
	if req.TTLSeconds != 3600 {
		t.Fatalf("TTLSeconds=%d want 3600", req.TTLSeconds)
	}
}

func TestApplyEphemeralDefaults_noOpWithoutEphemeralTag(t *testing.T) {
	req := &CreateRequest{Tags: []string{"architecture", "decision"}}
	ApplyEphemeralDefaults(req)
	if req.TTLSeconds != 0 {
		t.Fatalf("TTLSeconds=%d want 0", req.TTLSeconds)
	}
}

func TestApplyEphemeralDefaults_prefixTag(t *testing.T) {
	req := &CreateRequest{Tags: []string{"ephemeral:proof"}}
	ApplyEphemeralDefaults(req)
	if req.TTLSeconds != DefaultEphemeralTTLSec {
		t.Fatalf("TTLSeconds=%d want %d", req.TTLSeconds, DefaultEphemeralTTLSec)
	}
}
