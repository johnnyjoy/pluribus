package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// Context strategy names (deterministic routing from task text).
const (
	ctxStrategyContinuity = "continuity"
	ctxStrategyConstraint = "constraint_focus"
	ctxStrategyFailure    = "failure_focus"
	ctxStrategyPattern    = "pattern_focus"
	ctxStrategyEpisodic   = "episodic_thread"
)

var (
	constraintCueSubstrings = []string{
		"must not", "never ", " forbid", "forbidden", "required", "shall ", "constraint", "rule", "policy",
		"prohibit", "block ", "binding", "compliance", "violation",
	}
	failureCueSubstrings = []string{
		"fail", "error", "bug", "crash", "incident", "outage", "regression", "timeout", "denied", "broken",
	}
	patternCueSubstrings = []string{
		"pattern", "always ", "repeat", "convention", "idiom", "habit", "reuse",
	}
	episodicCueSubstrings = []string{
		"last time", "previously", "we tried", "learned when", "earlier", "yesterday",
	}
)

func countCueHits(low string, cues []string) int {
	n := 0
	for _, c := range cues {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if strings.Contains(low, c) {
			n++
		}
	}
	return n
}

// inferContextStrategy picks compile shaping from task text or explicit mode override (deterministic).
func inferContextStrategy(task, modeOverride string) (strategy, compileMode string, limits map[string]any) {
	modeOverride = strings.ToLower(strings.TrimSpace(modeOverride))
	switch modeOverride {
	case "constraint":
		return ctxStrategyConstraint, "continuity", map[string]any{"constraints": 18}
	case "failure":
		return ctxStrategyFailure, "continuity", map[string]any{"failures": 18}
	case "pattern":
		return ctxStrategyPattern, "continuity", map[string]any{"patterns": 18}
	case "episodic", "thread":
		return ctxStrategyEpisodic, "thread", nil
	case "continuity":
		return ctxStrategyContinuity, "continuity", nil
	default:
		if modeOverride != "" {
			return modeOverride, modeOverride, nil
		}
	}

	low := strings.ToLower(task)
	cs := countCueHits(low, constraintCueSubstrings)
	fs := countCueHits(low, failureCueSubstrings)
	ps := countCueHits(low, patternCueSubstrings)
	es := countCueHits(low, episodicCueSubstrings)

	if es >= 2 || (es >= 1 && cs+fs+ps == 0) {
		return ctxStrategyEpisodic, "thread", nil
	}

	type bucket struct {
		name   string
		n      int
		limits map[string]any
	}
	cands := []bucket{
		{ctxStrategyConstraint, cs, map[string]any{"constraints": 18}},
		{ctxStrategyFailure, fs, map[string]any{"failures": 18}},
		{ctxStrategyPattern, ps, map[string]any{"patterns": 18}},
	}
	best := cands[0]
	for _, c := range cands[1:] {
		if c.n > best.n {
			best = c
		}
	}
	if best.n == 0 {
		// Default path: bias retrieval slots toward actionable order (constraints → failures → patterns).
		return ctxStrategyContinuity, "continuity", map[string]any{
			"constraints": 14,
			"failures":    12,
			"patterns":    10,
			"decisions":   10,
		}
	}
	return best.name, "continuity", best.limits
}

// buildMemoryContextResolveCompileBody builds POST /v1/recall/compile JSON and metadata for agents.
func buildMemoryContextResolveCompileBody(arguments json.RawMessage) ([]byte, map[string]any, error) {
	if len(bytes.TrimSpace(arguments)) == 0 {
		return nil, nil, fmt.Errorf("recall_context/memory_context_resolve requires arguments")
	}
	var m map[string]any
	if err := json.Unmarshal(arguments, &m); err != nil {
		return nil, nil, err
	}
	task := strings.TrimSpace(firstString(m, "task", "task_description", "query", "retrieval_query"))
	if task == "" {
		return nil, nil, fmt.Errorf("recall_context/memory_context_resolve requires task or task_description (raw task text)")
	}
	modeOverride := strings.TrimSpace(firstString(m, "mode"))
	tags := parseStringSliceField(m, "tags")
	tags = append(tags, parseStringSliceField(m, "entities")...)

	strategy, compileMode, lims := inferContextStrategy(task, modeOverride)
	out := map[string]any{
		"retrieval_query": task,
		"mode":            compileMode,
	}
	if len(tags) > 0 {
		out["tags"] = dedupeStrings(tags)
	}
	if lims != nil {
		out["variant_modifier"] = map[string]any{"limits": lims}
	}
	cid := strings.TrimSpace(firstString(m, "correlation_id", "session_id"))
	if cid != "" {
		out["correlation_id"] = cid
	}
	meta := map[string]any{
		"strategy":               strategy,
		"compile_mode":           compileMode,
		"retrieval_query":        task,
		"memory_signal_priority": []string{"constraints", "failures", "patterns", "decisions"},
		"why":                    "Deterministic routing: " + strategy + " (keyword scores over task text; no LLM). Tiered recall defaults favor constraints, then failures, then patterns. Optional correlation_id boosts session-tagged memories (mcp:session:*) in ranking.",
	}
	if cid != "" {
		meta["correlation_id"] = cid
	}
	body, err := json.Marshal(out)
	return body, meta, err
}

