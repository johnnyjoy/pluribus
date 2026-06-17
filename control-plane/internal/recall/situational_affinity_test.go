package recall

import (
	"strings"
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

func TestSituationAffinityScore_pluribusQuery(t *testing.T) {
	query := "Hostile review of Pluribus product architecture integrations ops documentation gaps"
	in := SituationAffinityInput{SituationQuery: query}

	onguard := SituationAffinityScore(
		"Archived onguard-guide-generic-hostile-review: hostile VAN found campus guide failed generic install",
		nil, in,
	)
	pluribus := SituationAffinityScore(
		"Pluribus Claude Code plugin SessionStart loads wakeup_context; integrations matrix documents MCP hooks",
		[]string{"pluribus", "integration"}, in,
	)
	if pluribus <= onguard {
		t.Fatalf("pluribus affinity %v should exceed onguard %v for pluribus query", pluribus, onguard)
	}
}

func TestSituationAffinityScore_repoRootBoost(t *testing.T) {
	in := SituationAffinityInput{
		SituationQuery: "architecture review",
		RepoRoot:       "/projects/pluribus",
	}
	withRepo := SituationAffinityScore("Pluribus recall compiler ranking", []string{"pluribus"}, in)
	in.RepoRoot = ""
	withoutRepo := SituationAffinityScore("Pluribus recall compiler ranking", []string{"pluribus"}, in)
	if withRepo < withoutRepo {
		t.Fatalf("repo root should not reduce affinity: with=%v without=%v", withRepo, withoutRepo)
	}
}

func TestFailureOverlap_requiresRequestTags(t *testing.T) {
	w := DefaultRankingWeights()
	now := time.Now()
	req := ScoreRequest{
		SituationQuery: "hostile review of pluribus integrations",
	}
	fail := memory.MemoryObject{
		ID:        uuid.New(),
		Kind:      api.MemoryKindFailure,
		Statement: "hostile VAN review failed for onguard guide",
		Authority: 2,
		UpdatedAt: now,
		Tags:      []string{"onguard"},
	}
	sNoTags := scoreAt(fail, req, w, 10, now)
	req.Tags = []string{"onguard"}
	sWithTags := scoreAt(fail, req, w, 10, now)
	if sWithTags <= sNoTags {
		t.Fatalf("failure overlap should apply only with overlapping request tags: noTags=%v withTags=%v", sNoTags, sWithTags)
	}
}

func TestScoreAndSort_hostilePluribusReviewQuery(t *testing.T) {
	query := "Hostile review of Pluribus product architecture integrations ops documentation gaps"
	w := DefaultRankingWeights()
	now := time.Now()
	req := ScoreRequest{SituationQuery: query, RepoRoot: "/projects/pluribus"}

	onguardFail := memory.MemoryObject{
		ID:        uuid.New(),
		Kind:      api.MemoryKindFailure,
		Statement: "Archived onguard-guide-generic-hostile-review: hostile VAN found campus guide failed generic install",
		Authority: 2,
		UpdatedAt: now,
	}
	pluribusFail := memory.MemoryObject{
		ID:        uuid.New(),
		Kind:      api.MemoryKindFailure,
		Statement: "Pluribus recall returned OnGuard memories for a Pluribus architecture review query; situational affinity ranking added",
		Authority: 2,
		UpdatedAt: now,
		Tags:      []string{"pluribus", "recall"},
	}
	pluribusConstraint := memory.MemoryObject{
		ID:        uuid.New(),
		Kind:      api.MemoryKindConstraint,
		Statement: "Pluribus global memory pool uses situational ranking not project partitions; recall before multi-step work",
		Authority: 3,
		UpdatedAt: now,
		Tags:      []string{"pluribus", "doctrine"},
	}

	scored := ScoreAndSortWithReason([]memory.MemoryObject{onguardFail, pluribusFail, pluribusConstraint}, req, w, 10)
	if len(scored) < 2 {
		t.Fatal("expected scored results")
	}
	top := scored[0].Object.Statement
	if top != pluribusConstraint.Statement && top != pluribusFail.Statement {
		t.Fatalf("top result should be pluribus memory, got: %q (reason=%s score=%v)", top, scored[0].Reason, scored[0].Score)
	}
}

// situationalAffinityEvalCases is the validation dataset for before/after ranking checks (Phase 5).
var situationalAffinityEvalCases = []struct {
	name       string
	query      string
	repoRoot   string
	tags       []string
	wantTopSub string
}{
	{
		name:       "pluribus_architecture",
		query:      "Pluribus architecture recall ranking integrations",
		repoRoot:   "/projects/pluribus",
		wantTopSub: "Pluribus",
	},
	{
		name:       "onguard_schema",
		query:      "OnGuard schema reference search fuse JavaScript",
		wantTopSub: "OnGuard",
	},
	{
		name:       "uda_database",
		query:      "UDA Database Driver cache PHP 8.2 validation",
		wantTopSub: "UDA",
	},
}

func TestSituationalAffinity_evalDataset(t *testing.T) {
	w := DefaultRankingWeights()
	now := time.Now()
	memories := []memory.MemoryObject{
		{ID: uuid.New(), Kind: api.MemoryKindFailure, Statement: "OnGuard schema reference search failed fuse_init JavaScript", Authority: 2, UpdatedAt: now, Tags: []string{"onguard"}},
		{ID: uuid.New(), Kind: api.MemoryKindFailure, Statement: "UDA Driver cache PHP 8.2 Docker validation passed PHPStan", Authority: 2, UpdatedAt: now, Tags: []string{"uda"}},
		{ID: uuid.New(), Kind: api.MemoryKindFailure, Statement: "Pluribus recall uses global pool with situational affinity not partitions", Authority: 2, UpdatedAt: now, Tags: []string{"pluribus"}},
	}
	for _, tc := range situationalAffinityEvalCases {
		t.Run(tc.name, func(t *testing.T) {
			req := ScoreRequest{
				SituationQuery: tc.query,
				RepoRoot:       tc.repoRoot,
				Tags:           tc.tags,
			}
			scored := ScoreAndSortWithReason(memories, req, w, 10)
			if len(scored) == 0 {
				t.Fatal("no scores")
			}
			top := scored[0].Object.Statement
			if !strings.Contains(strings.ToLower(top), strings.ToLower(tc.wantTopSub)) {
				t.Fatalf("want top containing %q, got %q (reason=%s)", tc.wantTopSub, top, scored[0].Reason)
			}
		})
	}
}
