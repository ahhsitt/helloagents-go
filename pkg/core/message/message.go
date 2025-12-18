// Package message 定义对话消息相关的类型
package message

import (
	"time"
)

// Role 表示消息的角色类型
type Role string

const (
	// RoleSystem 系统消息
	RoleSystem Role = "system"
	// RoleUser 用户消息
	RoleUser Role = "user"
	// RoleAssistant AI 助手消息
	RoleAssistant Role = "assistant"
	// RoleTool 工具调用结果消息
	RoleTool Role = "tool"
)

// IsValid 检查 Role 是否为有效值
func (r Role) IsValid() bool {
	switch r {
	case RoleSystem, RoleUser, RoleAssistant, RoleTool:
		return true
	default:
		return false
	}
}

// ToolCall 表示 LLM 请求的工具调用
type ToolCall struct {
	// ID 调用标识
	ID string `json:"id"`
	// Name 工具名称
	Name string `json:"name"`
	// Arguments 调用参数
	Arguments map[string]interface{} `json:"arguments"`
}

// Message 表示对话中的一条消息
type Message struct {
	// ID 消息唯一标识
	ID string `json:"id,omitempty"`
	// Role 消息角色
	Role Role `json:"role"`
	// Content 消息内容
	Content string `json:"content"`
	// Name 名称（当 Role=tool 时为工具名称）
	Name string `json:"name,omitempty"`
	// ToolCalls 工具调用请求（当 Role=assistant 时）
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	// ToolCallID 对应的工具调用 ID（当 Role=tool 时）
	ToolCallID string `json:"tool_call_id,omitempty"`
	// Metadata 元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	// Timestamp 时间戳
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// NewMessage 创建新消息
func NewMessage(role Role, content string) Message {
	return Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// NewSystemMessage 创建系统消息
func NewSystemMessage(content string) Message {
	return NewMessage(RoleSystem, content)
}

// NewUserMessage 创建用户消息
func NewUserMessage(content string) Message {
	return NewMessage(RoleUser, content)
}

// NewAssistantMessage 创建助手消息
func NewAssistantMessage(content string) Message {
	return NewMessage(RoleAssistant, content)
}

// NewToolMessage 创建工具结果消息
func NewToolMessage(toolCallID, name, content string) Message {
	return Message{
		Role:       RoleTool,
		Content:    content,
		Name:       name,
		ToolCallID: toolCallID,
		Timestamp:  time.Now(),
	}
}

// Validate 验证消息是否有效
func (m *Message) Validate() error {
	if !m.Role.IsValid() {
		return ErrInvalidRole
	}
	// Content 可以为空（当 Role=assistant 且有 ToolCalls 时）
	if m.Content == "" && m.Role != RoleAssistant {
		return ErrEmptyContent
	}
	if m.Content == "" && m.Role == RoleAssistant && len(m.ToolCalls) == 0 {
		return ErrEmptyContent
	}
	// 当 Role=tool 时，ToolCallID 必须非空
	if m.Role == RoleTool && m.ToolCallID == "" {
		return ErrMissingToolCallID
	}
	return nil
}

// HasToolCalls 检查消息是否包含工具调用
func (m *Message) HasToolCalls() bool {
	return len(m.ToolCalls) > 0
}
