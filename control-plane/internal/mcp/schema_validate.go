package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ValidateToolArguments validates MCP tool arguments against registered inputSchema and semantic rules.
func ValidateToolArguments(toolName string, arguments json.RawMessage) error {
	spec, ok := ToolSpecByName(toolName)
	if !ok {
		return fmt.Errorf("unknown tool: %s", toolName)
	}
	args := arguments
	if len(bytesTrimSpace(args)) == 0 {
		args = json.RawMessage(`{}`)
	}
	var m map[string]any
	if err := json.Unmarshal(args, &m); err != nil {
		return fmt.Errorf("invalid arguments JSON for tool %s: %w", toolName, err)
	}
	if err := validateAgainstSchema(spec.InputSchema, m, toolName); err != nil {
		return err
	}
	return validateToolSemantics(toolName, m)
}

func bytesTrimSpace(b json.RawMessage) json.RawMessage {
	return json.RawMessage(strings.TrimSpace(string(b)))
}

func validateAgainstSchema(schema map[string]any, data map[string]any, toolName string) error {
	if schema == nil {
		return fmt.Errorf("tool %s has no inputSchema registered", toolName)
	}
	typ, _ := schema["type"].(string)
	if typ != "object" {
		return fmt.Errorf("tool %s: unsupported root schema type %q", toolName, typ)
	}
	if ap, ok := schema["additionalProperties"].(bool); ok && !ap {
		props, _ := schema["properties"].(map[string]any)
		for k := range data {
			if k == "" {
				continue
			}
			if props == nil || props[k] == nil {
				return fmt.Errorf("unexpected argument %q for tool %s (additionalProperties=false)", k, toolName)
			}
		}
	}
	if req, ok := schema["required"].([]any); ok {
		for _, r := range req {
			key, _ := r.(string)
			if key == "" {
				continue
			}
			if !hasNonEmptyValue(data, key) {
				return fmt.Errorf("missing required argument: %s", key)
			}
		}
	} else if reqStr, ok := schema["required"].([]string); ok {
		for _, key := range reqStr {
			if !hasNonEmptyValue(data, key) {
				return fmt.Errorf("missing required argument: %s", key)
			}
		}
	}
	for k, v := range data {
		props, _ := schema["properties"].(map[string]any)
		if props == nil {
			continue
		}
		ps, ok := props[k].(map[string]any)
		if !ok {
			continue
		}
		if err := validatePropertyType(k, ps, v); err != nil {
			return fmt.Errorf("invalid argument %q for tool %s: %w", k, toolName, err)
		}
	}
	return nil
}

func hasNonEmptyValue(m map[string]any, key string) bool {
	v, ok := m[key]
	if !ok || v == nil {
		return false
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s) != ""
	}
	return true
}

func validatePropertyType(key string, propSchema map[string]any, value any) error {
	if value == nil {
		return nil
	}
	typ, _ := propSchema["type"].(string)
	switch typ {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string")
		}
		if enum, ok := propSchema["enum"].([]any); ok {
			s := value.(string)
			found := false
			for _, e := range enum {
				if es, ok := e.(string); ok && es == s {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("value not in enum")
			}
		} else if enumStr, ok := propSchema["enum"].([]string); ok {
			s := value.(string)
			found := false
			for _, es := range enumStr {
				if es == s {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("value not in enum")
			}
		}
	case "integer":
		switch value.(type) {
		case float64, int, int64:
		default:
			return fmt.Errorf("expected integer")
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean")
		}
	case "array":
		arr, ok := value.([]any)
		if !ok {
			return fmt.Errorf("expected array")
		}
		items, _ := propSchema["items"].(map[string]any)
		if items != nil {
			itemType, _ := items["type"].(string)
			if itemType == "string" {
				for i, el := range arr {
					if _, ok := el.(string); !ok {
						return fmt.Errorf("array item %d expected string", i)
					}
				}
			}
		}
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return fmt.Errorf("expected object")
		}
	}
	return nil
}

