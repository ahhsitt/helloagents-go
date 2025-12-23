package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EpisodicMemoryStore 情景记忆存储实现
//
// 用于存储和检索特定事件或经历，支持会话管理、模式识别和时间线视图。
type EpisodicMemoryStore struct {
	episodes []Episode
	sessions map[string][]string // sessionID -> episodeIDs
	tfidf    *TFIDFVectorizer    // 用于本地语义检索
	mu       sync.RWMutex
}

// NewEpisodicMemory 创建情景记忆存储
func NewEpisodicMemory() *EpisodicMemoryStore {
	return &EpisodicMemoryStore{
		episodes: make([]Episode, 0),
		sessions: make(map[string][]string),
		tfidf:    NewTFIDFVectorizer(),
	}
}

// AddEpisode 添加事件
func (m *EpisodicMemoryStore) AddEpisode(ctx context.Context, episode Episode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 生成 ID（如果未提供）
	if episode.ID == "" {
		episode.ID = uuid.New().String()
	}

	// 设置时间戳（如果未提供）
	if episode.Timestamp == 0 {
		episode.Timestamp = time.Now().UnixMilli()
	}

	m.episodes = append(m.episodes, episode)

	// 维护 sessions 索引
	if episode.SessionID != "" {
		m.sessions[episode.SessionID] = append(m.sessions[episode.SessionID], episode.ID)
	}

	// 重建 TF-IDF
	m.rebuildTFIDF()

	return nil
}

// rebuildTFIDF 重建 TF-IDF 向量化器
func (m *EpisodicMemoryStore) rebuildTFIDF() {
	if len(m.episodes) == 0 {
		m.tfidf.Clear()
		return
	}

	docs := make([]string, len(m.episodes))
	for i, ep := range m.episodes {
		docs[i] = ep.Content
	}

	vectors := m.tfidf.FitTransform(docs)
	for i := range m.episodes {
		if i < len(vectors) {
			m.episodes[i].Vector = vectors[i]
		}
	}
}

// GetEpisodes 获取事件列表
func (m *EpisodicMemoryStore) GetEpisodes(ctx context.Context, filter *EpisodeFilter) ([]Episode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Episode, 0)

	for _, ep := range m.episodes {
		// 应用过滤条件
		if filter != nil {
			// 类型过滤
			if len(filter.Types) > 0 && !containsString(filter.Types, ep.Type) {
				continue
			}
			// 重要性过滤
			if filter.MinImportance > 0 && ep.Importance < filter.MinImportance {
				continue
			}
		}
		result = append(result, ep)
	}

	// 按时间戳降序排序（最新的在前）
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp > result[j].Timestamp
	})

	// 应用限制
	if filter != nil && filter.Limit > 0 && len(result) > filter.Limit {
		result = result[:filter.Limit]
	}

	return result, nil
}

// GetByTimeRange 按时间范围获取事件
func (m *EpisodicMemoryStore) GetByTimeRange(ctx context.Context, start, end int64) ([]Episode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Episode, 0)

	for _, ep := range m.episodes {
		if ep.Timestamp >= start && ep.Timestamp <= end {
			result = append(result, ep)
		}
	}

	// 按时间戳升序排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp < result[j].Timestamp
	})

	return result, nil
}

// GetByType 按类型获取事件
func (m *EpisodicMemoryStore) GetByType(ctx context.Context, typ string, limit int) ([]Episode, error) {
	return m.GetEpisodes(ctx, &EpisodeFilter{
		Types: []string{typ},
		Limit: limit,
	})
}

// GetMostImportant 获取最重要的事件
func (m *EpisodicMemoryStore) GetMostImportant(ctx context.Context, limit int) ([]Episode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Episode, len(m.episodes))
	copy(result, m.episodes)

	// 按重要性降序排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].Importance > result[j].Importance
	})

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

// Clear 清空所有事件
func (m *EpisodicMemoryStore) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.episodes = make([]Episode, 0)
	m.sessions = make(map[string][]string)
	m.tfidf.Clear()
	return nil
}

