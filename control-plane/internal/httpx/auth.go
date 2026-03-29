package httpx

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
)

// PluribusAPIKeyEnv is the only environment variable used for control-plane HTTP authentication (RC1).
const PluribusAPIKeyEnv = "PLURIBUS_API_KEY"

const (
	errAuthRequired              = "Missing API key: send X-API-Key with the value from PLURIBUS_API_KEY, or for POST /v1/mcp only pass the same value as ?token=."
	errAuthInvalidAPIKey         = "Invalid API key: the key does not match PLURIBUS_API_KEY."
	errAuthTokenQueryForbidden   = "Invalid query: ?token= is only allowed on POST /v1/mcp; use the X-API-Key header on all other routes."
	errAuthLegacyQueryParam      = "Unsupported auth query parameters (api_key, apikey, auth): use the X-API-Key header instead."
	errAuthAuthorizationRejected = "Authorization header is not supported: use X-API-Key with your Pluribus API key (Bearer and other schemes are not accepted)."
)

// MCPPath is the only path that may accept ?token= for API key (MCP clients without custom headers).
const MCPPath = "/v1/mcp"

// LoadPluribusAPIKey reads PLURIBUS_API_KEY from the environment.
// Empty or whitespace-only values are treated as unset (returns nil, auth disabled).
func LoadPluribusAPIKey() []byte {
	v := strings.TrimSpace(os.Getenv(PluribusAPIKeyEnv))
	if v == "" {
		return nil
	}
	return []byte(v)
}

// WrapWithPluribusAuth wraps the root HTTP handler (including MCP). When secret is nil, all requests pass.
// When secret is non-nil, protected routes require X-API-Key or (only for /v1/mcp) ?token=.
// /healthz and /readyz are never protected.
func WrapWithPluribusAuth(next http.Handler, secret []byte) http.Handler {
	if len(secret) == 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz", "/readyz":
			next.ServeHTTP(w, r)
			return
		}

		q := r.URL.Query()
		if legacyAuthQuery(q) {
			WriteError(w, http.StatusForbidden, errAuthLegacyQueryParam)
			return
		}
		if strings.TrimSpace(r.Header.Get("Authorization")) != "" {
			WriteError(w, http.StatusForbidden, errAuthAuthorizationRejected)
			return
		}

		token := strings.TrimSpace(q.Get("token"))
		if token != "" && r.URL.Path != MCPPath {
			WriteError(w, http.StatusForbidden, errAuthTokenQueryForbidden)
			return
		}

		headerKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
		var presented string
		switch {
		case headerKey != "":
			presented = headerKey
		case r.URL.Path == MCPPath && token != "":
			presented = token
		default:
			WriteError(w, http.StatusUnauthorized, errAuthRequired)
			return
		}

		if !secureEqual(secret, presented) {
			WriteError(w, http.StatusForbidden, errAuthInvalidAPIKey)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func legacyAuthQuery(q map[string][]string) bool {
	for k := range q {
		switch strings.ToLower(k) {
		case "apikey", "api_key", "auth":
			return true
		}
	}
	return false
}

func secureEqual(expected []byte, actual string) bool {
	a := []byte(actual)
	if len(expected) != len(a) {
		return false
	}
	return subtle.ConstantTimeCompare(expected, a) == 1
}
