package utility

import (
	"context"
	"testing"

	"control-plane/pkg/api"

	"github.com/google/uuid"
)

type fakeMemoryReader struct {
	exists bool
	kind   api.MemoryKind
	app    api.Applicability
	status api.Status
}

func (f *fakeMemoryReader) GetByID(ctx context.Context, id uuid.UUID) (bool, api.MemoryKind, api.Applicability, api.Status, error) {
	if !f.exists {
		return false, "", "", "", nil
	}
	return true, f.kind, f.app, f.status, nil
}

func (f *fakeMemoryReader) SetStatusPending(ctx context.Context, id uuid.UUID) error {
	f.status = api.StatusPending
	return nil
}

func TestRecordFeedbackWrongRequiresReason(t *testing.T) {
	svc := &Service{
		Repo:   &Repo{},
		Memory: &fakeMemoryReader{exists: true},
	}
	_, err := svc.RecordFeedback(context.Background(), uuid.New(), FeedbackRequest{EventType: EventWrong})
	if err != ErrReasonRequired {
		t.Fatalf("want reason required, got %v", err)
	}
}

func TestRecordFeedbackInvalidType(t *testing.T) {
	svc := &Service{Repo: &Repo{}, Memory: &fakeMemoryReader{exists: true}}
	_, err := svc.RecordFeedback(context.Background(), uuid.New(), FeedbackRequest{EventType: "bogus"})
	if err != ErrInvalidEventType {
		t.Fatalf("want invalid type, got %v", err)
	}
}

func TestRecordFeedbackMemoryNotFound(t *testing.T) {
	svc := &Service{Repo: &Repo{}, Memory: &fakeMemoryReader{exists: false}}
	_, err := svc.RecordFeedback(context.Background(), uuid.New(), FeedbackRequest{EventType: EventHelpful})
	if err != ErrMemoryNotFound {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestRecordFeedbackWrongDecreasesMoreThanHarmful(t *testing.T) {
	if EventWeight(EventWrong) >= EventWeight(EventHarmful) {
		t.Fatal("wrong should be more negative than harmful")
	}
}

func TestNegativeFeedbackRequiresReason(t *testing.T) {
	for _, et := range []string{EventHarmful, EventWrong, EventOutdated} {
		if err := ValidateFeedbackRequest(FeedbackRequest{EventType: et}); err != ErrReasonRequired {
			t.Fatalf("%s needs reason", et)
		}
	}
}

func TestUtilityScoreBounded(t *testing.T) {
	if ApplyScoreDelta(9, 5) != MaxUtilityScore {
		t.Fatal("high bound")
	}
	if ApplyScoreDelta(-9, -5) != MinUtilityScore {
		t.Fatal("low bound")
	}
}
