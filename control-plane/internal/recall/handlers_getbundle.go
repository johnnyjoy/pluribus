package recall

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"control-plane/internal/httpx"
)

// GetBundle handles GET /v1/recall/ — read-only recall bundle (same semantics as POST /compile).
// Query: retrieval_query (situation / intent; alias: query), tags (repeat or comma-separated),
// symbols (repeat or comma-separated), max_per_kind, max_total, max_tokens (optional ints).
func (h *Handlers) GetBundle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "GET only: use GET /v1/recall/ with query parameters")
		return
	}
	q := r.URL.Query()
	req := CompileRequest{
		Tags:    parseQueryStringList(q, "tags"),
		Symbols: parseQueryStringList(q, "symbols"),
	}
	if rq := strings.TrimSpace(q.Get("retrieval_query")); rq != "" {
		req.RetrievalQuery = rq
	} else if rq := strings.TrimSpace(q.Get("query")); rq != "" {
		req.RetrievalQuery = rq
	}
	if v := strings.TrimSpace(q.Get("max_per_kind")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid max_per_kind")
			return
		}
		req.MaxPerKind = n
	}
	if v := strings.TrimSpace(q.Get("max_total")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid max_total")
			return
		}
		req.MaxTotal = n
	}
	if v := strings.TrimSpace(q.Get("max_tokens")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid max_tokens")
			return
		}
		req.MaxTokens = n
	}

	bundle, err := h.Service.Compile(r.Context(), req)
	if err != nil {
		if err == ErrNoCompiler {
			httpx.WriteError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, bundle)
}

// parseQueryStringList merges repeated query keys and comma-separated values (deduped, trimmed, empty dropped).
func parseQueryStringList(q url.Values, key string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, raw := range q[key] {
		for _, part := range strings.Split(raw, ",") {
			s := strings.TrimSpace(part)
			if s == "" {
				continue
			}
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}
