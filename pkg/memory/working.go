package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/easyops/helloagents-go/pkg/core/message"
)

// workingMessage 带有重要性和向量的消息
type workingMessage struct {
	Message    message.Message
	Importance float32
	Vector     []float32 // TF-IDF 向量
}

// WorkingMemory 工作记忆实现
//
// 基于内存的对话历史存储，支持容量限制、TTL、重要性评分和语义检索。
type WorkingMemory struct {
	messages   []workingMessage
	maxSize    int
	tokenLimit int
	ttl        time.Duration
	tfidf      *TFIDFVectorizer // TF-IDF 向量化器
	mu         sync.RWMutex
}

// WorkingMemoryOption 配置选项
type WorkingMemoryOption func(*WorkingMemory)

// NewWorkingMemory 创建工作记忆
func NewWorkingMemory(opts ...WorkingMemoryOption) *WorkingMemory {
	m := &WorkingMemory{
		messages:   make([]workingMessage, 0),
		maxSize:    100,  // 默认最多 100 条消息
		tokenLimit: 4000, // 默认 4000 token 限制
		ttl:        0,    // 默认不过期
		tfidf:      NewTFIDFVectorizer(),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// WithMaxSize 设置最大消息数量
func WithMaxSize(size int) WorkingMemoryOption {
	return func(m *WorkingMemory) {
		m.maxSize = size
	}
}

// WithTokenLimit 设置 token 限制
func WithTokenLimit(limit int) WorkingMemoryOption {
	return func(m *WorkingMemory) {
		m.tokenLimit = limit
	}
}

// WithTTL 设置消息过期时间
func WithTTL(ttl time.Duration) WorkingMemoryOption {
	return func(m *WorkingMemory) {
		m.ttl = ttl
	}
}

// AddMessage 添加消息到记忆
func (m *WorkingMemory) AddMessage(ctx context.Context, msg message.Message) error {
	return m.AddMessageWithImportance(ctx, msg, 0.5)
}

// AddMessageWithImportance 添加带重要性的消息到记忆
func (m *WorkingMemory) AddMessageWithImportance(ctx context.Context, msg message.Message, importance float32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 设置时间戳
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	// 限制重要性范围
	if importance < 0 {
		importance = 0
	}
	if importance > 1 {
		importance = 1
	}

	wm := workingMessage{
		Message:    msg,
		Importance: importance,
	}

	m.messages = append(m.messages, wm)

	// 应用 LRU 清理
	if m.maxSize > 0 && len(m.messages) > m.maxSize {
		m.messages = m.messages[len(m.messages)-m.maxSize:]
	}

	// 重建 TF-IDF（增量更新）
	m.rebuildTFIDF()

	return nil
}

// rebuildTFIDF 重建 TF-IDF 向量化器
func (m *WorkingMemory) rebuildTFIDF() {
	if len(m.messages) == 0 {
		m.tfidf.Clear()
		return
	}

	// 收集所有文档
	docs := make([]string, len(m.messages))
	for i, wm := range m.messages {
		docs[i] = wm.Message.Content
	}

	// 重新训练和转换
	vectors := m.tfidf.FitTransform(docs)

	// 更新向量
	for i := range m.messages {
		if i < len(vectors) {
			m.messages[i].Vector = vectors[i]
		}
	}
}

// GetHistory 获取对话历史
func (m *WorkingMemory) GetHistory(ctx context.Context, limit int) ([]message.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 清理过期消息
	messages := m.filterExpired()

	if limit <= 0 || limit >= len(messages) {
		result := make([]message.Message, len(messages))
		for i, wm := range messages {
			result[i] = wm.Message
		}
		return result, nil
	}

	// 返回最近的 limit 条
	start := len(messages) - limit
	result := make([]message.Message, limit)
	for i, wm := range messages[start:] {
		result[i] = wm.Message
	}
	return result, nil
}

// GetRecentHistory 获取最近 n 条消息
func (m *WorkingMemory) GetRecentHistory(ctx context.Context, n int) ([]message.Message, error) {
	return m.GetHistory(ctx, n)
}

// Clear 清空记忆
func (m *WorkingMemory) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = make([]workingMessage, 0)
	m.tfidf.Clear()
	return nil
}

// Size 返回当前消息数量
func (m *WorkingMemory) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.messages)
}