// Size 返回事件数量
func (m *EpisodicMemoryStore) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.episodes)
}

// containsString 检查字符串切片是否包含指定字符串
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// compile-time interface check
var _ EpisodicMemory = (*EpisodicMemoryStore)(nil)
var _ Memory = (*EpisodicMemoryStore)(nil)

// Add 添加记忆项（实现 Memory 接口）
func (m *EpisodicMemoryStore) Add(ctx context.Context, item *MemoryItem) (string, error) {
	if err := item.Validate(); err != nil {
		return "", err
	}

	episode := Episode{
		ID:         item.ID,
		Type:       item.GetMetadataString("type"),
		Content:    item.Content,
		Timestamp:  item.Timestamp.UnixMilli(),
		Metadata:   item.Metadata,
		Importance: item.Importance,
		UserID:     item.UserID,
		SessionID:  item.GetMetadataString("session_id"),
		Outcome:    item.GetMetadataString("outcome"),
	}

	if ctx, ok := item.Metadata["context"].(map[string]interface{}); ok {
		episode.Context = ctx
	}

	return episode.ID, m.AddEpisode(ctx, episode)
}

// Retrieve 检索记忆（实现 Memory 接口）
//
// 使用 TF-IDF 语义检索，失败时回退到关键词匹配。
func (m *EpisodicMemoryStore) Retrieve(ctx context.Context, query string, opts ...RetrieveOption) ([]*MemoryItem, error) {
	options := &retrieveOptions{
		limit: 10,
	}
	for _, opt := range opts {
		opt(options)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.episodes) == 0 {
		return nil, nil
	}

	// 尝试 TF-IDF 检索
	results := m.tfidfSearch(query, options.limit)
	if len(results) == 0 {
		// 回退到关键词检索
		results = m.keywordSearch(query, options.limit)
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

// scoredEpisode 带分数的事件
type scoredEpisode struct {
	episode    Episode
	score      float32
	similarity float32
}

// tfidfSearch TF-IDF 语义检索
func (m *EpisodicMemoryStore) tfidfSearch(query string, limit int) []*MemoryItem {
	if m.tfidf.VocabularySize() == 0 {
		return nil
	}

	queryVector := m.tfidf.Transform(query)
	if queryVector == nil {
		return nil
	}

	scored := make([]scoredEpisode, 0, len(m.episodes))
	now := time.Now().UnixMilli()

	for _, ep := range m.episodes {
		if ep.Vector == nil {
			continue
		}
		similarity := m.tfidf.CosineSimilarity(queryVector, ep.Vector)
		ageDays := float32(now-ep.Timestamp) / (24 * 60 * 60 * 1000)
		score := m.calculateScore(similarity, ageDays, ep.Importance)
		scored = append(scored, scoredEpisode{
			episode:    ep,
			score:      score,
			similarity: similarity,
		})
	}

	// 按分数排序
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	if limit > 0 && limit < len(scored) {
		scored = scored[:limit]
	}

	results := make([]*MemoryItem, len(scored))
	for i, se := range scored {
		results[i] = m.episodeToItem(se.episode, se.score)
	}

	return results
}

// keywordSearch 关键词匹配检索
func (m *EpisodicMemoryStore) keywordSearch(query string, limit int) []*MemoryItem {
	query = strings.ToLower(query)
	keywords := strings.Fields(query)

	scored := make([]scoredEpisode, 0, len(m.episodes))
	now := time.Now().UnixMilli()

	for _, ep := range m.episodes {
		content := strings.ToLower(ep.Content)
		matchCount := 0
		for _, kw := range keywords {
			if strings.Contains(content, kw) {
				matchCount++
			}
		}

		if matchCount > 0 {
			similarity := float32(matchCount) / float32(len(keywords))
			ageDays := float32(now-ep.Timestamp) / (24 * 60 * 60 * 1000)
			score := m.calculateScore(similarity, ageDays, ep.Importance)
			scored = append(scored, scoredEpisode{
				episode:    ep,
				score:      score,
				similarity: similarity,
			})
		}
	}

	// 按分数排序
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	if limit > 0 && limit < len(scored) {
		scored = scored[:limit]
	}

	results := make([]*MemoryItem, len(scored))
	for i, se := range scored {
		results[i] = m.episodeToItem(se.episode, se.score)
	}

	return results
}

// calculateScore 计算综合得分
//
// 公式：(向量相似度 × 0.8 + 时间近因性 × 0.2) × (0.8 + 重要性 × 0.4)
func (m *EpisodicMemoryStore) calculateScore(similarity, ageDays, importance float32) float32 {
	// 时间近因性：1.0 / (1.0 + age_days)
	recency := 1.0 / (1.0 + ageDays)

	// 混合得分
	mixedScore := similarity*0.8 + recency*0.2

	// 重要性权重：范围 [0.8, 1.2]
	importanceWeight := 0.8 + importance*0.4

	return mixedScore * importanceWeight
}

// episodeToItem 将事件转换为 MemoryItem
func (m *EpisodicMemoryStore) episodeToItem(ep Episode, score float32) *MemoryItem {
	metadata := ep.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["type"] = ep.Type
	metadata["session_id"] = ep.SessionID
	metadata["outcome"] = ep.Outcome
	metadata["score"] = score

	return &MemoryItem{
		ID:         ep.ID,
		Content:    ep.Content,
		MemoryType: MemoryTypeEpisodic,
		UserID:     ep.UserID,
		Timestamp:  time.UnixMilli(ep.Timestamp),
		Importance: ep.Importance,
		Metadata:   metadata,
	}
}

// Update 更新记忆（实现 Memory 接口）
func (m *EpisodicMemoryStore) Update(ctx context.Context, id string, opts ...UpdateOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	options := &updateOptions{}
	for _, opt := range opts {
		opt(options)
	}

	for i := range m.episodes {
		if m.episodes[i].ID == id {
			if options.content != nil {
				m.episodes[i].Content = *options.content
			}
			if options.importance != nil {
				m.episodes[i].Importance = *options.importance
			}
			if options.metadata != nil {
				m.episodes[i].Metadata = options.metadata
			}
			m.rebuildTFIDF()
			return nil
		}
	}

	return ErrNotFound
}

// Remove 删除记忆（实现 Memory 接口）
func (m *EpisodicMemoryStore) Remove(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.episodes {
		if m.episodes[i].ID == id {
			// 从 sessions 索引中移除
			sessionID := m.episodes[i].SessionID
			if sessionID != "" {
				ids := m.sessions[sessionID]
				for j, eid := range ids {
					if eid == id {
						m.sessions[sessionID] = append(ids[:j], ids[j+1:]...)
						break
					}
				}
			}
			m.episodes = append(m.episodes[:i], m.episodes[i+1:]...)
			m.rebuildTFIDF()
			return nil
		}
	}

	return ErrNotFound
}

// Has 检查记忆是否存在（实现 Memory 接口）
func (m *EpisodicMemoryStore) Has(ctx context.Context, id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, ep := range m.episodes {
		if ep.ID == id {
			return true
		}
	}
	return false
}

// GetStats 获取统计信息（实现 Memory 接口）
func (m *EpisodicMemoryStore) GetStats(ctx context.Context) (*MemoryStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.episodes) == 0 {
		return &MemoryStats{Count: 0}, nil
	}

	var totalImportance float32
	var oldestTs, newestTs int64

	for i, ep := range m.episodes {
		totalImportance += ep.Importance
		if i == 0 || ep.Timestamp < oldestTs {
			oldestTs = ep.Timestamp
		}
		if i == 0 || ep.Timestamp > newestTs {
			newestTs = ep.Timestamp
		}
	}

	return &MemoryStats{
		Count:           len(m.episodes),
		OldestTimestamp: oldestTs,
		NewestTimestamp: newestTs,
		AvgImportance:   totalImportance / float32(len(m.episodes)),
	}, nil
}

// GetSessionEpisodes 获取指定会话的所有事件
func (m *EpisodicMemoryStore) GetSessionEpisodes(ctx context.Context, sessionID string) ([]Episode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids, ok := m.sessions[sessionID]
	if !ok {
		return nil, nil
	}

	result := make([]Episode, 0, len(ids))
	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}

	for _, ep := range m.episodes {
		if _, ok := idSet[ep.ID]; ok {
			result = append(result, ep)
		}
	}

	// 按时间戳升序排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp < result[j].Timestamp
	})

	return result, nil
}

