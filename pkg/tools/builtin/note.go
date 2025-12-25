// Package builtin 提供框架内置的常用工具
package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	agentctx "github.com/easyops/helloagents-go/pkg/context"
	"github.com/easyops/helloagents-go/pkg/tools"
)

// NoteType 定义笔记类型
type NoteType string

const (
	// NoteTypeTaskState 任务状态
	NoteTypeTaskState NoteType = "task_state"
	// NoteTypeConclusion 结论
	NoteTypeConclusion NoteType = "conclusion"
	// NoteTypeBlocker 阻塞项
	NoteTypeBlocker NoteType = "blocker"
	// NoteTypeAction 行动计划
	NoteTypeAction NoteType = "action"
	// NoteTypeReference 参考资料
	NoteTypeReference NoteType = "reference"
	// NoteTypeGeneral 通用笔记
	NoteTypeGeneral NoteType = "general"
)

// Note 表示一条笔记
type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Type      NoteType  `json:"type"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NoteIndexEntry 索引中的笔记条目
type NoteIndexEntry struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Type      NoteType  `json:"type"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
}

// NoteIndex 笔记索引
type NoteIndex struct {
	Notes    []NoteIndexEntry `json:"notes"`
	Metadata struct {
		CreatedAt  time.Time `json:"created_at"`
		TotalNotes int       `json:"total_notes"`
	} `json:"metadata"`
}

// NoteTool 笔记工具
//
// 为 Agent 提供结构化笔记管理能力，支持多种笔记类型：
//   - task_state: 任务状态
//   - conclusion: 关键结论
//   - blocker: 阻塞项
//   - action: 行动计划
//   - reference: 参考资料
//   - general: 通用笔记
//
// 用法示例：
//
//	noteTool := builtin.NewNoteTool(
//	    builtin.WithNoteWorkspace("./project_notes"),
//	)
//
//	// 创建笔记
//	result, _ := noteTool.Execute(ctx, map[string]interface{}{
//	    "action":    "create",
//	    "title":     "项目进展",
//	    "content":   "已完成需求分析",
//	    "note_type": "task_state",
//	    "tags":      []string{"milestone"},
//	})
type NoteTool struct {
	workspace string
	maxNotes  int
	index     *NoteIndex
	indexFile string
	mu        sync.RWMutex
	noteCount int
}

// NoteToolOption 配置 NoteTool
type NoteToolOption func(*NoteTool)

// WithNoteWorkspace 设置工作目录
func WithNoteWorkspace(workspace string) NoteToolOption {
	return func(n *NoteTool) {
		n.workspace = workspace
	}
}

// WithMaxNotes 设置最大笔记数量
func WithMaxNotes(max int) NoteToolOption {
	return func(n *NoteTool) {
		n.maxNotes = max
	}
}

// NewNoteTool 创建笔记工具
func NewNoteTool(opts ...NoteToolOption) (*NoteTool, error) {
	n := &NoteTool{
		workspace: "./notes",
		maxNotes:  1000,
	}

	for _, opt := range opts {
		opt(n)
	}

	// 确保工作目录存在
	if err := os.MkdirAll(n.workspace, 0755); err != nil {
		return nil, fmt.Errorf("创建笔记目录失败: %w", err)
	}

	n.indexFile = filepath.Join(n.workspace, "notes_index.json")

	// 加载索引
	if err := n.loadIndex(); err != nil {
		return nil, fmt.Errorf("加载索引失败: %w", err)
	}

	return n, nil
}

// Name 返回工具名称
func (n *NoteTool) Name() string {
	return "note"
}

// Description 返回工具描述
func (n *NoteTool) Description() string {
	return "笔记工具 - 创建、读取、更新、删除结构化笔记，支持任务状态、结论、阻塞项等类型。" +
		"操作类型: create(创建), read(读取), update(更新), delete(删除), list(列表), search(搜索), summary(摘要)"
}

