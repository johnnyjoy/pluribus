package chores

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"control-plane/internal/contradiction"
	"control-plane/internal/memory"

	"github.com/google/uuid"
)

// DefaultMinResolvers requires two distinct agents before a chore applies.
const DefaultMinResolvers = 2

// Service coordinates chore listing, voting, and corroborated application.
type Service struct {
	Repo           *Repo
	Memory         *memory.Service
	Contradictions *contradiction.Repo
	// MinResolvers distinct agent hashes must agree before an action applies.
	// 0 means DefaultMinResolvers; operators may set 1 for a small hive.
	MinResolvers int
	// NearDup gates the embedding near-duplicate pair scan (curation-loop pass).
	NearDup NearDupConfig
}

// NearDupConfig configures the embedding near-duplicate pair scan.
type NearDupConfig struct {
	Enabled       bool
	MinSimilarity float64
	WindowDays    int
	Limit         int
}

// RunChorePass is the curation-loop entry point: sync review chores from
// contradictions/quarantine and, when enabled, scan for near-duplicate pairs.
func (s *Service) RunChorePass(ctx context.Context) (int, error) {
	opened, err := s.SyncReviewChores(ctx)
	if s.NearDup.Enabled {
		n, scanErr := s.ScanNearDuplicates(ctx, s.NearDup.MinSimilarity, s.NearDup.WindowDays, s.NearDup.Limit)
		opened += n
		if err == nil {
			err = scanErr
		}
	}
	return opened, err
}

func (s *Service) minResolvers() int {
	if s.MinResolvers > 0 {
		return s.MinResolvers
	}
	return DefaultMinResolvers
}

// List syncs review chores from their sources and returns open chores.
func (s *Service) List(ctx context.Context, limit int) ([]Chore, error) {
	// Best-effort sync so freshly detected contradictions/quarantines appear
	// without waiting for the background loop.
	_, _ = s.SyncReviewChores(ctx)
	return s.Repo.ListOpen(ctx, limit)
}

// NextOpen returns the oldest open chore, or nil when the pool is clean.
func (s *Service) NextOpen(ctx context.Context) (*Chore, error) {
	list, err := s.Repo.ListOpen(ctx, 1)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return &list[0], nil
}

// Resolve records one agent's vote and applies the action once min_resolvers
// distinct non-author agents agree on it.
func (s *Service) Resolve(ctx context.Context, choreID uuid.UUID, req ResolveRequest) (*ResolveResponse, error) {
	agentID := strings.TrimSpace(req.AgentID)
	if agentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}
	ch, err := s.Repo.Get(ctx, choreID)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, fmt.Errorf("chore not found")
	}
	if ch.State != StateOpen {
		return nil, fmt.Errorf("chore is %s", ch.State)
	}
	if !actionAllowed(ch.Type, req.Action) {
		return nil, fmt.Errorf("action %q not allowed for %s (use %s)",
			req.Action, ch.Type, strings.Join(AllowedActions[ch.Type], "|"))
	}

	subject, related, err := s.loadMemories(ctx, ch)
	if err != nil {
		return nil, err
	}

	voterHash := memory.AgentUsageKey(agentID)
	authorHashes := authorHashSet(subject, related)
	resp := &ResolveResponse{
		ChoreID:      ch.ID,
		MinResolvers: s.minResolvers(),
		State:        ch.State,
		Counted:      true,
	}
	if _, isAuthor := authorHashes[voterHash]; isAuthor {
		resp.Counted = false
		resp.Note = "vote recorded but not counted: the memory's own author cannot corroborate its curation"
	}

	inserted, err := s.Repo.InsertVote(ctx, ch.ID, agentID, voterHash, req.Action, req.Reason)
	if err != nil {
		return nil, err
	}
	resp.Recorded = inserted
	if !inserted {
		resp.Counted = false
		resp.Note = "this agent already voted on this chore; votes are immutable"
	}

	exclude := make([]string, 0, len(authorHashes))
	for h := range authorHashes {
		exclude = append(exclude, h)
	}
	votes, err := s.Repo.CountVotesForAction(ctx, ch.ID, req.Action, exclude)
	if err != nil {
		return nil, err
	}
	resp.VotesForAction = votes
	if votes < s.minResolvers() {
		return resp, nil
	}

	finalState, err := s.apply(ctx, ch, req.Action, subject, related)
	if err != nil {
		return nil, fmt.Errorf("chore corroborated but apply failed: %w", err)
	}
	if err := s.Repo.MarkResolved(ctx, ch.ID, req.Action, finalState, time.Now()); err != nil {
		return nil, err
	}
	resp.Applied = true
	resp.State = finalState
	return resp, nil
}

// loadMemories fetches the subject (and related, when set) rows so the
// resolver can compare authors and authority.
func (s *Service) loadMemories(ctx context.Context, ch *Chore) (subject, related *memory.MemoryObject, err error) {
	subject, err = s.Memory.Repo.GetByID(ctx, ch.SubjectMemoryID)
	if err != nil {
		return nil, nil, err
	}
	if subject == nil {
		return nil, nil, fmt.Errorf("subject memory not found")
	}
	if ch.RelatedMemoryID != nil {
		related, err = s.Memory.Repo.GetByID(ctx, *ch.RelatedMemoryID)
		if err != nil {
			return nil, nil, err
		}
		if related == nil {
			return nil, nil, fmt.Errorf("related memory not found")
		}
	}
	return subject, related, nil
}

