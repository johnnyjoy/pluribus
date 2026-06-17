# Recall Scoring Principles

Phase 4 scoring changes follow these principles. All are implemented generally in `control-plane/internal/recall/relevance_scoring.go` and `scorer.go` — not benchmark-specific.

## 1. Relevance First, Authority Second

Total score is driven by a **relevance core** (lexical, situational affinity, tag alignment, symbol overlap). Authority applies as a **multiplier** (`0.45 + 0.55 × authNorm`), not as a dominant additive base. Sort order is **score descending**, with authority as tie-breaker only.

## 2. Domain Affinity Is Not a Silo

Wrong-domain and missing-anchor penalties reduce cross-project leakage. Repo-root basename affinity boosts same-workspace memories. Global governing constraints are not filtered out; they must still score on relevance when applicable.

## 3. Empty Tags Must Not Mean No Context

When request tags are empty, tag overlap is inferred from situation text tokens matched against memory tags. Query tokens, repo root, and memory tags jointly drive situational affinity.

## 4. Wrong-Domain Penalties Must Be Explainable

`ScoreComponentBreakdown` exposes `wrong_domain_penalty`, `generic_term_penalty`, `relevance_score`, `lexical_score`, `situational_score`, `tag_match_score`, and `matched_terms` on each ranked hit (via `JustificationMeta.components` and benchmark `RankedHit` fields).

## 5. Generic Queries Require Caution

`vagueQueryDampening` reduces scores when fewer than two distinctive query tokens exist. `FlattenBundleByScore` drops results below 8% of the top score (minimum 0.06) to avoid padding top-K with zero-relevance noise.

## 6. Status Matters

Active search is default. Lifecycle queries (`deprecated`, `superseded`, `sqlite` + `still`, etc.) also merge **superseded** candidates. Superseded rows are heavily down-ranked unless the query is lifecycle-oriented.

## 7. Global Constraints Must Survive When Relevant

RIU and global pool doctrine unchanged. Tag-aligned transferable memories can outrank higher-authority local rows when total score is higher.

## 8. Benchmark Metrics Decide

Threshold gate and committed baseline JSON are the acceptance judge. No benchmark label hardcoding; `bench:` and `domain:` infra tags are excluded from scoring.
