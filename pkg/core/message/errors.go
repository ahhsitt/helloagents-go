package message

import "errors"

// 消息验证相关错误
var (
	// ErrInvalidRole 无效的角色类型
	ErrInvalidRole = errors.New("invalid message role")
	// ErrEmptyContent 消息内容为空
	ErrEmptyContent = errors.New("message content cannot be empty")
	// ErrMissingToolCallID 缺少工具调用 ID
	ErrMissingToolCallID = errors.New("tool message requires tool_call_id")
)
