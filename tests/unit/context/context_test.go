package context_test

import (
	"context"
	"testing"
	"time"

	agentctx "github.com/easyops/helloagents-go/pkg/context"
	"github.com/easyops/helloagents-go/pkg/core/message"
)

func TestEstimatedCounter_Count(t *testing.T) {
	counter := agentctx.NewEstimatedCounter()

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "empty string",
			text:     "",
			expected: 0,
		},
		{
			name:     "short text",
			text:     "hello",
			expected: 1, // 5 chars / 4 = 1
		},
		{
			name:     "longer text",
			text:     "hello world, this is a test",
			expected: 6, // 27 chars / 4 = 6
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := counter.Count(tt.text)
			if result != tt.expected {
				t.Errorf("Count(%q) = %d, want %d", tt.text, result, tt.expected)
			}
		})
	}
}

func TestEstimatedCounter_CountMessages(t *testing.T) {
	counter := agentctx.NewEstimatedCounter()

	messages := []message.Message{
		{Role: message.RoleUser, Content: "Hello"},
		{Role: message.RoleAssistant, Content: "Hi there"},
	}

	result := counter.CountMessages(messages)
	// Should include message overhead
	if result <= 0 {
		t.Errorf("CountMessages should return positive count, got %d", result)
	}
}

