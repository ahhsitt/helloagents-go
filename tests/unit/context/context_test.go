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

// mockNoteRetriever 实现 NoteRetriever 接口用于测试
type mockNoteRetriever struct {
	notes []agentctx.NoteResult
}

func (m *mockNoteRetriever) ListNotes(noteType string, limit int) ([]agentctx.NoteResult, error) {
	var results []agentctx.NoteResult
	for _, note := range m.notes {
		if noteType == "" || note.Type == noteType {
			results = append(results, note)
		}
		if limit > 0 && len(results) >= limit {
			break
		}
	}
	return results, nil
}

func (m *mockNoteRetriever) SearchNotes(query string, limit int) ([]agentctx.NoteResult, error) {
	var results []agentctx.NoteResult
	for _, note := range m.notes {
		if containsSubstring(note.Title, query) || containsSubstring(note.Content, query) {
			results = append(results, note)
		}
		if limit > 0 && len(results) >= limit {
			break
		}
	}
	return results, nil
}

func TestNoteGatherer_Gather(t *testing.T) {
	now := time.Now()
	retriever := &mockNoteRetriever{
		notes: []agentctx.NoteResult{
			{ID: "note1", Title: "任务状态", Content: "正在开发功能A", Type: "task_state", UpdatedAt: now},
			{ID: "note2", Title: "结论", Content: "功能A设计完成", Type: "conclusion", UpdatedAt: now},
		},
	}

	gatherer := agentctx.NewNoteGatherer(retriever, 5)
	ctx := context.Background()

	input := &agentctx.GatherInput{
		Query: "功能A",
	}

	packets, err := gatherer.Gather(ctx, input)
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}

	if len(packets) == 0 {
		t.Error("Gather() 应该返回笔记包")
	}

	// 验证包含搜索结果
	found := false
	for _, p := range packets {
		if containsSubstring(p.Content, "功能A") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Gather() 应该返回包含查询关键词的笔记")
	}
}

func TestNoteGatherer_EmptyQuery(t *testing.T) {
	retriever := &mockNoteRetriever{
		notes: []agentctx.NoteResult{
			{ID: "note1", Title: "测试", Content: "内容", Type: "general"},
		},
	}

	gatherer := agentctx.NewNoteGatherer(retriever, 5)
	ctx := context.Background()

	input := &agentctx.GatherInput{
		Query: "",
	}

	packets, err := gatherer.Gather(ctx, input)
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}

	if len(packets) != 0 {
		t.Error("空查询应该返回空的 Packet 列表")
	}
}

func TestNoteGatherer_NilRetriever(t *testing.T) {
	gatherer := agentctx.NewNoteGatherer(nil, 5)
	ctx := context.Background()

	input := &agentctx.GatherInput{
		Query: "test",
	}

	packets, err := gatherer.Gather(ctx, input)
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}

	if len(packets) != 0 {
		t.Error("nil retriever 应该返回空的 Packet 列表")
	}
}

func TestNoteGatherer_PacketTypeMapping(t *testing.T) {
	now := time.Now()
	retriever := &mockNoteRetriever{
		notes: []agentctx.NoteResult{
			{ID: "note1", Title: "任务状态", Content: "test task_state", Type: "task_state", UpdatedAt: now},
			{ID: "note2", Title: "阻塞项", Content: "test blocker", Type: "blocker", UpdatedAt: now},
			{ID: "note3", Title: "行动", Content: "test action", Type: "action", UpdatedAt: now},
			{ID: "note4", Title: "结论", Content: "test conclusion", Type: "conclusion", UpdatedAt: now},
			{ID: "note5", Title: "参考", Content: "test reference", Type: "reference", UpdatedAt: now},
			{ID: "note6", Title: "通用", Content: "test general", Type: "general", UpdatedAt: now},
		},
	}

	gatherer := agentctx.NewNoteGatherer(retriever, 10)
	ctx := context.Background()

	input := &agentctx.GatherInput{
		Query: "test",
	}

	packets, err := gatherer.Gather(ctx, input)
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}

	// 验证类型映射
	taskStateTypes := []string{"task_state", "blocker", "action"}
	evidenceTypes := []string{"conclusion", "reference", "general"}

	for _, p := range packets {
		noteType, ok := p.Metadata["note_type"].(string)
		if !ok {
			t.Error("Packet 应该包含 note_type 元数据")
			continue
		}

		isTaskStateType := false
		for _, tt := range taskStateTypes {
			if noteType == tt {
				isTaskStateType = true
				break
			}
		}

		isEvidenceType := false
		for _, tt := range evidenceTypes {
			if noteType == tt {
				isEvidenceType = true
				break
			}
		}

		if isTaskStateType && p.Type != agentctx.PacketTypeTaskState {
			t.Errorf("笔记类型 %s 应该映射到 PacketTypeTaskState，得到 %v", noteType, p.Type)
		}

		if isEvidenceType && p.Type != agentctx.PacketTypeEvidence {
			t.Errorf("笔记类型 %s 应该映射到 PacketTypeEvidence，得到 %v", noteType, p.Type)
		}
	}
}

