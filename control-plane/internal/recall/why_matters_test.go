package recall

import "testing"

func TestBuildWhyMattersLine_constraint(t *testing.T) {
	s := BuildWhyMattersLine("constraint", "tag_match", 0, false)
	if s == "" {
		t.Fatal("empty")
	}
}

func TestSessionTagMatches(t *testing.T) {
	if !sessionTagMatches("abc", []string{"mcp:session:abc", "other"}) {
		t.Fatal("expected match")
	}
	if sessionTagMatches("abc", []string{"mcp:session:other"}) {
		t.Fatal("expected no match")
	}
}