func TestNewPacket(t *testing.T) {
	content := "test content"
	packet := agentctx.NewPacket(content)

	if packet.Content != content {
		t.Errorf("Content = %q, want %q", packet.Content, content)
	}

	if packet.Type != agentctx.PacketTypeCustom {
		t.Errorf("Type = %v, want %v", packet.Type, agentctx.PacketTypeCustom)
	}

	if packet.TokenCount == 0 {
		t.Error("TokenCount should be auto-calculated")
	}

	if packet.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestNewPacket_WithOptions(t *testing.T) {
	content := "test content"
	ts := time.Now().Add(-1 * time.Hour)

	packet := agentctx.NewPacket(content,
		agentctx.WithPacketType(agentctx.PacketTypeEvidence),
		agentctx.WithTimestamp(ts),
		agentctx.WithRelevanceScore(0.8),
		agentctx.WithSource("test"),
	)

	if packet.Type != agentctx.PacketTypeEvidence {
		t.Errorf("Type = %v, want %v", packet.Type, agentctx.PacketTypeEvidence)
	}

	if !packet.Timestamp.Equal(ts) {
		t.Errorf("Timestamp = %v, want %v", packet.Timestamp, ts)
	}

	if packet.RelevanceScore != 0.8 {
		t.Errorf("RelevanceScore = %f, want 0.8", packet.RelevanceScore)
	}

	if packet.Source != "test" {
		t.Errorf("Source = %q, want %q", packet.Source, "test")
	}
}

func TestPacketType_Priority(t *testing.T) {
	tests := []struct {
		packetType agentctx.PacketType
		expected   int
	}{
		{agentctx.PacketTypeInstructions, 0},
		{agentctx.PacketTypeTask, 1},
		{agentctx.PacketTypeTaskState, 1},
		{agentctx.PacketTypeEvidence, 2},
		{agentctx.PacketTypeHistory, 3},
		{agentctx.PacketTypeCustom, 4},
	}

	for _, tt := range tests {
		t.Run(string(tt.packetType), func(t *testing.T) {
			result := tt.packetType.Priority()
			if result != tt.expected {
				t.Errorf("Priority() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := agentctx.DefaultConfig()

	if config.MaxTokens != 8000 {
		t.Errorf("MaxTokens = %d, want 8000", config.MaxTokens)
	}

	if config.ReserveRatio != 0.15 {
		t.Errorf("ReserveRatio = %f, want 0.15", config.ReserveRatio)
	}

	if config.MinRelevance != 0.3 {
		t.Errorf("MinRelevance = %f, want 0.3", config.MinRelevance)
	}
}

func TestConfig_GetAvailableTokens(t *testing.T) {
	config := agentctx.NewConfig(
		agentctx.WithMaxTokens(10000),
		agentctx.WithReserveRatio(0.2),
	)

	expected := 8000 // 10000 * 0.8
	result := config.GetAvailableTokens()

	if result != expected {
		t.Errorf("GetAvailableTokens() = %d, want %d", result, expected)
	}
}

func TestRelevanceScorer_Score(t *testing.T) {
	scorer := agentctx.NewRelevanceScorer()

	tests := []struct {
		name     string
		content  string
		query    string
		minScore float64
		maxScore float64
	}{
		{
			name:     "no overlap",
			content:  "hello world",
			query:    "foo bar",
			minScore: 0.0,
			maxScore: 0.1,
		},
		{
			name:     "full overlap",
			content:  "hello world",
			query:    "hello world",
			minScore: 0.9,
			maxScore: 1.1,
		},
		{
			name:     "partial overlap",
			content:  "hello world today",
			query:    "hello there",
			minScore: 0.4,
			maxScore: 0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet := agentctx.NewPacket(tt.content)
			result := scorer.Score(packet, tt.query)
			if result < tt.minScore || result > tt.maxScore {
				t.Errorf("Score() = %f, want between %f and %f", result, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestRecencyScorer_Score(t *testing.T) {
	scorer := agentctx.NewRecencyScorer(3600) // 1 hour tau

	// Recent packet should have high score
	recentPacket := agentctx.NewPacket("test", agentctx.WithTimestamp(time.Now()))
	recentScore := scorer.Score(recentPacket, "")
	if recentScore < 0.9 {
		t.Errorf("Recent packet score = %f, want > 0.9", recentScore)
	}

	// Old packet should have low score
	oldPacket := agentctx.NewPacket("test", agentctx.WithTimestamp(time.Now().Add(-24*time.Hour)))
	oldScore := scorer.Score(oldPacket, "")
	if oldScore > 0.1 {
		t.Errorf("Old packet score = %f, want < 0.1", oldScore)
	}
}

func TestGSSCBuilder_Build(t *testing.T) {
	builder := agentctx.NewGSSCBuilder()

	input := &agentctx.BuildInput{
		Query:              "What is Go?",
		SystemInstructions: "You are a helpful assistant.",
		History: []message.Message{
			{Role: message.RoleUser, Content: "Hello"},
			{Role: message.RoleAssistant, Content: "Hi there"},
		},
	}

	ctx := context.Background()
	result, err := builder.Build(ctx, input)

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if result == "" {
		t.Error("Build() returned empty string")
	}

	// Check that sections are present
	if !containsSubstring(result, "[Role & Policies]") {
		t.Error("Result should contain [Role & Policies] section")
	}

	if !containsSubstring(result, "[Task]") {
		t.Error("Result should contain [Task] section")
	}

	if !containsSubstring(result, "[Output]") {
		t.Error("Result should contain [Output] section")
	}
}

func TestGSSCBuilder_BuildMessages(t *testing.T) {
	builder := agentctx.NewGSSCBuilder()

	input := &agentctx.BuildInput{
		Query:              "What is Go?",
		SystemInstructions: "You are a helpful assistant.",
	}

	ctx := context.Background()
	messages, err := builder.BuildMessages(ctx, input)

	if err != nil {
		t.Fatalf("BuildMessages() error = %v", err)
	}

	if len(messages) == 0 {
		t.Error("BuildMessages() returned empty slice")
	}

	// First message should be system
	if messages[0].Role != message.RoleSystem {
		t.Errorf("First message role = %v, want %v", messages[0].Role, message.RoleSystem)
	}

	// Last message should be user
	if messages[len(messages)-1].Role != message.RoleUser {
		t.Errorf("Last message role = %v, want %v", messages[len(messages)-1].Role, message.RoleUser)
	}
}

func TestDefaultStructurer_Structure(t *testing.T) {
	structurer := agentctx.NewDefaultStructurer()
	config := agentctx.DefaultConfig()

	packets := []*agentctx.Packet{
		agentctx.NewInstructionsPacket("Be helpful"),
		agentctx.NewTaskPacket("What is Go?"),
		agentctx.NewHistoryPacket("Previous conversation", time.Now().Add(-5*time.Minute)),
	}

	result := structurer.Structure(packets, "What is Go?", config)

	if result == "" {
		t.Error("Structure() returned empty string")
	}

	if !containsSubstring(result, "[Role & Policies]") {
		t.Error("Result should contain [Role & Policies] section")
	}

	if !containsSubstring(result, "Be helpful") {
		t.Error("Result should contain instructions content")
	}
}

func TestTruncateCompressor_Compress(t *testing.T) {
	compressor := agentctx.NewTruncateCompressor()

	// Test with content within budget
	config := agentctx.NewConfig(agentctx.WithMaxTokens(10000))
	shortContent := "Short content"
	result := compressor.Compress(shortContent, config)
	if result != shortContent {
		t.Error("Should not compress content within budget")
	}

	// Test with content exceeding budget (use very small budget)
	smallConfig := agentctx.NewConfig(agentctx.WithMaxTokens(20)) // Very small budget
	longContent := "[Role & Policies]\nVery long content that exceeds the budget and should be truncated to fit within the available tokens. This is additional text to make sure we exceed the budget significantly."
	result = compressor.Compress(longContent, smallConfig)

	// The result should be shorter than the original
	if result == longContent {
		t.Error("Should compress content exceeding budget")
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
