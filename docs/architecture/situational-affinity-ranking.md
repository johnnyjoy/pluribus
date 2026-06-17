# Situational affinity ranking

**Status:** Implemented in `control-plane/internal/recall/situational_affinity.go` (additive ranking term).

## What it is

An **additive** score component in the existing recall compiler/ranker:

```
total_score = existing_recall_score + (weight_situational_affinity × situation_affinity)
```

`situation_affinity` is **query coverage**: |Q ∩ M| / |Q| where:

- **Q** = domain tokens from `retrieval_query`, optional `repo_root` basename, and request `tags`
- **M** = domain tokens from memory `statement` and memory `tags`

Domain tokens are lowercase, length ≥ 4, stopword-stripped (same family as lexical similarity).

## What it is not

- **Not** a partition, filter, or second memory store
- **Not** hard exclusion of cross-project memories
- **Not** a replacement for authority, supersession, or contradiction policy

Memories from other projects can still surface when they genuinely match the situation (including shared words like “architecture” or “integration”).

## Why it exists

Evidence from the hostile-review incident (2026-05):

1. **`recall_context`** sent **no request tags** → `tagMatchScore` returned **1.0 for every row** (neutral boost, but mislabeled as `tag_match` in justifications).
2. **`FailureOverlap`** applied to **all failures** whenever `tagMatch > 0` → with empty tags, **every failure** received +0.5.
3. Query tokens **`hostile`** + **`review`** strongly matched OnGuard/CampusCard failure rows that literally contain “hostile VAN review”.
4. **Semantic retrieval** was unavailable (`no_embedder`) → lexical path only.
5. **Pluribus-tagged corpus** in the shared pool is sparse relative to other projects.

Result: confident, irrelevant recall — worse than empty recall.

## Related fix (same change set)

**Failure overlap** now requires **non-empty request tags** and real tag overlap. Empty-tag MCP recalls no longer give all failures a free +0.5.

**Dominant reason** no longer reports `tag_match` when request tags are empty (avoids misleading “Why surfaced: tag_match”).

## Configuration

`recall.ranking.weight_situational_affinity` (default **0.35** in `configs/config.yaml`).

Set to **0** to disable the term (other ranking unchanged).

## Request fields

- `retrieval_query` / task text (required for affinity)
- `repo_root` (optional basename token, e.g. `pluribus`) — MCP `recall_context` accepts `repo_root`, `workspace_root`, or `project_root`
- `tags` (optional; add tokens to Q-side coverage, still no filter)

## Evaluation

Table-driven regression: `TestSituationalAffinity_evalDataset` in `situational_affinity_test.go`.

Metrics encoded in tests:

- Top-1 contains expected domain label (Pluribus / OnGuard / UDA)
- Pluribus hostile-review fixture ranks Pluribus rows above OnGuard fixture
- Failure overlap gated on request tags

Run:

```bash
cd control-plane && go test ./internal/recall/ -run SituationalAffinity -v
```

## Doctrine alignment

See [memory-doctrine.md](memory-doctrine.md) §E: recall is **situation-shaped**, not silo-partitioned. Affinity strengthens **situation** signal without introducing forbidden container partitions.