// Parameters 返回参数 Schema
func (n *NoteTool) Parameters() tools.ParameterSchema {
	return tools.ParameterSchema{
		Type: "object",
		Properties: map[string]tools.PropertySchema{
			"action": {
				Type: "string",
				Description: "操作类型: create(创建), read(读取), update(更新), " +
					"delete(删除), list(列表), search(搜索), summary(摘要)",
				Enum: []string{"create", "read", "update", "delete", "list", "search", "summary"},
			},
			"title": {
				Type:        "string",
				Description: "笔记标题（create/update时使用）",
			},
			"content": {
				Type:        "string",
				Description: "笔记内容（create/update时使用）",
			},
			"note_type": {
				Type: "string",
				Description: "笔记类型: task_state(任务状态), conclusion(结论), " +
					"blocker(阻塞项), action(行动计划), reference(参考), general(通用)",
				Enum:    []string{"task_state", "conclusion", "blocker", "action", "reference", "general"},
				Default: "general",
			},
			"tags": {
				Type:        "array",
				Description: "标签列表（可选）",
				Items: &tools.PropertySchema{
					Type: "string",
				},
			},
			"note_id": {
				Type:        "string",
				Description: "笔记ID（read/update/delete时必需）",
			},
			"query": {
				Type:        "string",
				Description: "搜索关键词（search时必需）",
			},
			"limit": {
				Type:        "integer",
				Description: "返回结果数量限制（默认10）",
				Default:     10,
			},
		},
		Required: []string{"action"},
	}
}

// Execute 执行工具
func (n *NoteTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	action, ok := args["action"].(string)
	if !ok {
		return "", fmt.Errorf("缺少必需参数: action")
	}

	switch action {
	case "create":
		return n.createNote(args)
	case "read":
		return n.readNote(args)
	case "update":
		return n.updateNote(args)
	case "delete":
		return n.deleteNote(args)
	case "list":
		return n.listNotes(args)
	case "search":
		return n.searchNotes(args)
	case "summary":
		return n.getSummary()
	default:
		return "", fmt.Errorf("不支持的操作: %s", action)
	}
}

// Validate 验证参数
func (n *NoteTool) Validate(args map[string]interface{}) error {
	action, ok := args["action"].(string)
	if !ok {
		return fmt.Errorf("缺少必需参数: action")
	}

	switch action {
	case "create":
		if _, ok := args["title"].(string); !ok {
			return fmt.Errorf("create 操作需要 title 参数")
		}
		if _, ok := args["content"].(string); !ok {
			return fmt.Errorf("create 操作需要 content 参数")
		}
	case "read", "update", "delete":
		if _, ok := args["note_id"].(string); !ok {
			return fmt.Errorf("%s 操作需要 note_id 参数", action)
		}
	case "search":
		if _, ok := args["query"].(string); !ok {
			return fmt.Errorf("search 操作需要 query 参数")
		}
	}

	return nil
}

// loadIndex 加载笔记索引
func (n *NoteTool) loadIndex() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if _, err := os.Stat(n.indexFile); os.IsNotExist(err) {
		// 创建新索引
		n.index = &NoteIndex{}
		n.index.Metadata.CreatedAt = time.Now()
		n.index.Metadata.TotalNotes = 0
		return n.saveIndexLocked()
	}

	data, err := os.ReadFile(n.indexFile)
	if err != nil {
		return err
	}

	n.index = &NoteIndex{}
	if err := json.Unmarshal(data, n.index); err != nil {
		return err
	}

	n.noteCount = len(n.index.Notes)
	return nil
}

