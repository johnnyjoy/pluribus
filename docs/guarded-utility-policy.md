# Guarded Utility Application Policy

Phase 11K doctrine for evaluator-gated utility mutation.

## Principles

- **Recall is not usefulness.** Exposure frequency must not increase utility.
- **Self-report is not usefulness.** Agent claims without evaluator validation are rejected.
- **Citation is not usefulness.** Citing a memory does not prove helpful output.
- **Historical-only use is not current-guidance success.** Background context stays neutral by default.
- **One successful use is not enough to dominate ranking.** Caps and bounded deltas apply.
- **Refuted/superseded use is negative evidence.** Misuse demotes faster than promotion.
- **Evaluator validation is mandatory.** Only telemetry with `evaluation_id` and deterministic evaluator pass/fail may apply.
- **Policy approval is mandatory before mutation.** `safe_to_apply` plus policy gates required for positive apply.
- **All utility applications are auditable and reversible.** `agent_utility_applications` ledger + rollback tokens.

## Terms

| Term | Meaning |
|------|---------|
| utility candidate | Evaluator-generated signal from persisted telemetry (`agent_utility_candidates`) |
| utility event | Append-only ledger row in `memory_utility_events` when policy applies score change |
| utility application | Policy decision record in `agent_utility_applications` |
| positive signal | `used_correctly`, `helped_output` with evaluator pass + `safe_to_apply` |
| negative signal | `misused`, `unsafe_use`, `harmed_output`, `unsupported_claim`, refuted/superseded/wrong-scope |
| neutral signal | `ignored_correctly`, `historical_only_correctly` → record-only |
| review-needed signal | Failed/stale/tampered/unsafe positive without `safe_to_apply` |
| safe_to_apply | Candidate flag; true only for evaluator-validated positive signals at generation |
| apply_policy | Deterministic `utilitypolicy` engine (`phase11k-v1`) |
| promotion threshold | Max +0.5 delta per positive apply |
| demotion threshold | Max -1.0 delta per negative apply (stronger demotion) |
| cooldown | 7-day stale candidate window → review_required |
| per-session cap | Default 2 positive applies per memory per session |
| per-agent cap | Default 3 positive applies per memory per agent |
| confidence accumulation | Caps prevent single-session/agent domination |
| poisoning risk | Recall-only, self-report, citation-only, historical promotion |
| audit trail | Every policy decision persisted, including reject/record-only/review |
| rollback/reversal | `POST /v1/agent/utility/policy/revert-application` restores prior score |

## Policy decisions

- `apply_positive` — bounded score increase
- `apply_negative` — bounded score decrease
- `record_only` — audit only, no mutation
- `review_required` — audit only, human review
- `reject` — blocked (duplicate, caps, poisoning signals)

## Interfaces

REST (`/v1/agent/utility/policy/*`) and MCP (`agent_utility_*` tools) expose the same behavior.

## What does not exist

- No automatic utility mutation from recall exposure
- No LLM judge
- No production proof that arbitrary agents obey the contract
- Utility scores are not objective truth — they are policy-gated reputation signals
