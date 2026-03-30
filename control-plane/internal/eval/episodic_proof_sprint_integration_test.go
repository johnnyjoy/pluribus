//go:build integration
// +build integration

package eval

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"control-plane/internal/apiserver"
	"control-plane/internal/app"
	"control-plane/internal/migrate"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// TestEpisodicProofSprintREST_Postgres runs adversarial episodic scenarios through the real HTTP router (REST only).
// Complements proof-*.json with multi-step stateful checks (conflict, time skew, soak loops, boundary separation).
func TestEpisodicProofSprintREST_Postgres(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	pgdb, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	if err := pgdb.PingContext(context.Background()); err != nil {
		_ = pgdb.Close()
		t.Fatalf("ping postgres: %v", err)
	}
	if err := migrate.MaybeResetPublicSchemaForIntegrationTests(context.Background(), pgdb); err != nil {
		_ = pgdb.Close()
		t.Fatal(err)
	}
	if err := migrate.RequireProofHarnessCleanPostgres(context.Background(), pgdb); err != nil {
		_ = pgdb.Close()
		t.Fatal(err)
	}
	_ = pgdb.Close()

	base, cleanup := episodicProofServer(t, dsn, nil)
	defer cleanup()
	hc := &http.Client{Timeout: 60 * time.Second}

	t.Run("conflicting_evidence_two_distinct_candidates", func(t *testing.T) {
		tag := "proof:episodic:conflict:" + uuid.New().String()
		a := `CONFLICT-A-` + uuid.New().String() + ` payment gateway timeout error caused double charge incident`
		b := `CONFLICT-B-` + uuid.New().String() + ` inventory sync failure error lost warehouse stock counts`
		episodicProofLog(t, "conflicting_evidence", "distill", "start", "-")
		mustDistillInline(t, hc, base, a, tag)
		mustDistillInline(t, hc, base, b, tag)
		body := mustGET(t, hc, base+"/v1/curation/pending")
		if !strings.Contains(body, strings.Split(a, " ")[0]) || !strings.Contains(body, strings.Split(b, " ")[0]) {
			t.Fatalf("pending queue should surface both conflicting statement markers; body=%s", truncateEpisodicBody(body))
		}
		episodicProofLog(t, "conflicting_evidence", "pending", "pass", "both markers present")
	})

	t.Run("time_skew_occurred_at_not_ingest_time", func(t *testing.T) {
		rid := uuid.New().String()[:8]
		tag := "proof:episodic:skew:" + rid
		summary := `TIME SKEW MARKER ` + rid + ` redis cluster failover error during overnight maintenance window`
		body := fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q],"occurred_at":"2001-07-08T02:00:00Z","entities":["redis"]}`,
			summary, tag)
		episodicProofLog(t, "time_skew", "ingest", "start", rid)
		_ = mustPOST(t, hc, base+"/v1/advisory-episodes", body, http.StatusCreated)
		q := fmt.Sprintf(`{"query":"time skew marker %s redis failover","tags":[%q],"occurred_after":"2001-01-01T00:00:00Z","occurred_before":"2001-12-31T23:59:59Z","max_results":5}`,
			rid, tag)
		sim := mustPOST(t, hc, base+"/v1/advisory-episodes/similar", q, http.StatusOK)
		if !strings.Contains(sim, "TIME SKEW MARKER") {
			t.Fatalf("similar should find episode by occurred_at window, got %s", truncateEpisodicBody(sim))
		}
		episodicProofLog(t, "time_skew", "similar", "pass", rid)
	})

	t.Run("advisory_boundary_recall_ignores_episode_until_canon", func(t *testing.T) {
		rid := uuid.New().String()[:8]
		tag := "proof:episodic:boundary:" + rid
		marker := `ADVISORY ONLY MARKER ` + rid + ` never deploy on friday error caused outage pattern`
		episodicProofLog(t, "boundary", "ingest", "start", rid)
		epBody := fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q]}`, marker, tag)
		eb := mustPOST(t, hc, base+"/v1/advisory-episodes", epBody, http.StatusCreated)
		var ep struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal([]byte(eb), &ep)
		rb := mustPOST(t, hc, base+"/v1/recall/compile",
			fmt.Sprintf(`{"retrieval_query":%q,"tags":[%q]}`, marker, tag), http.StatusOK)
		if strings.Contains(rb, marker) {
			t.Fatalf("recall must not surface raw advisory episode text before promotion; bundle snippet=%s", truncateEpisodicBody(rb))
		}
		db := mustPOST(t, hc, base+"/v1/episodes/distill", fmt.Sprintf(`{"episode_id":%q}`, ep.ID), http.StatusOK)
		var dr struct {
			Candidates []struct {
				CandidateID string `json:"candidate_id"`
			} `json:"candidates"`
		}
		_ = json.Unmarshal([]byte(db), &dr)
		if len(dr.Candidates) < 1 {
			t.Fatalf("distill expected candidates: %s", db)
		}
		cid := dr.Candidates[0].CandidateID
		_ = mustPOST(t, hc, base+"/v1/curation/candidates/"+cid+"/materialize", `{}`, http.StatusCreated)
		ra := mustPOST(t, hc, base+"/v1/recall/compile",
			fmt.Sprintf(`{"retrieval_query":%q,"tags":[%q]}`, marker, tag), http.StatusOK)
		if !strings.Contains(ra, "ADVISORY ONLY MARKER") {
			t.Fatalf("after materialize, canonical memory should be recallable; got %s", truncateEpisodicBody(ra))
		}
		episodicProofLog(t, "boundary", "recall", "pass", rid)
	})

	t.Run("enforcement_stable_on_repeat", func(t *testing.T) {
		rid := uuid.New().String()[:8]
		tag := "proof:episodic:stab:" + rid
		summary := `STAB ENF ` + rid + ` never store primary credentials in plaintext error led to breach incident`
		eb := mustPOST(t, hc, base+"/v1/advisory-episodes",
			fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q]}`, summary, tag), http.StatusCreated)
		var ep struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal([]byte(eb), &ep)
		db := mustPOST(t, hc, base+"/v1/episodes/distill", fmt.Sprintf(`{"episode_id":%q}`, ep.ID), http.StatusOK)
		var dr struct {
			Candidates []struct {
				CandidateID string `json:"candidate_id"`
			} `json:"candidates"`
		}
		_ = json.Unmarshal([]byte(db), &dr)
		if len(dr.Candidates) < 1 {
			t.Fatal("no candidate")
		}
		cid := dr.Candidates[0].CandidateID
		_ = mustPOST(t, hc, base+"/v1/curation/candidates/"+cid+"/materialize", `{}`, http.StatusCreated)
		prop := fmt.Sprintf(`We will store user passwords in plaintext in config files STAB ENF %s.`, rid)
		b1 := mustPOST(t, hc, base+"/v1/enforcement/evaluate",
			fmt.Sprintf(`{"proposal_text":%q,"tags":[%q]}`, prop, tag), http.StatusOK)
		b2 := mustPOST(t, hc, base+"/v1/enforcement/evaluate",
			fmt.Sprintf(`{"proposal_text":%q,"tags":[%q]}`, prop, tag), http.StatusOK)
		d1, d2 := gjsonString(b1, "decision"), gjsonString(b2, "decision")
		if d1 == "" || d1 != d2 {
			t.Fatalf("enforcement decisions differ on repeat: %q vs %q (b1=%s b2=%s)", d1, d2, truncateEpisodicBody(b1), truncateEpisodicBody(b2))
		}
		episodicProofLog(t, "enforcement_repeat", "evaluate", "pass", d1)
	})

	t.Run("soak_distill_merge_idempotent_support_monotonic", func(t *testing.T) {
		rid := uuid.New().String()[:8]
		tag := "proof:episodic:soak:" + rid
		shared := fmt.Sprintf(`SOAK MERGE %s pipeline timeout error retried three times then rollback failure`, rid)
		var lastSupport int
		for i := 0; i < 4; i++ {
			eb := mustPOST(t, hc, base+"/v1/advisory-episodes",
				fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q,%q]}`, shared, tag, fmt.Sprintf("soak-iter-%d", i)),
				http.StatusCreated)
			var ep struct {
				ID string `json:"id"`
			}
			_ = json.Unmarshal([]byte(eb), &ep)
			db := mustPOST(t, hc, base+"/v1/episodes/distill", fmt.Sprintf(`{"episode_id":%q}`, ep.ID), http.StatusOK)
			var dr struct {
				Candidates []struct {
					DistillSupportCount int `json:"distill_support_count"`
					Merged              bool `json:"merged"`
				} `json:"candidates"`
			}
			_ = json.Unmarshal([]byte(db), &dr)
			if len(dr.Candidates) < 1 {
				t.Fatalf("iter %d: no candidates %s", i, db)
			}
			sc := dr.Candidates[0].DistillSupportCount
			if sc < lastSupport {
				t.Fatalf("iter %d: support count regressed %d -> %d", i, lastSupport, sc)
			}
			lastSupport = sc
			if i > 0 && !dr.Candidates[0].Merged {
				t.Logf("note: merged flag false on iter %d (multi-kind split possible)", i)
			}
		}
		if lastSupport < 4 {
			t.Fatalf("expected support count at least 4 after four identical distill merges, got %d", lastSupport)
		}
		episodicProofLog(t, "soak_merge", "distill_loop", "pass", fmt.Sprintf("support=%d", lastSupport))
	})

	t.Run("inverted_time_window_similar_returns_400", func(t *testing.T) {
		episodicProofLog(t, "time_inverted", "similar", "start", "-")
		body := `{"query":"episodic inverted window proof","occurred_after":"2030-12-31T00:00:00Z","occurred_before":"2030-01-01T00:00:00Z","max_results":3}`
		req, _ := http.NewRequest(http.MethodPost, base+"/v1/advisory-episodes/similar", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := hc.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("want 400 inverted window, got %d %s", resp.StatusCode, string(b))
		}
		if !strings.Contains(string(b), "occurred_after") {
			t.Fatalf("body should mention occurred_after: %s", string(b))
		}
		episodicProofLog(t, "time_inverted", "similar", "pass", "400")
	})

	t.Run("weak_inline_distill_yields_no_candidates", func(t *testing.T) {
		tag := "proof:episodic:weak:" + uuid.New().String()
		body, _ := json.Marshal(map[string]interface{}{"summary": "short", "tags": []string{tag}})
		out := mustPOST(t, hc, base+"/v1/episodes/distill", string(body), http.StatusOK)
		if strings.Contains(out, `"candidate_id"`) {
			t.Fatalf("expected no distill candidates for vague short summary; got %s", truncateEpisodicBody(out))
		}
		episodicProofLog(t, "weak_signal", "distill", "pass", "no candidates")
	})

	t.Run("auto_distill_on_ingest_pending_without_explicit_distill", func(t *testing.T) {
		baseAuto, cleanupAuto := episodicProofServer(t, dsn, func(cfg *app.Config) {
			cfg.Distillation.AutoFromAdvisoryEpisodes = true
		})
		defer cleanupAuto()
		rid := uuid.New().String()[:8]
		tag := "proof:episodic:autod:" + rid
		marker := `AUTO DISTILL INGEST ` + rid + ` cache stampede error overwhelmed origin during flash sale incident`
		episodicProofLog(t, "auto_distill", "ingest", "start", rid)
		_ = mustPOST(t, hc, baseAuto+"/v1/advisory-episodes",
			fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q]}`, marker, tag), http.StatusCreated)
		pend := mustGET(t, hc, baseAuto+"/v1/curation/pending")
		if !strings.Contains(pend, marker) || !strings.Contains(pend, `"pluribus_distill_origin":"auto"`) {
			t.Fatalf("expected auto-distilled pending row with origin auto; body=%s", truncateEpisodicBody(pend))
		}
		rc := mustPOST(t, hc, baseAuto+"/v1/recall/compile",
			fmt.Sprintf(`{"retrieval_query":%q,"tags":[%q],"max_per_kind":8,"max_total":40}`, marker, tag), http.StatusOK)
		if strings.Contains(rc, marker) {
			t.Fatalf("recall must not surface advisory/candidate text before materialize; got %s", truncateEpisodicBody(rc))
		}
		episodicProofLog(t, "auto_distill", "pending_recall_boundary", "pass", rid)
	})

	t.Run("enforcement_ignores_pending_distilled_candidate", func(t *testing.T) {
		rid := uuid.New().String()[:8]
		tag := "proof:episodic:pendef:" + rid
		summary := `PENDING CAND ENF ` + rid + ` never expose root api keys in client bundles error led to credential leak incident`
		eb := mustPOST(t, hc, base+"/v1/advisory-episodes",
			fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q]}`, summary, tag), http.StatusCreated)
		var ep struct{ ID string `json:"id"` }
		_ = json.Unmarshal([]byte(eb), &ep)
		_ = mustPOST(t, hc, base+"/v1/episodes/distill", fmt.Sprintf(`{"episode_id":%q}`, ep.ID), http.StatusOK)
		prop := fmt.Sprintf(`We will embed root api keys in the mobile client bundle PENDING CAND ENF %s.`, rid)
		ev := mustPOST(t, hc, base+"/v1/enforcement/evaluate",
			fmt.Sprintf(`{"proposal_text":%q,"tags":[%q]}`, prop, tag), http.StatusOK)
		var evj struct {
			Triggered []json.RawMessage `json:"triggered_memories"`
		}
		if err := json.Unmarshal([]byte(ev), &evj); err != nil {
			t.Fatalf("enforcement json: %v", err)
		}
		if len(evj.Triggered) != 0 {
			t.Fatalf("enforcement must not bind pending candidate-only text; triggered=%d body=%s", len(evj.Triggered), truncateEpisodicBody(ev))
		}
		episodicProofLog(t, "boundary", "enforcement_pending", "pass", rid)
	})

	t.Run("recall_compile_identical_requests_match", func(t *testing.T) {
		rid := uuid.New().String()[:8]
		tag := "proof:episodic:det:" + rid
		summary := `DET RECALL ` + rid + ` rotate database credentials quarterly failure when manual run skipped caused stale access incident`
		eb := mustPOST(t, hc, base+"/v1/advisory-episodes",
			fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q]}`, summary, tag), http.StatusCreated)
		var ep struct{ ID string `json:"id"` }
		_ = json.Unmarshal([]byte(eb), &ep)
		db := mustPOST(t, hc, base+"/v1/episodes/distill", fmt.Sprintf(`{"episode_id":%q}`, ep.ID), http.StatusOK)
		var dr struct {
			Candidates []struct{ CandidateID string `json:"candidate_id"` } `json:"candidates"`
		}
		_ = json.Unmarshal([]byte(db), &dr)
		if len(dr.Candidates) < 1 {
			t.Fatalf("distill: %s", db)
		}
		cid := dr.Candidates[0].CandidateID
		_ = mustPOST(t, hc, base+"/v1/curation/candidates/"+cid+"/materialize", `{}`, http.StatusCreated)
		q := fmt.Sprintf(`{"retrieval_query":%q,"tags":[%q],"max_per_kind":8,"max_total":40}`, summary, tag)
		r1 := mustPOST(t, hc, base+"/v1/recall/compile", q, http.StatusOK)
		r2 := mustPOST(t, hc, base+"/v1/recall/compile", q, http.StatusOK)
		if r1 != r2 {
			t.Fatalf("two identical recall.compile calls differ (len %d vs %d)", len(r1), len(r2))
		}
		episodicProofLog(t, "determinism", "recall_compile", "pass", rid)
	})

	t.Run("review_supporting_episodes_cap_three_with_four_merges", func(t *testing.T) {
		rid := uuid.New().String()[:8]
		tag := "proof:episodic:revcap:" + rid
		shared := fmt.Sprintf(`REVIEW CAP %s kafka broker under-replicated partitions error during rolling restart incident`, rid)
		var candID string
		for i := 0; i < 4; i++ {
			eb := mustPOST(t, hc, base+"/v1/advisory-episodes",
				fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q,"rev-%d"]}`, shared, tag, i), http.StatusCreated)
			var ep struct{ ID string `json:"id"` }
			_ = json.Unmarshal([]byte(eb), &ep)
			db := mustPOST(t, hc, base+"/v1/episodes/distill", fmt.Sprintf(`{"episode_id":%q}`, ep.ID), http.StatusOK)
			var dr struct {
				Candidates []struct{ CandidateID string `json:"candidate_id"` } `json:"candidates"`
			}
			_ = json.Unmarshal([]byte(db), &dr)
			if len(dr.Candidates) < 1 {
				t.Fatalf("iter %d: %s", i, db)
			}
			candID = dr.Candidates[0].CandidateID
		}
		rb := mustGET(t, hc, base+"/v1/curation/candidates/"+candID+"/review")
		n := jsonArrayLen(t, rb, "supporting_episodes")
		if n != 3 {
			t.Fatalf("supporting_episodes cap want 3 got %d body=%s", n, truncateEpisodicBody(rb))
		}
		var rev struct {
			Explanation string `json:"explanation"`
		}
		_ = json.Unmarshal([]byte(rb), &rev)
		if !strings.Contains(rev.Explanation, "4 supporting") {
			t.Fatalf("explanation should mention merged support count 4: %q", rev.Explanation)
		}
		episodicProofLog(t, "review_cap", "get_review", "pass", fmt.Sprintf("len=%d", n))
	})

	t.Run("promotion_auto_promote_disabled_403", func(t *testing.T) {
		out := mustPOST(t, hc, base+"/v1/curation/auto-promote", `{}`, http.StatusForbidden)
		if strings.TrimSpace(out) == "" {
			t.Fatal("auto-promote disabled: expected JSON error body")
		}
		episodicProofLog(t, "promotion_pressure", "auto_promote_off", "pass", "403")
	})

	t.Run("promotion_auto_promote_enabled_materializes_eligible", func(t *testing.T) {
		base2, cleanup2 := episodicProofServer(t, dsn, func(cfg *app.Config) {
			cfg.Promotion.AutoPromote = true
			cfg.Promotion.AutoMinSupportCount = 2
			cfg.Promotion.AutoMinSalience = 0.5
			cfg.Promotion.AutoAllowedKinds = []string{"failure", "pattern"}
		})
		defer cleanup2()
		tag := "proof:episodic:autop:" + uuid.New().String()[:8]
		shared := "failure error timeout when connecting to redis duplicate charge autop proof " + tag
		for i := 0; i < 2; i++ {
			eb := mustPOST(t, hc, base2+"/v1/advisory-episodes",
				fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q]}`, shared, tag), http.StatusCreated)
			var ep struct{ ID string `json:"id"` }
			_ = json.Unmarshal([]byte(eb), &ep)
			_ = mustPOST(t, hc, base2+"/v1/episodes/distill", fmt.Sprintf(`{"episode_id":%q}`, ep.ID), http.StatusOK)
		}
		_ = mustPOST(t, hc, base2+"/v1/curation/auto-promote", `{}`, http.StatusOK)
		rc := mustPOST(t, hc, base2+"/v1/recall/compile",
			fmt.Sprintf(`{"retrieval_query":%q,"tags":[%q],"max_per_kind":8,"max_total":40}`, shared, tag), http.StatusOK)
		if !strings.Contains(rc, "redis") || !strings.Contains(rc, "timeout") {
			t.Fatalf("after auto-promote, recall should surface canonical failure text; got %s", truncateEpisodicBody(rc))
		}
		episodicProofLog(t, "promotion_pressure", "auto_promote_on", "pass", tag)
	})

	t.Run("entity_overlap_unrelated_summary_not_dominant", func(t *testing.T) {
		rid := uuid.New().String()[:8]
		tag := "proof:episodic:entnoise:" + rid
		toka := `ENTITYNOISE-A-` + rid + ` payment refund queue deadlock error stalled payouts for merchants`
		tokb := `ENTITYNOISE-B-` + rid + ` shipping label printer firmware crash error delayed warehouse outbound`
		_ = mustPOST(t, hc, base+"/v1/advisory-episodes",
			fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q],"entities":["shared-ent-%s"]}`, toka, tag, rid), http.StatusCreated)
		_ = mustPOST(t, hc, base+"/v1/advisory-episodes",
			fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q],"entities":["shared-ent-%s"]}`, tokb, tag, rid), http.StatusCreated)
		q := fmt.Sprintf(`{"query":"entitynoise-a %s payment refund queue deadlock","tags":[%q],"entities":["shared-ent-%s"],"max_results":5}`,
			rid, tag, rid)
		sim := mustPOST(t, hc, base+"/v1/advisory-episodes/similar", q, http.StatusOK)
		var wrap struct {
			Cases []struct {
				Summary string  `json:"summary"`
				Score   float64 `json:"resemblance_score"`
			} `json:"advisory_similar_cases"`
		}
		_ = json.Unmarshal([]byte(sim), &wrap)
		if len(wrap.Cases) < 1 || !strings.Contains(wrap.Cases[0].Summary, "ENTITYNOISE-A") {
			t.Fatalf("top similar case should be lexically aligned episode A: %s", truncateEpisodicBody(sim))
		}
		for _, c := range wrap.Cases {
			if strings.Contains(c.Summary, "ENTITYNOISE-B") && c.Score >= wrap.Cases[0].Score {
				t.Fatalf("unrelated B should rank below A on A-tuned query (A=%.4f B=%.4f): %s",
					wrap.Cases[0].Score, c.Score, truncateEpisodicBody(sim))
			}
		}
		episodicProofLog(t, "entity_noise", "similar", "pass", fmt.Sprintf("top_score=%.3f", wrap.Cases[0].Score))
	})

	t.Run("historical_occurred_at_found_despite_recent_ingest", func(t *testing.T) {
		rid := uuid.New().String()[:8]
		tag := "proof:episodic:hist:" + rid
		summary := `HIST OCC ` + rid + ` ldap replication lag error caused stale group membership during cutover`
		_ = mustPOST(t, hc, base+"/v1/advisory-episodes",
			fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q],"occurred_at":"1995-11-05T08:00:00Z"}`, summary, tag), http.StatusCreated)
		q := fmt.Sprintf(`{"query":"hist occ %s ldap replication lag","tags":[%q],"occurred_after":"1995-01-01T00:00:00Z","occurred_before":"1995-12-31T23:59:59Z","max_results":5}`, rid, tag)
		sim := mustPOST(t, hc, base+"/v1/advisory-episodes/similar", q, http.StatusOK)
		if !strings.Contains(sim, "HIST OCC") {
			t.Fatalf("similar should use occurred_at not ingest time: %s", truncateEpisodicBody(sim))
		}
		episodicProofLog(t, "time_skew", "historical_window", "pass", rid)
	})

	t.Run("duplicate_entities_normalized_ingest", func(t *testing.T) {
		rid := uuid.New().String()[:8]
		tag := "proof:episodic:dupent:" + rid
		body := fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q],"entities":["Alpha","alpha","ALPHA","beta"]}`,
			`DUP ENT `+rid+` ci runner disk full error failed deploy artifact promotion`, tag)
		out := mustPOST(t, hc, base+"/v1/advisory-episodes", body, http.StatusCreated)
		if !strings.Contains(out, `"entities"`) || (!strings.Contains(out, `"alpha"`) && !strings.Contains(out, "alpha")) {
			t.Fatalf("expected entities array with normalized values in response: %s", truncateEpisodicBody(out))
		}
		episodicProofLog(t, "backward_compat", "duplicate_entities", "pass", rid)
	})

	t.Run("soak_recall_stable_three_iterations", func(t *testing.T) {
		rid := uuid.New().String()[:8]
		tag := "proof:episodic:soakrec:" + rid
		summary := `SOAK REC ` + rid + ` feature flag evaluation order bug caused wrong rollout cohort incident`
		eb := mustPOST(t, hc, base+"/v1/advisory-episodes",
			fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q]}`, summary, tag), http.StatusCreated)
		var ep struct{ ID string `json:"id"` }
		_ = json.Unmarshal([]byte(eb), &ep)
		db := mustPOST(t, hc, base+"/v1/episodes/distill", fmt.Sprintf(`{"episode_id":%q}`, ep.ID), http.StatusOK)
		var dr struct {
			Candidates []struct{ CandidateID string `json:"candidate_id"` } `json:"candidates"`
		}
		_ = json.Unmarshal([]byte(db), &dr)
		if len(dr.Candidates) < 1 {
			t.Fatal(db)
		}
		cid := dr.Candidates[0].CandidateID
		_ = mustPOST(t, hc, base+"/v1/curation/candidates/"+cid+"/materialize", `{}`, http.StatusCreated)
		q := fmt.Sprintf(`{"retrieval_query":%q,"tags":[%q],"max_per_kind":8,"max_total":40}`, summary, tag)
		var last string
		for i := 0; i < 3; i++ {
			rb := mustPOST(t, hc, base+"/v1/recall/compile", q, http.StatusOK)
			if last != "" && rb != last {
				t.Fatalf("recall drift on iteration %d", i)
			}
			last = rb
		}
		episodicProofLog(t, "soak_recall", "compile_loop", "pass", "3x stable")
	})
}

