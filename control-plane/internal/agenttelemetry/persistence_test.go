package agenttelemetry

import (
	"testing"
)

func TestHostileTelemetryPersistenceCases(t *testing.T) {
	cases, err := LoadPersistenceCases("")
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) < 28 {
		t.Fatalf("need >=28 cases, got %d", len(cases))
	}
	for _, c := range cases {
		t.Run(c.CaseID, func(t *testing.T) {
			svc := NewTestService()
			iface := c.Interface
			if c.ExpectedMCPRESTParity {
				restFr, err := RunRESTFlow(svc, c)
				if err != nil {
					t.Fatalf("rest: %v", err)
				}
				mcpSvc := NewTestService()
				mcpFr, err := RunMCPFlow(mcpSvc, c)
				if err != nil {
					t.Fatalf("mcp: %v", err)
				}
				if restFr.EvalPassed != mcpFr.EvalPassed {
					t.Fatalf("parity eval: rest=%v mcp=%v", restFr.EvalPassed, mcpFr.EvalPassed)
				}
				return
			}
			var fr FlowResult
			var err error
			if iface == "mcp" {
				fr, err = RunMCPFlow(svc, c)
			} else {
				fr, err = RunRESTFlow(svc, c)
			}
			if err != nil {
				t.Fatal(err)
			}
			if c.RejectStep == "unknown_session" && !fr.Rejected {
				t.Fatal("expected unknown session rejection")
			}
			if c.RejectStep == "missing_citation" && !fr.Rejected {
				t.Fatal("expected missing citation rejection")
			}
			if c.RejectStep == "missing_output" && !fr.Rejected {
				t.Fatal("expected missing output rejection")
			}
			if c.SelfReportObedient && !fr.SelfReportRejected {
				t.Fatal("expected self-report rejection flag")
			}
			if c.RecallOnly && !fr.RecallOnlyNoPositive {
				t.Fatal("recall-only should not generate positive utility")
			}
			if err := VerifyCase(c, fr); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestExternalMemoryIDAccepted(t *testing.T) {
	svc := NewTestService()
	cases, _ := LoadPersistenceCases("")
	var c PersistenceCase
	for _, x := range cases {
		if x.CaseID == "telemetry_rejects_malformed_memory_id_or_marks_external_test" {
			c = x
			break
		}
	}
	fr, err := RunRESTFlow(svc, c)
	if err != nil {
		t.Fatal(err)
	}
	if !fr.SessionPersisted || !fr.EvaluationPersisted {
		t.Fatalf("external memory flow failed: %+v", fr)
	}
}
