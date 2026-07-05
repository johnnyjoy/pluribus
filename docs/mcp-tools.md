# Pluribus MCP tools

Generated from `control-plane/internal/mcp/tool_registry.go`. Do not edit tool rows by hand — run `UPDATE_MCP_TOOLS_DOC=1 go test ./internal/mcp/ -run TestMCPToolsDocMatchesRegistry` from `control-plane/` to refresh.

## Tool tiers (`tools/list` only)

Set **`PLURIBUS_TOOLS`** env or **`mcp.tools_tier`** in config (`core` | `standard` | `all`). Default **`all`** lists every tool. **`tools/call`** still accepts all registered names regardless of tier.

- **`core`** — loop + housekeeping: `enforcement_evaluate`, `health`, `list_chores`, `mcp_episode_ingest`, `memory_context_resolve`, `memory_feedback`, `recall_context`, `record_experience`, `resolve_chore`, `wakeup_context`- **`standard`** — core plus: `auto_log_episode_if_relevant`, `curation_auto_promote`, `curation_digest`, `curation_materialize`, `curation_pending`, `curation_promote_candidate`, `curation_promotion_suggestions`, `curation_reject_candidate`, `curation_review_candidate`, `curation_strengthened`, `memory_create`, `memory_delete`, `memory_log_if_relevant`, `memory_promote`, `memory_quarantine`

