package lexical

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"control-plane/internal/app"
)

// Handlers serves experimental lexical HTTP when enabled in config.
type Handlers struct {
	DB  *sql.DB
	Cfg *app.LexicalConfig
}

// NewHandlers wires BM25 search. Cfg must be non-nil with ExperimentalHTTP true.
func NewHandlers(db *sql.DB, cfg *app.LexicalConfig) *Handlers {
	return &Handlers{DB: db, Cfg: cfg}
}

type searchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

// Search POST /v1/experimental/lexical/search — returns BM25-ranked memory_id rows from the projection table.
func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	if h.Cfg == nil || !h.Cfg.ExperimentalHTTP {
		http.Error(w, `{"error":"lexical experimental API disabled"}`, http.StatusNotFound)
		return
	}
	table := h.Cfg.ProjectionTable
	if table == "" {
		table = DefaultProjectionTable
	}
	var req searchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	hits, err := Search(r.Context(), h.DB, table, req.Query, req.Limit)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"hits": hits})
}
