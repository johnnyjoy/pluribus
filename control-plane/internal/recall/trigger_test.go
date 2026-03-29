package recall

import "testing"

func TestDetectTriggers_risk(t *testing.T) {
	in := TriggerInput{
		ProposalText: "We will deploy the migration to production on Friday",
		TaskTitle:    "Ship billing",
		TargetGoal:   "Reduce downtime",
	}
	got := DetectTriggers(in, 4)
	if len(got) < 1 || got[0].Kind != TriggerKindRisk {
		t.Fatalf("expected risk trigger, got %+v", got)
	}
}

func TestDetectTriggers_decision(t *testing.T) {
	in := TriggerInput{
		ProposalText: "We should choose between Postgres and SQLite for the cache layer",
		TaskTitle:    "Pick storage backend",
		TargetGoal:   "Improve reliability",
	}
	got := DetectTriggers(in, 4)
	var saw bool
	for _, g := range got {
		if g.Kind == TriggerKindDecision {
			saw = true
			break
		}
	}
	if !saw {
		t.Fatalf("expected decision trigger, got %+v", got)
	}
}

func TestDetectTriggers_similarity_richContext(t *testing.T) {
	in := TriggerInput{
		ProposalText: "Processing inbound webhooks requires idempotency keys and retry queues for stable ingestion",
		TaskTitle:    "Integrate billing webhooks",
		TargetGoal:   "Normalize events before dispatch",
	}
	got := DetectTriggers(in, 4)
	var saw bool
	for _, g := range got {
		if g.Kind == TriggerKindSimilarity {
			saw = true
			break
		}
	}
	if !saw {
		t.Fatalf("expected similarity trigger for rich context, got %+v", got)
	}
}

func TestDetectTriggers_lowSignalEmpty(t *testing.T) {
	in := TriggerInput{ProposalText: "ok", TaskTitle: "x"}
	if got := DetectTriggers(in, 4); len(got) != 0 {
		t.Fatalf("expected no triggers, got %+v", got)
	}
}

func TestValidTriggerKind(t *testing.T) {
	if !ValidTriggerKind("risk") || !ValidTriggerKind("decision") || !ValidTriggerKind("similarity") {
		t.Fatal("expected valid kinds")
	}
	if ValidTriggerKind("other") {
		t.Fatal("expected invalid")
	}
}
