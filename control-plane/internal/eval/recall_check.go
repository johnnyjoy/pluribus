package eval

import (
	"context"
	"strings"

	"control-plane/internal/memory"
	"control-plane/internal/recall"
)

type evalMemorySearcher struct {
	objs []memory.MemoryObject
}

func (f *evalMemorySearcher) Search(ctx context.Context, req memory.SearchRequest) ([]memory.MemoryObject, error) {
	return f.objs, nil
}

func (f *evalMemorySearcher) SearchMemories(ctx context.Context, req memory.MemoriesSearchRequest) ([]memory.MemoryObject, error) {
	// For evaluation, we keep the harness deterministic by returning the same memory pool for any token/query.
	return f.objs, nil
}

// RunRecall runs the explicit arm only (Service.Compile), for backward compatibility with single-path callers.
func RunRecall(s Scenario, writes []memory.MemoryObject) (*recall.RecallBundle, error) {
	ctx := context.Background()
	svc := newEvalRecallService(writes)
	return runScenarioExplicit(ctx, svc, s)
}

func runScenarioExplicit(ctx context.Context, svc *recall.Service, s Scenario) (*recall.RecallBundle, error) {
	req := buildCompileRequest(s)
	return svc.Compile(ctx, req)
}

func runScenarioTriggered(ctx context.Context, svc *recall.Service, s Scenario) (*recall.RecallBundle, *recall.TriggerMetadata, error) {
	req := buildCompileRequest(s)
	return svc.CompileTriggered(ctx, req)
}

func ValidateRecall(s Scenario, b *recall.RecallBundle) CheckResult {
	var details []string
	lookup := map[string][]string{
		"continuity":  toStatements(b.Continuity),
		"constraints": toStatements(b.Constraints),
		"experience":  toStatements(b.Experience),
	}
	for _, req := range s.RecallExpectations.MustInclude {
		parts := strings.SplitN(req, "::", 2)
		if len(parts) != 2 {
			details = append(details, "invalid must_include format: "+req)
			continue
		}
		bucket := strings.TrimSpace(parts[0])
		statement := strings.TrimSpace(parts[1])
		if !containsNormalized(lookup[bucket], statement) {
			details = append(details, "missing in "+bucket+": "+statement)
		}
	}
	for _, req := range s.RecallExpectations.MustBeFirst {
		parts := strings.SplitN(req, "::", 2)
		if len(parts) != 2 {
			details = append(details, "invalid must_be_first format: "+req)
			continue
		}
		bucket := strings.TrimSpace(parts[0])
		want := strings.TrimSpace(parts[1])
		got := firstStatementInBucket(b, bucket)
		if got == "" {
			details = append(details, "must_be_first empty bucket: "+bucket)
			continue
		}
		if !strings.Contains(strings.ToLower(got), strings.ToLower(want)) {
			details = append(details, "must_be_first "+bucket+": want "+want+", got "+got)
		}
	}
	return CheckResult{Pass: len(details) == 0, Details: details}
}

func firstStatementInBucket(b *recall.RecallBundle, bucket string) string {
	switch bucket {
	case "continuity":
		if len(b.Continuity) == 0 {
			return ""
		}
		return b.Continuity[0].Statement
	case "constraints":
		if len(b.Constraints) == 0 {
			return ""
		}
		return b.Constraints[0].Statement
	case "experience":
		if len(b.Experience) == 0 {
			return ""
		}
		return b.Experience[0].Statement
	default:
		return ""
	}
}

func toStatements(items []recall.MemoryItem) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Statement)
	}
	return out
}

func ptrWeights(w recall.RankingWeights) *recall.RankingWeights { return &w }
