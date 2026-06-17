# Agent Memory Contract Field Matrix

Canonical field matrix driving Phase 11G parity comparator and contract evaluator tests.

| Field | Required When | REST Path | MCP Path | Equality Rule | Allowed Omission | Safety Consequence |
|-------|---------------|-----------|----------|---------------|------------------|-------------------|
| memory_id | always | `id` | `recall_bundle.*.id` | exact string | never | cannot dedupe or audit memory |
| statement | always | `statement` | same | exact string | never | agent cannot read guidance |
| schema_type | always | `schema_type` | same | exact string | never | type confusion |
| lifecycle_role | always | `lifecycle_role` | same | exact string | never | historical treated as current |
| status | always | `status` | same | exact string | never | lifecycle ambiguity |
| applicability | always | `applicability` | same | exact string | empty only when unset at persist | wrong gating |
| scope | current_guidance | `scope` | same | exact string | optional for non-guidance | wrong-scope use |
| negative_scope | current_guidance | `negative_scope` | same | sorted set equality | optional for non-guidance | suppression failure |
| retrieval_cues | optional | `retrieval_cues` | same | sorted set | always optional | weaker retrieval only |
| use_instruction | current_guidance | `use_instruction` | same | exact string | optional for non-guidance | unsafe application |
| misuse_warning | historical_context | `misuse_warning` | same | exact string | optional for current_guidance | contamination |
| provenance_summary | always | `source_type` + `authority_basis` | same | both required | never both empty | untrusted memory |
| source_type | always | `source_type` | same | exact string | never | provenance loss |
| authority_basis | always | `authority_basis` | same | exact string | never | authority unexplained |
| authority | always | `authority` | same | int equality | never | ranking corruption |
| utility_score | always | `utility_score` | same | float epsilon 1e-9 | never | utility blindness |
| quality_state | always | `quality_state` | same | exact string | never | quality gate bypass |
| quality_score | always | `quality_score` | same | float epsilon 1e-9 | never | quality blindness |
| superseded_by | superseded_context | `superseded_by` | same | exact string | optional otherwise | supersession hidden |
| contradiction_marker | refuted_context | via `lifecycle_role=refuted_context` | same | exact role | N/A when not refuted | refuted as current |
| occurred_at | optional | `occurred_at` | same | time equality | optional | timeline only |
| source_created_at | optional | `source_created_at` | same | time equality | optional | timeline only |
| rank/score | optional | `justification.score` | same | float epsilon | optional | ranking transparency only |
| mcp_context | MCP recall_context only | N/A | `content[0].json.mcp_context` | envelope only | REST N/A | wrapper metadata |

## Normalization rules

- Float scores: epsilon `1e-9`.
- `negative_scope`: sorted before compare.
- MCP wrapper envelope (`mcp_context`, `recall_bundle` key) is not compared as memory fields.
- Empty optional strings omitted on both sides are allowed.

## Unsafe omissions

Missing lifecycle, scope, use/misuse, provenance, quality, supersession markers, or MCP JSON structured content are unsafe and fail contract evaluation.
