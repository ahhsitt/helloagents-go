package context

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/easyops/helloagents-go/pkg/core/message"
)

// Gatherer 定义从来源收集上下文包的接口。
type Gatherer interface {
	// Gather 从来源收集上下文包。
	Gather(ctx context.Context, input *GatherInput) ([]*Packet, error)
}

// GatherInput 包含收集上下文的输入数据。
type GatherInput struct {
	// Query 是当前用户查询。
	Query string

	// SystemInstructions 是系统提示/指令。
	SystemInstructions string

	// History 是对话历史。
	History []message.Message

	// Config 是上下文配置。
	Config *Config
}

// InstructionsGatherer 收集系统指令作为上下文包。
type InstructionsGatherer struct{}

// NewInstructionsGatherer 创建新的 InstructionsGatherer。
func NewInstructionsGatherer() *InstructionsGatherer {
	return &InstructionsGatherer{}
}

// Gather 收集系统指令。
func (g *InstructionsGatherer) Gather(_ context.Context, input *GatherInput) ([]*Packet, error) {
	if input.SystemInstructions == "" {
		return nil, nil
	}

	packet := NewInstructionsPacket(input.SystemInstructions)
	return []*Packet{packet}, nil
}

// TaskGatherer 收集当前任务/查询作为上下文包。
type TaskGatherer struct{}

// NewTaskGatherer 创建新的 TaskGatherer。
func NewTaskGatherer() *TaskGatherer {
	return &TaskGatherer{}
}

// Gather 收集当前任务。
func (g *TaskGatherer) Gather(_ context.Context, input *GatherInput) ([]*Packet, error) {
	if input.Query == "" {
		return nil, nil
	}

	packet := NewTaskPacket(input.Query)
	return []*Packet{packet}, nil
}

// HistoryGatherer 收集对话历史作为上下文包。
type HistoryGatherer struct {
	// MaxMessages 限制要包含的消息数量。
	MaxMessages int
}

// NewHistoryGatherer 创建新的 HistoryGatherer。
func NewHistoryGatherer(maxMessages int) *HistoryGatherer {
	if maxMessages <= 0 {
		maxMessages = 10
	}
	return &HistoryGatherer{
		MaxMessages: maxMessages,
	}
}

// Gather 收集对话历史。
func (g *HistoryGatherer) Gather(_ context.Context, input *GatherInput) ([]*Packet, error) {
	if len(input.History) == 0 {
		return nil, nil
	}

	// 获取最近的消息
	messages := input.History
	if len(messages) > g.MaxMessages {
		messages = messages[len(messages)-g.MaxMessages:]
	}

	// 将历史格式化为单个包
	var content string
	for _, msg := range messages {
		content += fmt.Sprintf("[%s] %s\n", msg.Role, msg.Content)
	}

	if content == "" {
		return nil, nil
	}

	packet := NewPacket(content,
		WithPacketType(PacketTypeHistory),
		WithSource("history"),
		WithMetadata(map[string]interface{}{
			"message_count": len(messages),
		}),
	)

	return []*Packet{packet}, nil
}

// MemoryGatherer 收集相关记忆作为上下文包。
// 这是一个用户可以实现以集成其记忆系统的接口。
type MemoryGatherer struct {
	// RetrieveFunc 是检索相关记忆的函数。
	RetrieveFunc func(ctx context.Context, query string, limit int) ([]MemoryResult, error)

	// Limit 是要检索的最大记忆数量。
	Limit int
}

// MemoryResult 表示记忆检索结果。
type MemoryResult struct {
	Content    string
	Importance float64
	Timestamp  time.Time
	Type       string // 例如 "task_state"、"fact"、"episode"
}

// NewMemoryGatherer 创建新的 MemoryGatherer。
func NewMemoryGatherer(retrieveFunc func(ctx context.Context, query string, limit int) ([]MemoryResult, error), limit int) *MemoryGatherer {
	if limit <= 0 {
		limit = 5
	}
	return &MemoryGatherer{
		RetrieveFunc: retrieveFunc,
		Limit:        limit,
	}
}