// GetSessions 获取所有会话 ID
func (m *EpisodicMemoryStore) GetSessions(ctx context.Context) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		result = append(result, id)
	}
	return result
}

// Pattern 模式结构
type Pattern struct {
	// Keywords 关键词列表
	Keywords []string `json:"keywords"`
	// Frequency 出现频率
	Frequency int `json:"frequency"`
	// Examples 示例事件 ID
	Examples []string `json:"examples"`
}

// PatternOption 模式识别选项
type PatternOption func(*patternOptions)

type patternOptions struct {
	minFrequency int
	maxPatterns  int
}

// WithMinFrequency 设置最小频率
func WithMinFrequency(freq int) PatternOption {
	return func(o *patternOptions) {
		o.minFrequency = freq
	}
}

// WithMaxPatterns 设置最大模式数
func WithMaxPatterns(max int) PatternOption {
	return func(o *patternOptions) {
		o.maxPatterns = max
	}
}

// FindPatterns 识别模式
//
// 基于关键词频率分析，找出重复出现的模式。
func (m *EpisodicMemoryStore) FindPatterns(ctx context.Context, opts ...PatternOption) ([]Pattern, error) {
	options := &patternOptions{
		minFrequency: 2,
		maxPatterns:  10,
	}
	for _, opt := range opts {
		opt(options)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.episodes) == 0 {
		return nil, nil
	}

	// 统计关键词频率
	wordFreq := make(map[string]int)
	wordEpisodes := make(map[string][]string)

	for _, ep := range m.episodes {
		words := strings.Fields(strings.ToLower(ep.Content))
		seen := make(map[string]struct{})
		for _, word := range words {
			if len(word) < 2 {
				continue
			}
			if _, ok := seen[word]; !ok {
				wordFreq[word]++
				wordEpisodes[word] = append(wordEpisodes[word], ep.ID)
				seen[word] = struct{}{}
			}
		}
	}

	// 过滤并排序
	type wordCount struct {
		word  string
		count int
	}
	counts := make([]wordCount, 0)
	for word, count := range wordFreq {
		if count >= options.minFrequency {
			counts = append(counts, wordCount{word, count})
		}
	}

	sort.Slice(counts, func(i, j int) bool {
		return counts[i].count > counts[j].count
	})

	if options.maxPatterns > 0 && len(counts) > options.maxPatterns {
		counts = counts[:options.maxPatterns]
	}

	// 构建模式
	patterns := make([]Pattern, len(counts))
	for i, wc := range counts {
		examples := wordEpisodes[wc.word]
		if len(examples) > 3 {
			examples = examples[:3]
		}
		patterns[i] = Pattern{
			Keywords:  []string{wc.word},
			Frequency: wc.count,
			Examples:  examples,
		}
	}

	return patterns, nil
}

