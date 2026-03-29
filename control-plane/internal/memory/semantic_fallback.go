package memory

// Semantic fallback reason codes (compile response + logs). Not an exhaustive enum of all errors.
const (
	SemanticFallbackNoEmbedder          = "no_embedder"
	SemanticFallbackDimensionMismatch   = "dimension_mismatch"
	SemanticFallbackEmptyQuery          = "empty_query"
	SemanticFallbackRetrievalDisabled   = "semantic_retrieval_disabled"
	SemanticFallbackEmbeddingFailed     = "embedding_failed"
	SemanticFallbackVectorSearchFailed  = "vector_search_failed"
	SemanticFallbackBackendUnsupported  = "backend_unsupported"
)
