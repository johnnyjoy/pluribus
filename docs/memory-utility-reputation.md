# Memory Utility and Reputation

Phase 7 foundation for structured memory-quality feedback. This is **not** a voting system, popularity contest, or truth oracle.

## Authority

Existing concept. A **priority/importance/control signal** used by enforcement and legacy ranking multipliers.

**Authority is not reputation.** Authority is not utility.

## Utility Score

New concept (Phase 7). A **bounded memory-quality influence signal** derived from append-only feedback events.

- Range: **-10 to +10** (neutral = 0)
- Stored in `memory_utility_scores.utility_score`
- Separate column from `memories.authority`
- Affects recall ranking via a **bounded additive term** — does not replace lexical relevance

## Feedback Event

Structured row in `memory_utility_events` attached to a `memory_id`, optionally correlated with recall session/bundle context.

### Exposed types (Phase 7)

| Type | Meaning | Default weight |
|------|---------|----------------|
| `helpful` | Materially improved answer, decision, or task | +1 |
| `harmful` | Contributed to bad outcome or wasted work | -2 |
| `wrong` | Factually incorrect or contradicted | -3 |
| `outdated` | Was true but no longer current | -1 |
| `irrelevant` | Recalled but did not help in context | -0.5 |

### Schema-reserved (not exposed via REST/MCP in Phase 7)

| Type | Meaning | Default weight |
|------|---------|----------------|
| `confirmed` | Stronger positive than helpful | +2 |
| `refuted` | Stronger negative; used for contradiction demotion | -4 |

Negative feedback (`harmful`, `wrong`, `outdated`) **requires a non-empty reason**.

## Confirmation and Refutation

- **Confirmation** — positive signal stronger than helpful; Phase 7 reserves `confirmed` in schema.
- **Refutation** — negative signal; contradiction create records `refuted` for involved memories.

## What Is Not Utility

```text
Recall frequency is not utility.
Duplicate repetition is not confirmation.
Evidence attachment is not verification (Phase 7).
Authority is not reputation.
```

## Recall Reinforcement (Phase 7 default)

```yaml
memory:
  utility:
    reinforce_on_recall: false          # default: recall alone does not bump authority
    reinforce_duplicate_authority: false # default: duplicate write does not bump authority
    utility_ranking_weight: 0.12
```

Legacy behavior is **opt-in** via explicit `true` in config.

## APIs

### REST

- `POST /v1/memory/{id}/feedback` — record feedback
- `GET /v1/memory/{id}/feedback` — list events
- `GET /v1/memory/{id}/utility` — read score + counts

### MCP

- `memory_feedback` — same validation as REST

## Agent Loop

After meaningful recall use, agents may submit `memory_feedback` with `memory_id` from the recall bundle. Feedback may include `correlation_id` / `recall_bundle_id` for audit linkage.

## Contradiction Demotion

Creating a contradiction record triggers `refuted` utility events for both involved memories (bounded score decrease). High-risk governing constraints may move to `pending`.

## Honest Limits (Phase 7)

- Feedback is agent-supplied and may be imperfect
- No source trust model
- No semantic truth verification
- Utility affects **ranking**, not enforcement authority (unless legacy reinforce flags enabled)
- Bad memories are still possible; influence should decrease when marked wrong
