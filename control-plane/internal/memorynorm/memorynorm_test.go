package memorynorm

import (
	"testing"
)

func TestPipelineVersion(t *testing.T) {
	if PipelineVersion == "" {
		t.Fatal("PipelineVersion must be set")
	}
}

func TestStatementCanonical(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "only_spaces", in: "   \t\n  ", want: ""},
		{name: "simple", in: "Hello World", want: "hello world"},
		{name: "collapse_spaces", in: "  a   b\tc  ", want: "a b c"},
		{name: "unicode_mixed_case", in: "Straße Wide", want: "straße wide"},
		{name: "nfc_input", in: "caf\u0065\u0301", want: "café"},                   // e + combining acute → café NFC then fold
		{name: "cjk", in: "  你好  世界 ", want: "你好 世界"},
		{name: "punctuation_preserved", in: "Use Postgres; not SQLite.", want: "use postgres; not sqlite."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StatementCanonical(tt.in)
			if got != tt.want {
				t.Fatalf("StatementCanonical(%q) = %q; want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestStatementCanonical_deterministic(t *testing.T) {
	a := "  The   DATABASE  must  be  Postgres  "
	b := "the database must be postgres"
	if StatementCanonical(a) != StatementCanonical(b) {
		t.Fatalf("expected equal canonical forms")
	}
}

func TestStatementKey(t *testing.T) {
	if StatementKey("") != "" {
		t.Fatal("empty canonical => empty key")
	}
	k1 := StatementKey("Hello")
	k2 := StatementKey("  hello  ")
	if k1 == "" || k2 == "" {
		t.Fatal("non-empty input should yield key")
	}
	if k1 != k2 {
		t.Fatalf("same canonical => same key: %q vs %q", k1, k2)
	}
	if k1 == StatementKey("goodbye") {
		t.Fatal("different statement should differ")
	}
	// fixed length
	if len(k1) != 64 {
		t.Fatalf("sha256 hex length: got %d", len(k1))
	}
}

