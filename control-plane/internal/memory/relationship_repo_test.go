package memory

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestRelationshipRepo_CreateRelationship_invalidType(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	r := &RelationshipRepo{DB: db}
	_, err = r.CreateRelationship(context.Background(), uuid.New(), uuid.New(), RelationshipType("nope"), "", "")
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}
