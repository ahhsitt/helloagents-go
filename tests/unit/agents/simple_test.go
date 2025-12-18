package agents_test

import (
	"context"
	"testing"
	"time"

	"github.com/easyops/helloagents-go/pkg/agents"
	"github.com/easyops/helloagents-go/pkg/core/llm"
	"github.com/easyops/helloagents-go/pkg/core/message"
)

// mockProvider implements llm.Provider for testing
type mockProvider struct {
	name        string
	model       string
	generateFn  func(ctx context.Context, req llm.Request) (llm.Response, error)
	streamFn    func(ctx context.Context, req llm.Request) (<-chan llm.StreamChunk, <-chan error)
	embedFn     func(ctx context.Context, texts []string) ([][]float32, error)
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Model() string { return m.model }
func (m *mockProvider) Close() error { return nil }

func (m *mockProvider) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	if m.generateFn != nil {
		return m.generateFn(ctx, req)
	}
	return llm.Response{
		Content:      "Hello! I'm a helpful assistant.",
		FinishReason: "stop",
		TokenUsage: message.TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 8,
			TotalTokens:      18,
		},
	}, nil
}

func (m *mockProvider) GenerateStream(ctx context.Context, req llm.Request) (<-chan llm.StreamChunk, <-chan error) {
	if m.streamFn != nil {
		return m.streamFn(ctx, req)
	}
	chunkCh := make(chan llm.StreamChunk, 3)
	errCh := make(chan error, 1)
	go func() {
		defer close(chunkCh)
		defer close(errCh)
		chunkCh <- llm.StreamChunk{Content: "Hello"}
		chunkCh <- llm.StreamChunk{Content: " World"}
		chunkCh <- llm.StreamChunk{Content: "!", Done: true, FinishReason: "stop"}
	}()
	return chunkCh, errCh
}

func (m *mockProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if m.embedFn != nil {
		return m.embedFn(ctx, texts)
	}
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = make([]float32, 128)
	}
	return result, nil
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		name:  "mock",
		model: "mock-model",
	}
}

func TestNewSimple_ValidProvider(t *testing.T) {
	provider := newMockProvider()
	agent, err := agents.NewSimple(provider)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if agent == nil {
		t.Fatal("expected agent to be non-nil")
	}
}

func TestNewSimple_NilProvider(t *testing.T) {
	_, err := agents.NewSimple(nil)
	if err == nil {
		t.Fatal("expected error for nil provider")
	}
}

