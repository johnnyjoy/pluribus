package compliance

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	HeaderSessionID     = "X-Pluribus-Session-Id"
	HeaderCorrelationID = "X-Pluribus-Correlation-Id"
	HeaderRepoRoot      = "X-Pluribus-Repo-Root"
)

// Service records MCP telemetry and evaluates compliance.
type Service struct {
	Repo *Repo
	// In-memory fallback for tests when Repo.DB is nil
	mu      sync.Mutex
	memSess map[uuid.UUID]Session
	memEv   map[uuid.UUID][]Event
}

// NewService wires compliance telemetry.
func NewService(db *Repo) *Service {
	return &Service{
		Repo:    db,
		memSess: map[uuid.UUID]Session{},
		memEv:   map[uuid.UUID][]Event{},
	}
}

// MCPContext carries correlation from HTTP MCP request.
type MCPContext struct {
	SessionID     uuid.UUID
	CorrelationID string
	ClientName    string
	ClientVersion string
	Transport     string
	RepoRoot      string
	WorkspaceHint string
	AuthMode      string
	RemoteAddr    string
}

// ContextFromRequest extracts or generates session/correlation from headers and initialize params.
func ContextFromRequest(r *http.Request, initClientName, initClientVersion string) MCPContext {
	ctx := MCPContext{Transport: "http_mcp"}
	if r != nil {
		ctx.RemoteAddr = r.RemoteAddr
		if strings.TrimSpace(r.Header.Get("X-API-Key")) != "" || strings.TrimSpace(r.URL.Query().Get("token")) != "" {
			ctx.AuthMode = "api_key"
		} else {
			ctx.AuthMode = "none"
		}
		if sid := strings.TrimSpace(r.Header.Get(HeaderSessionID)); sid != "" {
			if id, err := uuid.Parse(sid); err == nil {
				ctx.SessionID = id
			}
		}
		ctx.CorrelationID = strings.TrimSpace(r.Header.Get(HeaderCorrelationID))
		ctx.RepoRoot = strings.TrimSpace(r.Header.Get(HeaderRepoRoot))
	}
	if initClientName != "" {
		ctx.ClientName = initClientName
	}
	if initClientVersion != "" {
		ctx.ClientVersion = initClientVersion
	}
	if ctx.SessionID == uuid.Nil {
		ctx.SessionID = uuid.New()
	}
	if ctx.CorrelationID == "" {
		ctx.CorrelationID = ctx.SessionID.String()
	}
	return ctx
}

