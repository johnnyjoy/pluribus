package memory

import (
	"encoding/json"
	"testing"
)

func TestMergeSalienceForContext_distinct(t *testing.T) {
	a := []byte(`{"other":1}`)
	out1, err := mergeSalienceForContext(a, "abc12345")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(out1, &m); err != nil {
		t.Fatal(err)
	}
	var sal SaliencePayload
	if err := json.Unmarshal(m["salience"], &sal); err != nil {
		t.Fatal(err)
	}
	if sal.DistinctContexts != 1 || len(sal.ContextHashes) != 1 || sal.ContextHashes[0] != "abc12345" {
		t.Fatalf("first merge: %+v", sal)
	}
	out2, err := mergeSalienceForContext(out1, "abc12345")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(out2, &m); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(m["salience"], &sal); err != nil {
		t.Fatal(err)
	}
	if sal.DistinctContexts != 1 {
		t.Fatalf("duplicate context should not increment: %+v", sal)
	}
	out3, err := mergeSalienceForContext(out2, "fedcba09")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(out3, &m); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(m["salience"], &sal); err != nil {
		t.Fatal(err)
	}
	if sal.DistinctContexts != 2 {
		t.Fatalf("second distinct context: %+v", sal)
	}
}

func TestMergeSaliencePayload_distinctAgents(t *testing.T) {
	a := []byte(`{"other":1}`)
	out1, err := mergeSaliencePayload(a, "", "aa11bb22")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(out1, &m); err != nil {
		t.Fatal(err)
	}
	var sal SaliencePayload
	if err := json.Unmarshal(m["salience"], &sal); err != nil {
		t.Fatal(err)
	}
	if sal.DistinctAgents != 1 || len(sal.AgentHashes) != 1 || sal.AgentHashes[0] != "aa11bb22" {
		t.Fatalf("first agent merge: %+v", sal)
	}
	out2, err := mergeSaliencePayload(out1, "", "cc33dd44")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(out2, &m); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(m["salience"], &sal); err != nil {
		t.Fatal(err)
	}
	if sal.DistinctAgents != 2 {
		t.Fatalf("second distinct agent: %+v", sal)
	}
}