func dedupeStrings(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// textBlockHasRelevantSignals is deterministic; delegates to AutoLogEpisodeIfRelevant.
func textBlockHasRelevantSignals(text string) bool {
	ok, _ := AutoLogEpisodeIfRelevant(text)
	return ok
}

// execMemoryContextResolve runs POST /v1/recall/compile and wraps JSON with mcp_context.
func execMemoryContextResolve(client *http.Client, base, apiKey string, arguments json.RawMessage) map[string]any {
	body, meta, err := buildMemoryContextResolveCompileBody(arguments)
	if err != nil {
		return ToolResultErr(err.Error())
	}
	req, err := http.NewRequest(http.MethodPost, base+"/v1/recall/compile", bytes.NewReader(body))
	if err != nil {
		return ToolResultErr(err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return ToolResultErr(fmt.Sprintf("http error: %v", err))
	}
	defer resp.Body.Close()
	rawBody, _ := io.ReadAll(resp.Body)
	if len(rawBody) > 4*1024*1024 {
		rawBody = append(rawBody[:4*1024*1024], []byte("\n...truncated")...)
	}
	if resp.StatusCode >= 400 {
		text := fmt.Sprintf("%s\nHTTP %s\n%s", mustJSONMeta(meta), resp.Status, string(rawBody))
		return map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": text},
			},
			"isError": true,
		}
	}
	var bundle json.RawMessage
	if err := json.Unmarshal(rawBody, &bundle); err != nil {
		return ToolResultErr("recall compile: invalid JSON response")
	}
	enrichMCPContextFromRecallBundle(meta, bundle)
	applyMCPRecallBehaviorHints(meta)
	wrap := map[string]any{
		"mcp_context":   meta,
		"recall_bundle": bundle,
	}
	out, err := json.Marshal(wrap)
	if err != nil {
		return ToolResultErr(err.Error())
	}
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": string(out)},
		},
		"isError": false,
	}
}

func mustJSONMeta(meta map[string]any) string {
	b, err := json.Marshal(meta)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// execMemoryLogIfRelevant ingests an advisory episode when deterministic signals match (same policy as mcp_episode_ingest).
func execMemoryLogIfRelevant(client *http.Client, base, apiKey string, arguments json.RawMessage, pol *MemoryFormationPolicy) map[string]any {
	if len(bytes.TrimSpace(arguments)) == 0 {
		return ToolResultErr("memory_log_if_relevant requires arguments with text_block")
	}
	var m map[string]any
	if err := json.Unmarshal(arguments, &m); err != nil {
		return ToolResultErr(err.Error())
	}
	text, _ := m["text_block"].(string)
	text = strings.TrimSpace(text)
	if text == "" {
		return ToolResultErr("memory_log_if_relevant requires text_block")
	}
	if ok, r := AutoLogEpisodeIfRelevant(text); !ok {
		skip := map[string]any{
			"skipped": true,
			"reason":  "no_deterministic_learning_signals",
			"detail":  r,
			"hint":    "text_block did not match learning signals, repetition cues, or repeated tokens (deterministic; no LLM).",
		}
		b, _ := json.Marshal(skip)
		return map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": string(b)},
			},
			"isError": false,
		}
	}
	argMap := map[string]any{"summary": text}
	if cid, ok := m["correlation_id"].(string); ok && strings.TrimSpace(cid) != "" {
		argMap["correlation_id"] = strings.TrimSpace(cid)
	}
	if tags := parseStringSliceField(m, "tags"); len(tags) > 0 {
		argMap["tags"] = tags
	}
	raw, _ := json.Marshal(argMap)
	payload, vErr := buildAdvisoryEpisodeMCPBody(raw, pol)
	if vErr != nil {
		return ToolResultErr(vErr.Error())
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return ToolResultErr(err.Error())
	}
	req, err := http.NewRequest(http.MethodPost, base+"/v1/advisory-episodes", bytes.NewReader(b))
	if err != nil {
		return ToolResultErr(err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return ToolResultErr(fmt.Sprintf("http error: %v", err))
	}
	defer resp.Body.Close()
	rawBody, _ := io.ReadAll(resp.Body)
	statusErr := resp.StatusCode >= 400
	if !statusErr {
		rawBody = augmentAdvisoryEpisodeSuccessJSON(rawBody)
	}
	textOut := fmt.Sprintf(`{"memory_log_if_relevant":{"ingested":true}}` + "\n" + string(rawBody))
	if statusErr {
		textOut = fmt.Sprintf("HTTP %s\n%s", resp.Status, string(rawBody))
	}
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": textOut},
		},
		"isError": statusErr,
	}
}

