package tools

import (
	"sync"

	"github.com/easyops/helloagents-go/pkg/core/errors"
)

// Registry 工具注册表
//
// 用于管理和查找已注册的工具。支持并发安全的注册和查询。
type Registry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

// NewRegistry 创建新的工具注册表
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register 注册工具
//
// 如果工具名已存在，将返回错误。
func (r *Registry) Register(tool Tool) error {
	if tool == nil {
		return errors.ErrInvalidTool
	}

	name := tool.Name()
	if name == "" {
		return errors.ErrInvalidTool
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; exists {
		return errors.ErrToolAlreadyRegistered
	}

	r.tools[name] = tool
	return nil
}

// MustRegister 注册工具，失败则 panic
func (r *Registry) MustRegister(tool Tool) {
	if err := r.Register(tool); err != nil {
		panic(err)
	}
}

// RegisterAll 批量注册工具
//
// 如果任一工具注册失败，将停止注册并返回错误。
// 已成功注册的工具不会被回滚。
func (r *Registry) RegisterAll(tools ...Tool) error {
	for _, tool := range tools {
		if err := r.Register(tool); err != nil {
			return err
		}
	}
	return nil
}

// Get 获取工具
//
// 如果工具不存在，返回 nil 和 ErrToolNotFound。
func (r *Registry) Get(name string) (Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, errors.ErrToolNotFound
	}

	return tool, nil
}

// Has 检查工具是否存在
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.tools[name]
	return exists
}

// Unregister 取消注册工具
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return errors.ErrToolNotFound
	}

	delete(r.tools, name)
	return nil
}

// List 返回所有已注册工具的名称
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// All 返回所有已注册的工具
func (r *Registry) All() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// Count 返回已注册工具数量
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// Clear 清空所有已注册工具
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools = make(map[string]Tool)
}

// ToDefinitions 将所有工具转换为定义列表
func (r *Registry) ToDefinitions() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defs := make([]ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		defs = append(defs, ToDefinition(tool))
	}
	return defs
}

// DefaultRegistry 默认的全局工具注册表
var DefaultRegistry = NewRegistry()
