package agenttelemetry

import (
	"context"
	"fmt"
	"strings"

	"control-plane/internal/agentobedience"
	"control-plane/internal/recall"
)

// RecordAutoRecall implements recall.TelemetryRecorder.
func (s *Service) RecordAutoRecall(ctx context.Context, in recall.AutoRecallInput) (recall.RecallTelemetry, error) {
	disabled := recall.RecallTelemetry{TelemetryEnabled: false}
	if s == nil {
		return disabled, nil
	}
	sid := strings.TrimSpace(in.SessionID)
	if sid == "" {
		return disabled, nil
	}
	iface := strings.TrimSpace(in.Interface)
	if iface == "" {
		iface = agentobedience.InterfaceREST
	}
	if _, err := parseUUID(sid); err != nil {
		return disabled, fmt.Errorf("%w: session_id", ErrMalformedPayload)
	}
	if !s.hasSession(ctx, sid) {
		if _, err := s.StartSession(ctx, StartSessionRequest{
			SessionID: sid,
			Interface: iface,
		}); err != nil {
			return disabled, err
		}
	}
	var bundleJSON map[string]any
	var recalled []string
	if in.Bundle != nil {
		bundleJSON = BundleJSONFromRecall(in.Bundle)
		recalled = recalledIDsFromBundle(in.Bundle)
	} else if in.Wakeup != nil {
		bundleJSON = BundleJSONFromWakeup(in.Wakeup)
		recalled = recalledIDsFromWakeup(in.Wakeup)
	}
	if len(recalled) == 0 {
		recalled = RecalledIDsFromBundleJSON(bundleJSON)
	}
	if len(recalled) == 0 {
		recalled = []string{"telemetry:empty-recall"}
	}
	reqHash := HashRecallRequest(sid, in.RecallRequest)
	if existing, ok := s.findRecallByRequestHash(ctx, sid, reqHash); ok {
		return recall.RecallTelemetry{
			TelemetryEnabled:   true,
			TelemetrySessionID: sid,
			RecallEventID:      existing.ID.String(),
			RecallBundleID:     existing.RecallBundleID,
			RecallRequestHash:  reqHash,
		}, nil
	}
	bundleID := ""
	if bundleJSON != nil {
		if bid, ok := bundleJSON["bundle_id"].(string); ok {
			bundleID = bid
		}
	}
	if bundleID == "" {
		bundleID = StableBundleID(in.RecallRequest)
	}
	mode := strings.TrimSpace(in.RecallMode)
	if mode == "" {
		mode = "current"
	}
	if in.RecallRequest == nil {
		in.RecallRequest = map[string]any{}
	}
	in.RecallRequest["_recall_request_hash"] = reqHash
	ev, err := s.RecordRecall(ctx, RecordRecallRequest{
		SessionID:         sid,
		TaskID:            in.TaskID,
		Interface:         iface,
		RecallRequest:     in.RecallRequest,
		RecallBundleID:    bundleID,
		RecalledMemoryIDs: recalled,
		RecallBundle:      bundleJSON,
		RecallMode:        mode,
	})
	if err != nil {
		return disabled, err
	}
	return recall.RecallTelemetry{
		TelemetryEnabled:   true,
		TelemetrySessionID: sid,
		RecallEventID:      ev.ID.String(),
		RecallBundleID:     ev.RecallBundleID,
		RecallRequestHash:  reqHash,
	}, nil
}

func recalledIDsFromWakeup(w *recall.WakeupResponse) []string {
	if w == nil {
		return nil
	}
	ids := make([]string, 0, len(w.Identity)+len(w.GoverningMemory))
	for _, it := range w.Identity {
		if strings.TrimSpace(it.ID) != "" {
			ids = append(ids, it.ID)
		}
	}
	for _, it := range w.GoverningMemory {
		if strings.TrimSpace(it.ID) != "" {
			ids = append(ids, it.ID)
		}
	}
	return ids
}

func (s *Service) hasSession(ctx context.Context, sessionID string) bool {
	return s.HasSession(ctx, sessionID)
}

func (s *Service) findRecallByRequestHash(ctx context.Context, sessionID, hash string) (*RecallEvent, bool) {
	sid, err := parseUUID(sessionID)
	if err != nil {
		return nil, false
	}
	recalls := s.listRecallsBySession(ctx, sid)
	for i := range recalls {
		r := &recalls[i]
		if h := HashRecallRequest(sessionID, r.RecallRequestJSON); h == hash {
			return r, true
		}
		if rh, ok := r.RecallRequestJSON["_recall_request_hash"].(string); ok && rh == hash {
			return r, true
		}
	}
	return nil, false
}
