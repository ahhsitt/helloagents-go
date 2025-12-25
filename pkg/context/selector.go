package context

import (
	"math"
	"sort"
	"strings"
	"time"
)

// Selector 定义筛选和评分上下文包的接口。
type Selector interface {
	// Select 过滤和评分包，返回在预算内选中的包。
	Select(packets []*Packet, query string, config *Config) []*Packet
}

// Scorer 定义对单个包进行评分的接口。
type Scorer interface {
	// Score 根据查询计算包的分数。
	Score(packet *Packet, query string) float64
}

// RelevanceScorer 基于与查询的关键词重叠对包进行评分。
type RelevanceScorer struct{}

// NewRelevanceScorer 创建新的 RelevanceScorer。
func NewRelevanceScorer() *RelevanceScorer {
	return &RelevanceScorer{}
}

// Score 基于关键词重叠计算相关性。
func (s *RelevanceScorer) Score(packet *Packet, query string) float64 {
	if query == "" {
		return 0.0
	}

	queryTokens := tokenize(query)
	if len(queryTokens) == 0 {
		return 0.0
	}

	contentTokens := tokenize(packet.Content)
	if len(contentTokens) == 0 {
		return 0.0
	}

	// 计算重叠的词元
	querySet := make(map[string]struct{}, len(queryTokens))
	for _, token := range queryTokens {
		querySet[token] = struct{}{}
	}

	overlap := 0
	for _, token := range contentTokens {
		if _, exists := querySet[token]; exists {
			overlap++
		}
	}

	// 返回类 Jaccard 的重叠比率
	return float64(overlap) / float64(len(queryTokens))
}

// RecencyScorer 基于包的新近程度进行评分。
type RecencyScorer struct {
	// Tau 是指数衰减的时间常数（秒）。
	Tau float64
}

// NewRecencyScorer 创建新的 RecencyScorer。
func NewRecencyScorer(tau float64) *RecencyScorer {
	if tau <= 0 {
		tau = 3600 // 默认：1 小时
	}
	return &RecencyScorer{Tau: tau}
}

// Score 使用指数衰减计算新近性。
func (s *RecencyScorer) Score(packet *Packet, _ string) float64 {
	if packet.Timestamp.IsZero() {
		return 0.5 // 无时间戳的包默认分数
	}

	delta := time.Since(packet.Timestamp).Seconds()
	if delta < 0 {
		delta = 0
	}

	return math.Exp(-delta / s.Tau)
}

// CompositeScorer 组合多个评分器并使用权重。
type CompositeScorer struct {
	scorers []Scorer
	weights []float64
}

// NewCompositeScorer 创建新的 CompositeScorer。
func NewCompositeScorer(scorers []Scorer, weights []float64) *CompositeScorer {
	// 如果需要则归一化权重
	if len(weights) != len(scorers) {
		weights = make([]float64, len(scorers))
		for i := range weights {
			weights[i] = 1.0 / float64(len(scorers))
		}
	}

	return &CompositeScorer{
		scorers: scorers,
		weights: weights,
	}
}

// Score 计算加权复合分数。
func (s *CompositeScorer) Score(packet *Packet, query string) float64 {
	var total float64
	for i, scorer := range s.scorers {
		score := scorer.Score(packet, query)
		total += score * s.weights[i]
	}
	return total
}

// DefaultSelector 实现默认的筛选逻辑。
type DefaultSelector struct {
	scorer Scorer
}

// NewDefaultSelector 创建带有默认评分的新 DefaultSelector。
func NewDefaultSelector(config *Config) *DefaultSelector {
	relevanceScorer := NewRelevanceScorer()
	recencyScorer := NewRecencyScorer(config.RecencyTau)

	scorer := NewCompositeScorer(
		[]Scorer{relevanceScorer, recencyScorer},
		[]float64{config.RelevanceWeight, config.RecencyWeight},
	)

	return &DefaultSelector{scorer: scorer}
}

// Select 过滤和评分包，返回在预算内选中的包。
func (s *DefaultSelector) Select(packets []*Packet, query string, config *Config) []*Packet {
	if len(packets) == 0 {
		return nil
	}

	// 1. 对所有包评分
	for _, packet := range packets {
		// 如果已设置则使用现有相关性分数，否则计算
		if packet.RelevanceScore == 0 {
			packet.RelevanceScore = NewRelevanceScorer().Score(packet, query)
		}

		// 计算新近性分数
		packet.RecencyScore = NewRecencyScorer(config.RecencyTau).Score(packet, query)

		// 计算复合分数
		packet.CompositeScore = s.scorer.Score(packet, query)
	}

	// 2. 按优先级分类 - P0（指令）始终包含
	var p0Packets []*Packet
	var otherPackets []*Packet

	for _, packet := range packets {
		if packet.Type == PacketTypeInstructions || packet.Type == PacketTypeTask {
			p0Packets = append(p0Packets, packet)
		} else {
			otherPackets = append(otherPackets, packet)
		}
	}

	// 3. 按最低相关性过滤（非 P0 包）
	var filtered []*Packet
	for _, packet := range otherPackets {
		if packet.RelevanceScore >= config.MinRelevance || packet.Type == PacketTypeTaskState {
			filtered = append(filtered, packet)
		}
	}

	// 4. 按复合分数排序（降序）
	sort.Slice(filtered, func(i, j int) bool {
		// 首先按优先级（越低越好）
		if filtered[i].Type.Priority() != filtered[j].Type.Priority() {
			return filtered[i].Type.Priority() < filtered[j].Type.Priority()
		}
		// 然后按复合分数
		return filtered[i].CompositeScore > filtered[j].CompositeScore
	})

	// 5. 在预算内选择
	availableTokens := config.GetAvailableTokens()
	selected := make([]*Packet, 0, len(p0Packets)+len(filtered))
	usedTokens := 0

	// 始终首先包含 P0 包
	for _, packet := range p0Packets {
		if usedTokens+packet.TokenCount <= availableTokens {
			selected = append(selected, packet)
			usedTokens += packet.TokenCount
		}
	}

	// 根据分数和预算添加其他包
	for _, packet := range filtered {
		if usedTokens+packet.TokenCount > availableTokens {
			continue
		}
		selected = append(selected, packet)
		usedTokens += packet.TokenCount
	}

	return selected
}

// tokenize 将文本分割为小写词元用于比较。
func tokenize(text string) []string {
	text = strings.ToLower(text)

	// 按空格和标点符号进行简单分词
	var tokens []string
	var current strings.Builder

	for _, r := range text {
		if isTokenChar(r) {
			current.WriteRune(r)
		} else if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// isTokenChar 返回该字符是否应该是词元的一部分。
func isTokenChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= '0' && r <= '9') ||
		r >= 0x4E00 && r <= 0x9FFF // 中文字符
}

// 编译时接口检查
var _ Selector = (*DefaultSelector)(nil)
var _ Scorer = (*RelevanceScorer)(nil)
var _ Scorer = (*RecencyScorer)(nil)
var _ Scorer = (*CompositeScorer)(nil)
