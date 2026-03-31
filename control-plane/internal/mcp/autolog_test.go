package mcp

import (
	"strings"
	"testing"
)

func TestAutoLogEpisodeIfRelevant_learningSignal(t *testing.T) {
	ok, r := AutoLogEpisodeIfRelevant(strings.Repeat("x", 10) + " deployment error during release")
	if !ok || r != "learning_signal" {
		t.Fatalf("got %v %q", ok, r)
	}
}

func TestAutoLogEpisodeIfRelevant_repeatedBehaviorPhrase(t *testing.T) {
	s := strings.Repeat("word", 4) + " we tried the same approach again and " + strings.Repeat("z", 12)
	ok, r := AutoLogEpisodeIfRelevant(s)
	if !ok || r != "repeated_behavior" {
		t.Fatalf("got %v %q", ok, r)
	}
}

func TestAutoLogEpisodeIfRelevant_repeatedToken(t *testing.T) {
	s := "superlongtoken superlongtoken superlongtoken superlongtoken filler text here"
	ok, r := AutoLogEpisodeIfRelevant(s)
	if !ok || r != "repeated_token" {
		t.Fatalf("got %v %q", ok, r)
	}
}

func TestAutoLogEpisodeIfRelevant_tooShort(t *testing.T) {
	ok, r := AutoLogEpisodeIfRelevant("short")
	if ok || r != "too_short" {
		t.Fatalf("got %v %q", ok, r)
	}
}
