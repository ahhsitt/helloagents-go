package store

import "errors"

// Store errors
var (
	// ErrNotFound 未找到
	ErrNotFound = errors.New("not found")
	// ErrInvalidInput 无效输入
	ErrInvalidInput = errors.New("invalid input")
	// ErrConnectionFailed 连接失败
	ErrConnectionFailed = errors.New("connection failed")
	// ErrCollectionNotExists 集合不存在
	ErrCollectionNotExists = errors.New("collection not exists")
)
