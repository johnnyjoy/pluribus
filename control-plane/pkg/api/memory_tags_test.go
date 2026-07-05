package api

import (
	"reflect"
	"testing"
)

func TestSituationTagsForSemanticConsolidate_excludesHygieneTags(t *testing.T) {
	got := SituationTagsForSemanticConsolidate([]string{
		TagEphemeral, TagProofScenario, "proof-multi-agent-abc",
	})
	want := []string{"proof-multi-agent-abc"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestSituationTagsForSemanticConsolidate_emptyWhenOnlyHygiene(t *testing.T) {
	got := SituationTagsForSemanticConsolidate([]string{TagEphemeral, TagProofScenario})
	if len(got) != 0 {
		t.Fatalf("got %v want empty", got)
	}
}

func TestHasDisposableAutomationTags(t *testing.T) {
	if !HasDisposableAutomationTags([]string{TagEphemeral, TagProofScenario}) {
		t.Fatal("expected disposable")
	}
	if HasDisposableAutomationTags([]string{TagProofScenario}) {
		t.Fatal("proof-scenario alone is not disposable")
	}
	if HasDisposableAutomationTags([]string{TagEphemeral}) {
		t.Fatal("ephemeral alone is not disposable pair")
	}
}
