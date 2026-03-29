package ingest

import (
	"testing"
)

func TestNormalizeFactToken_trimAndLowerASCII(t *testing.T) {
	got := NormalizeFactToken("  Hello WORLD  ")
	if got != "hello world" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeFactToken_nfcComposed(t *testing.T) {
	// e + combining acute → single é in NFC
	decomposed := "caf\u0065\u0301" // e + combining acute
	got := NormalizeFactToken(decomposed)
	// NFC of café lowercased
	if got != "café" && got != "cafe\u0301" {
		// after NFC should be single char é
		if len(got) < 4 {
			t.Fatalf("unexpected %q len %d", got, len(got))
		}
	}
	// Stable hash check: same semantic input normalizes same way
	a := NormalizeFactToken("caf\u0065\u0301")
	b := NormalizeFactToken("café")
	if a != b {
		t.Fatalf("NFC mismatch: %q vs %q", a, b)
	}
}

func TestNormalizedFactHash_stable(t *testing.T) {
	h1 := normalizedFactHash("a", "b", "c")
	h2 := normalizedFactHash("a", "b", "c")
	if h1 != h2 {
		t.Fatal("hash not deterministic")
	}
	h3 := normalizedFactHash("a", "b", "d")
	if h1 == h3 {
		t.Fatal("hash should differ for different object")
	}
}
