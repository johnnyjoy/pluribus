// Package pgtextsearch automates pg_textsearch evaluation: seed, projection ETL, BM25 index, query suite.
// Canonical rows live in memories; lexical_memory_projection is derived only.
package pgtextsearch

// EvalTag marks memories created by the evaluation seed generator.
const EvalTag = "experiment:pg-textsearch-eval"

// DefaultProjectionTable matches lexical.DefaultProjectionTable.
const DefaultProjectionTable = "lexical_memory_projection"

const bm25IndexName = "lexical_memory_projection_bm25"