// filterExpired 过滤过期消息（内部使用，需要持有锁）
func (m *WorkingMemory) filterExpired() []workingMessage {
	if m.ttl == 0 {
		return m.messages
	}

	cutoff := time.Now().Add(-m.ttl)
	result := make([]workingMessage, 0, len(m.messages))

	for _, wm := range m.messages {
		if wm.Message.Timestamp.After(cutoff) {
			result = append(result, wm)
		}
	}

	return result
}

// GetMessagesWithinTokenLimit 获取不超过 token 限制的消息
//
// 从最新消息开始，向前累计直到达到 token 限制。
// 注意：此方法使用简化的 token 计算（按字符数估算）。
func (m *WorkingMemory) GetMessagesWithinTokenLimit(ctx context.Context) ([]message.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.tokenLimit <= 0 {
		result := make([]message.Message, len(m.messages))
		for i, wm := range m.messages {
			result[i] = wm.Message
		}
		return result, nil
	}

	messages := m.filterExpired()
	result := make([]message.Message, 0)
	totalTokens := 0

	// 从最新消息向前遍历
	for i := len(messages) - 1; i >= 0; i-- {
		wm := messages[i]
		// 简化的 token 估算：1 token ≈ 4 字符（英文），中文约 1-2 字符
		tokens := len(wm.Message.Content) / 3
		if totalTokens+tokens > m.tokenLimit {
			break
		}
		totalTokens += tokens
		result = append([]message.Message{wm.Message}, result...)
	}

	return result, nil
}

// compile-time interface check
var _ ConversationMemory = (*WorkingMemory)(nil)
var _ Memory = (*WorkingMemory)(nil)

// Add 添加记忆项（实现 Memory 接口）
func (m *WorkingMemory) Add(ctx context.Context, item *MemoryItem) (string, error) {
	if err := item.Validate(); err != nil {
		return "", err
	}

	msg := message.Message{
		ID:        item.ID,
		Role:      message.RoleUser, // 默认角色
		Content:   item.Content,
		Metadata:  item.Metadata,
		Timestamp: item.Timestamp,
	}

	// 尝试从元数据获取角色
	if role, ok := item.Metadata["role"].(string); ok {
		msg.Role = message.Role(role)
	}

	if err := m.AddMessageWithImportance(ctx, msg, item.Importance); err != nil {
		return "", err
	}

	return item.ID, nil
}

