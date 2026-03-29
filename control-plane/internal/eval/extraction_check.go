package eval

import (
	"strings"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

func RunInitialScenarioExtraction(s Scenario) []memory.MemoryObject {
	all := strings.Join([]string{s.Context, s.Trap}, "\n")
	lines := strings.Split(all, "\n")
	var out []memory.MemoryObject
	appendKind := func(kind api.MemoryKind, stmt string) {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			return
		}
		out = append(out, memory.MemoryObject{
			ID:        uuid.New(),
			Kind:      kind,
			Statement: stmt,
			Authority: 8,
			Status:    api.StatusActive,
			UpdatedAt: time.Now(),
			CreatedAt: time.Now(),
		})
	}
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		switch {
		case strings.HasPrefix(ln, "STATE:"):
			appendKind(api.MemoryKindState, strings.TrimSpace(strings.TrimPrefix(ln, "STATE:")))
		case strings.HasPrefix(ln, "DECISION:"):
			appendKind(api.MemoryKindDecision, strings.TrimSpace(strings.TrimPrefix(ln, "DECISION:")))
		case strings.HasPrefix(ln, "FAILURE:"):
			appendKind(api.MemoryKindFailure, strings.TrimSpace(strings.TrimPrefix(ln, "FAILURE:")))
		case strings.HasPrefix(ln, "PATTERN:"):
			appendKind(api.MemoryKindPattern, strings.TrimSpace(strings.TrimPrefix(ln, "PATTERN:")))
		case strings.HasPrefix(ln, "CONSTRAINT:"):
			appendKind(api.MemoryKindConstraint, strings.TrimSpace(strings.TrimPrefix(ln, "CONSTRAINT:")))
		}
	}
	return out
}

func ValidateExtraction(s Scenario, writes []memory.MemoryObject) CheckResult {
	allowed := map[api.MemoryKind]bool{
		api.MemoryKindState:      true,
		api.MemoryKindDecision:   true,
		api.MemoryKindFailure:    true,
		api.MemoryKindPattern:    true,
		api.MemoryKindConstraint: true,
	}
	gotByKind := map[api.MemoryKind][]string{}
	var details []string
	for _, w := range writes {
		if !allowed[w.Kind] {
			details = append(details, "invalid kind written: "+string(w.Kind))
		}
		gotByKind[w.Kind] = append(gotByKind[w.Kind], w.Statement)
	}
	checkContains := func(kind api.MemoryKind, want []string) {
		got := gotByKind[kind]
		for _, ws := range want {
			if !containsNormalized(got, ws) {
				details = append(details, "missing "+string(kind)+": "+ws)
			}
		}
	}
	checkContains(api.MemoryKindState, s.ExpectedExtraction.State)
	checkContains(api.MemoryKindDecision, s.ExpectedExtraction.Decision)
	checkContains(api.MemoryKindFailure, s.ExpectedExtraction.Failure)
	checkContains(api.MemoryKindPattern, s.ExpectedExtraction.Pattern)
	checkContains(api.MemoryKindConstraint, s.ExpectedExtraction.Constraint)

	return CheckResult{Pass: len(details) == 0, Details: details}
}

func containsNormalized(got []string, want string) bool {
	wn := strings.ToLower(strings.TrimSpace(want))
	for _, g := range got {
		if strings.ToLower(strings.TrimSpace(g)) == wn {
			return true
		}
	}
	return false
}