func episodicProofServer(t *testing.T, dsn string, tweak func(*app.Config)) (baseURL string, cleanup func()) {
	t.Helper()
	cfg, err := app.LoadConfig(proofIntegrationConfigPath(t))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Postgres.DSN = dsn
	cfg.Similarity.Enabled = true
	cfg.Similarity.MinResemblance = 0.05
	cfg.Distillation.Enabled = true
	cfg.Enforcement.MinBindingAuthority = 3
	if tweak != nil {
		tweak(cfg)
	}
	container, err := app.Boot(cfg)
	if err != nil {
		t.Fatalf("boot: %v", err)
	}
	rtr, err := apiserver.NewRouter(cfg, container)
	if err != nil {
		_ = container.DB.Close()
		t.Fatalf("router: %v", err)
	}
	srv := httptest.NewServer(rtr)
	return srv.URL, func() {
		srv.Close()
		container.DB.Close()
	}
}

func episodicProofLog(t *testing.T, scenario, phase, status, detail string) {
	t.Helper()
	d := strings.TrimSpace(detail)
	if d == "" {
		d = "-"
	}
	t.Logf("[EPISODIC PROOF] scenario=%s phase=%s status=%s detail=%s", scenario, phase, status, d)
}

func truncateEpisodicBody(s string) string {
	if len(s) <= 600 {
		return s
	}
	return s[:600] + "..."
}

