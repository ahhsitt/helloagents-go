package memory

import (
	"context"
	"testing"
)

// mockMemory is a mock implementation of the Memory interface for testing
type mockMemory struct {
	items   map[string]*MemoryItem
	addErr  error
	retErr  error
	memType MemoryType
}

func newMockMemory(memType MemoryType) *mockMemory {
	return &mockMemory{
		items:   make(map[string]*MemoryItem),
		memType: memType,
	}
}

func (m *mockMemory) Add(ctx context.Context, item *MemoryItem) (string, error) {
	if m.addErr != nil {
		return "", m.addErr
	}
	m.items[item.ID] = item
	return item.ID, nil
}

func (m *mockMemory) Retrieve(ctx context.Context, query string, opts ...RetrieveOption) ([]*MemoryItem, error) {
	if m.retErr != nil {
		return nil, m.retErr
	}
	result := make([]*MemoryItem, 0, len(m.items))
	for _, item := range m.items {
		result = append(result, item)
	}
	return result, nil
}

func (m *mockMemory) Update(ctx context.Context, id string, opts ...UpdateOption) error {
	if _, exists := m.items[id]; !exists {
		return ErrNotFound
	}
	return nil
}

func (m *mockMemory) Remove(ctx context.Context, id string) error {
	if _, exists := m.items[id]; !exists {
		return ErrNotFound
	}
	delete(m.items, id)
	return nil
}

func (m *mockMemory) Has(ctx context.Context, id string) bool {
	_, exists := m.items[id]
	return exists
}

func (m *mockMemory) Clear(ctx context.Context) error {
	m.items = make(map[string]*MemoryItem)
	return nil
}

func (m *mockMemory) GetStats(ctx context.Context) (*MemoryStats, error) {
	return &MemoryStats{
		Count: len(m.items),
	}, nil
}

func TestNewMemoryManager(t *testing.T) {
	config := DefaultMemoryConfig()
	manager := NewMemoryManager(config)

	if manager == nil {
		t.Fatal("expected non-nil manager")
	}
	if manager.config != config {
		t.Error("expected config to be set")
	}
	if manager.memoryTypes == nil {
		t.Error("expected memoryTypes to be initialized")
	}
}

func TestNewMemoryManagerWithNilConfig(t *testing.T) {
	manager := NewMemoryManager(nil)

	if manager == nil {
		t.Fatal("expected non-nil manager")
	}
	if manager.config == nil {
		t.Error("expected default config to be set")
	}
}

func TestNewMemoryManagerWithUserID(t *testing.T) {
	userID := "user123"
	manager := NewMemoryManager(nil, WithManagerUserID(userID))

	if manager.UserID() != userID {
		t.Errorf("expected userID %q, got %q", userID, manager.UserID())
	}
}

