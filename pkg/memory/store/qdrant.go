package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// QdrantVectorStore Qdrant 向量存储
//
// 基于 Qdrant REST API 的向量存储实现。
type QdrantVectorStore struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	dimensions int
}

// QdrantConfig Qdrant 配置
type QdrantConfig struct {
	URL        string
	APIKey     string
	Dimensions int
	Timeout    time.Duration
}

// NewQdrantVectorStore 创建 Qdrant 向量存储
func NewQdrantVectorStore(config QdrantConfig) (*QdrantVectorStore, error) {
	if config.URL == "" {
		config.URL = "http://localhost:6333"
	}
	if config.Dimensions <= 0 {
		config.Dimensions = 128
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}

	store := &QdrantVectorStore{
		baseURL:    config.URL,
		apiKey:     config.APIKey,
		dimensions: config.Dimensions,
		httpClient: &http.Client{Timeout: config.Timeout},
	}

	return store, nil
}

// ensureCollection 确保集合存在
func (s *QdrantVectorStore) ensureCollection(ctx context.Context, collection string) error {
	// 检查集合是否存在
	req, err := s.newRequest(ctx, "GET", fmt.Sprintf("/collections/%s", collection), nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check collection: %w", err)
	}
	defer resp.Body.Close()

	// 集合存在
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	// 创建集合
	createBody := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     s.dimensions,
			"distance": "Cosine",
		},
	}

	req, err = s.newRequest(ctx, "PUT", fmt.Sprintf("/collections/%s", collection), createBody)
	if err != nil {
		return err
	}

	resp, err = s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection: %s", string(body))
	}

	return nil
}

// AddVectors 批量添加向量
func (s *QdrantVectorStore) AddVectors(ctx context.Context, collection string, vectors []VectorRecord) error {
	if err := s.ensureCollection(ctx, collection); err != nil {
		return err
	}

	// 构建 upsert 请求
	points := make([]map[string]interface{}, len(vectors))
	for i, v := range vectors {
		points[i] = map[string]interface{}{
			"id":      v.ID,
			"vector":  v.Vector,
			"payload": s.buildPayload(v),
		}
	}

	body := map[string]interface{}{
		"points": points,
	}

	req, err := s.newRequest(ctx, "PUT", fmt.Sprintf("/collections/%s/points", collection), body)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upsert vectors: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upsert vectors: %s", string(respBody))
	}

	return nil
}

// buildPayload 构建 payload
func (s *QdrantVectorStore) buildPayload(v VectorRecord) map[string]interface{} {
	payload := make(map[string]interface{})
	if v.MemoryID != "" {
		payload["memory_id"] = v.MemoryID
	}
	for k, val := range v.Payload {
		payload[k] = val
	}
	return payload
}

// SearchSimilar 相似度搜索
func (s *QdrantVectorStore) SearchSimilar(ctx context.Context, collection string, vector []float32, topK int, filter *VectorFilter) ([]VectorSearchResult, error) {
	body := map[string]interface{}{
		"vector":      vector,
		"limit":       topK,
		"with_payload": true,
	}

	if filter != nil {
		qdrantFilter := s.buildQdrantFilter(filter)
		if len(qdrantFilter) > 0 {
			body["filter"] = qdrantFilter
		}
	}

	req, err := s.newRequest(ctx, "POST", fmt.Sprintf("/collections/%s/points/search", collection), body)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed: %s", string(respBody))
	}

	var result struct {
		Result []struct {
			ID      string                 `json:"id"`
			Score   float32                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]VectorSearchResult, len(result.Result))
	for i, r := range result.Result {
		memoryID := ""
		if mid, ok := r.Payload["memory_id"].(string); ok {
			memoryID = mid
		}
		results[i] = VectorSearchResult{
			ID:       r.ID,
			Score:    r.Score,
			Payload:  r.Payload,
			MemoryID: memoryID,
		}
	}

	return results, nil
}

