// Package protocols 提供智能体通信协议的接口定义和实现
//
// 本模块支持多种协议：
//   - MCP (Model Context Protocol): 模型上下文协议，用于工具/资源/提示词管理
package protocols

// ProtocolType 协议类型枚举
type ProtocolType string

const (
	// ProtocolMCP Model Context Protocol
	ProtocolMCP ProtocolType = "mcp"
	// ProtocolA2A Agent-to-Agent Protocol (保留，未实现)
	ProtocolA2A ProtocolType = "a2a"
	// ProtocolANP Agent Network Protocol (保留，未实现)
	ProtocolANP ProtocolType = "anp"
)

// String 返回协议类型的字符串表示
func (p ProtocolType) String() string {
	return string(p)
}

// Protocol 协议基础接口（概念性，实际实现可不继承）
//
// 这个接口定义了协议的基本概念，但各协议根据自己的特点独立实现。
type Protocol interface {
	// ProtocolName 返回协议名称
	ProtocolName() string
	// Version 返回协议版本
	Version() string
}

// BaseProtocol 协议基类（可选使用）
type BaseProtocol struct {
	protocolType ProtocolType
	version      string
}

// NewBaseProtocol 创建协议基类
func NewBaseProtocol(pt ProtocolType, version string) *BaseProtocol {
	return &BaseProtocol{
		protocolType: pt,
		version:      version,
	}
}

// ProtocolName 返回协议名称
func (p *BaseProtocol) ProtocolName() string {
	return p.protocolType.String()
}

// Version 返回协议版本
func (p *BaseProtocol) Version() string {
	return p.version
}
