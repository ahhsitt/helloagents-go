package memory

import (
	"testing"
	"time"
)

func TestDefaultMemoryConfig(t *testing.T) {
	config := DefaultMemoryConfig()

	if config.StoragePath != "./memory_data" {
		t.Errorf("expected StoragePath './memory_data', got %q", config.StoragePath)
	}
	if config.MaxCapacity != 1000 {
		t.Errorf("expected MaxCapacity 1000, got %d", config.MaxCapacity)
	}
	if config.ImportanceThreshold != 0.1 {
		t.Errorf("expected ImportanceThreshold 0.1, got %f", config.ImportanceThreshold)
	}
	if config.DecayFactor != 0.95 {
		t.Errorf("expected DecayFactor 0.95, got %f", config.DecayFactor)
	}
	if config.WorkingMemoryCapacity != 100 {
		t.Errorf("expected WorkingMemoryCapacity 100, got %d", config.WorkingMemoryCapacity)
	}
	if config.WorkingMemoryTokens != 4000 {
		t.Errorf("expected WorkingMemoryTokens 4000, got %d", config.WorkingMemoryTokens)
	}
	if config.WorkingMemoryTTL != 2*time.Hour {
		t.Errorf("expected WorkingMemoryTTL 2h, got %v", config.WorkingMemoryTTL)
	}
	if config.EpisodicMemoryCapacity != 10000 {
		t.Errorf("expected EpisodicMemoryCapacity 10000, got %d", config.EpisodicMemoryCapacity)
	}
	if config.SemanticMemoryCapacity != 10000 {
		t.Errorf("expected SemanticMemoryCapacity 10000, got %d", config.SemanticMemoryCapacity)
	}
	if config.EmbedderType != "openai" {
		t.Errorf("expected EmbedderType 'openai', got %q", config.EmbedderType)
	}
	if config.EmbedderModel != "text-embedding-3-small" {
		t.Errorf("expected EmbedderModel 'text-embedding-3-small', got %q", config.EmbedderModel)
	}
}

func TestNewMemoryConfig(t *testing.T) {
	config := NewMemoryConfig()

	// Should have same defaults as DefaultMemoryConfig
	if config.MaxCapacity != 1000 {
		t.Errorf("expected MaxCapacity 1000, got %d", config.MaxCapacity)
	}
}

func TestWithStoragePath(t *testing.T) {
	path := "/custom/path"
	config := NewMemoryConfig(WithStoragePath(path))

	if config.StoragePath != path {
		t.Errorf("expected StoragePath %q, got %q", path, config.StoragePath)
	}
}

func TestWithMaxCapacity(t *testing.T) {
	capacity := 500
	config := NewMemoryConfig(WithMaxCapacity(capacity))

	if config.MaxCapacity != capacity {
		t.Errorf("expected MaxCapacity %d, got %d", capacity, config.MaxCapacity)
	}
}

func TestWithImportanceThreshold(t *testing.T) {
	threshold := float32(0.3)
	config := NewMemoryConfig(WithImportanceThreshold(threshold))

	if config.ImportanceThreshold != threshold {
		t.Errorf("expected ImportanceThreshold %f, got %f", threshold, config.ImportanceThreshold)
	}
}

func TestWithDecayFactor(t *testing.T) {
	factor := float32(0.9)
	config := NewMemoryConfig(WithDecayFactor(factor))

	if config.DecayFactor != factor {
		t.Errorf("expected DecayFactor %f, got %f", factor, config.DecayFactor)
	}
}

func TestWithWorkingMemoryConfig(t *testing.T) {
	capacity := 50
	tokens := 2000
	ttl := 1 * time.Hour

	config := NewMemoryConfig(WithWorkingMemoryConfig(capacity, tokens, ttl))

	if config.WorkingMemoryCapacity != capacity {
		t.Errorf("expected WorkingMemoryCapacity %d, got %d", capacity, config.WorkingMemoryCapacity)
	}
	if config.WorkingMemoryTokens != tokens {
		t.Errorf("expected WorkingMemoryTokens %d, got %d", tokens, config.WorkingMemoryTokens)
	}
	if config.WorkingMemoryTTL != ttl {
		t.Errorf("expected WorkingMemoryTTL %v, got %v", ttl, config.WorkingMemoryTTL)
	}
}

