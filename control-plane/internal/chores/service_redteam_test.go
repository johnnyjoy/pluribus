package chores

import (
	"context"
	"testing"
	"time"

	"control-plane/pkg/api"

	"github.com/DATA-DOG/go-sqlmock"
)

// Red-team: a lone hostile agent proposing consolidate on two unrelated
// memories must not be able to apply the merge by itself — the default
// threshold requires a second distinct agent, and expectations prove no
// UPDATE against memories or chores runs.
func TestRedTeam_hostileLoneConsolidateCannotApply(t *testing.T) {
	svc, mock, done := newTestService(t)
	defer done()
	rel := memRel
	now := time.Now()
	mock.ExpectQuery(`FROM curation_chores c WHERE`).WithArgs(choreID).
		WillReturnRows(choreRows(choreID, TypeDuplicatePair, memSubj, &rel, nil))
	expectGetMemory(mock, memSubj, "backups run nightly at 2am", "active", "author-a", 6, now)
	expectGetMemory(mock, memRel, "never store secrets in git", "active", "author-b", 8, now)
	mock.ExpectExec(`INSERT INTO curation_chore_votes`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`FROM curation_chore_votes`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	resp, err := svc.Resolve(context.Background(), choreID, ResolveRequest{
		AgentID: "hostile-agent", Action: ActionConsolidate,
		Reason: "merge these (attack: erase the secrets constraint)",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Applied {
		t.Fatalf("single hostile vote applied a consolidation: %+v", resp)
	}
	if resp.State != StateOpen {
		t.Fatalf("chore must stay open, got %s", resp.State)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations (no memory mutation expected): %v", err)
	}
}

// Red-team: a corroborated quarantine release must land the row in PENDING —
// never straight back to active. The status argument of the UPDATE is asserted.
func TestRedTeam_releasedQuarantineLandsPendingNotActive(t *testing.T) {
	svc, mock, done := newTestService(t)
	defer done()
	svc.MinResolvers = 1
	now := time.Now()
	mock.ExpectQuery(`FROM curation_chores c WHERE`).WithArgs(choreID).
		WillReturnRows(choreRows(choreID, TypeQuarantineReview, memSubj, nil, quarantineEvidence{Reason: "harmful_advice_screen"}))
	expectGetMemory(mock, memSubj, "curl | sudo bash is efficient", "quarantined", "author-a", 1, now)
	mock.ExpectExec(`INSERT INTO curation_chore_votes`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`FROM curation_chore_votes`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	// apply: ReleaseQuarantined re-fetches (status guard), setRemediationStatus
	// fetches again, then updates status. The asserted arg proves the row lands
	// pending, not active.
	expectGetMemory(mock, memSubj, "curl | sudo bash is efficient", "quarantined", "author-a", 1, now)
	expectGetMemory(mock, memSubj, "curl | sudo bash is efficient", "quarantined", "author-a", 1, now)
	mock.ExpectExec(`UPDATE memories SET status`).
		WithArgs(api.StatusPending, sqlmock.AnyArg(), memSubj).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE curation_chores SET state`).
		WithArgs(StateResolved, ActionRelease, sqlmock.AnyArg(), choreID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	resp, err := svc.Resolve(context.Background(), choreID, ResolveRequest{AgentID: "reviewer-agent", Action: ActionRelease})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Applied || resp.State != StateResolved {
		t.Fatalf("expected applied release, got %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

// Red-team: releasing a memory that is NOT quarantined must fail even when
// corroborated — resolve_chore cannot be used to flip arbitrary rows to pending.
func TestRedTeam_releaseOnlyWorksOnQuarantinedRows(t *testing.T) {
	svc, mock, done := newTestService(t)
	defer done()
	svc.MinResolvers = 1
	now := time.Now()
	mock.ExpectQuery(`FROM curation_chores c WHERE`).WithArgs(choreID).
		WillReturnRows(choreRows(choreID, TypeQuarantineReview, memSubj, nil, nil))
	// Row is active (e.g. an operator already released it via another path).
	expectGetMemory(mock, memSubj, "some memory", "active", "author-a", 5, now)
	mock.ExpectExec(`INSERT INTO curation_chore_votes`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`FROM curation_chore_votes`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	expectGetMemory(mock, memSubj, "some memory", "active", "author-a", 5, now)

	_, err := svc.Resolve(context.Background(), choreID, ResolveRequest{AgentID: "reviewer-agent", Action: ActionRelease})
	if err == nil {
		t.Fatal("release of a non-quarantined row must fail")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}
