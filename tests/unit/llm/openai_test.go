package llm_test

import (
	"testing"

	"github.com/easyops/helloagents-go/pkg/core/errors"
	"github.com/easyops/helloagents-go/pkg/core/llm"
	"github.com/easyops/helloagents-go/pkg/core/message"
)

func TestNewOpenAI_ValidAPIKey(t *testing.T) {
	client, err := llm.NewOpenAI(llm.WithAPIKey("test-api-key"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client == nil {
		t.Fatal("expected client to be non-nil")
	}
}

func TestNewOpenAI_EmptyAPIKey(t *testing.T) {
	_, err := llm.NewOpenAI()
	if err != errors.ErrInvalidAPIKey {
		t.Fatalf("expected ErrInvalidAPIKey, got %v", err)
	}
}

func TestNewOpenAI_DefaultModel(t *testing.T) {
	client, err := llm.NewOpenAI(llm.WithAPIKey("test-api-key"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client.Model() != "gpt-4o" {
		t.Fatalf("expected default model 'gpt-4o', got %s", client.Model())
	}
}

func TestNewOpenAI_CustomModel(t *testing.T) {
	client, err := llm.NewOpenAI(
		llm.WithAPIKey("test-api-key"),
		llm.WithModel("gpt-4o-mini"),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client.Model() != "gpt-4o-mini" {
		t.Fatalf("expected model 'gpt-4o-mini', got %s", client.Model())
	}
}

func TestOpenAIClient_Name(t *testing.T) {
	client, _ := llm.NewOpenAI(llm.WithAPIKey("test-api-key"))
	if client.Name() != "openai" {
		t.Fatalf("expected name 'openai', got %s", client.Name())
	}
}

func TestOpenAIClient_Close(t *testing.T) {
	client, _ := llm.NewOpenAI(llm.WithAPIKey("test-api-key"))
	err := client.Close()
	if err != nil {
		t.Fatalf("expected no error on close, got %v", err)
	}
}

func TestNewOpenAI_CustomBaseURL(t *testing.T) {
	client, err := llm.NewOpenAI(
		llm.WithAPIKey("test-api-key"),
		llm.WithBaseURL("https://custom-api.example.com/v1"),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client == nil {
		t.Fatal("expected client to be non-nil")
	}
}

func TestNewOpenAI_WithAllOptions(t *testing.T) {
	client, err := llm.NewOpenAI(
		llm.WithAPIKey("test-api-key"),
		llm.WithModel("gpt-4"),
		llm.WithBaseURL("https://custom.example.com"),
		llm.WithTemperature(0.8),
		llm.WithMaxTokens(2000),
		llm.WithMaxRetries(5),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client.Model() != "gpt-4" {
		t.Fatalf("expected model 'gpt-4', got %s", client.Model())
	}
}

func TestOpenAIClient_ImplementsProvider(t *testing.T) {
	client, _ := llm.NewOpenAI(llm.WithAPIKey("test-api-key"))

	// Verify client implements Provider interface
	var _ llm.Provider = client
}

func TestBuildRequest_WithMessages(t *testing.T) {
	client, _ := llm.NewOpenAI(llm.WithAPIKey("test-api-key"))

	// Test that client can be created with valid messages
	msgs := []message.Message{
		{Role: message.RoleSystem, Content: "You are a helpful assistant."},
		{Role: message.RoleUser, Content: "Hello!"},
	}

	// Verify messages are valid
	for _, msg := range msgs {
		if !msg.Role.IsValid() {
			t.Fatalf("invalid role: %s", msg.Role)
		}
	}

	// Verify client is ready
	if client.Name() != "openai" {
		t.Fatalf("expected name 'openai', got %s", client.Name())
	}
}

func TestMessageRoles(t *testing.T) {
	tests := []struct {
		role    message.Role
		valid   bool
	}{
		{message.RoleSystem, true},
		{message.RoleUser, true},
		{message.RoleAssistant, true},
		{message.RoleTool, true},
		{message.Role("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if tt.role.IsValid() != tt.valid {
				t.Errorf("expected IsValid() = %v for role %s", tt.valid, tt.role)
			}
		})
	}
}

// Note: Integration tests that require actual API calls should be placed
// in tests/integration/ and use environment variables for API keys