func TestWithEpisodicMemoryCapacity(t *testing.T) {
	capacity := 5000
	config := NewMemoryConfig(WithEpisodicMemoryCapacity(capacity))

	if config.EpisodicMemoryCapacity != capacity {
		t.Errorf("expected EpisodicMemoryCapacity %d, got %d", capacity, config.EpisodicMemoryCapacity)
	}
}

func TestWithSemanticMemoryCapacity(t *testing.T) {
	capacity := 5000
	config := NewMemoryConfig(WithSemanticMemoryCapacity(capacity))

	if config.SemanticMemoryCapacity != capacity {
		t.Errorf("expected SemanticMemoryCapacity %d, got %d", capacity, config.SemanticMemoryCapacity)
	}
}

func TestWithEmbedderConfig(t *testing.T) {
	embedderType := "dashscope"
	model := "text-embedding-v2"
	apiKey := "test-api-key"

	config := NewMemoryConfig(WithEmbedderConfig(embedderType, model, apiKey))

	if config.EmbedderType != embedderType {
		t.Errorf("expected EmbedderType %q, got %q", embedderType, config.EmbedderType)
	}
	if config.EmbedderModel != model {
		t.Errorf("expected EmbedderModel %q, got %q", model, config.EmbedderModel)
	}
	if config.EmbedderAPIKey != apiKey {
		t.Errorf("expected EmbedderAPIKey %q, got %q", apiKey, config.EmbedderAPIKey)
	}
}

func TestWithSQLiteConfig(t *testing.T) {
	path := "/custom/memory.db"
	config := NewMemoryConfig(WithSQLiteConfig(path))

	if config.SQLitePath != path {
		t.Errorf("expected SQLitePath %q, got %q", path, config.SQLitePath)
	}
}

func TestWithQdrantConfig(t *testing.T) {
	url := "https://qdrant.example.com:6333"
	apiKey := "qdrant-api-key"
	collection := "custom_collection"

	config := NewMemoryConfig(WithQdrantConfig(url, apiKey, collection))

	if config.QdrantURL != url {
		t.Errorf("expected QdrantURL %q, got %q", url, config.QdrantURL)
	}
	if config.QdrantAPIKey != apiKey {
		t.Errorf("expected QdrantAPIKey %q, got %q", apiKey, config.QdrantAPIKey)
	}
	if config.QdrantCollection != collection {
		t.Errorf("expected QdrantCollection %q, got %q", collection, config.QdrantCollection)
	}
}

func TestWithNeo4jConfig(t *testing.T) {
	uri := "bolt://localhost:7687"
	username := "neo4j"
	password := "password123"

	config := NewMemoryConfig(WithNeo4jConfig(uri, username, password))

	if config.Neo4jURI != uri {
		t.Errorf("expected Neo4jURI %q, got %q", uri, config.Neo4jURI)
	}
	if config.Neo4jUsername != username {
		t.Errorf("expected Neo4jUsername %q, got %q", username, config.Neo4jUsername)
	}
	if config.Neo4jPassword != password {
		t.Errorf("expected Neo4jPassword %q, got %q", password, config.Neo4jPassword)
	}
}

func TestMultipleOptions(t *testing.T) {
	config := NewMemoryConfig(
		WithStoragePath("/custom"),
		WithMaxCapacity(200),
		WithWorkingMemoryConfig(20, 1000, 30*time.Minute),
		WithEmbedderConfig("local", "model", "key"),
	)

	if config.StoragePath != "/custom" {
		t.Errorf("expected StoragePath '/custom', got %q", config.StoragePath)
	}
	if config.MaxCapacity != 200 {
		t.Errorf("expected MaxCapacity 200, got %d", config.MaxCapacity)
	}
	if config.WorkingMemoryCapacity != 20 {
		t.Errorf("expected WorkingMemoryCapacity 20, got %d", config.WorkingMemoryCapacity)
	}
	if config.EmbedderType != "local" {
		t.Errorf("expected EmbedderType 'local', got %q", config.EmbedderType)
	}
}
