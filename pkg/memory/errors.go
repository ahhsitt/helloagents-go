package memory

import "errors"

// 记忆系统相关错误
var (
	// ErrNotFound 记录未找到
	ErrNotFound = errors.New("record not found")
	// ErrEmbeddingFailed 嵌入生成失败
	ErrEmbeddingFailed = errors.New("embedding generation failed")
	// ErrMemoryFull 记忆已满
	ErrMemoryFull = errors.New("memory is full")
	// ErrInvalidInput 输入无效
	ErrInvalidInput = errors.New("invalid input")
)
