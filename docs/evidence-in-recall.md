# Evidence-in-recall bundles

**What:** When enabled, `POST /v1/recall/compile`, `GET /v1/recall/`, and `POST /v1/recall/compile-multi` attach **`supporting_evidence`** to each **`MemoryItem`** in the bundle. Each entry is a compact **`EvidenceRef`** (id, kind, title, summary, optional path hint)—**not** full artifact bodies.

**Why:** Curated memory stays primary; evidence provides **bounded “receipts”** so agents can trust or apply a rule without turning recall into a transcript or log dump.

**Authority:** **`MemoryItem.authority`** is unchanged. Evidence count does **not** increase authority; promotion gates and memory authority policy remain separate.

## Configuration

Under **`recall.evidence_in_bundle`** in the YAML loaded via **`CONFIG`** (fields in **`config.example.yaml`**). The default image uses committed **`configs/config.yaml`**; for host dev use **`configs/config.local.yaml`** copied from the example.

| Field | Default (when `enabled: true`) | Meaning |
|-------|----------------------------------|---------|
| `enabled` | `false` | Master switch (backward compatible). |
| `max_per_memory` | `2` | Max evidence refs per memory item. |
| `max_per_bundle` | `12` | Global cap across all items in bucket order. |
| `summary_max_chars` | `256` | Max runes for the synthetic summary line. |

Defaults are applied by `ApplyEvidenceInBundleDefaults` when the section is present and `enabled: true`.

## Selection

Linked evidence is loaded per memory via **`memory_evidence_links`**. Records are sorted by **`BaseScore(kind)`** (test > benchmark > log > observation), then **`created_at`** descending. The first **`max_per_memory`** refs are taken until **`max_per_bundle`** is exhausted. If the bundle budget runs out before every memory is hydrated, **`evidence_budget_applied`** is set on **`RecallBundle`**.

## Cache

Redis recall bundle keys include an **evidence fingerprint** so cached bundles with evidence do not collide with bundles compiled without this feature or with different caps.

## What this is not

- Not episodic / similarity retrieval.
- Not automatic full-file or log expansion in recall.
- Not a replacement for `GET /v1/evidence?memory_id=` when you need full metadata.

## Proof scenario (manual)

1. Link at least one evidence record to a memory (`POST /v1/evidence/{id}/link` with `memory_id`).
2. Set `recall.evidence_in_bundle.enabled: true` and restart.
3. Call recall compile with **tags** (and **`retrieval_query`**) that surface that memory in the bundle.
4. Confirm `supporting_evidence` on the corresponding `MemoryItem` and that disabling the feature removes it (new compile, new cache key).