type recallItemLite struct {
	Statement     string             `json:"statement"`
	Justification *justificationLite `json:"justification,omitempty"`
}

type justificationLite struct {
	Reason string `json:"reason"`
}

// enrichMCPContextFromRecallBundle adds why_now, bundle_counts, and primary_signal from the recall bundle (deterministic).
func enrichMCPContextFromRecallBundle(meta map[string]any, bundle json.RawMessage) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(bundle, &raw); err != nil {
		return
	}
	if v, ok := raw["recall_preamble"]; ok {
		var pre string
		if json.Unmarshal(v, &pre) == nil && strings.TrimSpace(pre) != "" {
			meta["recall_preamble"] = strings.TrimSpace(pre)
		}
	}
	countKeys := []string{
		"governing_constraints", "known_failures", "applicable_patterns",
		"decisions", "constraints", "continuity", "experience",
	}
	counts := map[string]int{}
	total := 0
	for _, k := range countKeys {
		if v, ok := raw[k]; ok {
			var arr []json.RawMessage
			if json.Unmarshal(v, &arr) != nil {
				continue
			}
			counts[k] = len(arr)
			total += len(arr)
		}
	}
	meta["bundle_counts"] = counts
	if total == 0 {
		meta["why_now"] = "No memories matched this situation in the pool yet; episodic ingest strengthens the next recall."
		return
	}

	tryWhy := func(items []recallItemLite, prefix string) bool {
		if len(items) == 0 {
			return false
		}
		st := strings.TrimSpace(items[0].Statement)
		if st == "" {
			return false
		}
		if len([]rune(st)) > 160 {
			r := []rune(st)
			st = string(r[:160]) + "…"
		}
		meta["why_now"] = prefix + st
		if items[0].Justification != nil && strings.TrimSpace(items[0].Justification.Reason) != "" {
			meta["primary_signal"] = strings.TrimSpace(items[0].Justification.Reason)
		}
		return true
	}

	if v, ok := raw["governing_constraints"]; ok {
		var items []recallItemLite
		if json.Unmarshal(v, &items) == nil && tryWhy(items, "Constraint applies: ") {
			return
		}
	}
	if v, ok := raw["known_failures"]; ok {
		var items []recallItemLite
		if json.Unmarshal(v, &items) == nil && tryWhy(items, "Known failure to avoid: ") {
			return
		}
	}
	if v, ok := raw["applicable_patterns"]; ok {
		var items []recallItemLite
		if json.Unmarshal(v, &items) == nil && tryWhy(items, "Relevant pattern: ") {
			return
		}
	}
	if v, ok := raw["decisions"]; ok {
		var items []recallItemLite
		if json.Unmarshal(v, &items) == nil && tryWhy(items, "Prior decision: ") {
			return
		}
	}
	meta["why_now"] = "Recall returned " + strconv.Itoa(total) + " memory item(s); see bundle_counts for distribution."
}

const (
	mcpDecisionHintAlways      = "Use this context before making changes or repeating similar work."
	mcpRelevanceHintStrong     = "Prior decisions or patterns may affect this task."
	mcpAfterWorkHintWeakPool   = "No strong prior memory found. Consider recording the outcome after completing this task."
	mcpAfterWorkHintStrongPool = "After completing meaningful work, consider recording the outcome."
)

// bundleCountTotal sums bundle_counts from enrichMCPContextFromRecallBundle (0 if missing or unparseable).
func bundleCountTotal(meta map[string]any) int {
	v, ok := meta["bundle_counts"]
	if !ok || v == nil {
		return 0
	}
	switch m := v.(type) {
	case map[string]int:
		n := 0
		for _, c := range m {
			n += c
		}
		return n
	case map[string]any:
		n := 0
		for _, c := range m {
			switch x := c.(type) {
			case float64:
				n += int(x)
			case int:
				n += x
			case int64:
				n += int(x)
			}
		}
		return n
	default:
		return 0
	}
}

// applyMCPRecallBehaviorHints adds low-noise timing cues to mcp_context after a successful recall compile.
// Strong pool = at least one memory item in any counted recall bucket (see enrichMCPContextFromRecallBundle).
func applyMCPRecallBehaviorHints(meta map[string]any) {
	meta["decision_hint"] = mcpDecisionHintAlways
	total := bundleCountTotal(meta)
	if total > 0 {
		meta["relevance_hint"] = mcpRelevanceHintStrong
		meta["after_work_hint"] = mcpAfterWorkHintStrongPool
		return
	}
	meta["after_work_hint"] = mcpAfterWorkHintWeakPool
}
