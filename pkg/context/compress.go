package context

import (
	"strings"
)

// Compressor 定义在预算内压缩上下文的接口。
type Compressor interface {
	// Compress 缩减上下文字符串以适应 Token 预算。
	Compress(context string, config *Config) string
}

// TruncateCompressor 通过截断超出预算的内容进行压缩。
type TruncateCompressor struct {
	// PreserveStructure 尝试保持分段边界完整。
	PreserveStructure bool
}

// NewTruncateCompressor 创建新的 TruncateCompressor。
func NewTruncateCompressor() *TruncateCompressor {
	return &TruncateCompressor{
		PreserveStructure: true,
	}
}

// Compress 截断上下文以适应 Token 预算。
func (c *TruncateCompressor) Compress(context string, config *Config) string {
	if !config.EnableCompression {
		return context
	}

	counter := config.GetTokenCounter()
	currentTokens := counter.Count(context)
	availableTokens := config.GetAvailableTokens()

	if currentTokens <= availableTokens {
		return context
	}

	if c.PreserveStructure {
		return c.compressWithStructure(context, config)
	}

	return c.simpleCompress(context, config)
}

// simpleCompress 逐行截断直到符合预算。
func (c *TruncateCompressor) simpleCompress(context string, config *Config) string {
	counter := config.GetTokenCounter()
	availableTokens := config.GetAvailableTokens()

	lines := strings.Split(context, "\n")
	result := make([]string, 0, len(lines))
	usedTokens := 0

	for _, line := range lines {
		lineTokens := counter.Count(line + "\n")
		if usedTokens+lineTokens > availableTokens {
			break
		}
		result = append(result, line)
		usedTokens += lineTokens
	}

	return strings.Join(result, "\n")
}

// compressWithStructure 在截断时保持分段结构。
func (c *TruncateCompressor) compressWithStructure(context string, config *Config) string {
	counter := config.GetTokenCounter()
	availableTokens := config.GetAvailableTokens()

	// 解析分段
	sections := parseSections(context)

	// 截断的优先级顺序（先 P3，最后 P0）
	priorities := []string{
		"[Context]",         // P3
		"[Evidence]",        // P2
		"[State]",           // P1
		"[Output]",          // 辅助
		"[Task]",            // P1
		"[Role & Policies]", // P0
	}

	// 计算当前总量
	currentTokens := counter.Count(context)

	// 从最低优先级开始截断分段直到符合预算
	for _, priority := range priorities {
		if currentTokens <= availableTokens {
			break
		}

		if section, exists := sections[priority]; exists && section != "" {
			sectionTokens := counter.Count(section)

			// 先尝试部分截断
			truncated := truncateSection(section, sectionTokens/2, counter)
			sections[priority] = truncated

			currentTokens = counter.Count(rebuildContext(sections, priorities))

			// 如果仍超出预算，则完全删除
			if currentTokens > availableTokens {
				delete(sections, priority)
				currentTokens = counter.Count(rebuildContext(sections, priorities))
			}
		}
	}

	return rebuildContext(sections, priorities)
}

// parseSections 按标题将上下文分割成分段。
func parseSections(context string) map[string]string {
	sections := make(map[string]string)

	// 已知的分段标题
	headers := []string{
		"[Role & Policies]",
		"[Task]",
		"[State]",
		"[Evidence]",
		"[Context]",
		"[Output]",
	}

	// 查找每个分段
	for i, header := range headers {
		startIdx := strings.Index(context, header)
		if startIdx == -1 {
			continue
		}

		// 查找此分段的结束位置（下一个分段的开始或字符串结尾）
		endIdx := len(context)
		for j := i + 1; j < len(headers); j++ {
			nextIdx := strings.Index(context, headers[j])
			if nextIdx != -1 && nextIdx > startIdx && nextIdx < endIdx {
				endIdx = nextIdx
			}
		}

		sections[header] = strings.TrimSpace(context[startIdx:endIdx])
	}

	return sections
}

// truncateSection 将分段截断到大约目标 Token 数量。
func truncateSection(section string, targetTokens int, counter TokenCounter) string {
	lines := strings.Split(section, "\n")
	if len(lines) <= 2 {
		return section // 保留标题和至少一行内容
	}

	// 保留标题并截断内容
	header := lines[0]
	var result []string
	result = append(result, header)
	usedTokens := counter.Count(header)

	for i := 1; i < len(lines) && usedTokens < targetTokens; i++ {
		lineTokens := counter.Count(lines[i])
		if usedTokens+lineTokens > targetTokens {
			break
		}
		result = append(result, lines[i])
		usedTokens += lineTokens
	}

	// 添加截断指示
	if len(result) < len(lines) {
		result = append(result, "... (内容已截断)")
	}

	return strings.Join(result, "\n")
}

// rebuildContext 按正确顺序重新组装分段。
func rebuildContext(sections map[string]string, order []string) string {
	var parts []string

	// 反向重建（使 [Role & Policies] 排在最前面）
	for i := len(order) - 1; i >= 0; i-- {
		header := order[i]
		if section, exists := sections[header]; exists && section != "" {
			parts = append([]string{section}, parts...)
		}
	}

	return strings.Join(parts, "\n\n")
}

// NoOpCompressor 是一个不做任何操作的压缩器。
type NoOpCompressor struct{}

// NewNoOpCompressor 创建新的 NoOpCompressor。
func NewNoOpCompressor() *NoOpCompressor {
	return &NoOpCompressor{}
}

// Compress 原样返回上下文。
func (c *NoOpCompressor) Compress(context string, _ *Config) string {
	return context
}

// 编译时接口检查
var _ Compressor = (*TruncateCompressor)(nil)
var _ Compressor = (*NoOpCompressor)(nil)
