package rag

import (
	"context"
	"io"
)

// DocumentLoader 文档加载器接口
type DocumentLoader interface {
	// Load 加载文档
	Load(ctx context.Context) ([]Document, error)
	// SupportedExtensions 支持的文件扩展名
	SupportedExtensions() []string
}

// TextLoader 文本文件加载器
type TextLoader struct {
	source string
	reader io.Reader
}

// NewTextLoader 从 io.Reader 创建文本加载器
func NewTextLoader(source string, reader io.Reader) *TextLoader {
	return &TextLoader{
		source: source,
		reader: reader,
	}
}

// Load 加载文档
func (l *TextLoader) Load(ctx context.Context) ([]Document, error) {
	content, err := io.ReadAll(l.reader)
	if err != nil {
		return nil, err
	}

	doc := Document{
		ID:      generateID(),
		Content: string(content),
		Metadata: DocumentMetadata{
			Source: l.source,
		},
	}

	return []Document{doc}, nil
}

// SupportedExtensions 支持的文件扩展名
func (l *TextLoader) SupportedExtensions() []string {
	return []string{".txt", ".md", ".text"}
}

// StringLoader 字符串加载器（用于直接加载字符串内容）
type StringLoader struct {
	content  string
	metadata DocumentMetadata
}

// NewStringLoader 创建字符串加载器
func NewStringLoader(content string, metadata DocumentMetadata) *StringLoader {
	return &StringLoader{
		content:  content,
		metadata: metadata,
	}
}

// Load 加载文档
func (l *StringLoader) Load(ctx context.Context) ([]Document, error) {
	doc := Document{
		ID:       generateID(),
		Content:  l.content,
		Metadata: l.metadata,
	}
	return []Document{doc}, nil
}

// SupportedExtensions 支持的文件扩展名
func (l *StringLoader) SupportedExtensions() []string {
	return []string{}
}

// MultiLoader 多文档加载器
type MultiLoader struct {
	loaders []DocumentLoader
}

// NewMultiLoader 创建多文档加载器
func NewMultiLoader(loaders ...DocumentLoader) *MultiLoader {
	return &MultiLoader{loaders: loaders}
}

// Load 加载所有文档
func (l *MultiLoader) Load(ctx context.Context) ([]Document, error) {
	var docs []Document

	for _, loader := range l.loaders {
		loadedDocs, err := loader.Load(ctx)
		if err != nil {
			return nil, err
		}
		docs = append(docs, loadedDocs...)
	}

	return docs, nil
}

// SupportedExtensions 支持的文件扩展名
func (l *MultiLoader) SupportedExtensions() []string {
	extSet := make(map[string]struct{})
	for _, loader := range l.loaders {
		for _, ext := range loader.SupportedExtensions() {
			extSet[ext] = struct{}{}
		}
	}

	exts := make([]string, 0, len(extSet))
	for ext := range extSet {
		exts = append(exts, ext)
	}
	return exts
}

// compile-time interface check
var _ DocumentLoader = (*TextLoader)(nil)
var _ DocumentLoader = (*StringLoader)(nil)
var _ DocumentLoader = (*MultiLoader)(nil)
