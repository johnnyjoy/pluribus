package lexical

import "testing"

func TestValidateProjectionTable(t *testing.T) {
	if err := ValidateProjectionTable("lexical_memory_projection"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateProjectionTable(""); err == nil {
		t.Fatal("want error for empty")
	}
	if err := ValidateProjectionTable("LexicalBad"); err == nil {
		t.Fatal("want error for uppercase")
	}
	if err := ValidateProjectionTable("drop table"); err == nil {
		t.Fatal("want error for invalid chars")
	}
}