func TestRegisterMemory(t *testing.T) {
	manager := NewMemoryManager(nil)
	mock := newMockMemory(MemoryTypeWorking)

	err := manager.RegisterMemory(MemoryTypeWorking, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Try to register again
	err = manager.RegisterMemory(MemoryTypeWorking, mock)
	if err != ErrMemoryTypeExists {
		t.Errorf("expected ErrMemoryTypeExists, got %v", err)
	}
}

func TestUnregisterMemory(t *testing.T) {
	manager := NewMemoryManager(nil)
	mock := newMockMemory(MemoryTypeWorking)

	// Register first
	err := manager.RegisterMemory(MemoryTypeWorking, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Unregister
	err = manager.UnregisterMemory(MemoryTypeWorking)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Try to unregister again
	err = manager.UnregisterMemory(MemoryTypeWorking)
	if err != ErrMemoryTypeNotFound {
		t.Errorf("expected ErrMemoryTypeNotFound, got %v", err)
	}
}

func TestGetMemory(t *testing.T) {
	manager := NewMemoryManager(nil)
	mock := newMockMemory(MemoryTypeWorking)

	err := manager.RegisterMemory(MemoryTypeWorking, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	memory, err := manager.GetMemory(MemoryTypeWorking)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if memory != mock {
		t.Error("expected same mock memory")
	}

	// Try to get non-existent
	_, err = manager.GetMemory(MemoryTypeEpisodic)
	if err != ErrMemoryTypeNotFound {
		t.Errorf("expected ErrMemoryTypeNotFound, got %v", err)
	}
}

func TestAddMemory(t *testing.T) {
	ctx := context.Background()
	manager := NewMemoryManager(nil, WithManagerUserID("user1"))
	mock := newMockMemory(MemoryTypeWorking)

	err := manager.RegisterMemory(MemoryTypeWorking, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	id, err := manager.AddMemory(ctx, "test content", WithAddMemoryType(MemoryTypeWorking))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty ID")
	}

	// Verify item was added
	if len(mock.items) != 1 {
		t.Errorf("expected 1 item, got %d", len(mock.items))
	}
}

func TestAddMemoryWithOptions(t *testing.T) {
	ctx := context.Background()
	manager := NewMemoryManager(nil, WithManagerUserID("user1"))
	mock := newMockMemory(MemoryTypeWorking)

	err := manager.RegisterMemory(MemoryTypeWorking, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	metadata := map[string]interface{}{"key": "value"}
	_, err = manager.AddMemory(ctx, "test content",
		WithAddMemoryType(MemoryTypeWorking),
		WithAddImportance(0.9),
		WithAddMetadata(metadata),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify item properties
	for _, item := range mock.items {
		if item.Importance != 0.9 {
			t.Errorf("expected importance 0.9, got %f", item.Importance)
		}
		if item.Metadata["key"] != "value" {
			t.Errorf("expected metadata key=value, got %v", item.Metadata)
		}
	}
}

func TestAddMemoryNotFound(t *testing.T) {
	ctx := context.Background()
	manager := NewMemoryManager(nil)

	_, err := manager.AddMemory(ctx, "test content", WithAddMemoryType(MemoryTypeWorking))
	if err != ErrMemoryTypeNotFound {
		t.Errorf("expected ErrMemoryTypeNotFound, got %v", err)
	}
}

func TestRetrieveMemories(t *testing.T) {
	ctx := context.Background()
	manager := NewMemoryManager(nil)

	workingMock := newMockMemory(MemoryTypeWorking)
	episodicMock := newMockMemory(MemoryTypeEpisodic)

	_ = manager.RegisterMemory(MemoryTypeWorking, workingMock)
	_ = manager.RegisterMemory(MemoryTypeEpisodic, episodicMock)

	// Add items to both mocks
	workingMock.items["w1"] = NewMemoryItem("working content", MemoryTypeWorking, WithImportance(0.8))
	episodicMock.items["e1"] = NewMemoryItem("episodic content", MemoryTypeEpisodic, WithImportance(0.6))

	// Retrieve from all
	results, err := manager.RetrieveMemories(ctx, "query")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Results should be sorted by importance (0.8 first)
	if results[0].Importance < results[1].Importance {
		t.Error("expected results sorted by importance descending")
	}
}

func TestRetrieveMemoriesWithTypeFilter(t *testing.T) {
	ctx := context.Background()
	manager := NewMemoryManager(nil)

	workingMock := newMockMemory(MemoryTypeWorking)
	episodicMock := newMockMemory(MemoryTypeEpisodic)

	_ = manager.RegisterMemory(MemoryTypeWorking, workingMock)
	_ = manager.RegisterMemory(MemoryTypeEpisodic, episodicMock)

	workingMock.items["w1"] = NewMemoryItem("working content", MemoryTypeWorking)
	episodicMock.items["e1"] = NewMemoryItem("episodic content", MemoryTypeEpisodic)

	// Retrieve only from working memory
	results, err := manager.RetrieveMemories(ctx, "query", WithMemoryTypeFilter(MemoryTypeWorking))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if results[0].MemoryType != MemoryTypeWorking {
		t.Errorf("expected working memory type, got %s", results[0].MemoryType)
	}
}

func TestRetrieveMemoriesWithLimit(t *testing.T) {
	ctx := context.Background()
	manager := NewMemoryManager(nil)

	mock := newMockMemory(MemoryTypeWorking)
	manager.RegisterMemory(MemoryTypeWorking, mock)

	// Add multiple items
	for i := 0; i < 10; i++ {
		item := NewMemoryItem("content", MemoryTypeWorking)
		mock.items[item.ID] = item
	}

	results, err := manager.RetrieveMemories(ctx, "query", WithLimit(5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}
}

func TestUpdateMemory(t *testing.T) {
	ctx := context.Background()
	manager := NewMemoryManager(nil)

	mock := newMockMemory(MemoryTypeWorking)
	manager.RegisterMemory(MemoryTypeWorking, mock)

	item := NewMemoryItem("content", MemoryTypeWorking)
	mock.items[item.ID] = item

	err := manager.UpdateMemory(ctx, MemoryTypeWorking, item.ID, WithContentUpdate("new content"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateMemoryNotFound(t *testing.T) {
	ctx := context.Background()
	manager := NewMemoryManager(nil)

	mock := newMockMemory(MemoryTypeWorking)
	manager.RegisterMemory(MemoryTypeWorking, mock)

	err := manager.UpdateMemory(ctx, MemoryTypeWorking, "nonexistent", WithContentUpdate("new content"))
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRemoveMemory(t *testing.T) {
	ctx := context.Background()
	manager := NewMemoryManager(nil)

	mock := newMockMemory(MemoryTypeWorking)
	manager.RegisterMemory(MemoryTypeWorking, mock)

	item := NewMemoryItem("content", MemoryTypeWorking)
	mock.items[item.ID] = item

	err := manager.RemoveMemory(ctx, MemoryTypeWorking, item.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.items) != 0 {
		t.Error("expected item to be removed")
	}
}

func TestGetStats(t *testing.T) {
	ctx := context.Background()
	manager := NewMemoryManager(nil)

	workingMock := newMockMemory(MemoryTypeWorking)
	episodicMock := newMockMemory(MemoryTypeEpisodic)

	manager.RegisterMemory(MemoryTypeWorking, workingMock)
	manager.RegisterMemory(MemoryTypeEpisodic, episodicMock)

	// Add items
	workingMock.items["w1"] = NewMemoryItem("content", MemoryTypeWorking)
	workingMock.items["w2"] = NewMemoryItem("content", MemoryTypeWorking)
	episodicMock.items["e1"] = NewMemoryItem("content", MemoryTypeEpisodic)

	stats, err := manager.GetStats(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.TotalCount != 3 {
		t.Errorf("expected total count 3, got %d", stats.TotalCount)
	}
	if stats.ByType[MemoryTypeWorking].Count != 2 {
		t.Errorf("expected working count 2, got %d", stats.ByType[MemoryTypeWorking].Count)
	}
	if stats.ByType[MemoryTypeEpisodic].Count != 1 {
		t.Errorf("expected episodic count 1, got %d", stats.ByType[MemoryTypeEpisodic].Count)
	}
}

func TestConfig(t *testing.T) {
	config := DefaultMemoryConfig()
	manager := NewMemoryManager(config)

	if manager.Config() != config {
		t.Error("expected same config")
	}
}

func TestClassifyMemoryType(t *testing.T) {
	manager := NewMemoryManager(nil)

	tests := []struct {
		name     string
		content  string
		expected MemoryType
	}{
		{"default", "any content", MemoryTypeWorking},
		{"episodic_event", "Something happened yesterday", MemoryTypeEpisodic},
		{"episodic_meeting", "We had a meeting today", MemoryTypeEpisodic},
		{"semantic_definition", "The definition of AI is...", MemoryTypeSemantic},
		{"semantic_concept", "This concept refers to...", MemoryTypeSemantic},
		{"chinese_event", "昨天发生了一件事", MemoryTypeEpisodic},
		{"chinese_concept", "这个概念是指...", MemoryTypeSemantic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memType := manager.classifyMemoryType(tt.content)
			if memType != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, memType)
			}
		})
	}
}

func TestCalculateImportance(t *testing.T) {
	manager := NewMemoryManager(nil)

	tests := []struct {
		name     string
		content  string
		metadata map[string]interface{}
		minExp   float32
		maxExp   float32
	}{
		{"short", "hi", nil, 0.3, 0.5},                          // < 20 chars, -0.1
		{"medium", string(make([]byte, 100)), nil, 0.4, 0.6},    // normal
		{"long", string(make([]byte, 300)), nil, 0.5, 0.7},      // > 200, +0.1
		{"very_long", string(make([]byte, 600)), nil, 0.6, 0.8}, // > 500, +0.2
		{"important", "This is very important message", nil, 0.5, 0.8},
		{"maybe", "Maybe this is trivial stuff", nil, 0.3, 0.5},
		{"metadata_high", "content", map[string]interface{}{"priority": "high"}, 0.6, 0.8},
		{"metadata_low", "content", map[string]interface{}{"priority": "low"}, 0.2, 0.4},
		{"explicit", "content", map[string]interface{}{"importance": float32(0.9)}, 0.9, 0.9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			importance := manager.calculateImportance(tt.content, tt.metadata)
			if importance < tt.minExp || importance > tt.maxExp {
				t.Errorf("expected importance between %f and %f, got %f", tt.minExp, tt.maxExp, importance)
			}
		})
	}
}

func TestForgetStrategies(t *testing.T) {
	if ForgetByImportance != "importance_based" {
		t.Errorf("expected ForgetByImportance = 'importance_based', got %q", ForgetByImportance)
	}
	if ForgetByTime != "time_based" {
		t.Errorf("expected ForgetByTime = 'time_based', got %q", ForgetByTime)
	}
	if ForgetByCapacity != "capacity_based" {
		t.Errorf("expected ForgetByCapacity = 'capacity_based', got %q", ForgetByCapacity)
	}
}

func TestForgetOptions(t *testing.T) {
	opts := &forgetOptions{}

	WithThreshold(0.5)(opts)
	if opts.threshold != 0.5 {
		t.Errorf("expected threshold 0.5, got %f", opts.threshold)
	}

	WithMaxAgeDays(30)(opts)
	if opts.maxAgeDays != 30 {
		t.Errorf("expected maxAgeDays 30, got %d", opts.maxAgeDays)
	}

	WithTargetCapacity(100)(opts)
	if opts.targetCapacity != 100 {
		t.Errorf("expected targetCapacity 100, got %d", opts.targetCapacity)
	}
}

func TestRetrieveOptions(t *testing.T) {
	opts := &retrieveOptions{}

	WithLimit(10)(opts)
	if opts.limit != 10 {
		t.Errorf("expected limit 10, got %d", opts.limit)
	}

	WithMinScore(0.5)(opts)
	if opts.minScore != 0.5 {
		t.Errorf("expected minScore 0.5, got %f", opts.minScore)
	}

	WithMemoryTypeFilter(MemoryTypeWorking)(opts)
	if opts.memoryType != MemoryTypeWorking {
		t.Errorf("expected memoryType working, got %s", opts.memoryType)
	}

	WithUserIDFilter("user1")(opts)
	if opts.userID != "user1" {
		t.Errorf("expected userID user1, got %s", opts.userID)
	}
}

func TestUpdateOptions(t *testing.T) {
	opts := &updateOptions{}

	content := "new content"
	WithContentUpdate(content)(opts)
	if *opts.content != content {
		t.Errorf("expected content %q, got %q", content, *opts.content)
	}

	importance := float32(0.9)
	WithImportanceUpdate(importance)(opts)
	if *opts.importance != importance {
		t.Errorf("expected importance %f, got %f", importance, *opts.importance)
	}

	metadata := map[string]interface{}{"key": "value"}
	WithMetadataUpdate(metadata)(opts)
	if opts.metadata["key"] != "value" {
		t.Errorf("expected metadata key=value, got %v", opts.metadata)
	}
}
