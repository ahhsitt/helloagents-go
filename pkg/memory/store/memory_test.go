package store

import (
	"context"
	"testing"
	"time"
)

// ============================================================================
// Document Store Tests
// ============================================================================

func TestNewMemoryDocumentStore(t *testing.T) {
	store := NewMemoryDocumentStore()
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestMemoryDocumentStore_PutGet(t *testing.T) {
	store := NewMemoryDocumentStore()
	ctx := context.Background()

	doc := Document{
		Content:  "test content",
		Metadata: map[string]interface{}{"key": "value"},
	}

	err := store.Put(ctx, "test", "doc1", doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	retrieved, err := store.Get(ctx, "test", "doc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retrieved.ID != "doc1" {
		t.Errorf("expected ID 'doc1', got %s", retrieved.ID)
	}
	if retrieved.Content != "test content" {
		t.Errorf("expected content 'test content', got %s", retrieved.Content)
	}
}

func TestMemoryDocumentStore_GetNotFound(t *testing.T) {
	store := NewMemoryDocumentStore()
	ctx := context.Background()

	_, err := store.Get(ctx, "test", "nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryDocumentStore_Delete(t *testing.T) {
	store := NewMemoryDocumentStore()
	ctx := context.Background()

	doc := Document{Content: "test"}
	_ = store.Put(ctx, "test", "doc1", doc)

	err := store.Delete(ctx, "test", "doc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = store.Get(ctx, "test", "doc1")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete")
	}
}

func TestMemoryDocumentStore_Query(t *testing.T) {
	store := NewMemoryDocumentStore()
	ctx := context.Background()

	_ = store.Put(ctx, "test", "doc1", Document{Content: "hello world", Metadata: map[string]interface{}{"type": "greeting"}})
	_ = store.Put(ctx, "test", "doc2", Document{Content: "goodbye world", Metadata: map[string]interface{}{"type": "farewell"}})
	_ = store.Put(ctx, "test", "doc3", Document{Content: "hello there", Metadata: map[string]interface{}{"type": "greeting"}})

	// Query all
	results, err := store.Query(ctx, "test", Filter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// Query with filter
	results, err = store.Query(ctx, "test", Filter{Field: "type", Op: "eq", Value: "greeting"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestMemoryDocumentStore_QueryWithLimit(t *testing.T) {
	store := NewMemoryDocumentStore()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		_ = store.Put(ctx, "test", "doc"+string(rune('0'+i)), Document{Content: "content"})
	}

	results, err := store.Query(ctx, "test", Filter{}, WithQueryLimit(5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}
}

func TestMemoryDocumentStore_Count(t *testing.T) {
	store := NewMemoryDocumentStore()
	ctx := context.Background()

	_ = store.Put(ctx, "test", "doc1", Document{Content: "a", Metadata: map[string]interface{}{"type": "a"}})
	_ = store.Put(ctx, "test", "doc2", Document{Content: "b", Metadata: map[string]interface{}{"type": "b"}})
	_ = store.Put(ctx, "test", "doc3", Document{Content: "c", Metadata: map[string]interface{}{"type": "a"}})

	count, err := store.Count(ctx, "test", Filter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}

	count, err = store.Count(ctx, "test", Filter{Field: "type", Op: "eq", Value: "a"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
}

func TestMemoryDocumentStore_Clear(t *testing.T) {
	store := NewMemoryDocumentStore()
	ctx := context.Background()

	_ = store.Put(ctx, "test", "doc1", Document{Content: "a"})
	_ = store.Put(ctx, "test", "doc2", Document{Content: "b"})

	err := store.Clear(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count, _ := store.Count(ctx, "test", Filter{})
	if count != 0 {
		t.Errorf("expected count 0 after clear, got %d", count)
	}
}

func TestMemoryDocumentStore_ContainsFilter(t *testing.T) {
	store := NewMemoryDocumentStore()
	ctx := context.Background()

	_ = store.Put(ctx, "test", "doc1", Document{Content: "hello world"})
	_ = store.Put(ctx, "test", "doc2", Document{Content: "goodbye world"})

	results, err := store.Query(ctx, "test", Filter{Field: "content", Op: "contains", Value: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

// ============================================================================
// Vector Store Tests
// ============================================================================

func TestNewMemoryVectorStore(t *testing.T) {
	store := NewMemoryVectorStore()
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestMemoryVectorStore_AddAndSearch(t *testing.T) {
	store := NewMemoryVectorStore()
	ctx := context.Background()

	vectors := []VectorRecord{
		{ID: "v1", Vector: []float32{1, 0, 0}, MemoryID: "m1"},
		{ID: "v2", Vector: []float32{0, 1, 0}, MemoryID: "m2"},
		{ID: "v3", Vector: []float32{0.9, 0.1, 0}, MemoryID: "m3"},
	}

	err := store.AddVectors(ctx, "test", vectors)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Search for similar to [1, 0, 0]
	results, err := store.SearchSimilar(ctx, "test", []float32{1, 0, 0}, 2, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	// v1 should be first (exact match)
	if results[0].ID != "v1" {
		t.Errorf("expected first result to be v1, got %s", results[0].ID)
	}
}

func TestMemoryVectorStore_SearchWithFilter(t *testing.T) {
	store := NewMemoryVectorStore()
	ctx := context.Background()

	vectors := []VectorRecord{
		{ID: "v1", Vector: []float32{1, 0, 0}, MemoryID: "m1", Payload: map[string]interface{}{"user_id": "user1"}},
		{ID: "v2", Vector: []float32{0.9, 0.1, 0}, MemoryID: "m2", Payload: map[string]interface{}{"user_id": "user2"}},
	}

	_ = store.AddVectors(ctx, "test", vectors)

	filter := &VectorFilter{UserID: "user1"}
	results, err := store.SearchSimilar(ctx, "test", []float32{1, 0, 0}, 10, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result with filter, got %d", len(results))
	}
}

func TestMemoryVectorStore_DeleteVectors(t *testing.T) {
	store := NewMemoryVectorStore()
	ctx := context.Background()

	vectors := []VectorRecord{
		{ID: "v1", Vector: []float32{1, 0, 0}},
		{ID: "v2", Vector: []float32{0, 1, 0}},
	}
	_ = store.AddVectors(ctx, "test", vectors)

	err := store.DeleteVectors(ctx, "test", []string{"v1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stats, _ := store.GetStats(ctx, "test")
	if stats.VectorCount != 1 {
		t.Errorf("expected 1 vector after delete, got %d", stats.VectorCount)
	}
}

func TestMemoryVectorStore_DeleteByFilter(t *testing.T) {
	store := NewMemoryVectorStore()
	ctx := context.Background()

	vectors := []VectorRecord{
		{ID: "v1", Vector: []float32{1, 0, 0}, MemoryID: "m1"},
		{ID: "v2", Vector: []float32{0, 1, 0}, MemoryID: "m2"},
	}
	_ = store.AddVectors(ctx, "test", vectors)

	err := store.DeleteByFilter(ctx, "test", &VectorFilter{MemoryID: "m1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stats, _ := store.GetStats(ctx, "test")
	if stats.VectorCount != 1 {
		t.Errorf("expected 1 vector after delete, got %d", stats.VectorCount)
	}
}

func TestMemoryVectorStore_Clear(t *testing.T) {
	store := NewMemoryVectorStore()
	ctx := context.Background()

	vectors := []VectorRecord{
		{ID: "v1", Vector: []float32{1, 0, 0}},
		{ID: "v2", Vector: []float32{0, 1, 0}},
	}
	_ = store.AddVectors(ctx, "test", vectors)

	err := store.Clear(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stats, _ := store.GetStats(ctx, "test")
	if stats.VectorCount != 0 {
		t.Errorf("expected 0 vectors after clear, got %d", stats.VectorCount)
	}
}

func TestMemoryVectorStore_GetStats(t *testing.T) {
	store := NewMemoryVectorStore()
	ctx := context.Background()

	vectors := []VectorRecord{
		{ID: "v1", Vector: []float32{1, 0, 0, 0}},
		{ID: "v2", Vector: []float32{0, 1, 0, 0}},
	}
	_ = store.AddVectors(ctx, "test", vectors)

	stats, err := store.GetStats(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.VectorCount != 2 {
		t.Errorf("expected 2 vectors, got %d", stats.VectorCount)
	}
	if stats.Dimensions != 4 {
		t.Errorf("expected 4 dimensions, got %d", stats.Dimensions)
	}
}

func TestMemoryVectorStore_HealthCheck(t *testing.T) {
	store := NewMemoryVectorStore()
	ctx := context.Background()

	err := store.HealthCheck(ctx)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// ============================================================================
// Graph Store Tests
// ============================================================================

func TestNewMemoryGraphStore(t *testing.T) {
	store := NewMemoryGraphStore()
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestMemoryGraphStore_AddGetEntity(t *testing.T) {
	store := NewMemoryGraphStore()
	ctx := context.Background()

	entity := &GraphEntity{
		ID:   "e1",
		Name: "Test Entity",
		Type: "concept",
	}

	err := store.AddEntity(ctx, entity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	retrieved, err := store.GetEntity(ctx, "e1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved.Name != "Test Entity" {
		t.Errorf("expected name 'Test Entity', got %s", retrieved.Name)
	}
}

func TestMemoryGraphStore_SearchEntities(t *testing.T) {
	store := NewMemoryGraphStore()
	ctx := context.Background()

	_ = store.AddEntity(ctx, &GraphEntity{ID: "e1", Name: "Apple Inc", Type: "organization"})
	_ = store.AddEntity(ctx, &GraphEntity{ID: "e2", Name: "Apple Store", Type: "location"})
	_ = store.AddEntity(ctx, &GraphEntity{ID: "e3", Name: "Microsoft", Type: "organization"})

	results, err := store.SearchEntities(ctx, "apple", "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// With type filter
	results, err = store.SearchEntities(ctx, "apple", "organization", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result with type filter, got %d", len(results))
	}
}

func TestMemoryGraphStore_DeleteEntity(t *testing.T) {
	store := NewMemoryGraphStore()
	ctx := context.Background()

	_ = store.AddEntity(ctx, &GraphEntity{ID: "e1", Name: "Entity 1", Type: "concept"})
	_ = store.AddEntity(ctx, &GraphEntity{ID: "e2", Name: "Entity 2", Type: "concept"})
	_ = store.AddRelation(ctx, &GraphRelation{ID: "r1", FromEntityID: "e1", ToEntityID: "e2", Type: "related_to"})

	err := store.DeleteEntity(ctx, "e1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = store.GetEntity(ctx, "e1")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete")
	}

	// Relation should also be deleted
	stats, _ := store.GetStats(ctx)
	if stats.RelationCount != 0 {
		t.Errorf("expected 0 relations after entity delete, got %d", stats.RelationCount)
	}
}

func TestMemoryGraphStore_AddGetRelation(t *testing.T) {
	store := NewMemoryGraphStore()
	ctx := context.Background()

	_ = store.AddEntity(ctx, &GraphEntity{ID: "e1", Name: "A", Type: "concept"})
	_ = store.AddEntity(ctx, &GraphEntity{ID: "e2", Name: "B", Type: "concept"})

	rel := &GraphRelation{
		ID:           "r1",
		FromEntityID: "e1",
		ToEntityID:   "e2",
		Type:         "related_to",
		Strength:     0.8,
	}

	err := store.AddRelation(ctx, rel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	relations, err := store.GetRelations(ctx, "e1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(relations) != 1 {
		t.Errorf("expected 1 relation, got %d", len(relations))
	}
}

func TestMemoryGraphStore_FindRelatedEntities(t *testing.T) {
	store := NewMemoryGraphStore()
	ctx := context.Background()

	// Create a graph: A -> B -> C
	_ = store.AddEntity(ctx, &GraphEntity{ID: "a", Name: "A", Type: "concept"})
	_ = store.AddEntity(ctx, &GraphEntity{ID: "b", Name: "B", Type: "concept"})
	_ = store.AddEntity(ctx, &GraphEntity{ID: "c", Name: "C", Type: "concept"})
	_ = store.AddRelation(ctx, &GraphRelation{ID: "r1", FromEntityID: "a", ToEntityID: "b", Type: "knows", Strength: 1.0})
	_ = store.AddRelation(ctx, &GraphRelation{ID: "r2", FromEntityID: "b", ToEntityID: "c", Type: "knows", Strength: 1.0})

	results, err := store.FindRelatedEntities(ctx, "a", "", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 related entities, got %d", len(results))
	}

	// Check depths
	depths := make(map[string]int)
	for _, r := range results {
		depths[r.Entity.Name] = r.Depth
	}
	if depths["B"] != 1 {
		t.Errorf("expected B at depth 1, got %d", depths["B"])
	}
	if depths["C"] != 2 {
		t.Errorf("expected C at depth 2, got %d", depths["C"])
	}
}

func TestMemoryGraphStore_GetShortestPath(t *testing.T) {
	store := NewMemoryGraphStore()
	ctx := context.Background()

	// Create: A -> B -> C
	_ = store.AddEntity(ctx, &GraphEntity{ID: "a", Name: "A", Type: "concept"})
	_ = store.AddEntity(ctx, &GraphEntity{ID: "b", Name: "B", Type: "concept"})
	_ = store.AddEntity(ctx, &GraphEntity{ID: "c", Name: "C", Type: "concept"})
	_ = store.AddRelation(ctx, &GraphRelation{ID: "r1", FromEntityID: "a", ToEntityID: "b", Type: "knows"})
	_ = store.AddRelation(ctx, &GraphRelation{ID: "r2", FromEntityID: "b", ToEntityID: "c", Type: "knows"})

	entities, relations, err := store.GetShortestPath(ctx, "a", "c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entities) != 3 {
		t.Errorf("expected 3 entities in path, got %d", len(entities))
	}
	if len(relations) != 2 {
		t.Errorf("expected 2 relations in path, got %d", len(relations))
	}
}

func TestMemoryGraphStore_GetShortestPath_SameNode(t *testing.T) {
	store := NewMemoryGraphStore()
	ctx := context.Background()

	_ = store.AddEntity(ctx, &GraphEntity{ID: "a", Name: "A", Type: "concept"})

	entities, relations, err := store.GetShortestPath(ctx, "a", "a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entities) != 1 {
		t.Errorf("expected 1 entity for same node path, got %d", len(entities))
	}
	if len(relations) != 0 {
		t.Errorf("expected 0 relations for same node path, got %d", len(relations))
	}
}

func TestMemoryGraphStore_DeleteRelation(t *testing.T) {
	store := NewMemoryGraphStore()
	ctx := context.Background()

	_ = store.AddEntity(ctx, &GraphEntity{ID: "e1", Name: "A", Type: "concept"})
	_ = store.AddEntity(ctx, &GraphEntity{ID: "e2", Name: "B", Type: "concept"})
	_ = store.AddRelation(ctx, &GraphRelation{ID: "r1", FromEntityID: "e1", ToEntityID: "e2", Type: "knows"})

	err := store.DeleteRelation(ctx, "r1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stats, _ := store.GetStats(ctx)
	if stats.RelationCount != 0 {
		t.Errorf("expected 0 relations after delete, got %d", stats.RelationCount)
	}
}

func TestMemoryGraphStore_Clear(t *testing.T) {
	store := NewMemoryGraphStore()
	ctx := context.Background()

	_ = store.AddEntity(ctx, &GraphEntity{ID: "e1", Name: "A", Type: "concept"})
	_ = store.AddEntity(ctx, &GraphEntity{ID: "e2", Name: "B", Type: "concept"})
	_ = store.AddRelation(ctx, &GraphRelation{ID: "r1", FromEntityID: "e1", ToEntityID: "e2", Type: "knows"})

	err := store.Clear(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stats, _ := store.GetStats(ctx)
	if stats.EntityCount != 0 {
		t.Errorf("expected 0 entities after clear, got %d", stats.EntityCount)
	}
	if stats.RelationCount != 0 {
		t.Errorf("expected 0 relations after clear, got %d", stats.RelationCount)
	}
}

func TestMemoryGraphStore_GetStats(t *testing.T) {
	store := NewMemoryGraphStore()
	ctx := context.Background()

	_ = store.AddEntity(ctx, &GraphEntity{ID: "e1", Name: "A", Type: "person"})
	_ = store.AddEntity(ctx, &GraphEntity{ID: "e2", Name: "B", Type: "organization"})
	_ = store.AddRelation(ctx, &GraphRelation{ID: "r1", FromEntityID: "e1", ToEntityID: "e2", Type: "works_at"})

	stats, err := store.GetStats(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.EntityCount != 2 {
		t.Errorf("expected 2 entities, got %d", stats.EntityCount)
	}
	if stats.RelationCount != 1 {
		t.Errorf("expected 1 relation, got %d", stats.RelationCount)
	}
	if stats.EntityTypes["person"] != 1 {
		t.Errorf("expected 1 person entity, got %d", stats.EntityTypes["person"])
	}
	if stats.RelationTypes["works_at"] != 1 {
		t.Errorf("expected 1 works_at relation, got %d", stats.RelationTypes["works_at"])
	}
}

// ============================================================================
// Config Tests
// ============================================================================

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config == nil {
		t.Fatal("expected non-nil config")
	}
	if config.Type != StoreTypeMemory {
		t.Errorf("expected type memory, got %s", config.Type)
	}
	if config.VectorDimensions != 128 {
		t.Errorf("expected 128 dimensions, got %d", config.VectorDimensions)
	}
}

// ============================================================================
// Cosine Similarity Tests
// ============================================================================

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a, b     []float32
		expected float32
	}{
		{"identical", []float32{1, 0, 0}, []float32{1, 0, 0}, 1.0},
		{"orthogonal", []float32{1, 0, 0}, []float32{0, 1, 0}, 0.0},
		{"opposite", []float32{1, 0, 0}, []float32{-1, 0, 0}, -1.0},
		{"empty", []float32{}, []float32{}, 0.0},
		{"different_length", []float32{1, 0}, []float32{1, 0, 0}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			if result < tt.expected-0.01 || result > tt.expected+0.01 {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

// ============================================================================
// Timestamp Tests
// ============================================================================

func TestDocumentTimestamps(t *testing.T) {
	store := NewMemoryDocumentStore()
	ctx := context.Background()

	before := time.Now()
	_ = store.Put(ctx, "test", "doc1", Document{Content: "test"})
	after := time.Now()

	doc, _ := store.Get(ctx, "test", "doc1")

	if doc.CreatedAt.Before(before) || doc.CreatedAt.After(after) {
		t.Error("created_at timestamp out of range")
	}
	if doc.UpdatedAt.Before(before) || doc.UpdatedAt.After(after) {
		t.Error("updated_at timestamp out of range")
	}
}

func TestGraphEntityTimestamps(t *testing.T) {
	store := NewMemoryGraphStore()
	ctx := context.Background()

	before := time.Now()
	_ = store.AddEntity(ctx, &GraphEntity{ID: "e1", Name: "Test", Type: "concept"})
	after := time.Now()

	entity, _ := store.GetEntity(ctx, "e1")

	if entity.CreatedAt.Before(before) || entity.CreatedAt.After(after) {
		t.Error("created_at timestamp out of range")
	}
	if entity.UpdatedAt.Before(before) || entity.UpdatedAt.After(after) {
		t.Error("updated_at timestamp out of range")
	}
}