// CorrelationFromToolArgs reads correlation_id/session_id from tool arguments.
func CorrelationFromToolArgs(args json.RawMessage) string {
	var m map[string]any
	if len(args) == 0 {
		return ""
	}
	_ = json.Unmarshal(args, &m)
	for _, k := range []string{"correlation_id", "session_id"} {
		if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// EnsureSession upserts session row.
func (s *Service) EnsureSession(ctx context.Context, mc MCPContext) error {
	if s == nil {
		return nil
	}
	now := time.Now().UTC()
	sess := Session{
		ID:            mc.SessionID,
		StartedAt:     now,
		LastSeenAt:    now,
		ClientName:    mc.ClientName,
		ClientVersion: mc.ClientVersion,
		Transport:     mc.Transport,
		RepoRoot:      mc.RepoRoot,
		WorkspaceHint: mc.WorkspaceHint,
	}
	if s.Repo != nil && s.Repo.DB != nil {
		return s.Repo.UpsertSession(ctx, sess)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.memSess[mc.SessionID]; ok {
		sess.StartedAt = old.StartedAt
	}
	s.memSess[mc.SessionID] = sess
	return nil
}

// RecordEvent persists telemetry event.
func (s *Service) RecordEvent(ctx context.Context, e Event) error {
	if s == nil {
		return nil
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now().UTC()
	}
	if s.Repo != nil && s.Repo.DB != nil {
		_, err := s.Repo.InsertEvent(ctx, e)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	s.memEv[e.SessionID] = append(s.memEv[e.SessionID], e)
	return nil
}

// ListEvents for session.
func (s *Service) ListEvents(ctx context.Context, sessionID uuid.UUID) ([]Event, error) {
	if s == nil {
		return nil, nil
	}
	if s.Repo != nil && s.Repo.DB != nil {
		return s.Repo.ListEvents(ctx, sessionID, time.Time{}, time.Time{})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Event(nil), s.memEv[sessionID]...), nil
}

// GetSession by ID.
func (s *Service) GetSession(ctx context.Context, id uuid.UUID) (*Session, error) {
	if s == nil {
		return nil, nil
	}
	if s.Repo != nil && s.Repo.DB != nil {
		return s.Repo.GetSession(ctx, id)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.memSess[id]; ok {
		cp := sess
		return &cp, nil
	}
	return nil, nil
}

// ListSessions recent.
func (s *Service) ListSessions(ctx context.Context, limit int) ([]Session, error) {
	if s == nil {
		return nil, nil
	}
	if s.Repo != nil && s.Repo.DB != nil {
		return s.Repo.ListSessions(ctx, limit)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Session, 0, len(s.memSess))
	for _, sess := range s.memSess {
		out = append(out, sess)
	}
	return out, nil
}

// RecordMCPMethod logs non-tool MCP methods.
func (s *Service) RecordMCPMethod(ctx context.Context, mc MCPContext, method string, errCode int, errMsg string, duration time.Duration) {
	var et string
	switch method {
	case "initialize":
		et = EventMCPInitialize
	case "tools/list":
		et = EventMCPToolsList
	default:
		return
	}
	_ = s.EnsureSession(ctx, mc)
	status := "ok"
	code, msg := "", ""
	if errCode != 0 {
		status = "error"
		code = itoaErr(errCode)
		msg = errMsg
	}
	_ = s.RecordEvent(ctx, Event{
		SessionID:      mc.SessionID,
		EventType:      et,
		CorrelationID:  mc.CorrelationID,
		ResultStatus:   status,
		ErrorCode:      code,
		ErrorMessage:   msg,
		DurationMS:     int(duration.Milliseconds()),
		Metadata:       map[string]any{"method": method, "auth_mode": mc.AuthMode},
	})
}

// RecordToolCall logs MCP tools/call lifecycle.
func (s *Service) RecordToolCall(ctx context.Context, mc MCPContext, toolName string, args json.RawMessage, started time.Time, success bool, errCode int, errMsg, enforcementDecision string) {
	if s == nil {
		return
	}
	corr := CorrelationFromToolArgs(args)
	if corr != "" {
		mc.CorrelationID = corr
	}
	_ = s.EnsureSession(ctx, mc)

	loopRole := LoopRoleForTool(toolName)
	risk := RiskForTool(toolName)

	startEv := Event{
		SessionID:     mc.SessionID,
		EventType:     EventMCPToolCallStarted,
		ToolName:      toolName,
		LoopRole:      loopRole,
		RiskLevel:     risk,
		CorrelationID: mc.CorrelationID,
		RequestHash:   HashRequest(json.RawMessage(args)),
		RequestSummary: SummarizeToolCall(toolName, args),
		ResultStatus:  "started",
		Metadata:      map[string]any{"auth_mode": mc.AuthMode},
	}
	_ = s.RecordEvent(ctx, startEv)

	et := EventMCPToolCallCompleted
	status := "ok"
	code, msg := "", ""
	if !success {
		et = EventMCPToolCallFailed
		status = "error"
		code = itoaErr(errCode)
		msg = errMsg
	} else {
		et = ClassifyTool(toolName)
	}

	done := Event{
		SessionID:           mc.SessionID,
		OccurredAt:          time.Now().UTC(),
		EventType:           et,
		ToolName:            toolName,
		LoopRole:            loopRole,
		RiskLevel:           risk,
		CorrelationID:       mc.CorrelationID,
		RequestHash:         startEv.RequestHash,
		RequestSummary:      startEv.RequestSummary,
		ResultStatus:        status,
		ErrorCode:           code,
		ErrorMessage:        msg,
		DurationMS:          int(time.Since(started).Milliseconds()),
		EnforcementDecision: enforcementDecision,
		Metadata:            map[string]any{"auth_mode": mc.AuthMode},
	}
	_ = s.RecordEvent(ctx, done)
}

func itoaErr(code int) string {
	switch code {
	case -32602:
		return "-32602"
	case -32601:
		return "-32601"
	case -32000:
		return "-32000"
	default:
		return "error"
	}
}

// RecordMemoryFeedback logs utility feedback linkage for agent-loop audit (Phase 7).
func (s *Service) RecordMemoryFeedback(ctx context.Context, memoryID uuid.UUID, eventType, correlationID string) {
	if s == nil {
		return
	}
	sid := uuid.Nil
	if correlationID != "" {
		if id, err := uuid.Parse(correlationID); err == nil {
			sid = id
		}
	}
	if sid == uuid.Nil {
		sid = uuid.New()
	}
	_ = s.RecordEvent(ctx, Event{
		SessionID:     sid,
		EventType:     EventMemoryFeedback,
		ToolName:      "memory_feedback",
		LoopRole:      "post_outcome",
		RiskLevel:     "medium",
		CorrelationID: correlationID,
		MemoryID:      memoryID.String(),
		ResultStatus:  "ok",
		RequestSummary: eventType,
		Metadata: map[string]any{
			"event_type": eventType,
		},
	})
}