// TimelineEntry 时间线条目
type TimelineEntry struct {
	// Episode 事件
	Episode Episode `json:"episode"`
	// RelativeTime 相对时间描述
	RelativeTime string `json:"relative_time"`
}

// TimelineOption 时间线选项
type TimelineOption func(*timelineOptions)

type timelineOptions struct {
	startTime int64
	endTime   int64
	limit     int
}

// WithTimelineRange 设置时间范围
func WithTimelineRange(start, end int64) TimelineOption {
	return func(o *timelineOptions) {
		o.startTime = start
		o.endTime = end
	}
}

// WithTimelineLimit 设置返回数量限制
func WithTimelineLimit(limit int) TimelineOption {
	return func(o *timelineOptions) {
		o.limit = limit
	}
}

// GetTimeline 获取时间线视图
func (m *EpisodicMemoryStore) GetTimeline(ctx context.Context, opts ...TimelineOption) ([]TimelineEntry, error) {
	options := &timelineOptions{
		limit: 50,
	}
	for _, opt := range opts {
		opt(options)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// 过滤事件
	filtered := make([]Episode, 0, len(m.episodes))
	for _, ep := range m.episodes {
		if options.startTime > 0 && ep.Timestamp < options.startTime {
			continue
		}
		if options.endTime > 0 && ep.Timestamp > options.endTime {
			continue
		}
		filtered = append(filtered, ep)
	}

	// 按时间戳升序排序
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp < filtered[j].Timestamp
	})

	if options.limit > 0 && len(filtered) > options.limit {
		filtered = filtered[:options.limit]
	}

	// 构建时间线条目
	now := time.Now()
	entries := make([]TimelineEntry, len(filtered))
	for i, ep := range filtered {
		entries[i] = TimelineEntry{
			Episode:      ep,
			RelativeTime: formatRelativeTime(time.UnixMilli(ep.Timestamp), now),
		}
	}

	return entries, nil
}