// Retrieve 检索记忆（实现 Memory 接口）
//
// 使用 TF-IDF 语义检索，失败时回退到关键词匹配。
func (m *WorkingMemory) Retrieve(ctx context.Context, query string, opts ...RetrieveOption) ([]*MemoryItem, error) {
	options := &retrieveOptions{
		limit: 10,
	}
	for _, opt := range opts {
		opt(options)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	messages := m.filterExpired()
	if len(messages) == 0 {
		return nil, nil
	}

	// 尝试 TF-IDF 检索
	results, err := m.tfidfSearch(query, messages, options.limit)
	if err != nil || len(results) == 0 {
		// 回退到关键词检索
		results = m.keywordSearch(query, messages, options.limit)
	}

	// 过滤最小分数
	if options.minScore > 0 {
		filtered := make([]*MemoryItem, 0, len(results))
		for _, item := range results {
			if item.Importance >= options.minScore {
				filtered = append(filtered, item)
			}
		}
		return filtered, nil
	}

	return results, nil
}

// scoredMessage 带分数的消息
type scoredMessage struct {
	message    workingMessage
	score      float32
	similarity float32
}

// tfidfSearch TF-IDF 语义检索
func (m *WorkingMemory) tfidfSearch(query string, messages []workingMessage, limit int) ([]*MemoryItem, error) {
	if m.tfidf.VocabularySize() == 0 {
		return nil, nil
	}

	queryVector := m.tfidf.Transform(query)
	if queryVector == nil {
		return nil, nil
	}

	scored := make([]scoredMessage, 0, len(messages))
	for _, wm := range messages {
		if wm.Vector == nil {
			continue
		}
		similarity := m.tfidf.CosineSimilarity(queryVector, wm.Vector)
		score := m.calculateScore(similarity, float32(wm.Message.Timestamp.Sub(time.Now()).Hours()*-1), wm.Importance)
		scored = append(scored, scoredMessage{
			message:    wm,
			score:      score,
			similarity: similarity,
		})
	}

	// 按分数排序
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// 转换为 MemoryItem
	if limit > 0 && limit < len(scored) {
		scored = scored[:limit]
	}

	results := make([]*MemoryItem, len(scored))
	for i, sm := range scored {
		results[i] = m.messageToItem(sm.message, sm.score)
	}

	return results, nil
}

// keywordSearch 关键词匹配检索
func (m *WorkingMemory) keywordSearch(query string, messages []workingMessage, limit int) []*MemoryItem {
	query = strings.ToLower(query)
	keywords := strings.Fields(query)

	scored := make([]scoredMessage, 0, len(messages))
	for _, wm := range messages {
		content := strings.ToLower(wm.Message.Content)
		matchCount := 0
		for _, kw := range keywords {
			if strings.Contains(content, kw) {
				matchCount++
			}
		}

		if matchCount > 0 {
			similarity := float32(matchCount) / float32(len(keywords))
			ageHours := float32(time.Since(wm.Message.Timestamp).Hours())
			score := m.calculateScore(similarity, ageHours, wm.Importance)
			scored = append(scored, scoredMessage{
				message:    wm,
				score:      score,
				similarity: similarity,
			})
		}
	}

	// 按分数排序
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// 转换为 MemoryItem
	if limit > 0 && limit < len(scored) {
		scored = scored[:limit]
	}

	results := make([]*MemoryItem, len(scored))
	for i, sm := range scored {
		results[i] = m.messageToItem(sm.message, sm.score)
	}

	return results
}

// calculateScore 计算综合得分
//
// 公式：(语义相似度 × 时间衰减) × (0.8 + 重要性 × 0.4)
func (m *WorkingMemory) calculateScore(similarity, ageHours, importance float32) float32 {
	// 时间衰减：1.0 / (1.0 + age_hours / 24.0)
	timeDecay := 1.0 / (1.0 + ageHours/24.0)

	// 重要性权重：范围 [0.8, 1.2]
	importanceWeight := 0.8 + importance*0.4

	return similarity * timeDecay * importanceWeight
}

// messageToItem 将消息转换为 MemoryItem
func (m *WorkingMemory) messageToItem(wm workingMessage, score float32) *MemoryItem {
	metadata := wm.Message.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["role"] = string(wm.Message.Role)
	metadata["score"] = score

	return &MemoryItem{
		ID:         wm.Message.ID,
		Content:    wm.Message.Content,
		MemoryType: MemoryTypeWorking,
		Timestamp:  wm.Message.Timestamp,
		Importance: wm.Importance,
		Metadata:   metadata,
	}
}

// Update 更新记忆（实现 Memory 接口）
func (m *WorkingMemory) Update(ctx context.Context, id string, opts ...UpdateOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	options := &updateOptions{}
	for _, opt := range opts {
		opt(options)
	}

	for i := range m.messages {
		if m.messages[i].Message.ID == id {
			if options.content != nil {
				m.messages[i].Message.Content = *options.content
			}
			if options.importance != nil {
				m.messages[i].Importance = *options.importance
			}
			if options.metadata != nil {
				m.messages[i].Message.Metadata = options.metadata
			}
			// 重建 TF-IDF
			m.rebuildTFIDF()
			return nil
		}
	}

	return ErrNotFound
}

// Remove 删除记忆（实现 Memory 接口）
func (m *WorkingMemory) Remove(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.messages {
		if m.messages[i].Message.ID == id {
			m.messages = append(m.messages[:i], m.messages[i+1:]...)
			m.rebuildTFIDF()
			return nil
		}
	}

	return ErrNotFound
}

// Has 检查记忆是否存在（实现 Memory 接口）
func (m *WorkingMemory) Has(ctx context.Context, id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, wm := range m.messages {
		if wm.Message.ID == id {
			return true
		}
	}
	return false
}

// GetStats 获取统计信息（实现 Memory 接口）
func (m *WorkingMemory) GetStats(ctx context.Context) (*MemoryStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	messages := m.filterExpired()
	if len(messages) == 0 {
		return &MemoryStats{Count: 0}, nil
	}

	var totalImportance float32
	var oldestTs, newestTs int64
	totalTokens := 0

	for i, wm := range messages {
		totalImportance += wm.Importance
		totalTokens += len(wm.Message.Content) / 3

		ts := wm.Message.Timestamp.UnixMilli()
		if i == 0 || ts < oldestTs {
			oldestTs = ts
		}
		if i == 0 || ts > newestTs {
			newestTs = ts
		}
	}

	return &MemoryStats{
		Count:           len(messages),
		TotalTokens:     totalTokens,
		OldestTimestamp: oldestTs,
		NewestTimestamp: newestTs,
		AvgImportance:   totalImportance / float32(len(messages)),
	}, nil
}

// Forget 执行遗忘（实现 Memory 接口的扩展）
func (m *WorkingMemory) Forget(ctx context.Context, strategy ForgetStrategy, opts ...ForgetOption) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	options := &forgetOptions{}
	for _, opt := range opts {
		opt(options)
	}

	originalCount := len(m.messages)
	var remaining []workingMessage

	switch strategy {
	case ForgetByImportance:
		threshold := options.threshold
		if threshold <= 0 {
			threshold = 0.3 // 默认阈值
		}
		for _, wm := range m.messages {
			if wm.Importance >= threshold {
				remaining = append(remaining, wm)
			}
		}

	case ForgetByTime:
		maxAgeDays := options.maxAgeDays
		if maxAgeDays <= 0 {
			maxAgeDays = 7 // 默认 7 天
		}
		cutoff := time.Now().AddDate(0, 0, -maxAgeDays)
		for _, wm := range m.messages {
			if wm.Message.Timestamp.After(cutoff) {
				remaining = append(remaining, wm)
			}
		}

	case ForgetByCapacity:
		targetCapacity := options.targetCapacity
		if targetCapacity <= 0 {
			targetCapacity = m.maxSize / 2 // 默认减半
		}
		if len(m.messages) <= targetCapacity {
			return 0, nil
		}
		// 按重要性排序，保留最重要的
		sorted := make([]workingMessage, len(m.messages))
		copy(sorted, m.messages)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Importance > sorted[j].Importance
		})
		remaining = sorted[:targetCapacity]
		// 按时间重新排序
		sort.Slice(remaining, func(i, j int) bool {
			return remaining[i].Message.Timestamp.Before(remaining[j].Message.Timestamp)
		})
	}

	m.messages = remaining
	m.rebuildTFIDF()

	return originalCount - len(m.messages), nil
}

