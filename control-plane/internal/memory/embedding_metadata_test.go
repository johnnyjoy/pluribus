package memory_test

import (
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"
)

func TestEmbeddingMetadataWrittenOnCreate(t *testing.T) {
	meta := memory.NewEmbeddingWriteMeta(api.MemoryKindPattern, "", "deploy via make deploy", "http", "test-model", 8)
	if meta.Model != "test-model" {
		t.Fatalf("model=%q", meta.Model)
	}
	if meta.Dimension != 8 {
		t.Fatalf("dim=%d", meta.Dimension)
	}
	if meta.SourceHash == "" {
		t.Fatal("expected source hash")
	}
	if meta.Status != memory.EmbeddingStatusValid {
		t.Fatalf("status=%q", meta.Status)
	}
	if meta.CreatedAt.IsZero() || meta.UpdatedAt.IsZero() {
		t.Fatal("expected timestamps")
	}
}

func TestEmbeddingSourceHashMatchesEmbeddedText(t *testing.T) {
	kind := api.MemoryKindPattern
	stmt := "always run make test before deploy"
	h1 := memory.ComputeEmbeddingSourceHash(kind, "", stmt)
	h2 := memory.ComputeEmbeddingSourceHash(kind, "", stmt)
	if h1 != h2 {
		t.Fatal("hash not stable")
	}
	text := memory.EmbeddingTextForMemory(kind, "", stmt)
	if text == "" {
		t.Fatal("expected embedding text")
	}
	changed := memory.ComputeEmbeddingSourceHash(kind, "", stmt+" changed")
	if changed == h1 {
		t.Fatal("expected different hash after statement change")
	}
}

func TestEmbeddingModelRecorded(t *testing.T) {
	meta := memory.NewEmbeddingWriteMeta(api.MemoryKindDecision, "", "x", "openai", "text-embedding-3-small", 1536)
	if meta.Model != "text-embedding-3-small" {
		t.Fatalf("model=%q", meta.Model)
	}
}

func TestEmbeddingDimensionRecorded(t *testing.T) {
	meta := memory.NewEmbeddingWriteMeta(api.MemoryKindPattern, "", "x", "http", "m", 512)
	if meta.Dimension != 512 {
		t.Fatalf("dim=%d", meta.Dimension)
	}
}

func TestEmbeddingCreatedAtRecorded(t *testing.T) {
	before := time.Now().UTC()
	meta := memory.NewEmbeddingWriteMeta(api.MemoryKindPattern, "", "x", "http", "m", 8)
	if meta.CreatedAt.Before(before) {
		t.Fatal("created_at too early")
	}
}

func TestStaleEmbeddingDetectedAfterStatementChange(t *testing.T) {
	obj := memory.MemoryObject{Kind: api.MemoryKindPattern, Statement: "new statement"}
	meta := memory.EmbeddingMeta{
		Model:      "text-embedding-3-small",
		Dimension:  8,
		SourceHash: memory.ComputeEmbeddingSourceHash(api.MemoryKindPattern, "", "old statement"),
		Status:     memory.EmbeddingStatusValid,
	}
	profile := memory.EmbeddingConfigProfile{Model: "text-embedding-3-small", Dimension: 8}
	if r := memory.CheckEmbeddingStaleness(obj, meta, profile); r != memory.StalenessSourceHashMismatch {
		t.Fatalf("reason=%q want source_hash_mismatch", r)
	}
}

func TestStaleEmbeddingDetectedAfterModelChange(t *testing.T) {
	stmt := "stable text"
	obj := memory.MemoryObject{Kind: api.MemoryKindPattern, Statement: stmt}
	meta := memory.EmbeddingMeta{
		Model:      "old-model",
		Dimension:  8,
		SourceHash: memory.ComputeEmbeddingSourceHash(api.MemoryKindPattern, "", stmt),
		Status:     memory.EmbeddingStatusValid,
	}
	profile := memory.EmbeddingConfigProfile{Model: "new-model", Dimension: 8}
	if r := memory.CheckEmbeddingStaleness(obj, meta, profile); r != memory.StalenessModelMismatch {
		t.Fatalf("reason=%q want model_mismatch", r)
	}
}

func TestMissingEmbeddingDetected(t *testing.T) {
	obj := memory.MemoryObject{Kind: api.MemoryKindPattern, Statement: "x"}
	meta := memory.EmbeddingMeta{Model: "m", Dimension: 8, SourceHash: "abc", Status: memory.EmbeddingStatusValid}
	profile := memory.EmbeddingConfigProfile{Model: "m", Dimension: 8}
	if r := memory.CheckEmbeddingStalenessWithVector(false, obj, meta, profile); r != memory.StalenessMissingEmbedding {
		t.Fatalf("reason=%q want missing_embedding", r)
	}
}

func TestDimensionMismatchDetected(t *testing.T) {
	stmt := "x"
	obj := memory.MemoryObject{Kind: api.MemoryKindPattern, Statement: stmt}
	meta := memory.EmbeddingMeta{
		Model:      "m",
		Dimension:  512,
		SourceHash: memory.ComputeEmbeddingSourceHash(api.MemoryKindPattern, "", stmt),
		Status:     memory.EmbeddingStatusValid,
	}
	profile := memory.EmbeddingConfigProfile{Model: "m", Dimension: 1536}
	if r := memory.CheckEmbeddingStaleness(obj, meta, profile); r != memory.StalenessDimensionMismatch {
		t.Fatalf("reason=%q want dimension_mismatch", r)
	}
}

func TestSemanticRetrievalDefaultOff(t *testing.T) {
	cfg := &memory.SemanticRetrievalConfig{}
	if cfg.RetrievalEnabled() {
		t.Fatal("semantic retrieval must be disabled when enabled flag unset")
	}
	cfg.Enabled = ptrBool(false)
	if cfg.RetrievalEnabled() {
		t.Fatal("semantic retrieval must be disabled when enabled=false")
	}
}

func ptrBool(v bool) *bool { return &v }

func TestLiveEmbedderBenchmarkRequiresExplicitConfig(t *testing.T) {
	t.Setenv("PLURIBUS_EMBEDDER_ENDPOINT", "")
	_, _, err := memory.LoadLiveEmbedderFromEnv()
	if err == nil {
		t.Fatal("expected error without PLURIBUS_EMBEDDER_ENDPOINT")
	}
}
