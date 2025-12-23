package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/easyops/helloagents-go/pkg/memory"
)

// mockEmbedder implements memory.Embedder for testing
type mockEmbedder struct {
	embedFn func(ctx context.Context, texts []string) ([][]float32, error)
}

func (m *mockEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if m.embedFn != nil {
		return m.embedFn(ctx, texts)
	}
	// Return simple embeddings - each text gets a unique vector
	result := make([][]float32, len(texts))
	for i := range texts {
		vec := make([]float32, 128)
		// Simple hash-based embedding for testing
		for j, c := range texts[i] {
			if j < 128 {
				vec[j] = float32(c) / 256.0
			}
		}
		result[i] = vec
	}
	return result, nil
}

func newMockEmbedder() *mockEmbedder {
	return &mockEmbedder{}
}

func TestNewSemanticMemory(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	if mem == nil {
		t.Fatal("expected non-nil memory")
	}
	if mem.Size() != 0 {
		t.Fatalf("expected size 0, got %d", mem.Size())
	}
}

func TestSemanticMemory_Store(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	err := mem.Store(ctx, "id-1", "Hello world", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mem.Size() != 1 {
		t.Fatalf("expected size 1, got %d", mem.Size())
	}
}

func TestSemanticMemory_StoreWithAutoID(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	err := mem.Store(ctx, "", "Hello world", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mem.Size() != 1 {
		t.Fatalf("expected size 1, got %d", mem.Size())
	}
}

func TestSemanticMemory_StoreWithMetadata(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	metadata := map[string]interface{}{
		"source": "test",
		"type":   "preference",
	}

	err := mem.Store(ctx, "id-1", "User preference", metadata)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSemanticMemory_Search(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	_ = mem.Store(ctx, "id-1", "Hello world", nil)
	_ = mem.Store(ctx, "id-2", "Goodbye world", nil)
	_ = mem.Store(ctx, "id-3", "Hello there", nil)

	results, err := mem.Search(ctx, "Hello", 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestSemanticMemory_SearchWithThreshold(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	_ = mem.Store(ctx, "id-1", "Hello world", nil)
	_ = mem.Store(ctx, "id-2", "Something completely different", nil)

	results, err := mem.SearchWithThreshold(ctx, "Hello", 5, 0.9)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Only high-scoring results should be returned
	for _, r := range results {
		if r.Score < 0.9 {
			t.Fatalf("expected score >= 0.9, got %f", r.Score)
		}
	}
}

func TestSemanticMemory_Delete(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	_ = mem.Store(ctx, "id-1", "Hello world", nil)
	_ = mem.Store(ctx, "id-2", "Goodbye world", nil)

	err := mem.Delete(ctx, "id-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mem.Size() != 1 {
		t.Fatalf("expected size 1, got %d", mem.Size())
	}
}

func TestSemanticMemory_DeleteNotFound(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	err := mem.Delete(ctx, "non-existent")
	if err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestSemanticMemory_Clear(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	_ = mem.Store(ctx, "id-1", "Hello", nil)
	_ = mem.Store(ctx, "id-2", "World", nil)

	err := mem.Clear(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mem.Size() != 0 {
		t.Fatalf("expected size 0, got %d", mem.Size())
	}
}

func TestSemanticMemory_Update(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	_ = mem.Store(ctx, "id-1", "Original content", nil)
	_ = mem.Store(ctx, "id-1", "Updated content", nil)

	if mem.Size() != 1 {
		t.Fatalf("expected size 1 after update, got %d", mem.Size())
	}

	results, _ := mem.Search(ctx, "Updated", 1)
	if len(results) == 0 {
		t.Fatal("expected to find updated content")
	}
}

func TestSemanticMemory_SearchResultFields(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	metadata := map[string]interface{}{"key": "value"}
	_ = mem.Store(ctx, "id-1", "Test content", metadata)

	results, _ := mem.Search(ctx, "Test", 1)
	if len(results) != 1 {
		t.Fatal("expected 1 result")
	}

	r := results[0]
	if r.ID != "id-1" {
		t.Fatalf("expected ID 'id-1', got %s", r.ID)
	}
	if r.Content != "Test content" {
		t.Fatalf("expected content 'Test content', got %s", r.Content)
	}
	if r.Score <= 0 {
		t.Fatal("expected positive score")
	}
}

func TestSemanticMemory_ImplementsVectorMemory(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	var _ memory.VectorMemory = mem
}

func TestSemanticMemory_ImplementsMemory(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	var _ memory.Memory = mem
}

// ============================================================================
// Memory interface tests
// ============================================================================

func TestSemanticMemory_Add(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	item := memory.NewMemoryItem("Test content", memory.MemoryTypeSemantic,
		memory.WithImportance(0.8),
	)

	id, err := mem.Add(ctx, item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != item.ID {
		t.Errorf("expected id %s, got %s", item.ID, id)
	}
	if mem.Size() != 1 {
		t.Errorf("expected size 1, got %d", mem.Size())
	}
}

func TestSemanticMemory_Retrieve(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	_ = mem.Store(ctx, "id-1", "The cat sat on the mat", nil)
	_ = mem.Store(ctx, "id-2", "The dog ran in the park", nil)
	_ = mem.Store(ctx, "id-3", "Birds fly in the sky", nil)

	results, err := mem.Retrieve(ctx, "cat", memory.WithLimit(2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected at least one result")
	}
}

func TestSemanticMemory_UpdateMemory(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	_ = mem.Store(ctx, "id-1", "Original content", nil)

	newContent := "Updated content"
	err := mem.Update(ctx, "id-1", memory.WithContentUpdate(newContent))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify update
	results, _ := mem.Search(ctx, "Updated", 1)
	if len(results) == 0 {
		t.Error("expected to find updated content")
	}
}

func TestSemanticMemory_Remove(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	_ = mem.Store(ctx, "id-1", "Test content", nil)

	err := mem.Remove(ctx, "id-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mem.Size() != 0 {
		t.Errorf("expected size 0, got %d", mem.Size())
	}
}

func TestSemanticMemory_Has(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	_ = mem.Store(ctx, "id-1", "Test content", nil)

	if !mem.Has(ctx, "id-1") {
		t.Error("expected Has to return true for existing id")
	}
	if mem.Has(ctx, "nonexistent") {
		t.Error("expected Has to return false for nonexistent id")
	}
}

func TestSemanticMemory_GetStats(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	_ = mem.Store(ctx, "id-1", "Content 1", map[string]interface{}{"importance": float32(0.5)})
	_ = mem.Store(ctx, "id-2", "Content 2", map[string]interface{}{"importance": float32(0.8)})

	stats, err := mem.GetStats(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Count != 2 {
		t.Errorf("expected count 2, got %d", stats.Count)
	}
	expectedAvg := float32(0.65)
	if stats.AvgImportance < expectedAvg-0.01 || stats.AvgImportance > expectedAvg+0.01 {
		t.Errorf("expected avg importance around %f, got %f", expectedAvg, stats.AvgImportance)
	}
}

// ============================================================================
// Entity management tests
// ============================================================================

func TestSemanticMemory_AddEntity(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	entity := memory.NewEntity("John Doe", memory.EntityTypePerson)
	entity.Description = "A test person"

	err := mem.AddEntity(ctx, entity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mem.EntityCount() != 1 {
		t.Errorf("expected 1 entity, got %d", mem.EntityCount())
	}
}

func TestSemanticMemory_GetEntity(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	entity := memory.NewEntity("John Doe", memory.EntityTypePerson)
	_ = mem.AddEntity(ctx, entity)

	retrieved, err := mem.GetEntity(ctx, entity.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved.Name != "John Doe" {
		t.Errorf("expected name 'John Doe', got %s", retrieved.Name)
	}
}

func TestSemanticMemory_GetEntityByName(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	entity := memory.NewEntity("Acme Corp", memory.EntityTypeOrganization)
	_ = mem.AddEntity(ctx, entity)

	retrieved, err := mem.GetEntityByName(ctx, "acme corp") // Case insensitive
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved.Name != "Acme Corp" {
		t.Errorf("expected name 'Acme Corp', got %s", retrieved.Name)
	}
}

func TestSemanticMemory_SearchEntities(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	_ = mem.AddEntity(ctx, memory.NewEntity("Apple Inc", memory.EntityTypeOrganization))
	_ = mem.AddEntity(ctx, memory.NewEntity("Apple Store", memory.EntityTypeLocation))
	_ = mem.AddEntity(ctx, memory.NewEntity("Microsoft Corp", memory.EntityTypeOrganization))

	results, err := mem.SearchEntities(ctx, "apple", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestSemanticMemory_DeleteEntity(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	entity := memory.NewEntity("Test Entity", memory.EntityTypeConcept)
	_ = mem.AddEntity(ctx, entity)

	err := mem.DeleteEntity(ctx, entity.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mem.EntityCount() != 0 {
		t.Errorf("expected 0 entities, got %d", mem.EntityCount())
	}
}

func TestSemanticMemory_EntityFrequency(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	entity := memory.NewEntity("Recurring Entity", memory.EntityTypeConcept)
	_ = mem.AddEntity(ctx, entity)

	// Add same entity again - should increment frequency
	entity2 := memory.NewEntity("Recurring Entity", memory.EntityTypeConcept)
	_ = mem.AddEntity(ctx, entity2)

	// Should still be 1 entity but with higher frequency
	if mem.EntityCount() != 1 {
		t.Errorf("expected 1 entity, got %d", mem.EntityCount())
	}

	retrieved, _ := mem.GetEntityByName(ctx, "Recurring Entity")
	if retrieved.Frequency != 2 {
		t.Errorf("expected frequency 2, got %d", retrieved.Frequency)
	}
}

// ============================================================================
// Relation management tests
// ============================================================================

func TestSemanticMemory_AddRelation(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	entity1 := memory.NewEntity("Alice", memory.EntityTypePerson)
	entity2 := memory.NewEntity("Acme Corp", memory.EntityTypeOrganization)
	_ = mem.AddEntity(ctx, entity1)
	_ = mem.AddEntity(ctx, entity2)

	rel := memory.NewRelation(entity1.ID, entity2.ID, memory.RelationTypeWorksAt)
	err := mem.AddRelation(ctx, rel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mem.RelationCount() != 1 {
		t.Errorf("expected 1 relation, got %d", mem.RelationCount())
	}
}

func TestSemanticMemory_GetRelation(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	entity1 := memory.NewEntity("Alice", memory.EntityTypePerson)
	entity2 := memory.NewEntity("Bob", memory.EntityTypePerson)
	_ = mem.AddEntity(ctx, entity1)
	_ = mem.AddEntity(ctx, entity2)

	rel := memory.NewRelation(entity1.ID, entity2.ID, memory.RelationTypeKnows)
	_ = mem.AddRelation(ctx, rel)

	retrieved, err := mem.GetRelation(ctx, rel.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved.RelationType != memory.RelationTypeKnows {
		t.Errorf("expected relation type 'knows', got %s", retrieved.RelationType)
	}
}

func TestSemanticMemory_GetRelatedEntities(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	// Create a simple graph: Alice -> Bob -> Charlie
	alice := memory.NewEntity("Alice", memory.EntityTypePerson)
	bob := memory.NewEntity("Bob", memory.EntityTypePerson)
	charlie := memory.NewEntity("Charlie", memory.EntityTypePerson)
	_ = mem.AddEntity(ctx, alice)
	_ = mem.AddEntity(ctx, bob)
	_ = mem.AddEntity(ctx, charlie)

	rel1 := memory.NewRelation(alice.ID, bob.ID, memory.RelationTypeKnows)
	rel2 := memory.NewRelation(bob.ID, charlie.ID, memory.RelationTypeKnows)
	_ = mem.AddRelation(ctx, rel1)
	_ = mem.AddRelation(ctx, rel2)

	// Get entities related to Alice with depth 2
	results, err := mem.GetRelatedEntities(ctx, alice.ID, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 related entities, got %d", len(results))
	}

	// Bob should be at depth 1, Charlie at depth 2
	depths := make(map[string]int)
	for _, r := range results {
		depths[r.Entity.Name] = r.Depth
	}
	if depths["Bob"] != 1 {
		t.Errorf("expected Bob at depth 1, got %d", depths["Bob"])
	}
	if depths["Charlie"] != 2 {
		t.Errorf("expected Charlie at depth 2, got %d", depths["Charlie"])
	}
}

func TestSemanticMemory_DeleteRelation(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	entity1 := memory.NewEntity("A", memory.EntityTypeConcept)
	entity2 := memory.NewEntity("B", memory.EntityTypeConcept)
	_ = mem.AddEntity(ctx, entity1)
	_ = mem.AddEntity(ctx, entity2)

	rel := memory.NewRelation(entity1.ID, entity2.ID, memory.RelationTypeRelatedTo)
	_ = mem.AddRelation(ctx, rel)

	err := mem.DeleteRelation(ctx, rel.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mem.RelationCount() != 0 {
		t.Errorf("expected 0 relations, got %d", mem.RelationCount())
	}
}

func TestSemanticMemory_DeleteEntityRemovesRelations(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	entity1 := memory.NewEntity("A", memory.EntityTypeConcept)
	entity2 := memory.NewEntity("B", memory.EntityTypeConcept)
	entity3 := memory.NewEntity("C", memory.EntityTypeConcept)
	_ = mem.AddEntity(ctx, entity1)
	_ = mem.AddEntity(ctx, entity2)
	_ = mem.AddEntity(ctx, entity3)

	// A -- B, B -- C
	_ = mem.AddRelation(ctx, memory.NewRelation(entity1.ID, entity2.ID, memory.RelationTypeRelatedTo))
	_ = mem.AddRelation(ctx, memory.NewRelation(entity2.ID, entity3.ID, memory.RelationTypeRelatedTo))

	// Delete B - should remove both relations
	_ = mem.DeleteEntity(ctx, entity2.ID)

	if mem.RelationCount() != 0 {
		t.Errorf("expected 0 relations after deleting connected entity, got %d", mem.RelationCount())
	}
}

// ============================================================================
// Entity extraction tests
// ============================================================================

func TestSemanticMemory_ExtractEntities(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)

	content := "John Smith works at Google Inc located in San Francisco."
	entities := mem.ExtractEntities(content)

	if len(entities) == 0 {
		t.Error("expected to extract at least one entity")
	}

	// Check that we found person
	hasPerson := false
	for _, e := range entities {
		if e.Type == memory.EntityTypePerson && e.Name == "John Smith" {
			hasPerson = true
		}
	}
	if !hasPerson {
		t.Error("expected to find person entity 'John Smith'")
	}
}

func TestSemanticMemory_ExtractEntitiesChinese(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)

	content := "张明在北京市工作于阿里巴巴公司"
	entities := mem.ExtractEntities(content)

	if len(entities) == 0 {
		t.Error("expected to extract at least one Chinese entity")
	}

	// Check entity types
	hasLocation := false
	hasOrg := false
	for _, e := range entities {
		if e.Type == memory.EntityTypeLocation {
			hasLocation = true
		}
		if e.Type == memory.EntityTypeOrganization {
			hasOrg = true
		}
	}

	if !hasLocation {
		t.Error("expected to find location entity")
	}
	if !hasOrg {
		t.Error("expected to find organization entity")
	}
}

func TestSemanticMemory_ExtractRelations(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)

	content := "John Smith works at Anthropic Inc."
	entities := mem.ExtractEntities(content)
	relations := mem.ExtractRelations(content, entities)

	if len(entities) < 2 {
		t.Skip("not enough entities extracted for relation test")
	}

	if len(relations) == 0 {
		t.Error("expected to extract at least one relation")
	}
}

// ============================================================================
// Forget tests
// ============================================================================

func TestSemanticMemory_ForgetByImportance(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	_ = mem.Store(ctx, "id-1", "Low importance", map[string]interface{}{"importance": float32(0.1)})
	_ = mem.Store(ctx, "id-2", "High importance", map[string]interface{}{"importance": float32(0.9)})

	count, err := mem.Forget(ctx, memory.ForgetByImportance, memory.WithThreshold(0.5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 forgotten, got %d", count)
	}
	if mem.Size() != 1 {
		t.Errorf("expected size 1, got %d", mem.Size())
	}
}

func TestSemanticMemory_ForgetByTime(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	// Store will use current time, so both will be recent
	_ = mem.Store(ctx, "id-1", "Recent content", nil)
	_ = mem.Store(ctx, "id-2", "Also recent", nil)

	// With 30 day cutoff, nothing should be forgotten
	count, err := mem.Forget(ctx, memory.ForgetByTime, memory.WithMaxAgeDays(30))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 forgotten for recent items, got %d", count)
	}
}

func TestSemanticMemory_ForgetByCapacity(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	_ = mem.Store(ctx, "id-1", "Low 1", map[string]interface{}{"importance": float32(0.2)})
	_ = mem.Store(ctx, "id-2", "High", map[string]interface{}{"importance": float32(0.9)})
	_ = mem.Store(ctx, "id-3", "Low 2", map[string]interface{}{"importance": float32(0.3)})
	_ = mem.Store(ctx, "id-4", "Medium", map[string]interface{}{"importance": float32(0.6)})

	count, err := mem.Forget(ctx, memory.ForgetByCapacity, memory.WithTargetCapacity(2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 forgotten, got %d", count)
	}
	if mem.Size() != 2 {
		t.Errorf("expected size 2, got %d", mem.Size())
	}
}

// ============================================================================
// Hybrid search tests
// ============================================================================

func TestSemanticMemory_HybridSearch(t *testing.T) {
	// Test with nil embedder to verify TF-IDF fallback
	mem := memory.NewSemanticMemory(nil)
	ctx := context.Background()

	_ = mem.Store(ctx, "id-1", "The quick brown fox jumps over the lazy dog", nil)
	_ = mem.Store(ctx, "id-2", "A fast brown fox leaps over a sleeping hound", nil)
	_ = mem.Store(ctx, "id-3", "Cats and dogs are common pets", nil)

	results, err := mem.Search(ctx, "brown fox", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected at least one result from TF-IDF/keyword search")
	}
}

func TestSemanticMemory_KeywordFallback(t *testing.T) {
	mem := memory.NewSemanticMemory(nil)
	ctx := context.Background()

	_ = mem.Store(ctx, "id-1", "Hello world", nil)

	results, err := mem.Search(ctx, "hello", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result from keyword fallback, got %d", len(results))
	}
}

// ============================================================================
// Entity/Relation structure tests
// ============================================================================

func TestEntity_NewEntity(t *testing.T) {
	entity := memory.NewEntity("Test Entity", memory.EntityTypeConcept)

	if entity.Name != "Test Entity" {
		t.Errorf("expected name 'Test Entity', got %s", entity.Name)
	}
	if entity.Type != memory.EntityTypeConcept {
		t.Errorf("expected type 'concept', got %s", entity.Type)
	}
	if entity.ID == "" {
		t.Error("expected non-empty ID")
	}
	if entity.Frequency != 1 {
		t.Errorf("expected frequency 1, got %d", entity.Frequency)
	}
}

func TestEntity_SetProperty(t *testing.T) {
	entity := memory.NewEntity("Test", memory.EntityTypeConcept)
	entity.SetProperty("key", "value")

	val, ok := entity.GetProperty("key")
	if !ok {
		t.Error("expected property to exist")
	}
	if val != "value" {
		t.Errorf("expected 'value', got %v", val)
	}
}

func TestRelation_NewRelation(t *testing.T) {
	rel := memory.NewRelation("from-id", "to-id", memory.RelationTypeRelatedTo)

	if rel.FromEntityID != "from-id" {
		t.Errorf("expected from 'from-id', got %s", rel.FromEntityID)
	}
	if rel.ToEntityID != "to-id" {
		t.Errorf("expected to 'to-id', got %s", rel.ToEntityID)
	}
	if rel.Strength != 1.0 {
		t.Errorf("expected strength 1.0, got %f", rel.Strength)
	}
}

func TestRelation_AddEvidence(t *testing.T) {
	rel := memory.NewRelation("a", "b", memory.RelationTypeRelatedTo)
	rel.AddEvidence("mem-1")
	rel.AddEvidence("mem-2")
	rel.AddEvidence("mem-1") // Duplicate

	if len(rel.Evidence) != 2 {
		t.Errorf("expected 2 evidence items, got %d", len(rel.Evidence))
	}
}

func TestRelation_UpdateStrength(t *testing.T) {
	rel := memory.NewRelation("a", "b", memory.RelationTypeRelatedTo)

	rel.UpdateStrength(0.5)
	if rel.Strength != 0.5 {
		t.Errorf("expected strength 0.5, got %f", rel.Strength)
	}

	// Test bounds
	rel.UpdateStrength(1.5)
	if rel.Strength != 1.0 {
		t.Errorf("expected strength capped at 1.0, got %f", rel.Strength)
	}

	rel.UpdateStrength(-0.5)
	if rel.Strength != 0.0 {
		t.Errorf("expected strength capped at 0.0, got %f", rel.Strength)
	}
}

func TestNewEntityWithOptions(t *testing.T) {
	entity := memory.NewEntityWithOptions("Test", memory.EntityTypePerson,
		memory.WithEntityDescription("A test entity"),
		memory.WithEntityFrequency(5),
		memory.WithEntityProperties(map[string]interface{}{"age": 30}),
	)

	if entity.Description != "A test entity" {
		t.Errorf("expected description 'A test entity', got %s", entity.Description)
	}
	if entity.Frequency != 5 {
		t.Errorf("expected frequency 5, got %d", entity.Frequency)
	}
	if age, ok := entity.Properties["age"]; !ok || age != 30 {
		t.Error("expected property 'age' to be 30")
	}
}

func TestNewRelationWithOptions(t *testing.T) {
	rel := memory.NewRelationWithOptions("a", "b", memory.RelationTypeKnows,
		memory.WithRelationStrength(0.8),
		memory.WithRelationEvidence([]string{"ev1", "ev2"}),
		memory.WithRelationProperties(map[string]interface{}{"since": "2020"}),
	)

	if rel.Strength != 0.8 {
		t.Errorf("expected strength 0.8, got %f", rel.Strength)
	}
	if len(rel.Evidence) != 2 {
		t.Errorf("expected 2 evidence items, got %d", len(rel.Evidence))
	}
	if since, ok := rel.Properties["since"]; !ok || since != "2020" {
		t.Error("expected property 'since' to be '2020'")
	}
}

// ============================================================================
// Clear tests
// ============================================================================

func TestSemanticMemory_ClearAlsoRemovesEntitiesAndRelations(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	// Add records, entities, and relations
	_ = mem.Store(ctx, "id-1", "Content", nil)
	entity1 := memory.NewEntity("A", memory.EntityTypeConcept)
	entity2 := memory.NewEntity("B", memory.EntityTypeConcept)
	_ = mem.AddEntity(ctx, entity1)
	_ = mem.AddEntity(ctx, entity2)
	_ = mem.AddRelation(ctx, memory.NewRelation(entity1.ID, entity2.ID, memory.RelationTypeRelatedTo))

	// Clear
	_ = mem.Clear(ctx)

	if mem.Size() != 0 {
		t.Errorf("expected size 0, got %d", mem.Size())
	}
	if mem.EntityCount() != 0 {
		t.Errorf("expected 0 entities, got %d", mem.EntityCount())
	}
	if mem.RelationCount() != 0 {
		t.Errorf("expected 0 relations, got %d", mem.RelationCount())
	}
}

// ============================================================================
// Edge case tests
// ============================================================================

func TestSemanticMemory_AddRelationWithMissingEntity(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	rel := memory.NewRelation("nonexistent", "also-nonexistent", memory.RelationTypeRelatedTo)
	err := mem.AddRelation(ctx, rel)
	if err == nil {
		t.Error("expected error for relation with missing entities")
	}
}

func TestSemanticMemory_GetRelatedEntitiesNotFound(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	_, err := mem.GetRelatedEntities(ctx, "nonexistent", 2)
	if err == nil {
		t.Error("expected error for nonexistent entity")
	}
}

func TestSemanticMemory_SearchEmpty(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	results, err := mem.Search(ctx, "query", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil && len(results) != 0 {
		t.Errorf("expected nil or empty results, got %v", results)
	}
}

func TestSemanticMemory_StoreWithImportance(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	metadata := map[string]interface{}{
		"importance": float32(0.9),
		"user_id":    "user123",
	}
	_ = mem.Store(ctx, "id-1", "Important content", metadata)

	stats, _ := mem.GetStats(ctx)
	if stats.AvgImportance != 0.9 {
		t.Errorf("expected avg importance 0.9, got %f", stats.AvgImportance)
	}
}

// Ensure we properly handle timestamps
func TestSemanticMemory_StatsTimestamps(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	beforeStore := time.Now().UnixMilli()
	_ = mem.Store(ctx, "id-1", "Content", nil)
	afterStore := time.Now().UnixMilli()

	stats, _ := mem.GetStats(ctx)
	if stats.OldestTimestamp < beforeStore || stats.OldestTimestamp > afterStore {
		t.Error("timestamp outside expected range")
	}
}
