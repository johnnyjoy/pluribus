package migrate

import (
	"strings"
	"testing"
)

func TestSplitStatements_simple(t *testing.T) {
	sql := `
-- header
CREATE TABLE a (id INT);
SELECT 1;
`
	got := splitStatements(sql)
	if len(got) != 2 {
		t.Fatalf("got %d statements: %v", len(got), got)
	}
	if !strings.Contains(got[0], "CREATE TABLE") {
		t.Fatal(got[0])
	}
	if !strings.Contains(got[1], "SELECT 1") {
		t.Fatal(got[1])
	}
}

func TestSplitStatements_emptyCommentOnly(t *testing.T) {
	got := splitStatements(stripSQLLineComments("-- only\n"))
	if len(got) != 0 {
		t.Fatal(got)
	}
}

func TestStripComments_semicolonInComment(t *testing.T) {
	const sql = `-- x
ALTER TABLE t ADD COLUMN a INT;
-- status already supports any TEXT; use 'superseded' and 'archived'.
`
	stripped := stripSQLLineComments(sql)
	if strings.Contains(stripped, "use 'superseded'") {
		t.Fatal("comment body should be removed")
	}
	stmts := splitStatements(stripped)
	if len(stmts) != 1 || !strings.Contains(stmts[0], "ALTER TABLE") {
		t.Fatalf("got %#v", stmts)
	}
}
