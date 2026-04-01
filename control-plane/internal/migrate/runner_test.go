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

func TestSplitStatements_dollarQuotedPlPgSQL(t *testing.T) {
	sql := `DO $$
BEGIN
  IF EXISTS (SELECT 1) THEN
    ALTER TABLE a RENAME TO b;
  END IF;
END $$;
ALTER TABLE c ADD d INT;
`
	got := splitStatements(sql)
	if len(got) != 2 {
		t.Fatalf("want 2 statements, got %d: %#v", len(got), got)
	}
	if !strings.Contains(got[0], "DO $$") || !strings.Contains(got[0], "END $$") {
		t.Fatalf("first stmt should be whole DO block: %s", got[0])
	}
	if !strings.Contains(got[1], "ALTER TABLE c") {
		t.Fatalf("second stmt: %s", got[1])
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