func TestNoteGatherer_PriorityRetrieval(t *testing.T) {
	now := time.Now()
	retriever := &mockNoteRetriever{
		notes: []agentctx.NoteResult{
			{ID: "note1", Title: "阻塞项1", Content: "blocker content", Type: "blocker", UpdatedAt: now},
			{ID: "note2", Title: "行动1", Content: "action content", Type: "action", UpdatedAt: now},
			{ID: "note3", Title: "通用笔记", Content: "general content", Type: "general", UpdatedAt: now},
		},
	}

	gatherer := agentctx.NewNoteGatherer(retriever, 10)
	ctx := context.Background()

	input := &agentctx.GatherInput{
		Query: "content",
	}

	packets, err := gatherer.Gather(ctx, input)
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}

	// 验证 blocker 和 action 优先检索（应该在结果前面）
	if len(packets) < 2 {
		t.Fatal("应该返回至少 2 个包")
	}

	// 前两个应该是 blocker 和 action（优先检索）
	firstNoteType, _ := packets[0].Metadata["note_type"].(string)
	secondNoteType, _ := packets[1].Metadata["note_type"].(string)

	priorityTypes := map[string]bool{"blocker": true, "action": true}
	if !priorityTypes[firstNoteType] {
		t.Errorf("第一个包应该是 blocker 或 action 类型，得到 %s", firstNoteType)
	}
	if !priorityTypes[secondNoteType] {
		t.Errorf("第二个包应该是 blocker 或 action 类型，得到 %s", secondNoteType)
	}
}

func TestNoteGatherer_Deduplication(t *testing.T) {
	now := time.Now()
	// 同一笔记既是 blocker 类型，又会被搜索命中
	retriever := &mockNoteRetriever{
		notes: []agentctx.NoteResult{
			{ID: "note1", Title: "紧急阻塞", Content: "搜索关键词", Type: "blocker", UpdatedAt: now},
		},
	}

	gatherer := agentctx.NewNoteGatherer(retriever, 5)
	ctx := context.Background()

	input := &agentctx.GatherInput{
		Query: "搜索关键词",
	}

	packets, err := gatherer.Gather(ctx, input)
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}

	// 应该只返回一个包（去重）
	if len(packets) != 1 {
		t.Errorf("去重后应该只有 1 个包，得到 %d 个", len(packets))
	}
}