func TestNewSimple_WithName(t *testing.T) {
	provider := newMockProvider()
	agent, err := agents.NewSimple(provider, agents.WithName("TestAgent"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if agent.Name() != "TestAgent" {
		t.Fatalf("expected name 'TestAgent', got %s", agent.Name())
	}
}

func TestNewSimple_WithSystemPrompt(t *testing.T) {
	provider := newMockProvider()
	agent, err := agents.NewSimple(provider, agents.WithSystemPrompt("You are helpful."))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	config := agent.Config()
	if config.SystemPrompt != "You are helpful." {
		t.Fatalf("expected system prompt 'You are helpful.', got %s", config.SystemPrompt)
	}
}

func TestSimpleAgent_Run(t *testing.T) {
	provider := newMockProvider()
	agent, _ := agents.NewSimple(provider)

	ctx := context.Background()
	output, err := agent.Run(ctx, agents.Input{Query: "Hello"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if output.Response == "" {
		t.Fatal("expected non-empty response")
	}
}

func TestSimpleAgent_RunWithContext(t *testing.T) {
	provider := &mockProvider{
		name:  "mock",
		model: "mock-model",
		generateFn: func(ctx context.Context, req llm.Request) (llm.Response, error) {
			// Simulate slow response
			select {
			case <-ctx.Done():
				return llm.Response{}, ctx.Err()
			case <-time.After(10 * time.Millisecond):
				return llm.Response{Content: "Done", FinishReason: "stop"}, nil
			}
		},
	}

	agent, _ := agents.NewSimple(provider)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	output, err := agent.Run(ctx, agents.Input{Query: "Hello"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if output.Response != "Done" {
		t.Fatalf("expected 'Done', got %s", output.Response)
	}
}

func TestSimpleAgent_TokenUsage(t *testing.T) {
	provider := newMockProvider()
	agent, _ := agents.NewSimple(provider)

	ctx := context.Background()
	output, err := agent.Run(ctx, agents.Input{Query: "Hello"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if output.TokenUsage.TotalTokens == 0 {
		t.Fatal("expected non-zero token usage")
	}
}

func TestSimpleAgent_Duration(t *testing.T) {
	provider := newMockProvider()
	agent, _ := agents.NewSimple(provider)

	ctx := context.Background()
	output, err := agent.Run(ctx, agents.Input{Query: "Hello"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if output.Duration <= 0 {
		t.Fatal("expected positive duration")
	}
}

func TestSimpleAgent_GetHistory(t *testing.T) {
	provider := newMockProvider()
	agent, _ := agents.NewSimple(provider)

	// Initially empty
	history := agent.GetHistory()
	if len(history) != 0 {
		t.Fatalf("expected empty history, got %d messages", len(history))
	}

	// After one run
	ctx := context.Background()
	_, _ = agent.Run(ctx, agents.Input{Query: "Hello"})

	history = agent.GetHistory()
	if len(history) != 2 { // user + assistant
		t.Fatalf("expected 2 messages in history, got %d", len(history))
	}
}

func TestSimpleAgent_ClearHistory(t *testing.T) {
	provider := newMockProvider()
	agent, _ := agents.NewSimple(provider)

	ctx := context.Background()
	_, _ = agent.Run(ctx, agents.Input{Query: "Hello"})

	// Clear history
	agent.ClearHistory()

	history := agent.GetHistory()
	if len(history) != 0 {
		t.Fatalf("expected empty history after clear, got %d messages", len(history))
	}
}

func TestSimpleAgent_SetSystemPrompt(t *testing.T) {
	provider := newMockProvider()
	agent, _ := agents.NewSimple(provider, agents.WithSystemPrompt("Initial prompt"))

	agent.SetSystemPrompt("Updated prompt")

	config := agent.Config()
	if config.SystemPrompt != "Updated prompt" {
		t.Fatalf("expected 'Updated prompt', got %s", config.SystemPrompt)
	}
}

func TestSimpleAgent_MultiTurnConversation(t *testing.T) {
	responses := []string{"Hello!", "I'm fine, thanks!", "Goodbye!"}
	responseIdx := 0

	provider := &mockProvider{
		name:  "mock",
		model: "mock-model",
		generateFn: func(ctx context.Context, req llm.Request) (llm.Response, error) {
			resp := llm.Response{
				Content:      responses[responseIdx],
				FinishReason: "stop",
			}
			responseIdx++
			return resp, nil
		},
	}

	agent, _ := agents.NewSimple(provider)
	ctx := context.Background()

	// Turn 1
	out1, _ := agent.Run(ctx, agents.Input{Query: "Hi"})
	if out1.Response != "Hello!" {
		t.Fatalf("expected 'Hello!', got %s", out1.Response)
	}

	// Turn 2
	out2, _ := agent.Run(ctx, agents.Input{Query: "How are you?"})
	if out2.Response != "I'm fine, thanks!" {
		t.Fatalf("expected 'I'm fine, thanks!', got %s", out2.Response)
	}

	// Check history
	history := agent.GetHistory()
	if len(history) != 4 { // 2 turns * 2 messages
		t.Fatalf("expected 4 messages in history, got %d", len(history))
	}
}

func TestSimpleAgent_RunStream(t *testing.T) {
	provider := newMockProvider()
	agent, _ := agents.NewSimple(provider)

	ctx := context.Background()
	chunkCh, errCh := agent.RunStream(ctx, agents.Input{Query: "Hello"})

	var content string
	var done bool

	for chunk := range chunkCh {
		content += chunk.Content
		if chunk.Done {
			done = true
		}
	}

	// Check for errors
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	default:
	}

	if content != "Hello World!" {
		t.Fatalf("expected 'Hello World!', got %s", content)
	}
	if !done {
		t.Fatal("expected done to be true")
	}
}

func TestSimpleAgent_ImplementsAgent(t *testing.T) {
	provider := newMockProvider()
	agent, _ := agents.NewSimple(provider)

	// Verify agent implements Agent interface
	var _ agents.Agent = agent
}

func TestSimpleAgent_WithOptions(t *testing.T) {
	provider := newMockProvider()
	agent, err := agents.NewSimple(provider,
		agents.WithName("CustomAgent"),
		agents.WithSystemPrompt("Custom prompt"),
		agents.WithMaxIterations(5),
		agents.WithAgentTemperature(0.7),
		agents.WithAgentMaxTokens(1000),
		agents.WithAgentTimeout(30*time.Second),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if agent.Name() != "CustomAgent" {
		t.Fatalf("expected name 'CustomAgent', got %s", agent.Name())
	}

	config := agent.Config()
	if config.SystemPrompt != "Custom prompt" {
		t.Fatalf("expected system prompt 'Custom prompt'")
	}
}
