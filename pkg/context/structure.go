package context

import (
	"sort"
	"strings"
)

// Structurer 定义将包组织成结构化上下文的接口。
type Structurer interface {
	// Structure 将包组织成结构化的上下文字符串。
	Structure(packets []*Packet, query string, config *Config) string
}

// DefaultStructurer 实现基于 P0-P3 优先级的结构化。
type DefaultStructurer struct{}

// NewDefaultStructurer 创建新的 DefaultStructurer。
func NewDefaultStructurer() *DefaultStructurer {
	return &DefaultStructurer{}
}

// Structure 使用 P0-P3 分段将包组织成结构化上下文。
func (s *DefaultStructurer) Structure(packets []*Packet, query string, config *Config) string {
	if len(packets) == 0 {
		return ""
	}

	// 按类型分组包
	groups := make(map[PacketType][]*Packet)
	for _, packet := range packets {
		groups[packet.Type] = append(groups[packet.Type], packet)
	}

	var sections []string

	// [Role & Policies] - P0：系统指令
	if instructions := groups[PacketTypeInstructions]; len(instructions) > 0 {
		section := "[Role & Policies]\n"
		for _, p := range instructions {
			section += p.Content
		}
		sections = append(sections, section)
	}

	// [Task] - P1：当前任务/查询
	if tasks := groups[PacketTypeTask]; len(tasks) > 0 {
		section := "[Task]\n"
		section += "用户问题：" + query
		sections = append(sections, section)
	} else if query != "" {
		// 如果没有任务包则降级
		section := "[Task]\n"
		section += "用户问题：" + query
		sections = append(sections, section)
	}

	// [State] - P1：任务状态和关键结论
	if taskState := groups[PacketTypeTaskState]; len(taskState) > 0 {
		section := "[State]\n关键进展与未决问题：\n"
		for _, p := range taskState {
			section += p.Content + "\n"
		}
		sections = append(sections, section)
	}

	// [Evidence] - P2：来自 Memory/RAG 的事实证据
	if evidence := groups[PacketTypeEvidence]; len(evidence) > 0 {
		section := "[Evidence]\n事实与引用：\n"

		// 按相关性分数排序
		sorted := make([]*Packet, len(evidence))
		copy(sorted, evidence)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].RelevanceScore > sorted[j].RelevanceScore
		})

		for _, p := range sorted {
			source := p.Source
			if source == "" {
				source = "unknown"
			}
			section += "\n[来源: " + source + "]\n" + p.Content + "\n"
		}
		sections = append(sections, section)
	}

	// [Context] - P3：对话历史
	if history := groups[PacketTypeHistory]; len(history) > 0 {
		section := "[Context]\n对话历史与背景：\n"
		for _, p := range history {
			section += p.Content
		}
		sections = append(sections, section)
	}

	// [Output] - 输出约束
	outputTemplate := config.OutputTemplate
	if outputTemplate == "" {
		outputTemplate = defaultOutputTemplate
	}
	sections = append(sections, "[Output]\n"+outputTemplate)

	return strings.Join(sections, "\n\n")
}

// MinimalStructurer 提供不带分段标题的简单结构。
type MinimalStructurer struct{}

// NewMinimalStructurer 创建新的 MinimalStructurer。
func NewMinimalStructurer() *MinimalStructurer {
	return &MinimalStructurer{}
}

// Structure 将包组织成简洁的上下文字符串。
func (s *MinimalStructurer) Structure(packets []*Packet, query string, _ *Config) string {
	if len(packets) == 0 {
		return query
	}

	// 先按优先级排序，再按分数排序
	sorted := make([]*Packet, len(packets))
	copy(sorted, packets)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Type.Priority() != sorted[j].Type.Priority() {
			return sorted[i].Type.Priority() < sorted[j].Type.Priority()
		}
		return sorted[i].CompositeScore > sorted[j].CompositeScore
	})

	var parts []string
	for _, p := range sorted {
		if p.Content != "" {
			parts = append(parts, p.Content)
		}
	}

	return strings.Join(parts, "\n\n")
}

// CustomStructurer 允许基于自定义模板的结构化。
type CustomStructurer struct {
	// Template 是带有占位符的模板字符串，如 {{instructions}}、{{task}} 等。
	Template string
}

// NewCustomStructurer 使用给定模板创建新的 CustomStructurer。
func NewCustomStructurer(template string) *CustomStructurer {
	return &CustomStructurer{Template: template}
}

// Structure 将自定义模板应用于包。
func (s *CustomStructurer) Structure(packets []*Packet, query string, _ *Config) string {
	if s.Template == "" {
		return NewMinimalStructurer().Structure(packets, query, nil)
	}

	// 按类型分组包
	groups := make(map[PacketType][]*Packet)
	for _, packet := range packets {
		groups[packet.Type] = append(groups[packet.Type], packet)
	}

	result := s.Template

	// 替换占位符
	replacements := map[string]string{
		"{{instructions}}": joinPackets(groups[PacketTypeInstructions]),
		"{{task}}":         query,
		"{{task_state}}":   joinPackets(groups[PacketTypeTaskState]),
		"{{evidence}}":     joinPackets(groups[PacketTypeEvidence]),
		"{{history}}":      joinPackets(groups[PacketTypeHistory]),
		"{{custom}}":       joinPackets(groups[PacketTypeCustom]),
	}

	for placeholder, content := range replacements {
		result = strings.ReplaceAll(result, placeholder, content)
	}

	return result
}

// joinPackets 用换行符连接包内容。
func joinPackets(packets []*Packet) string {
	if len(packets) == 0 {
		return ""
	}

	var parts []string
	for _, p := range packets {
		if p.Content != "" {
			parts = append(parts, p.Content)
		}
	}

	return strings.Join(parts, "\n")
}

// 编译时接口检查
var _ Structurer = (*DefaultStructurer)(nil)
var _ Structurer = (*MinimalStructurer)(nil)
var _ Structurer = (*CustomStructurer)(nil)