func TestNoteGatherer_PacketMetadata(t *testing.T) {
	now := time.Now()
	retriever := &mockNoteRetriever{
		notes: []agentctx.NoteResult{
			{
				ID:        "note_123",
				Title:     "测试笔记",
				Content:   "测试内容",
				Type:      "task_state",
				Tags:      []string{"tag1", "tag2"},
				UpdatedAt: now,
			},
		},
	}

	gatherer := agentctx.NewNoteGatherer(retriever, 5)
	ctx := context.Background()

	input := &agentctx.GatherInput{
		Query: "测试",
	}

	packets, err := gatherer.Gather(ctx, input)
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}

	if len(packets) == 0 {
		t.Fatal("应该返回至少一个包")
	}

	p := packets[0]

	// 验证 Source
	if p.Source != "note" {
		t.Errorf("Source = %q, want %q", p.Source, "note")
	}

	// 验证 RelevanceScore
	if p.RelevanceScore != 0.75 {
		t.Errorf("RelevanceScore = %f, want 0.75", p.RelevanceScore)
	}

	// 验证 Metadata
	if noteID, ok := p.Metadata["note_id"].(string); !ok || noteID != "note_123" {
		t.Errorf("Metadata[note_id] = %v, want %q", p.Metadata["note_id"], "note_123")
	}

	if noteType, ok := p.Metadata["note_type"].(string); !ok || noteType != "task_state" {
		t.Errorf("Metadata[note_type] = %v, want %q", p.Metadata["note_type"], "task_state")
	}

	if tags, ok := p.Metadata["tags"].([]string); !ok || len(tags) != 2 {
		t.Errorf("Metadata[tags] = %v, want [tag1, tag2]", p.Metadata["tags"])
	}

	// 验证 Content 格式
	if !containsSubstring(p.Content, "[笔记:测试笔记]") {
		t.Errorf("Content 应该包含笔记标题，得到: %s", p.Content)
	}
	if !containsSubstring(p.Content, "测试内容") {
		t.Errorf("Content 应该包含笔记内容，得到: %s", p.Content)
	}
}

func TestNoteGatherer_Limit(t *testing.T) {
	now := time.Now()
	notes := make([]agentctx.NoteResult, 10)
	for i := 0; i < 10; i++ {
		notes[i] = agentctx.NoteResult{
			ID:        "note_" + string(rune('0'+i)),
			Title:     "笔记",
			Content:   "搜索内容",
			Type:      "general",
			UpdatedAt: now,
		}
	}

	retriever := &mockNoteRetriever{notes: notes}
	gatherer := agentctx.NewNoteGatherer(retriever, 3) // 限制为 3
	ctx := context.Background()

	input := &agentctx.GatherInput{
		Query: "搜索",
	}

	packets, err := gatherer.Gather(ctx, input)
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}

	if len(packets) > 3 {
		t.Errorf("返回的包数量 = %d, 应该不超过 3", len(packets))
	}
}

func TestNoteGatherer_WithCompositeGatherer(t *testing.T) {
	now := time.Now()
	retriever := &mockNoteRetriever{
		notes: []agentctx.NoteResult{
			{ID: "note1", Title: "笔记", Content: "笔记内容", Type: "task_state", UpdatedAt: now},
		},
	}

	noteGatherer := agentctx.NewNoteGatherer(retriever, 5)
	instructionsGatherer := agentctx.NewInstructionsGatherer()
	taskGatherer := agentctx.NewTaskGatherer()

	composite := agentctx.NewCompositeGatherer([]agentctx.Gatherer{
		instructionsGatherer,
		taskGatherer,
		noteGatherer,
	}, false) // 顺序执行

	ctx := context.Background()
	input := &agentctx.GatherInput{
		Query:              "笔记",
		SystemInstructions: "你是一个助手",
	}

	packets, err := composite.Gather(ctx, input)
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}

	// 验证返回了不同来源的包
	sources := make(map[string]bool)
	for _, p := range packets {
		if p.Source != "" {
			sources[p.Source] = true
		}
	}

	if !sources["note"] {
		t.Error("CompositeGatherer 应该包含来自 note 的包")
	}
}

func TestNoteGatherer_DefaultLimit(t *testing.T) {
	// 测试默认限制（limit <= 0 时使用默认值 5）
	gatherer := agentctx.NewNoteGatherer(&mockNoteRetriever{}, 0)
	if gatherer.Limit != 5 {
		t.Errorf("默认 Limit = %d, want 5", gatherer.Limit)
	}

	gatherer2 := agentctx.NewNoteGatherer(&mockNoteRetriever{}, -1)
	if gatherer2.Limit != 5 {
		t.Errorf("负数 Limit 应该使用默认值 5, 得到 %d", gatherer2.Limit)
	}
}