// buildQdrantFilter 构建 Qdrant 过滤器
func (s *QdrantVectorStore) buildQdrantFilter(filter *VectorFilter) map[string]interface{} {
	var mustConditions []map[string]interface{}

	if filter.MemoryID != "" {
		mustConditions = append(mustConditions, map[string]interface{}{
			"key":   "memory_id",
			"match": map[string]interface{}{"value": filter.MemoryID},
		})
	}

	if filter.UserID != "" {
		mustConditions = append(mustConditions, map[string]interface{}{
			"key":   "user_id",
			"match": map[string]interface{}{"value": filter.UserID},
		})
	}

	if filter.MemoryType != "" {
		mustConditions = append(mustConditions, map[string]interface{}{
			"key":   "memory_type",
			"match": map[string]interface{}{"value": filter.MemoryType},
		})
	}

	for k, v := range filter.Conditions {
		mustConditions = append(mustConditions, map[string]interface{}{
			"key":   k,
			"match": map[string]interface{}{"value": v},
		})
	}

	if len(mustConditions) == 0 {
		return nil
	}

	return map[string]interface{}{
		"must": mustConditions,
	}
}

// DeleteVectors 按 ID 删除向量
func (s *QdrantVectorStore) DeleteVectors(ctx context.Context, collection string, ids []string) error {
	body := map[string]interface{}{
		"points": ids,
	}

	req, err := s.newRequest(ctx, "POST", fmt.Sprintf("/collections/%s/points/delete", collection), body)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete vectors: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed: %s", string(respBody))
	}

	return nil
}

// DeleteByFilter 按条件删除
func (s *QdrantVectorStore) DeleteByFilter(ctx context.Context, collection string, filter *VectorFilter) error {
	qdrantFilter := s.buildQdrantFilter(filter)
	if qdrantFilter == nil {
		return nil
	}

	body := map[string]interface{}{
		"filter": qdrantFilter,
	}

	req, err := s.newRequest(ctx, "POST", fmt.Sprintf("/collections/%s/points/delete", collection), body)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete by filter: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete by filter failed: %s", string(respBody))
	}

	return nil
}

// Clear 清空集合
func (s *QdrantVectorStore) Clear(ctx context.Context, collection string) error {
	// 删除并重建集合
	req, err := s.newRequest(ctx, "DELETE", fmt.Sprintf("/collections/%s", collection), nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to clear collection: %w", err)
	}
	defer resp.Body.Close()

	// 忽略 404 错误（集合不存在）
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("clear failed: %s", string(respBody))
	}

	return nil
}

// GetStats 获取统计信息
func (s *QdrantVectorStore) GetStats(ctx context.Context, collection string) (*VectorStoreStats, error) {
	req, err := s.newRequest(ctx, "GET", fmt.Sprintf("/collections/%s", collection), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return &VectorStoreStats{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get stats failed: %s", string(respBody))
	}

	var result struct {
		Result struct {
			VectorsCount int `json:"vectors_count"`
			Config       struct {
				Params struct {
					Vectors struct {
						Size int `json:"size"`
					} `json:"vectors"`
				} `json:"params"`
			} `json:"config"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &VectorStoreStats{
		VectorCount:  result.Result.VectorsCount,
		Dimensions:   result.Result.Config.Params.Vectors.Size,
		IndexedCount: result.Result.VectorsCount,
	}, nil
}

// HealthCheck 健康检查
func (s *QdrantVectorStore) HealthCheck(ctx context.Context) error {
	req, err := s.newRequest(ctx, "GET", "/", nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qdrant not healthy: status %d", resp.StatusCode)
	}

	return nil
}

// Close 关闭连接
func (s *QdrantVectorStore) Close() error {
	s.httpClient.CloseIdleConnections()
	return nil
}

// newRequest 创建 HTTP 请求
func (s *QdrantVectorStore) newRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("api-key", s.apiKey)
	}

	return req, nil
}

// Compile-time interface check
var _ VectorStore = (*QdrantVectorStore)(nil)