// Gather 收集相关记忆。
func (g *MemoryGatherer) Gather(ctx context.Context, input *GatherInput) ([]*Packet, error) {
	if g.RetrieveFunc == nil {
		return nil, nil
	}

	results, err := g.RetrieveFunc(ctx, input.Query, g.Limit)
	if err != nil {
		return nil, err
	}

	packets := make([]*Packet, 0, len(results))
	for _, result := range results {
		packetType := PacketTypeEvidence
		if result.Type == "task_state" {
			packetType = PacketTypeTaskState
		}

		packet := NewPacket(result.Content,
			WithPacketType(packetType),
			WithSource("memory"),
			WithTimestamp(result.Timestamp),
			WithRelevanceScore(result.Importance),
		)
		packets = append(packets, packet)
	}

	return packets, nil
}

// RAGGatherer 从 RAG 收集相关文档作为上下文包。
type RAGGatherer struct {
	// RetrieveFunc 是检索相关文档的函数。
	RetrieveFunc func(ctx context.Context, query string, topK int) ([]RAGResult, error)

	// TopK 是要检索的最大文档数量。
	TopK int
}

// RAGResult 表示 RAG 检索结果。
type RAGResult struct {
	Content string
	Score   float64
	Source  string
}

// NewRAGGatherer 创建新的 RAGGatherer。
func NewRAGGatherer(retrieveFunc func(ctx context.Context, query string, topK int) ([]RAGResult, error), topK int) *RAGGatherer {
	if topK <= 0 {
		topK = 5
	}
	return &RAGGatherer{
		RetrieveFunc: retrieveFunc,
		TopK:         topK,
	}
}

// Gather 从 RAG 收集相关文档。
func (g *RAGGatherer) Gather(ctx context.Context, input *GatherInput) ([]*Packet, error) {
	if g.RetrieveFunc == nil {
		return nil, nil
	}

	results, err := g.RetrieveFunc(ctx, input.Query, g.TopK)
	if err != nil {
		return nil, err
	}

	packets := make([]*Packet, 0, len(results))
	for _, result := range results {
		source := result.Source
		if source == "" {
			source = "rag"
		}

		packet := NewEvidencePacket(result.Content, source, result.Score)
		packets = append(packets, packet)
	}

	return packets, nil
}

// CompositeGatherer 组合多个收集器。
type CompositeGatherer struct {
	gatherers []Gatherer
	parallel  bool
}

// NewCompositeGatherer 创建新的 CompositeGatherer。
func NewCompositeGatherer(gatherers []Gatherer, parallel bool) *CompositeGatherer {
	return &CompositeGatherer{
		gatherers: gatherers,
		parallel:  parallel,
	}
}

// Gather 从所有收集器收集包。
func (g *CompositeGatherer) Gather(ctx context.Context, input *GatherInput) ([]*Packet, error) {
	if g.parallel {
		return g.gatherParallel(ctx, input)
	}
	return g.gatherSequential(ctx, input)
}

func (g *CompositeGatherer) gatherSequential(ctx context.Context, input *GatherInput) ([]*Packet, error) {
	var allPackets []*Packet

	for _, gatherer := range g.gatherers {
		packets, err := gatherer.Gather(ctx, input)
		if err != nil {
			// 记录错误但继续其他收集器
			continue
		}
		allPackets = append(allPackets, packets...)
	}

	return allPackets, nil
}

func (g *CompositeGatherer) gatherParallel(ctx context.Context, input *GatherInput) ([]*Packet, error) {
	var (
		mu         sync.Mutex
		wg         sync.WaitGroup
		allPackets []*Packet
	)

	for _, gatherer := range g.gatherers {
		wg.Add(1)
		go func(gth Gatherer) {
			defer wg.Done()

			packets, err := gth.Gather(ctx, input)
			if err != nil {
				return
			}

			mu.Lock()
			allPackets = append(allPackets, packets...)
			mu.Unlock()
		}(gatherer)
	}

	wg.Wait()
	return allPackets, nil
}

// 编译时接口检查
var _ Gatherer = (*InstructionsGatherer)(nil)
var _ Gatherer = (*TaskGatherer)(nil)
var _ Gatherer = (*HistoryGatherer)(nil)
var _ Gatherer = (*MemoryGatherer)(nil)
var _ Gatherer = (*RAGGatherer)(nil)
var _ Gatherer = (*CompositeGatherer)(nil)