func authorHashSet(memories ...*memory.MemoryObject) map[string]struct{} {
	out := make(map[string]struct{})
	for _, m := range memories {
		if m == nil {
			continue
		}
		if h := memory.AgentUsageKey(m.AgentID); h != "" {
			out[h] = struct{}{}
		}
	}
	return out
}

const choreSource = "curation_chore"

// apply performs the corroborated action. It only ever moves memories to
// reversible states (superseded, pending, deleted-tombstone); nothing an
// agent vote does can mint an active memory.
func (s *Service) apply(ctx context.Context, ch *Chore, action string, subject, related *memory.MemoryObject) (finalState string, err error) {
	reason := fmt.Sprintf("agent-corroborated chore %s (%s)", ch.ID, action)
	switch ch.Type {
	case TypeContradiction:
		recID, _ := s.contradictionRecordID(ch)
		switch action {
		case ActionKeepSubject, ActionKeepRelated:
			winner, loser := subject, related
			if action == ActionKeepRelated {
				winner, loser = related, subject
			}
			if winner == nil || loser == nil {
				return "", fmt.Errorf("contradiction chore missing a side")
			}
			if err := s.Memory.SupersedeMemory(ctx, winner.ID, loser.ID, reason, choreSource); err != nil {
				return "", err
			}
			if recID != uuid.Nil {
				if err := s.Contradictions.UpdateResolution(ctx, recID, contradiction.ResolutionDeprecated); err != nil {
					return "", err
				}
			}
		case ActionCoexist:
			if recID != uuid.Nil {
				if err := s.Contradictions.UpdateResolution(ctx, recID, contradiction.ResolutionNarrowException); err != nil {
					return "", err
				}
			}
		}
		return StateResolved, nil

	case TypeQuarantineReview:
		switch action {
		case ActionRelease:
			// Pending, never straight to active: activation still requires the
			// normal promotion path.
			if _, err := s.Memory.ReleaseQuarantined(ctx, subject.ID, reason); err != nil {
				return "", err
			}
		case ActionDelete:
			if _, err := s.Memory.SoftDelete(ctx, subject.ID, reason); err != nil {
				return "", err
			}
		}
		return StateResolved, nil

	case TypeDuplicatePair:
		switch action {
		case ActionConsolidate:
			if related == nil {
				return "", fmt.Errorf("duplicate chore missing related memory")
			}
			winner, loser := pickSurvivor(subject, related)
			if err := s.Memory.SupersedeMemory(ctx, winner.ID, loser.ID, reason, choreSource); err != nil {
				return "", err
			}
			return StateResolved, nil
		case ActionDistinct:
			// Keep both; the pair is remembered (any-state dedup) so it is
			// never offered again.
			return StateDismissed, nil
		}
	}
	return "", fmt.Errorf("unhandled chore action %s/%s", ch.Type, action)
}

// pickSurvivor keeps the higher-authority row; on a tie the newer row wins
// (it usually carries the more current phrasing).
func pickSurvivor(a, b *memory.MemoryObject) (winner, loser *memory.MemoryObject) {
	if a.Authority != b.Authority {
		if a.Authority > b.Authority {
			return a, b
		}
		return b, a
	}
	if a.CreatedAt.After(b.CreatedAt) {
		return a, b
	}
	return b, a
}

func (s *Service) contradictionRecordID(ch *Chore) (uuid.UUID, error) {
	if len(ch.Evidence) == 0 {
		return uuid.Nil, nil
	}
	var ev contradictionEvidence
	if err := json.Unmarshal(ch.Evidence, &ev); err != nil {
		return uuid.Nil, err
	}
	return ev.ContradictionRecordID, nil
}

// HousekeepingLine formats one optional maintenance ask for a visiting agent.
func HousekeepingLine(ch *Chore) string {
	if ch == nil {
		return ""
	}
	const cut = 90
	switch ch.Type {
	case TypeContradiction:
		return fmt.Sprintf(
			"Housekeeping (optional): two memories are flagged as contradictory — subject %q vs related %q. If your context lets you judge, call resolve_chore with chore_id=%s and action keep_subject, keep_related, or coexist.",
			snippet(ch.SubjectStatement, cut), snippet(ch.RelatedStatement, cut), ch.ID)
	case TypeQuarantineReview:
		return fmt.Sprintf(
			"Housekeeping (optional): memory %q is quarantined and awaiting review. If you can judge it, call resolve_chore with chore_id=%s and action release (back to pending) or delete.",
			snippet(ch.SubjectStatement, cut), ch.ID)
	case TypeDuplicatePair:
		return fmt.Sprintf(
			"Housekeeping (optional): memories %q and %q look like near-duplicates. If you agree, call resolve_chore with chore_id=%s and action consolidate; otherwise action distinct.",
			snippet(ch.SubjectStatement, cut), snippet(ch.RelatedStatement, cut), ch.ID)
	}
	return ""
}

func snippet(s string, n int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
