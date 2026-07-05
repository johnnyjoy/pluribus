package memory

import (
	"control-plane/pkg/api"
)

// DefaultEphemeralTTLSec is applied on create when an ephemeral tag is present
// and the client did not set ttl_seconds. Keeps proof/smoke rows from lingering
// in the shared pool without operator cleanup scripts.
const DefaultEphemeralTTLSec = 86400 // 24h

// ApplyEphemeralDefaults sets ttl_seconds when tags include ephemeral and TTL is unset.
func ApplyEphemeralDefaults(req *CreateRequest) {
	if req == nil || req.TTLSeconds > 0 {
		return
	}
	for _, tag := range req.Tags {
		if api.IsEphemeralTag(tag) {
			req.TTLSeconds = DefaultEphemeralTTLSec
			return
		}
	}
}