// saveIndexLocked 保存笔记索引（需要持有锁）
func (n *NoteTool) saveIndexLocked() error {
	n.index.Metadata.TotalNotes = len(n.index.Notes)
	data, err := json.MarshalIndent(n.index, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(n.indexFile, data, 0600)
}

// generateNoteID 生成笔记ID
func (n *NoteTool) generateNoteID() string {
	timestamp := time.Now().Format("20060102_150405")
	n.noteCount++
	return fmt.Sprintf("note_%s_%d", timestamp, n.noteCount)
}

// getNotePath 获取笔记文件路径
func (n *NoteTool) getNotePath(noteID string) string {
	return filepath.Join(n.workspace, noteID+".md")
}

// noteToMarkdown 将笔记转换为Markdown格式
func (n *NoteTool) noteToMarkdown(note *Note) string {
	var sb strings.Builder

	// YAML frontmatter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("id: %s\n", note.ID))
	sb.WriteString(fmt.Sprintf("title: %s\n", note.Title))
	sb.WriteString(fmt.Sprintf("type: %s\n", note.Type))

	if len(note.Tags) > 0 {
		tagsJSON, _ := json.Marshal(note.Tags)
		sb.WriteString(fmt.Sprintf("tags: %s\n", string(tagsJSON)))
	}

	sb.WriteString(fmt.Sprintf("created_at: %s\n", note.CreatedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("updated_at: %s\n", note.UpdatedAt.Format(time.RFC3339)))
	sb.WriteString("---\n\n")

	// Markdown content
	sb.WriteString(fmt.Sprintf("# %s\n\n", note.Title))
	sb.WriteString(note.Content)

	return sb.String()
}

// markdownToNote 将Markdown文本解析为笔记对象
func (n *NoteTool) markdownToNote(markdown string) (*Note, error) {
	// 提取YAML frontmatter
	re := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n`)
	matches := re.FindStringSubmatch(markdown)
	if matches == nil {
		return nil, fmt.Errorf("无效的笔记格式：缺少YAML前置元数据")
	}

	frontmatter := matches[1]
	contentStart := len(matches[0])

	note := &Note{}

	// 解析YAML（简化版）
	for _, line := range strings.Split(frontmatter, "\n") {
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])

			switch key {
			case "id":
				note.ID = value
			case "title":
				note.Title = value
			case "type":
				note.Type = NoteType(value)
			case "tags":
				var tags []string
				if err := json.Unmarshal([]byte(value), &tags); err == nil {
					note.Tags = tags
				}
			case "created_at":
				if t, err := time.Parse(time.RFC3339, value); err == nil {
					note.CreatedAt = t
				}
			case "updated_at":
				if t, err := time.Parse(time.RFC3339, value); err == nil {
					note.UpdatedAt = t
				}
			}
		}
	}

	// 提取内容（去掉标题行）
	content := strings.TrimSpace(markdown[contentStart:])
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "# ") {
		content = strings.TrimSpace(strings.Join(lines[1:], "\n"))
	}
	note.Content = content

	return note, nil
}

// createNote 创建笔记
func (n *NoteTool) createNote(args map[string]interface{}) (string, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// 检查笔记数量限制
	if len(n.index.Notes) >= n.maxNotes {
		return "", fmt.Errorf("笔记数量已达上限 (%d)", n.maxNotes)
	}

	title, _ := args["title"].(string)
	content, _ := args["content"].(string)

	if title == "" || content == "" {
		return "", fmt.Errorf("创建笔记需要提供 title 和 content")
	}

	noteType := NoteTypeGeneral
	if t, ok := args["note_type"].(string); ok && t != "" {
		noteType = NoteType(t)
	}

	var tags []string
	if t, ok := args["tags"].([]interface{}); ok {
		for _, tag := range t {
			if s, ok := tag.(string); ok {
				tags = append(tags, s)
			}
		}
	} else if t, ok := args["tags"].([]string); ok {
		tags = t
	}

	noteID := n.generateNoteID()
	now := time.Now()

	note := &Note{
		ID:        noteID,
		Title:     title,
		Content:   content,
		Type:      noteType,
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// 保存笔记文件
	notePath := n.getNotePath(noteID)
	markdown := n.noteToMarkdown(note)
	if err := os.WriteFile(notePath, []byte(markdown), 0600); err != nil {
		return "", fmt.Errorf("保存笔记失败: %w", err)
	}

	// 更新索引
	n.index.Notes = append(n.index.Notes, NoteIndexEntry{
		ID:        noteID,
		Title:     title,
		Type:      noteType,
		Tags:      tags,
		CreatedAt: now,
	})

	if err := n.saveIndexLocked(); err != nil {
		return "", fmt.Errorf("更新索引失败: %w", err)
	}

	return fmt.Sprintf("✅ 笔记创建成功\nID: %s\n标题: %s\n类型: %s", noteID, title, noteType), nil
}

// readNote 读取笔记
func (n *NoteTool) readNote(args map[string]interface{}) (string, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	noteID, ok := args["note_id"].(string)
	if !ok || noteID == "" {
		return "", fmt.Errorf("读取笔记需要提供 note_id")
	}

	notePath := n.getNotePath(noteID)
	data, err := os.ReadFile(notePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("笔记不存在: %s", noteID)
		}
		return "", fmt.Errorf("读取笔记失败: %w", err)
	}

	note, err := n.markdownToNote(string(data))
	if err != nil {
		return "", err
	}

	return n.formatNote(note, false), nil
}

// updateNote 更新笔记
func (n *NoteTool) updateNote(args map[string]interface{}) (string, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	noteID, ok := args["note_id"].(string)
	if !ok || noteID == "" {
		return "", fmt.Errorf("更新笔记需要提供 note_id")
	}

	notePath := n.getNotePath(noteID)
	data, err := os.ReadFile(notePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("笔记不存在: %s", noteID)
		}
		return "", fmt.Errorf("读取笔记失败: %w", err)
	}

	note, err := n.markdownToNote(string(data))
	if err != nil {
		return "", err
	}

	// 更新字段
	if title, ok := args["title"].(string); ok && title != "" {
		note.Title = title
	}
	if content, ok := args["content"].(string); ok && content != "" {
		note.Content = content
	}
	if noteType, ok := args["note_type"].(string); ok && noteType != "" {
		note.Type = NoteType(noteType)
	}
	if tags, ok := args["tags"].([]interface{}); ok {
		note.Tags = nil
		for _, tag := range tags {
			if s, ok := tag.(string); ok {
				note.Tags = append(note.Tags, s)
			}
		}
	} else if tags, ok := args["tags"].([]string); ok {
		note.Tags = tags
	}

	note.UpdatedAt = time.Now()

	// 保存更新
	markdown := n.noteToMarkdown(note)
	if err := os.WriteFile(notePath, []byte(markdown), 0600); err != nil {
		return "", fmt.Errorf("保存笔记失败: %w", err)
	}

	// 更新索引
	for i, entry := range n.index.Notes {
		if entry.ID == noteID {
			n.index.Notes[i].Title = note.Title
			n.index.Notes[i].Type = note.Type
			n.index.Notes[i].Tags = note.Tags
			break
		}
	}

	if err := n.saveIndexLocked(); err != nil {
		return "", fmt.Errorf("更新索引失败: %w", err)
	}

	return fmt.Sprintf("✅ 笔记更新成功: %s", noteID), nil
}

// deleteNote 删除笔记
func (n *NoteTool) deleteNote(args map[string]interface{}) (string, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	noteID, ok := args["note_id"].(string)
	if !ok || noteID == "" {
		return "", fmt.Errorf("删除笔记需要提供 note_id")
	}

	notePath := n.getNotePath(noteID)
	if _, err := os.Stat(notePath); os.IsNotExist(err) {
		return "", fmt.Errorf("笔记不存在: %s", noteID)
	}

	// 删除文件
	if err := os.Remove(notePath); err != nil {
		return "", fmt.Errorf("删除笔记失败: %w", err)
	}

	// 更新索引
	newNotes := make([]NoteIndexEntry, 0, len(n.index.Notes)-1)
	for _, entry := range n.index.Notes {
		if entry.ID != noteID {
			newNotes = append(newNotes, entry)
		}
	}
	n.index.Notes = newNotes

	if err := n.saveIndexLocked(); err != nil {
		return "", fmt.Errorf("更新索引失败: %w", err)
	}

	return fmt.Sprintf("✅ 笔记已删除: %s", noteID), nil
}

// listNotes 列出笔记
func (n *NoteTool) listNotes(args map[string]interface{}) (string, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var noteType string
	if t, ok := args["note_type"].(string); ok {
		noteType = t
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	} else if l, ok := args["limit"].(int); ok {
		limit = l
	}

	// 过滤笔记
	var filtered []NoteIndexEntry
	for _, entry := range n.index.Notes {
		if noteType == "" || string(entry.Type) == noteType {
			filtered = append(filtered, entry)
		}
	}

	// 限制数量
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	if len(filtered) == 0 {
		return "📝 暂无笔记", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📝 笔记列表（共 %d 条）\n\n", len(filtered)))

	for _, entry := range filtered {
		sb.WriteString(fmt.Sprintf("• [%s] %s\n", entry.Type, entry.Title))
		sb.WriteString(fmt.Sprintf("  ID: %s\n", entry.ID))
		if len(entry.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("  标签: %s\n", strings.Join(entry.Tags, ", ")))
		}
		sb.WriteString(fmt.Sprintf("  创建时间: %s\n\n", entry.CreatedAt.Format("2006-01-02 15:04:05")))
	}

	return sb.String(), nil
}

// searchNotes 搜索笔记
func (n *NoteTool) searchNotes(args map[string]interface{}) (string, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("搜索需要提供 query")
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	} else if l, ok := args["limit"].(int); ok {
		limit = l
	}

	queryLower := strings.ToLower(query)
	var matched []*Note

	for _, entry := range n.index.Notes {
		notePath := n.getNotePath(entry.ID)
		data, err := os.ReadFile(notePath)
		if err != nil {
			continue
		}

		note, err := n.markdownToNote(string(data))
		if err != nil {
			continue
		}

		// 检查标题、内容、标签是否匹配
		titleMatch := strings.Contains(strings.ToLower(note.Title), queryLower)
		contentMatch := strings.Contains(strings.ToLower(note.Content), queryLower)

		var tagMatch bool
		for _, tag := range note.Tags {
			if strings.Contains(strings.ToLower(tag), queryLower) {
				tagMatch = true
				break
			}
		}

		if titleMatch || contentMatch || tagMatch {
			matched = append(matched, note)
		}
	}

	// 限制数量
	if len(matched) > limit {
		matched = matched[:limit]
	}

	if len(matched) == 0 {
		return fmt.Sprintf("📝 未找到匹配 '%s' 的笔记", query), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 搜索结果（共 %d 条）\n\n", len(matched)))

	for _, note := range matched {
		sb.WriteString(n.formatNote(note, true))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// getSummary 获取笔记摘要
func (n *NoteTool) getSummary() (string, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	total := len(n.index.Notes)

	// 按类型统计
	typeCounts := make(map[NoteType]int)
	for _, entry := range n.index.Notes {
		typeCounts[entry.Type]++
	}

	var sb strings.Builder
	sb.WriteString("📊 笔记摘要\n\n")
	sb.WriteString(fmt.Sprintf("总笔记数: %d\n\n", total))
	sb.WriteString("按类型统计:\n")

	typeOrder := []NoteType{
		NoteTypeTaskState, NoteTypeConclusion, NoteTypeBlocker,
		NoteTypeAction, NoteTypeReference, NoteTypeGeneral,
	}
	for _, t := range typeOrder {
		if count, ok := typeCounts[t]; ok {
			sb.WriteString(fmt.Sprintf("  • %s: %d\n", t, count))
		}
	}

	return sb.String(), nil
}

// formatNote 格式化笔记输出
func (n *NoteTool) formatNote(note *Note, compact bool) string {
	if compact {
		content := note.Content
		if len(content) > 100 {
			content = content[:100] + "..."
		}
		return fmt.Sprintf("[%s] %s\nID: %s\n内容: %s", note.Type, note.Title, note.ID, content)
	}

	var sb strings.Builder
	sb.WriteString("📝 笔记详情\n\n")
	sb.WriteString(fmt.Sprintf("ID: %s\n", note.ID))
	sb.WriteString(fmt.Sprintf("标题: %s\n", note.Title))
	sb.WriteString(fmt.Sprintf("类型: %s\n", note.Type))
	if len(note.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("标签: %s\n", strings.Join(note.Tags, ", ")))
	}
	sb.WriteString(fmt.Sprintf("创建时间: %s\n", note.CreatedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("更新时间: %s\n", note.UpdatedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("\n内容:\n%s\n", note.Content))

	return sb.String()
}

// ListNotes 列出笔记（实现 context.NoteRetriever 接口）
func (n *NoteTool) ListNotes(noteType string, limit int) ([]agentctx.NoteResult, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	capacity := limit
	if capacity <= 0 || capacity > len(n.index.Notes) {
		capacity = len(n.index.Notes)
	}
	results := make([]agentctx.NoteResult, 0, capacity)
	count := 0

	for _, entry := range n.index.Notes {
		if noteType != "" && string(entry.Type) != noteType {
			continue
		}

		notePath := n.getNotePath(entry.ID)
		data, err := os.ReadFile(notePath)
		if err != nil {
			continue
		}

		note, err := n.markdownToNote(string(data))
		if err != nil {
			continue
		}

		results = append(results, agentctx.NoteResult{
			ID:        note.ID,
			Title:     note.Title,
			Content:   note.Content,
			Type:      string(note.Type),
			Tags:      note.Tags,
			UpdatedAt: note.UpdatedAt,
		})

		count++
		if limit > 0 && count >= limit {
			break
		}
	}

	return results, nil
}

// SearchNotes 搜索笔记（实现 context.NoteRetriever 接口）
func (n *NoteTool) SearchNotes(query string, limit int) ([]agentctx.NoteResult, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if query == "" {
		return nil, nil
	}

	queryLower := strings.ToLower(query)
	var results []agentctx.NoteResult

	for _, entry := range n.index.Notes {
		notePath := n.getNotePath(entry.ID)
		data, err := os.ReadFile(notePath)
		if err != nil {
			continue
		}

		note, err := n.markdownToNote(string(data))
		if err != nil {
			continue
		}

		// 检查标题、内容、标签是否匹配
		titleMatch := strings.Contains(strings.ToLower(note.Title), queryLower)
		contentMatch := strings.Contains(strings.ToLower(note.Content), queryLower)

		var tagMatch bool
		for _, tag := range note.Tags {
			if strings.Contains(strings.ToLower(tag), queryLower) {
				tagMatch = true
				break
			}
		}

		if titleMatch || contentMatch || tagMatch {
			results = append(results, agentctx.NoteResult{
				ID:        note.ID,
				Title:     note.Title,
				Content:   note.Content,
				Type:      string(note.Type),
				Tags:      note.Tags,
				UpdatedAt: note.UpdatedAt,
			})
		}

		if limit > 0 && len(results) >= limit {
			break
		}
	}

	return results, nil
}

// 编译时接口检查
var _ tools.Tool = (*NoteTool)(nil)
var _ tools.ToolWithValidation = (*NoteTool)(nil)
var _ agentctx.NoteRetriever = (*NoteTool)(nil)
