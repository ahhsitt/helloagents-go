package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EpisodicMemoryStore 情景记忆存储实现
type EpisodicMemoryStore struct {
	episodes []Episode
	mu       sync.RWMutex
}

// NewEpisodicMemory 创建情景记忆存储
func NewEpisodicMemory() *EpisodicMemoryStore {
	return &EpisodicMemoryStore{
		episodes: make([]Episode, 0),
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
	return nil
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
