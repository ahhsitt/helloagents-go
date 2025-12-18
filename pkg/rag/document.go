// Package rag 提供检索增强生成（RAG）功能
package rag

import "time"

// Document 文档
type Document struct {
	// ID 文档唯一标识
	ID string `json:"id"`
	// Content 文档内容
	Content string `json:"content"`
	// Metadata 元数据
	Metadata DocumentMetadata `json:"metadata"`
}

// DocumentMetadata 文档元数据
type DocumentMetadata struct {
	// Source 来源（文件路径、URL 等）
	Source string `json:"source,omitempty"`
	// Title 标题
	Title string `json:"title,omitempty"`
	// Author 作者
	Author string `json:"author,omitempty"`
	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at,omitempty"`
	// Tags 标签
	Tags []string `json:"tags,omitempty"`
	// Custom 自定义元数据
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// DocumentChunk 文档分块
type DocumentChunk struct {
	// ID 分块唯一标识
	ID string `json:"id"`
	// DocumentID 所属文档 ID
	DocumentID string `json:"document_id"`
	// Content 分块内容
	Content string `json:"content"`
	// Index 分块索引（在文档中的位置）
	Index int `json:"index"`
	// StartOffset 在原文档中的起始位置
	StartOffset int `json:"start_offset"`
	// EndOffset 在原文档中的结束位置
	EndOffset int `json:"end_offset"`
	// Metadata 元数据（继承自文档）
	Metadata DocumentMetadata `json:"metadata"`
	// Vector 嵌入向量
	Vector []float32 `json:"vector,omitempty"`
}

// RetrievalResult 检索结果
type RetrievalResult struct {
	// Chunk 文档分块
	Chunk DocumentChunk `json:"chunk"`
	// Score 相关性分数 (0-1)
	Score float32 `json:"score"`
}

// RAGContext RAG 上下文（用于生成）
type RAGContext struct {
	// Query 原始查询
	Query string `json:"query"`
	// Results 检索结果
	Results []RetrievalResult `json:"results"`
	// TotalDocuments 检索的文档总数
	TotalDocuments int `json:"total_documents"`
}

// FormatContext 格式化上下文为提示文本
func (c *RAGContext) FormatContext() string {
	if len(c.Results) == 0 {
		return ""
	}

	var result string
	result = "Context information from relevant documents:\n\n"

	for i, r := range c.Results {
		result += "---\n"
		if r.Chunk.Metadata.Source != "" {
			result += "Source: " + r.Chunk.Metadata.Source + "\n"
		}
		result += r.Chunk.Content + "\n"
		if i < len(c.Results)-1 {
			result += "\n"
		}
	}

	result += "---\n\n"
	return result
}