func validateToolSemantics(toolName string, m map[string]any) error {
	raw, err := json.Marshal(m)
	if err != nil {
		return err
	}
	switch toolName {
	case "recall_context", "memory_context_resolve":
		task := strings.TrimSpace(firstString(m, "task", "task_description", "query", "retrieval_query"))
		if task == "" {
			return fmt.Errorf("missing required argument: task (or task_description, query, retrieval_query)")
		}
		if rm := strings.TrimSpace(firstString(m, "recall_mode")); rm != "" {
			switch strings.ToLower(rm) {
			case "current", "historical":
			default:
				return fmt.Errorf("invalid recall_mode %q: must be current or historical", rm)
			}
		}
		if err := validateOptionalRFC3339(m, "occurred_after"); err != nil {
			return err
		}
		if err := validateOptionalRFC3339(m, "occurred_before"); err != nil {
			return err
		}
	case "record_experience", "mcp_episode_ingest":
		if err := ValidateMcpEpisodeSummary(firstString(m, "summary"), DefaultMemoryFormationPolicy()); err != nil {
			return err
		}
	case "memory_log_if_relevant", "auto_log_episode_if_relevant":
		if strings.TrimSpace(firstString(m, "text_block")) == "" {
			return fmt.Errorf("missing required argument: text_block")
		}
	case "enforcement_evaluate":
		return ValidateEnforcementArguments(raw)
	case "curation_digest":
		return ValidateDigestArguments(raw)
	case "curation_materialize", "curation_promote_candidate", "curation_review_candidate", "curation_reject_candidate":
		if _, err := ParseCandidateID(raw); err != nil {
			return err
		}
	case "episode_search_similar":
		if strings.TrimSpace(firstString(m, "query", "summary_text")) == "" {
			return fmt.Errorf("missing required argument: query or summary_text")
		}
	case "episode_distill_explicit":
		if strings.TrimSpace(firstString(m, "episode_id")) == "" && strings.TrimSpace(firstString(m, "summary")) == "" {
			return fmt.Errorf("missing required argument: episode_id or summary")
		}
	case "memory_recall_advanced":
		if strings.TrimSpace(firstString(m, "query", "retrieval_query")) == "" {
			return fmt.Errorf("missing required argument: query")
		}
	case "recall_run_multi":
		if strings.TrimSpace(firstString(m, "retrieval_query")) == "" {
			return fmt.Errorf("missing required argument: retrieval_query")
		}
	case "memory_create":
		if strings.TrimSpace(firstString(m, "kind")) == "" || strings.TrimSpace(firstString(m, "statement")) == "" {
			return fmt.Errorf("missing required arguments: kind and statement")
		}
	case "memory_feedback":
		if strings.TrimSpace(firstString(m, "memory_id")) == "" {
			return fmt.Errorf("missing required argument: memory_id")
		}
		if strings.TrimSpace(firstString(m, "event_type")) == "" {
			return fmt.Errorf("missing required argument: event_type")
		}
		et := strings.ToLower(strings.TrimSpace(firstString(m, "event_type")))
		for _, neg := range []string{"harmful", "wrong", "outdated"} {
			if et == neg && strings.TrimSpace(firstString(m, "reason")) == "" {
				return fmt.Errorf("reason required for event_type %s", et)
			}
		}
	case "memory_detect_contradictions":
		if _, err := buildContradictionDetectBody(raw); err != nil {
			return err
		}
	case "evidence_list":
		if strings.TrimSpace(firstString(m, "memory_id")) == "" && strings.TrimSpace(firstString(m, "kind")) == "" {
			return fmt.Errorf("missing required argument: memory_id or kind")
		}
	case "evidence_attach":
		if strings.TrimSpace(firstString(m, "memory_id")) == "" {
			return fmt.Errorf("missing required argument: memory_id")
		}
		txt := strings.TrimSpace(firstString(m, "evidence_text", "text", "content"))
		if txt == "" {
			return fmt.Errorf("missing required argument: evidence_text")
		}
	case "memory_relationships_get":
		if _, err := parseRequiredUUIDArg(raw, "memory_id"); err != nil {
			return fmt.Errorf("missing required argument: memory_id")
		}
	case "memory_relationships_create":
		if strings.TrimSpace(firstString(m, "from_memory_id")) == "" || strings.TrimSpace(firstString(m, "to_memory_id")) == "" || strings.TrimSpace(firstString(m, "relationship_type")) == "" {
			return fmt.Errorf("missing required arguments: from_memory_id, to_memory_id, relationship_type")
		}
	}
	return nil
}

// SchemaQualityIssues returns human-readable problems for registry schemas (for tests).
func SchemaQualityIssues() []string {
	var issues []string
	for _, t := range toolRegistry() {
		if strings.TrimSpace(t.Description) == "" {
			issues = append(issues, t.Name+": empty description")
		}
		if t.InputSchema == nil {
			issues = append(issues, t.Name+": nil inputSchema")
			continue
		}
		typ, _ := t.InputSchema["type"].(string)
		if typ != "object" {
			issues = append(issues, t.Name+": root type not object")
		}
		props, hasProps := t.InputSchema["properties"].(map[string]any)
		_, hasAdditional := t.InputSchema["additionalProperties"]
		if !hasAdditional {
			issues = append(issues, t.Name+": missing additionalProperties")
		}
		if !hasProps && typ == "object" {
			if ap, ok := t.InputSchema["additionalProperties"].(bool); !ok || ap {
				issues = append(issues, t.Name+": empty properties without additionalProperties:false")
			}
		}
		if len(props) == 0 {
			if ap, ok := t.InputSchema["additionalProperties"].(bool); !ok || ap {
				issues = append(issues, t.Name+": schema is bare {type:object} without additionalProperties:false")
			}
		}
		if t.Mutates && t.Risk == "" {
			issues = append(issues, t.Name+": mutating tool missing risk level")
		}
	}
	return issues
}

func validateOptionalRFC3339(m map[string]any, field string) error {
	s := strings.TrimSpace(firstString(m, field))
	if s == "" {
		return nil
	}
	layouts := []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05Z07:00", "2006-01-02"}
	for _, layout := range layouts {
		if _, err := time.Parse(layout, s); err == nil {
			return nil
		}
	}
	return fmt.Errorf("invalid %s %q: use RFC3339 timestamp", field, s)
}