func mustPOST(t *testing.T, hc *http.Client, urlStr, jsonBody string, want int) string {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, urlStr, strings.NewReader(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := hc.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != want {
		t.Fatalf("POST %s want %d got %d body=%s", urlStr, want, resp.StatusCode, string(b))
	}
	return string(b)
}

func mustGET(t *testing.T, hc *http.Client, urlStr string) string {
	t.Helper()
	resp, err := hc.Get(urlStr)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s want 200 got %d body=%s", urlStr, resp.StatusCode, string(b))
	}
	return string(b)
}

func mustDistillInline(t *testing.T, hc *http.Client, base, summary, tag string) {
	t.Helper()
	body, _ := json.Marshal(map[string]interface{}{
		"summary": summary,
		"tags":    []string{tag},
	})
	_ = mustPOST(t, hc, base+"/v1/episodes/distill", string(body), http.StatusOK)
}

// gjsonString is a tiny JSON helper for flat decision field (no new deps).
func gjsonString(jsonText, key string) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(jsonText), &m); err != nil {
		return ""
	}
	raw, ok := m[key]
	if !ok {
		return ""
	}
	var s string
	_ = json.Unmarshal(raw, &s)
	return s
}

// jsonArrayLen returns the length of a top-level JSON array field, or -1 if missing or not an array.
func jsonArrayLen(t *testing.T, body, key string) int {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("json: %v", err)
	}
	raw, ok := m[key]
	if !ok {
		return -1
	}
	var a []json.RawMessage
	if err := json.Unmarshal(raw, &a); err != nil {
		return -1
	}
	return len(a)
}
