# Data model: canonical memory vs lexical projection

## Canonical source

| Table | Role |
|-------|------|
| `memories` | Source of truth for statements, authority, status, kinds |
| `memory_relationships` | Edges (not duplicated in lexical layer by default) |

## Lexical projection (recommended)

A dedicated table, e.g. **`lexical_memory_projection`**, **not** a replacement for `memories`:**

| Column | Purpose |
|--------|---------|
| `memory_id` | `UUID` PK / FK → `memories(id)` |
| `doc_text` | Single searchable text column for `USING bm25(doc_text)` |
| Optional | `kind`, `tags` JSONB, etc. — only for **pre-filter** indexes if needed |

### Source fields → `doc_text`

Typical extract (ETL):

- Primary: `statement` (or normalized statement field if you add one)
- Optional: high-signal keywords from tags (flattened), **not** full JSON blobs

**Do not** index raw blobs that duplicate enforcement-only payloads; keep governance in SQL/Go, not in BM25 noise.

### Updates

- On memory insert/update/delete: **enqueue** projection update (async) or synchronous upsert in the same transaction as you prefer (experimentation).
- **Reindex**: truncate projection + bulk backfill from `memories` — safe because canonical rows are untouched.

### What is not indexed

- Full `memories` row as JSON
- Advisory-only tables as canonical memory (unless explicitly part of a product decision)

## Direct indexing on `memories`?

Possible but **discouraged** for exploration: couples index lifecycle to the truth table DDL and complicates rollback. Prefer **projection**.
