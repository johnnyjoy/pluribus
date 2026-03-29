package httpx

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"control-plane/internal/migrate"

	"github.com/go-chi/chi/v5"
)

// RegisterReadiness adds GET /readyz: DB reachable and core schema present (memories table).
// /healthz remains liveness-only (process up).
func RegisterReadiness(r chi.Router, db *sql.DB) {
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			http.Error(w, "database unreachable (check postgres.dsn and network): "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		ok, err := migrate.CoreSchemaReady(ctx, db)
		if err != nil {
			http.Error(w, "readiness check failed: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		if !ok {
			http.Error(w, "database schema incomplete: baseline not applied (restart control-plane against a fresh Postgres database)", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
}