| Tool | Purpose | Agent-loop role | Required inputs | Optional inputs | Backend endpoint | Output summary | Risk level | Test coverage |
|------|---------|-----------------|-----------------|-----------------|------------------|----------------|------------|---------------|
| `recall_context` | Use at the start of a substantive task, after context changes, or when uncertain whether prior constraints, decisions... | session_start | (semantic) | agent_id, correlation_id, entities, mode, occurred_after, occurred_before, project_root, query, recall_mode, repo_root, retrieval_query, session_id, tags, task, task_description, workspace_root | POST /v1/recall/compile (wrapped) | mcp_context + recall_bundle JSON | low | both |
| `record_experience` | Use after solving a problem, encountering a failure, discovering a reusable pattern, or confirming a decision | post_outcome | summary | agent_id, correlation_id, entities, event_kind, repo_root, tags | POST /v1/advisory-episodes | advisory episode + formation status | medium | both |
| `memory_log_if_relevant` | Use when unsure if a text block contains learning signals | post_outcome | text_block | correlation_id, tags | POST /v1/advisory-episodes (conditional) | skipped or ingest result | medium | unit |
| `auto_log_episode_if_relevant` | Alias for memory_log_if_relevant | post_outcome | text_block | correlation_id, tags | POST /v1/advisory-episodes (conditional) | skipped or ingest result | medium | unit |
| `enforcement_evaluate` | Use before large refactors, policy-sensitive edits, or high-risk proposals when binding memory may apply | pre_change | proposal_text | intent | POST /v1/enforcement/evaluate | decision, triggered_memories, validation | low | both |
| `recall_compile` | Advanced: raw POST /v1/recall/compile with full CompileRequest body | none | (semantic) | include_status, mode, occurred_after, occurred_before, recall_mode, repo_root, retrieval_query, tags | POST /v1/recall/compile | recall bundle JSON | low | unit |
| `recall_get` | GET-shaped recall when your workflow requires it | none | (semantic) | max_per_kind, max_tokens, max_total, query, retrieval_query, tags | GET /v1/recall/ | recall bundle JSON | low | unit |
| `wakeup_context` | Use at session start for compact L0/L1 identity and governing memory via POST /v1/recall/wakeup | session_start | (semantic) | max_governing_total, max_per_kind, max_state | POST /v1/recall/wakeup | wakeup bundle (identity, governing_memory) | low | both |
| `recall_run_multi` | Advanced multi-variant recall orchestration | experimental | retrieval_query | tags, variants | POST /v1/recall/run-multi | run-multi response | medium | integration |
| `memory_create` | Author explicit durable memory (constraint, decision, etc | admin | kind, statement | agent_id, applicability, authority, payload, supersedes_id, tags | POST /v1/memory | memory object | high | integration |
| `memory_feedback` | Submit structured feedback about an existing memory's usefulness, harm, correctness, freshness, or relevance | post_outcome | event_type, memory_id | correlation_id, evidence_id, payload, reason, recall_bundle_id, source, source_session_id, source_tool | POST /v1/memory/{id}/feedback | feedback response with utility score | medium | unit |
| `memory_promote` | Structured promote path per server policy | admin | kind, statement | tags | POST /v1/memory/promote | promotion result | high | integration |
| `curation_digest` | Turn bounded work_summary into digest proposals (pending candidates) | curation | work_summary | tags | POST /v1/curation/digest | digest proposals | medium | integration |
| `curation_pending` | List pending candidates (not canonical memory) | curation | (semantic) | — | GET /v1/curation/pending | candidate list | low | integration |
| `curation_promotion_suggestions` | Promotion readiness hints | curation | (semantic) | — | GET /v1/curation/promotion-suggestions | suggestions | low | integration |
| `curation_strengthened` | Candidates reinforced by multiple distill signals | curation | (semantic) | min_support | GET /v1/curation/strengthened | candidate list | low | integration |
| `curation_review_candidate` | Review one candidate before promote | curation | (semantic) | candidate_id, id | GET /v1/curation/candidates/{id}/review | review payload | low | integration |
| `curation_materialize` | Promote validated candidate to durable memory | curation | (semantic) | candidate_id, id | POST /v1/curation/candidates/{id}/materialize | materialize result | high | integration |
| `curation_promote_candidate` | Alias for curation_materialize | curation | (semantic) | candidate_id, id | POST /v1/curation/candidates/{id}/materialize | materialize result | high | integration |
| `curation_reject_candidate` | Reject a candidate so it will not become memory | curation | (semantic) | candidate_id, id | POST /v1/curation/candidates/{id}/reject | reject result | medium | integration |
| `curation_auto_promote` | Batch auto-promote when server promotion | curation | (semantic) | — | POST /v1/curation/auto-promote | auto-promote result | critical | integration |
| `list_chores` | List open curation chores the hive would like reviewed (contradiction pairs, quarantined rows, near-duplicate pairs) | curation | (semantic) | limit | GET /v1/curation/chores | chore list with statements and allowed actions | low | unit |
| `resolve_chore` | Vote on a curation chore (from list_chores or the housekeeping line in recall/wakeup responses) | curation | action, agent_id | chore_id, id, reason | POST /v1/curation/chores/{id}/resolve | vote receipt (recorded, counted, applied, state) | medium | unit |
| `episode_search_similar` | Search similar advisory episodes (non-reject bucket) | diagnostic | (semantic) | max_results, occurred_after, occurred_before, query, summary_text, tags | POST /v1/advisory-episodes/similar | similar episodes | low | integration |
| `episode_distill_explicit` | Explicit distill to candidates when auto-distill off | curation | (semantic) | entities, episode_id, summary, tags | POST /v1/episodes/distill | distill result | medium | integration |
| `memory_recall_advanced` | Mode-shaped compile (continuity/constraint/pattern/episodic) | none | query | mode, repo_root, retrieval_query, symbols, tags | POST /v1/recall/compile | recall bundle | low | unit |
| `memory_preflight_check` | Quick risk hint from scope stats | diagnostic | (semantic) | changed_files_count, tags | POST /v1/recall/preflight | preflight stats | low | unit |
| `memory_quarantine` | Operator remediation: remove a memory from ALL recall (any mode) pending review | admin | (semantic) | id, memory_id, reason | POST /v1/memory/{id}/quarantine | updated memory object | high | unit |
| `memory_delete` | Operator remediation: soft-delete a memory (tombstone status deleted; excluded from all recall including historical) | admin | (semantic) | id, memory_id, reason | DELETE /v1/memory/{id} | updated memory object | high | unit |
| `memory_detect_contradictions` | Compare two memory UUIDs for conflicts | diagnostic | conflict_with_id, memory_id | — | POST /v1/contradictions/detect | detection result | medium | integration |
| `memory_list_contradictions` | Audit contradiction queue | diagnostic | (semantic) | limit, memory_id, resolution_state | GET /v1/contradictions/ | contradiction list | low | integration |
| `evidence_attach` | Create evidence and link to memory | admin | (semantic) | content, evidence_text, kind, memory_id, text | POST /v1/evidence + link | evidence ids | medium | unit |
| `evidence_list` | List evidence by memory_id or kind | diagnostic | (semantic) | kind, memory_id | GET /v1/evidence/ | evidence list | low | unit |
| `memory_relationships_get` | Inspect graph edges for a memory | diagnostic | memory_id | — | GET /v1/memory/{id}/relationships | relationships | low | integration |
| `memory_relationships_create` | Record typed edge between memories | admin | from_memory_id, relationship_type, to_memory_id | — | POST /v1/memory/relationships | relationship | medium | integration |
| `memory_context_resolve` | Compatibility alias for recall_context — identical behavior | session_start | (semantic) | agent_id, correlation_id, entities, mode, occurred_after, occurred_before, project_root, query, recall_mode, repo_root, retrieval_query, session_id, tags, task, task_description, workspace_root | POST /v1/recall/compile (wrapped) | mcp_context + recall_bundle | low | both |
| `mcp_episode_ingest` | Compatibility alias for record_experience — identical behavior | post_outcome | summary | agent_id, correlation_id, entities, event_kind, repo_root, tags | POST /v1/advisory-episodes | advisory episode | medium | both |
| `compliance_summary` | Phase 2: aggregate loop compliance counts across recent MCP sessions | diagnostic | (semantic) | — | GET /v1/compliance/summary | session counts by compliance status | low | unit |
| `compliance_session_get` | Phase 2: fetch one agent session row (client, transport, repo_root) | diagnostic | session_id | — | GET /v1/compliance/sessions/{id} | session object | low | unit |
| `compliance_session_events` | Phase 2: list redacted loop telemetry events for a session | diagnostic | session_id | — | GET /v1/compliance/sessions/{id}/events | events array | low | unit |
| `compliance_evaluate` | Phase 2: evaluate whether MCP-visible calls satisfy recall → enforcement → record loop for a session | diagnostic | session_id | recall_max_age_ms | POST /v1/compliance/evaluate | status, missing_steps, evidence | low | both |
| `agent_telemetry_start_session` | Phase 11I: start persisted memory-use telemetry session | session_start | interface | agent_id, client_name, session_id, tags | POST /v1/agent/telemetry/session/start | telemetry_session JSON | low | both |
| `agent_telemetry_record_recall` | Phase 11I: persist recall exposure event | session_start | recall_bundle, recalled_memory_ids, session_id | task_id | POST /v1/agent/telemetry/recall | telemetry_recall JSON | low | both |
| `agent_telemetry_record_decision` | Phase 11I: persist memory use decisions | post_outcome | decisions, recall_event_id, session_id | — | POST /v1/agent/telemetry/decision | telemetry_decision JSON | low | both |
| `agent_telemetry_record_output` | Phase 11I: persist agent output facts/actions | post_outcome | session_id | memory_citations, output_facts, recall_event_id | POST /v1/agent/telemetry/output | telemetry_output JSON | low | both |
| `agent_telemetry_evaluate` | Phase 11I: run deterministic obedience evaluator and persist violations/utility candidates | diagnostic | recall_event_id, session_id | output_id | POST /v1/agent/telemetry/evaluate | telemetry_evaluation JSON | low | both |
| `agent_telemetry_get_session` | Phase 11I: query full session telemetry summary | diagnostic | session_id | — | GET /v1/agent/telemetry/session/{id} | telemetry_session_summary JSON | low | both |
| `agent_telemetry_get_memory` | Phase 11I: query per-memory telemetry aggregates | diagnostic | memory_id | — | GET /v1/agent/telemetry/memory/{id} | telemetry_memory_summary JSON | low | both |
| `agent_telemetry_get_violations` | Phase 11I: query persisted violations | diagnostic | (semantic) | memory_id, violation_code | GET /v1/agent/telemetry/violations | telemetry_violations JSON | low | both |
| `agent_telemetry_get_utility_candidates` | Phase 11I: query utility candidates (not auto-applied) | diagnostic | (semantic) | memory_id | GET /v1/agent/telemetry/utility-candidates | telemetry_utility_candidates JSON | low | both |
| `agent_utility_evaluate_candidate` | Phase 11K: evaluate utility candidate through policy without mutation | diagnostic | candidate_id | — | POST /v1/agent/utility/policy/evaluate-candidate | policy_decision JSON | low | both |
| `agent_utility_apply_candidate` | Phase 11K: apply utility candidate through guarded policy | post_outcome | candidate_id | applied_by | POST /v1/agent/utility/policy/apply-candidate | utility_application JSON | medium | both |
| `agent_utility_apply_batch` | Phase 11K: batch apply utility candidates through policy caps | post_outcome | candidate_ids | applied_by | POST /v1/agent/utility/policy/apply-batch | utility_applications JSON | medium | both |
| `agent_utility_revert_application` | Phase 11K: revert policy utility application by rollback token | post_outcome | rollback_token | revert_reason | POST /v1/agent/utility/policy/revert-application | utility_application JSON | medium | both |
| `agent_utility_get_candidate` | Phase 11K: query utility application for candidate | diagnostic | candidate_id | — | GET /v1/agent/utility/policy/candidate/{id} | utility_application JSON | low | both |
| `agent_utility_get_memory_history` | Phase 11K: query utility application history for memory | diagnostic | memory_id | — | GET /v1/agent/utility/policy/memory/{id} | utility_applications JSON | low | both |
| `agent_utility_get_applications` | Phase 11K: list recent utility policy applications | diagnostic | (semantic) | — | GET /v1/agent/utility/policy/applications | utility_applications JSON | low | both |
| `agent_utility_get_policy_summary` | Phase 11K: policy application summary metrics | diagnostic | (semantic) | — | GET /v1/agent/utility/policy/summary | policy_summary JSON | low | both |
| `health` | Liveness probe (GET /healthz) | diagnostic | (semantic) | — | GET /healthz | ok JSON | low | unit |

## Direct tools/call coverage

| Tool | Coverage category | Test / proof | Pass status |
|------|-------------------|--------------|-------------|
| `recall_context` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `record_experience` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `memory_log_if_relevant` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `auto_log_episode_if_relevant` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `enforcement_evaluate` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `recall_compile` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `recall_get` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `wakeup_context` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `recall_run_multi` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `memory_create` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `memory_feedback` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `memory_promote` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `curation_digest` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `curation_pending` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `curation_promotion_suggestions` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `curation_strengthened` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `curation_review_candidate` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `curation_materialize` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `curation_promote_candidate` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `curation_reject_candidate` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `curation_auto_promote` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `list_chores` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `resolve_chore` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `episode_search_similar` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `episode_distill_explicit` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `memory_recall_advanced` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `memory_preflight_check` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `memory_quarantine` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `memory_delete` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `memory_detect_contradictions` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `memory_list_contradictions` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `evidence_attach` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `evidence_list` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `memory_relationships_get` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `memory_relationships_create` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `memory_context_resolve` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `mcp_episode_ingest` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `compliance_summary` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `compliance_session_get` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `compliance_session_events` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `compliance_evaluate` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `agent_telemetry_start_session` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `agent_telemetry_record_recall` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `agent_telemetry_record_decision` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `agent_telemetry_record_output` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `agent_telemetry_evaluate` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `agent_telemetry_get_session` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `agent_telemetry_get_memory` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `agent_telemetry_get_violations` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `agent_telemetry_get_utility_candidates` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `agent_utility_evaluate_candidate` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `agent_utility_apply_candidate` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `agent_utility_apply_batch` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `agent_utility_revert_application` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `agent_utility_get_candidate` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `agent_utility_get_memory_history` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `agent_utility_get_applications` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `agent_utility_get_policy_summary` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |
| `health` | unit_tools_call_tested | tool_call_coverage_test.go | pass (automated) |

## Aliases

| Alias | Canonical tool |
|-------|----------------|
| `auto_log_episode_if_relevant` | `memory_log_if_relevant` |
| `curation_promote_candidate` | `curation_materialize` |
| `mcp_episode_ingest` | `record_experience` |
| `memory_context_resolve` | `recall_context` |

## Verification

- Unit MCP tests: `make test-mcp` (repo root) or `cd control-plane && go test ./internal/mcp/...`
- Docker HTTP MCP proof: `make proof-mcp`
- Docker authenticated MCP proof: `make proof-mcp-auth`
- Stdio adapter proof: `make proof-mcp-stdio`
- Full Phase 1 close-out: `make proof-mcp-all`
