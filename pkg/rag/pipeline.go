package rag

import (
	"context"
	"fmt"
	"strings"
)

// RAGPipeline RAG 管道接口
type RAGPipeline interface {
	// Ingest 摄取文档
	Ingest(ctx context.Context, docs []Document) error
	// Query 查询并生成回答
	Query(ctx context.Context, query string, topK int) (*RAGResponse, error)
}

// RAGResponse RAG 响应
type RAGResponse struct {
	// Answer 生成的回答
	Answer string `json:"answer"`
	// Sources 来源引用
	Sources []Source `json:"sources"`
	// Context 检索到的上下文
	Context *RAGContext `json:"context"`
}

// Source 来源引用
type Source struct {
	// DocumentID 文档 ID
	DocumentID string `json:"document_id"`
	// ChunkID 块 ID
	ChunkID string `json:"chunk_id"`
	// Content 内容片段
	Content string `json:"content"`
	// Source 来源（如文件路径）
	Source string `json:"source"`
	// Score 相关性分数
	Score float32 `json:"score"`
}

// DefaultRAGPipeline 默认 RAG 管道
type DefaultRAGPipeline struct {
	loader    DocumentLoader
	chunker   DocumentChunker
	embedder  Embedder
	store     VectorStore
	retriever Retriever
	generator AnswerGenerator
}

// AnswerGenerator 回答生成器接口
type AnswerGenerator interface {
	// Generate 基于上下文生成回答
	Generate(ctx context.Context, query string, context *RAGContext) (string, error)
}

// RAGPipelineOption RAG 管道选项
type RAGPipelineOption func(*DefaultRAGPipeline)

// WithLoader 设置文档加载器
func WithLoader(loader DocumentLoader) RAGPipelineOption {
	return func(p *DefaultRAGPipeline) {
		p.loader = loader
	}
}

// WithChunker 设置文档分块器
func WithChunker(chunker DocumentChunker) RAGPipelineOption {
	return func(p *DefaultRAGPipeline) {
		p.chunker = chunker
	}
}

// WithEmbedder 设置嵌入器
func WithEmbedder(embedder Embedder) RAGPipelineOption {
	return func(p *DefaultRAGPipeline) {
		p.embedder = embedder
	}
}

// WithStore 设置向量存储
func WithStore(store VectorStore) RAGPipelineOption {
	return func(p *DefaultRAGPipeline) {
		p.store = store
	}
}

// WithRetriever 设置检索器
func WithRetriever(retriever Retriever) RAGPipelineOption {
	return func(p *DefaultRAGPipeline) {
		p.retriever = retriever
	}
}

// WithGenerator 设置回答生成器
func WithGenerator(generator AnswerGenerator) RAGPipelineOption {
	return func(p *DefaultRAGPipeline) {
		p.generator = generator
	}
}

// NewRAGPipeline 创建 RAG 管道
func NewRAGPipeline(opts ...RAGPipelineOption) *DefaultRAGPipeline {
	p := &DefaultRAGPipeline{}

	for _, opt := range opts {
		opt(p)
	}

	// 设置默认值
	if p.chunker == nil {
		p.chunker = NewRecursiveCharacterChunker(512, 50)
	}

	if p.store == nil {
		p.store = NewInMemoryVectorStore()
	}

	return p
}

// Ingest 摄取文档
func (p *DefaultRAGPipeline) Ingest(ctx context.Context, docs []Document) error {
	if p.embedder == nil {
		return fmt.Errorf("embedder is required for ingestion")
	}

	var allChunks []DocumentChunk

	// 分块处理每个文档
	for _, doc := range docs {
		chunks := p.chunker.Chunk(doc)
		allChunks = append(allChunks, chunks...)
	}

	if len(allChunks) == 0 {
		return nil
	}

	// 批量生成嵌入
	contents := make([]string, len(allChunks))
	for i, chunk := range allChunks {
		contents[i] = chunk.Content
	}

	embeddings, err := p.embedder.Embed(ctx, contents)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// 将嵌入向量附加到块
	for i := range allChunks {
		if i < len(embeddings) {
			allChunks[i].Vector = embeddings[i]
		}
	}

	// 存储到向量数据库
	if err := p.store.Add(ctx, allChunks); err != nil {
		return fmt.Errorf("failed to store chunks: %w", err)
	}

	return nil
}

// IngestFromLoader 从加载器摄取文档
func (p *DefaultRAGPipeline) IngestFromLoader(ctx context.Context, loader DocumentLoader) error {
	docs, err := loader.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load documents: %w", err)
	}

	return p.Ingest(ctx, docs)
}

// Query 查询并生成回答
func (p *DefaultRAGPipeline) Query(ctx context.Context, query string, topK int) (*RAGResponse, error) {
	if p.retriever == nil {
		return nil, fmt.Errorf("retriever is required for query")
	}

	// 检索相关文档
	results, err := p.retriever.Retrieve(ctx, query, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve documents: %w", err)
	}

	// 构建上下文
	ragContext := &RAGContext{
		Query:          query,
		Results:        results,
		TotalDocuments: len(results),
	}

	// 构建来源引用
	sources := make([]Source, len(results))
	for i, r := range results {
		sources[i] = Source{
			DocumentID: r.Chunk.DocumentID,
			ChunkID:    r.Chunk.ID,
			Content:    truncateContent(r.Chunk.Content, 200),
			Source:     r.Chunk.Metadata.Source,
			Score:      r.Score,
		}
	}

	response := &RAGResponse{
		Sources: sources,
		Context: ragContext,
	}

	// 生成回答
	if p.generator != nil {
		answer, err := p.generator.Generate(ctx, query, ragContext)
		if err != nil {
			return nil, fmt.Errorf("failed to generate answer: %w", err)
		}
		response.Answer = answer
	}

	return response, nil
}

// QueryWithoutGeneration 仅检索不生成回答
func (p *DefaultRAGPipeline) QueryWithoutGeneration(ctx context.Context, query string, topK int) ([]RetrievalResult, error) {
	if p.retriever == nil {
		return nil, fmt.Errorf("retriever is required for query")
	}

	return p.retriever.Retrieve(ctx, query, topK)
}

// SetRetriever 设置检索器
func (p *DefaultRAGPipeline) SetRetriever(retriever Retriever) {
	p.retriever = retriever
}

// SetGenerator 设置回答生成器
func (p *DefaultRAGPipeline) SetGenerator(generator AnswerGenerator) {
	p.generator = generator
}

// GetStore 获取向量存储
func (p *DefaultRAGPipeline) GetStore() VectorStore {
	return p.store
}

// truncateContent 截断内容
func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "..."
}

// SimpleAnswerGenerator 简单回答生成器（用于测试）
type SimpleAnswerGenerator struct{}

// Generate 基于上下文生成简单回答
func (g *SimpleAnswerGenerator) Generate(ctx context.Context, query string, ragContext *RAGContext) (string, error) {
	if len(ragContext.Results) == 0 {
		return "No relevant information found.", nil
	}

	var sb strings.Builder
	sb.WriteString("Based on the available information:\n\n")

	for i, r := range ragContext.Results {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, truncateContent(r.Chunk.Content, 500)))
		if r.Chunk.Metadata.Source != "" {
			sb.WriteString(fmt.Sprintf("   (Source: %s)\n", r.Chunk.Metadata.Source))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// compile-time interface check
var _ RAGPipeline = (*DefaultRAGPipeline)(nil)
var _ AnswerGenerator = (*SimpleAnswerGenerator)(nil)
