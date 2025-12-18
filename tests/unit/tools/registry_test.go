package tools_test

import (
	"context"
	"sync"
	"testing"

	"github.com/easyops/helloagents-go/pkg/core/errors"
	"github.com/easyops/helloagents-go/pkg/tools"
)

// mockTool implements tools.Tool for testing
type mockTool struct {
	name        string
	description string
	params      tools.ParameterSchema
}

func (m *mockTool) Name() string                   { return m.name }
func (m *mockTool) Description() string            { return m.description }
func (m *mockTool) Parameters() tools.ParameterSchema { return m.params }
func (m *mockTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	return "mock result", nil
}

func newMockTool(name string) *mockTool {
	return &mockTool{
		name:        name,
		description: "Mock tool for testing",
		params: tools.ParameterSchema{
			Type:       "object",
			Properties: map[string]tools.PropertySchema{},
		},
	}
}

func TestNewRegistry(t *testing.T) {
	registry := tools.NewRegistry()
	if registry == nil {
		t.Fatal("expected non-nil registry")
	}
	if registry.Count() != 0 {
		t.Fatalf("expected count 0, got %d", registry.Count())
	}
}

func TestRegistry_Register(t *testing.T) {
	registry := tools.NewRegistry()
	tool := newMockTool("test-tool")

	err := registry.Register(tool)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if registry.Count() != 1 {
		t.Fatalf("expected count 1, got %d", registry.Count())
	}
}

func TestRegistry_RegisterNil(t *testing.T) {
	registry := tools.NewRegistry()

	err := registry.Register(nil)
	if err != errors.ErrInvalidTool {
		t.Fatalf("expected ErrInvalidTool, got %v", err)
	}
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	registry := tools.NewRegistry()
	tool1 := newMockTool("test-tool")
	tool2 := newMockTool("test-tool")

	_ = registry.Register(tool1)
	err := registry.Register(tool2)

	if err != errors.ErrToolAlreadyRegistered {
		t.Fatalf("expected ErrToolAlreadyRegistered, got %v", err)
	}
}

func TestRegistry_MustRegister(t *testing.T) {
	registry := tools.NewRegistry()
	tool := newMockTool("test-tool")

	// Should not panic
	registry.MustRegister(tool)

	if registry.Count() != 1 {
		t.Fatalf("expected count 1, got %d", registry.Count())
	}
}

func TestRegistry_MustRegisterPanics(t *testing.T) {
	registry := tools.NewRegistry()
	tool := newMockTool("test-tool")
	_ = registry.Register(tool)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for duplicate registration")
		}
	}()

	// Should panic on duplicate
	registry.MustRegister(tool)
}

func TestRegistry_RegisterAll(t *testing.T) {
	registry := tools.NewRegistry()
	tools := []tools.Tool{
		newMockTool("tool-1"),
		newMockTool("tool-2"),
		newMockTool("tool-3"),
	}

	err := registry.RegisterAll(tools...)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if registry.Count() != 3 {
		t.Fatalf("expected count 3, got %d", registry.Count())
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := tools.NewRegistry()
	tool := newMockTool("test-tool")
	_ = registry.Register(tool)

	retrieved, err := registry.Get("test-tool")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if retrieved.Name() != "test-tool" {
		t.Fatalf("expected name 'test-tool', got %s", retrieved.Name())
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	registry := tools.NewRegistry()

	_, err := registry.Get("non-existent")
	if err != errors.ErrToolNotFound {
		t.Fatalf("expected ErrToolNotFound, got %v", err)
	}
}

func TestRegistry_Has(t *testing.T) {
	registry := tools.NewRegistry()
	tool := newMockTool("test-tool")
	_ = registry.Register(tool)

	if !registry.Has("test-tool") {
		t.Fatal("expected Has to return true for registered tool")
	}
	if registry.Has("non-existent") {
		t.Fatal("expected Has to return false for non-existent tool")
	}
}

func TestRegistry_Unregister(t *testing.T) {
	registry := tools.NewRegistry()
	tool := newMockTool("test-tool")
	_ = registry.Register(tool)

	err := registry.Unregister("test-tool")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if registry.Count() != 0 {
		t.Fatalf("expected count 0, got %d", registry.Count())
	}
}

func TestRegistry_UnregisterNotFound(t *testing.T) {
	registry := tools.NewRegistry()

	err := registry.Unregister("non-existent")
	if err != errors.ErrToolNotFound {
		t.Fatalf("expected ErrToolNotFound, got %v", err)
	}
}

func TestRegistry_List(t *testing.T) {
	registry := tools.NewRegistry()
	_ = registry.Register(newMockTool("tool-a"))
	_ = registry.Register(newMockTool("tool-b"))

	names := registry.List()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
}

func TestRegistry_All(t *testing.T) {
	registry := tools.NewRegistry()
	_ = registry.Register(newMockTool("tool-a"))
	_ = registry.Register(newMockTool("tool-b"))

	allTools := registry.All()
	if len(allTools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(allTools))
	}
}

func TestRegistry_Clear(t *testing.T) {
	registry := tools.NewRegistry()
	_ = registry.Register(newMockTool("tool-a"))
	_ = registry.Register(newMockTool("tool-b"))

	registry.Clear()

	if registry.Count() != 0 {
		t.Fatalf("expected count 0 after clear, got %d", registry.Count())
	}
}

func TestRegistry_ToDefinitions(t *testing.T) {
	registry := tools.NewRegistry()
	_ = registry.Register(newMockTool("tool-a"))
	_ = registry.Register(newMockTool("tool-b"))

	defs := registry.ToDefinitions()
	if len(defs) != 2 {
		t.Fatalf("expected 2 definitions, got %d", len(defs))
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	registry := tools.NewRegistry()
	var wg sync.WaitGroup

	// Concurrent registrations
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tool := newMockTool("tool-" + string(rune('a'+idx%26)) + "-" + string(rune('0'+idx%10)))
			_ = registry.Register(tool)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = registry.List()
			_ = registry.Count()
		}()
	}

	wg.Wait()
}

func TestDefaultRegistry(t *testing.T) {
	// Verify DefaultRegistry exists and is usable
	if tools.DefaultRegistry == nil {
		t.Fatal("expected DefaultRegistry to be non-nil")
	}
}
