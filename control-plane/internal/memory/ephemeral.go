package memory

import (
	"control-plane/pkg/api"
)

// DefaultEphemeralTTLSec is applied on create when disposable automation tags are present
// and the client did not set ttl_seconds.
const DefaultEphemeralTTLSec = 86400 // 24h

// ApplyEphemeralDefaults sets ttl_seconds when tags mark a disposable automation row and TTL is unset.
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
	if api.HasDisposableAutomationTags(req.Tags) {
		req.TTLSeconds = DefaultEphemeralTTLSec
	}
}
