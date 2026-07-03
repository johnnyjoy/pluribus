package mcp

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// MCPFullStubRouter returns a stub HTTP backend that satisfies all MCP tool proxy routes.
func MCPFullStubRouter() http.Handler {
	r := chi.NewRouter()
	jsonOK := func(w http.ResponseWriter, body string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, body)
	}

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { jsonOK(w, `{"ok":true}`) })
	r.Post("/v1/recall/compile", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"governing_constraints":[],"applicable_patterns":[],"failures":[],"decisions":[],"continuity":[]}`)
	})
	r.Post("/v1/recall/wakeup", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"identity":[],"governing_memory":[]}`)
	})
	r.Get("/v1/recall/", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"items":[]}`)
	})
	r.Post("/v1/recall/run-multi", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"variants":[]}`)
	})
	r.Post("/v1/recall/preflight", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"risk_hint":"low","binding_count":0}`)
	})
	r.Post("/v1/advisory-episodes", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"id":"00000000-0000-4000-8000-000000000001","summary_text":"ok","source":"mcp"}`)
	})
	r.Post("/v1/advisory-episodes/similar", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"episodes":[]}`)
	})
	r.Post("/v1/enforcement/evaluate", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"decision":"allow","evaluation_engine":"rule_based_heuristic_v1","triggered_memories":[]}`)
	})
	r.Post("/v1/memory", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"id":"11111111-1111-4111-8111-111111111111","kind":"constraint"}`)
	})
	r.Post("/v1/memory/promote", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"promoted":true}`)
	})
	r.Get("/v1/memory/{id}/relationships", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"relationships":[]}`)
	})
	r.Post("/v1/memory/{id}/feedback", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "00000000-0000-0000-0000-000000000000" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"error":"memory not found"}`)
			return
		}
		jsonOK(w, `{"memory_id":"`+id+`","event_id":"22222222-2222-4222-8222-222222222222","event_type":"helpful","new_utility_score":1.0,"counts":{"memory_id":"`+id+`","utility_score":1.0,"helpful_count":1}}`)
	})
	r.Get("/v1/memory/{id}/feedback", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"events":[]}`)
	})
	r.Get("/v1/memory/{id}/utility", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "00000000-0000-0000-0000-000000000000" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"error":"memory not found"}`)
			return
		}
		jsonOK(w, `{"memory_id":"`+id+`","utility_score":0.0}`)
	})
	r.Post("/v1/memory/relationships", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"id":"44444444-4444-4444-8444-444444444444"}`)
	})
	r.Post("/v1/memory/{id}/quarantine", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		jsonOK(w, `{"id":"`+id+`","status":"quarantined"}`)
	})
	r.Delete("/v1/memory/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		jsonOK(w, `{"id":"`+id+`","status":"deleted"}`)
	})
	r.Post("/v1/curation/digest", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"proposals":[]}`)
	})
	r.Get("/v1/curation/pending", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"candidates":[]}`)
	})
	r.Get("/v1/curation/promotion-suggestions", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"suggestions":[]}`)
	})
	r.Get("/v1/curation/strengthened", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"candidates":[]}`)
	})
	r.Get("/v1/curation/candidates/{id}/review", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"candidate":{"id":"33333333-3333-4333-8333-333333333333"}}`)
	})
	r.Post("/v1/curation/candidates/{id}/materialize", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"materialized":true}`)
	})
	r.Post("/v1/curation/candidates/{id}/reject", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"rejected":true}`)
	})
	r.Post("/v1/curation/auto-promote", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"promoted_count":0}`)
	})
	r.Get("/v1/curation/chores", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"chores":[]}`)
	})
	r.Post("/v1/curation/chores/{id}/resolve", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		jsonOK(w, `{"chore_id":"`+id+`","recorded":true,"counted":true,"votes_for_action":1,"min_resolvers":2,"applied":false,"state":"open"}`)
	})
	r.Post("/v1/episodes/distill", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"distilled":true}`)
	})
	r.Post("/v1/contradictions/detect", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"detected":false}`)
	})
	r.Get("/v1/contradictions", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"contradictions":[]}`)
	})
	r.Post("/v1/evidence", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"id":"55555555-5555-4555-8555-555555555555"}`)
	})
	r.Post("/v1/evidence/{id}/link", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"linked":true}`)
	})
	r.Get("/v1/evidence", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"evidence":[]}`)
	})
	r.Get("/v1/compliance/summary", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"total_sessions":0,"by_status":{},"evaluated_window":"recent_sessions_heuristic"}`)
	})
	r.Get("/v1/compliance/sessions", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"sessions":[]}`)
	})
	r.Get("/v1/compliance/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		jsonOK(w, `{"id":"`+id+`","client_name":"stub","transport":"http_mcp"}`)
	})
	r.Get("/v1/compliance/sessions/{id}/events", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"events":[]}`)
	})
	r.Post("/v1/compliance/evaluate", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"session_id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","status":"unknown","missing_steps":["insufficient_events"],"evidence":{"reason":"no telemetry events"}}`)
	})
	r.Post("/v1/agent/telemetry/session/start", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"session_id":"`+testSessionUUID+`","interface":"mcp","started_at":"2026-01-01T00:00:00Z"}`)
	})
	r.Post("/v1/agent/telemetry/recall", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"recall_event_id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb","session_id":"`+testSessionUUID+`"}`)
	})
	r.Post("/v1/agent/telemetry/decision", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"decisions":[{"decision_id":"cccccccc-cccc-4ccc-8ccc-cccccccccccc","memory_id":"mem_curr","decision":"used"}]}`)
	})
	r.Post("/v1/agent/telemetry/output", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"output_id":"dddddddd-dddd-4ddd-8ddd-dddddddddddd","session_id":"`+testSessionUUID+`"}`)
	})
	r.Post("/v1/agent/telemetry/evaluate", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"evaluation":{"evaluation_id":"eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee","obedience_passed":true,"obedience_score":1.0},"violations":[],"utility_candidates":[]}`)
	})
	r.Get("/v1/agent/telemetry/session/{session_id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "session_id")
		jsonOK(w, `{"session":{"session_id":"`+id+`"},"recall_events":[],"memory_decisions":[],"output_events":[],"obedience_evaluations":[],"violations":[],"utility_candidates":[]}`)
	})
	r.Get("/v1/agent/telemetry/memory/{memory_id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "memory_id")
		jsonOK(w, `{"memory_id":"`+id+`","recall_count":0,"used_count":0,"ignored_count":0,"violation_count":0,"obedience_pass_rate":0}`)
	})
	r.Get("/v1/agent/telemetry/violations", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"violations":[]}`)
	})
	r.Get("/v1/agent/telemetry/utility-candidates", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"utility_candidates":[]}`)
	})
	r.Post("/v1/agent/utility/policy/evaluate-candidate", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"candidate_id":"33333333-3333-4333-8333-333333333333","decision":"reject","delta":0,"reason":"stub","policy_version":"phase11k-v1","evidence":[]}`)
	})
	r.Post("/v1/agent/utility/policy/apply-candidate", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"application_id":"44444444-4444-4444-8444-444444444444","candidate_id":"33333333-3333-4333-8333-333333333333","memory_id":"test:mem","decision":"record_only","delta":0,"policy_version":"phase11k-v1","rollback_token":"stub-token"}`)
	})
	r.Post("/v1/agent/utility/policy/apply-batch", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"applications":[]}`)
	})
	r.Post("/v1/agent/utility/policy/revert-application", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"rollback_token":"stub-token","revert_reason":"stub"}`)
	})
	r.Get("/v1/agent/utility/policy/candidate/{candidate_id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "candidate_id")
		jsonOK(w, `{"candidate_id":"`+id+`","decision":"reject"}`)
	})
	r.Get("/v1/agent/utility/policy/memory/{memory_id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "memory_id")
		jsonOK(w, `{"applications":[],"memory_id":"`+id+`"}`)
	})
	r.Get("/v1/agent/utility/policy/applications", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"applications":[]}`)
	})
	r.Get("/v1/agent/utility/policy/summary", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, `{"total_applications":0,"policy_version":"phase11k-v1"}`)
	})
	return r
}