// formatRelativeTime 格式化相对时间
func formatRelativeTime(t, now time.Time) string {
	diff := now.Sub(t)

	if diff < time.Minute {
		return "刚刚"
	}
	if diff < time.Hour {
		return strings.TrimSuffix(diff.Round(time.Minute).String(), "0s") + "前"
	}
	if diff < 24*time.Hour {
		return strings.TrimSuffix(diff.Round(time.Hour).String(), "0m0s") + "前"
	}
	days := int(diff.Hours() / 24)
	if days == 1 {
		return "昨天"
	}
	if days < 7 {
		return strings.TrimSuffix((time.Duration(days)*24*time.Hour).String(), "0s") + "前"
	}
	if days < 30 {
		weeks := days / 7
		return strings.TrimSuffix((time.Duration(weeks*7*24)*time.Hour).String(), "0s") + "前"
	}
	return t.Format("2006-01-02")
}

// Forget 执行遗忘（实现扩展）
func (m *EpisodicMemoryStore) Forget(ctx context.Context, strategy ForgetStrategy, opts ...ForgetOption) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	options := &forgetOptions{}
	for _, opt := range opts {
		opt(options)
	}

	originalCount := len(m.episodes)
	var remaining []Episode

	switch strategy {
	case ForgetByImportance:
		threshold := options.threshold
		if threshold <= 0 {
			threshold = 0.3
		}
		for _, ep := range m.episodes {
			if ep.Importance >= threshold {
				remaining = append(remaining, ep)
			}
		}

	case ForgetByTime:
		maxAgeDays := options.maxAgeDays
		if maxAgeDays <= 0 {
			maxAgeDays = 30
		}
		cutoff := time.Now().AddDate(0, 0, -maxAgeDays).UnixMilli()
		for _, ep := range m.episodes {
			if ep.Timestamp >= cutoff {
				remaining = append(remaining, ep)
			}
		}

	case ForgetByCapacity:
		targetCapacity := options.targetCapacity
		if targetCapacity <= 0 {
			targetCapacity = 1000
		}
		if len(m.episodes) <= targetCapacity {
			return 0, nil
		}
		// 按重要性排序，保留最重要的
		sorted := make([]Episode, len(m.episodes))
		copy(sorted, m.episodes)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Importance > sorted[j].Importance
		})
		remaining = sorted[:targetCapacity]
		// 按时间重新排序
		sort.Slice(remaining, func(i, j int) bool {
			return remaining[i].Timestamp < remaining[j].Timestamp
		})
	}

	// 重建 sessions 索引
	m.sessions = make(map[string][]string)
	for _, ep := range remaining {
		if ep.SessionID != "" {
			m.sessions[ep.SessionID] = append(m.sessions[ep.SessionID], ep.ID)
		}
	}

	m.episodes = remaining
	m.rebuildTFIDF()

	return originalCount - len(m.episodes), nil
}