// GetImportant 获取高重要性记忆
func (m *WorkingMemory) GetImportant(ctx context.Context, limit int) ([]*MemoryItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	messages := m.filterExpired()
	if len(messages) == 0 {
		return nil, nil
	}

	// 按重要性排序
	sorted := make([]workingMessage, len(messages))
	copy(sorted, messages)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Importance > sorted[j].Importance
	})

	if limit > 0 && limit < len(sorted) {
		sorted = sorted[:limit]
	}

	results := make([]*MemoryItem, len(sorted))
	for i, wm := range sorted {
		results[i] = m.messageToItem(wm, wm.Importance)
	}

	return results, nil
}

// GetRecent 获取最近记忆
func (m *WorkingMemory) GetRecent(ctx context.Context, limit int) ([]*MemoryItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	messages := m.filterExpired()
	if len(messages) == 0 {
		return nil, nil
	}

	// 从最新开始
	start := 0
	if limit > 0 && limit < len(messages) {
		start = len(messages) - limit
	}

	results := make([]*MemoryItem, len(messages)-start)
	for i := start; i < len(messages); i++ {
		wm := messages[i]
		results[i-start] = m.messageToItem(wm, wm.Importance)
	}

	// 反转顺序（最新在前）
	for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
		results[i], results[j] = results[j], results[i]
	}

	return results, nil
}

// GetContextSummary 获取上下文摘要
func (m *WorkingMemory) GetContextSummary(ctx context.Context, maxLength int) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	messages := m.filterExpired()
	if len(messages) == 0 {
		return "", nil
	}

	var builder strings.Builder
	for _, wm := range messages {
		if maxLength > 0 && builder.Len()+len(wm.Message.Content) > maxLength {
			break
		}
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(string(wm.Message.Role))
		builder.WriteString(": ")
		builder.WriteString(wm.Message.Content)
	}

	return builder.String(), nil
}
