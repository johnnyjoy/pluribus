-- Smoke: pg_textsearch extension + BM25 index + ranked query.
-- Run inside the experimental Postgres container, e.g.:
--   docker compose -f docker-compose.yml -f docker-compose.pg-textsearch.yml exec postgres \
--     psql -U controlplane -d controlplane -v ON_ERROR_STOP=1 -f /path/to/pg_textsearch_smoke.sql
-- Or pipe: psql ... < scripts/experiments/pg_textsearch_smoke.sql

CREATE EXTENSION IF NOT EXISTS pg_textsearch;

DROP TABLE IF EXISTS pg_textsearch_smoke_docs;
CREATE TABLE pg_textsearch_smoke_docs (
  id bigserial PRIMARY KEY,
  content text NOT NULL
);

INSERT INTO pg_textsearch_smoke_docs (content) VALUES
  ('PostgreSQL is a powerful database system'),
  ('BM25 is an effective ranking function'),
  ('Full text search with custom scoring');

CREATE INDEX pg_textsearch_smoke_docs_bm25 ON pg_textsearch_smoke_docs USING bm25 (content)
  WITH (text_config = 'english');

-- Lower scores = better match (BM25 negative score convention)
SELECT id, content, content <@> 'database system' AS neg_bm25
FROM pg_textsearch_smoke_docs
ORDER BY content <@> 'database system'
LIMIT 5;
