package tools_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/easyops/helloagents-go/pkg/tools/builtin"
)

func setupNoteTool(t *testing.T) (*builtin.NoteTool, string) {
	t.Helper()
	tmpDir := t.TempDir()
	tool, err := builtin.NewNoteTool(builtin.WithNoteWorkspace(tmpDir))
	if err != nil {
		t.Fatalf("创建 NoteTool 失败: %v", err)
	}
	return tool, tmpDir
}

func TestNewNoteTool(t *testing.T) {
	tool, tmpDir := setupNoteTool(t)

	if tool.Name() != "note" {
		t.Errorf("expected name 'note', got %s", tool.Name())
	}

	if !strings.Contains(tool.Description(), "笔记工具") {
		t.Errorf("expected description to contain '笔记工具', got %s", tool.Description())
	}

	// 验证索引文件已创建
	indexPath := filepath.Join(tmpDir, "notes_index.json")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("索引文件应该已创建")
	}
}

func TestNoteTool_Parameters(t *testing.T) {
	tool, _ := setupNoteTool(t)
	params := tool.Parameters()

	if params.Type != "object" {
		t.Errorf("expected type 'object', got %s", params.Type)
	}

	requiredParams := params.Required
	if len(requiredParams) != 1 || requiredParams[0] != "action" {
		t.Errorf("expected required param 'action', got %v", requiredParams)
	}

	expectedProps := []string{"action", "title", "content", "note_type", "tags", "note_id", "query", "limit"}
	for _, prop := range expectedProps {
		if _, ok := params.Properties[prop]; !ok {
			t.Errorf("缺少参数属性: %s", prop)
		}
	}
}

func TestNoteTool_Create(t *testing.T) {
	tool, _ := setupNoteTool(t)
	ctx := context.Background()

	// 创建笔记
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action":    "create",
		"title":     "测试笔记",
		"content":   "这是测试内容",
		"note_type": "task_state",
		"tags":      []interface{}{"test", "unit"},
	})

	if err != nil {
		t.Fatalf("创建笔记失败: %v", err)
	}

	if !strings.Contains(result, "✅ 笔记创建成功") {
		t.Errorf("期望成功消息，得到: %s", result)
	}
	if !strings.Contains(result, "测试笔记") {
		t.Errorf("期望包含标题，得到: %s", result)
	}
	if !strings.Contains(result, "task_state") {
		t.Errorf("期望包含类型，得到: %s", result)
	}
}

func TestNoteTool_CreateWithDefaultType(t *testing.T) {
	tool, _ := setupNoteTool(t)
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"title":   "通用笔记",
		"content": "默认类型测试",
	})

	if err != nil {
		t.Fatalf("创建笔记失败: %v", err)
	}

	// 验证默认类型为 general
	if !strings.Contains(result, "general") {
		t.Errorf("期望默认类型 'general'，得到: %s", result)
	}
}

func TestNoteTool_CreateMissingFields(t *testing.T) {
	tool, _ := setupNoteTool(t)
	ctx := context.Background()

	// 缺少 title
	_, err := tool.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"content": "只有内容",
	})
	if err == nil {
		t.Error("缺少 title 应该返回错误")
	}

	// 缺少 content
	_, err = tool.Execute(ctx, map[string]interface{}{
		"action": "create",
		"title":  "只有标题",
	})
	if err == nil {
		t.Error("缺少 content 应该返回错误")
	}
}

func TestNoteTool_Read(t *testing.T) {
	tool, _ := setupNoteTool(t)
	ctx := context.Background()

	// 先创建笔记
	createResult, _ := tool.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"title":   "读取测试",
		"content": "这是要读取的内容",
	})

	// 从结果中提取 ID
	noteID := extractNoteID(createResult)

	// 读取笔记
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action":  "read",
		"note_id": noteID,
	})

	if err != nil {
		t.Fatalf("读取笔记失败: %v", err)
	}

	if !strings.Contains(result, "读取测试") {
		t.Errorf("期望包含标题，得到: %s", result)
	}
	if !strings.Contains(result, "这是要读取的内容") {
		t.Errorf("期望包含内容，得到: %s", result)
	}
}

func TestNoteTool_ReadNotFound(t *testing.T) {
	tool, _ := setupNoteTool(t)
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]interface{}{
		"action":  "read",
		"note_id": "nonexistent_id",
	})

	if err == nil {
		t.Error("读取不存在的笔记应该返回错误")
	}
	if !strings.Contains(err.Error(), "笔记不存在") {
		t.Errorf("错误消息应该包含 '笔记不存在'，得到: %v", err)
	}
}

func TestNoteTool_Update(t *testing.T) {
	tool, _ := setupNoteTool(t)
	ctx := context.Background()

	// 创建笔记
	createResult, _ := tool.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"title":   "原始标题",
		"content": "原始内容",
	})
	noteID := extractNoteID(createResult)

	// 更新笔记
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action":  "update",
		"note_id": noteID,
		"title":   "更新后标题",
		"content": "更新后内容",
	})

	if err != nil {
		t.Fatalf("更新笔记失败: %v", err)
	}
	if !strings.Contains(result, "✅ 笔记更新成功") {
		t.Errorf("期望成功消息，得到: %s", result)
	}

	// 验证更新
	readResult, _ := tool.Execute(ctx, map[string]interface{}{
		"action":  "read",
		"note_id": noteID,
	})
	if !strings.Contains(readResult, "更新后标题") {
		t.Errorf("期望更新后的标题，得到: %s", readResult)
	}
}

func TestNoteTool_Delete(t *testing.T) {
	tool, _ := setupNoteTool(t)
	ctx := context.Background()

	// 创建笔记
	createResult, _ := tool.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"title":   "要删除的笔记",
		"content": "内容",
	})
	noteID := extractNoteID(createResult)

	// 删除笔记
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action":  "delete",
		"note_id": noteID,
	})

	if err != nil {
		t.Fatalf("删除笔记失败: %v", err)
	}
	if !strings.Contains(result, "✅ 笔记已删除") {
		t.Errorf("期望成功消息，得到: %s", result)
	}

	// 验证已删除
	_, err = tool.Execute(ctx, map[string]interface{}{
		"action":  "read",
		"note_id": noteID,
	})
	if err == nil {
		t.Error("读取已删除的笔记应该返回错误")
	}
}

func TestNoteTool_List(t *testing.T) {
	tool, _ := setupNoteTool(t)
	ctx := context.Background()

	// 创建多条笔记
	for i := 0; i < 3; i++ {
		_, _ = tool.Execute(ctx, map[string]interface{}{
			"action":    "create",
			"title":     "列表测试笔记",
			"content":   "内容",
			"note_type": "task_state",
		})
	}

	// 列出所有笔记
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action": "list",
	})

	if err != nil {
		t.Fatalf("列出笔记失败: %v", err)
	}
	if !strings.Contains(result, "共 3 条") {
		t.Errorf("期望包含 3 条笔记，得到: %s", result)
	}
}

func TestNoteTool_ListByType(t *testing.T) {
	tool, _ := setupNoteTool(t)
	ctx := context.Background()

	// 创建不同类型的笔记
	_, _ = tool.Execute(ctx, map[string]interface{}{
		"action":    "create",
		"title":     "任务状态",
		"content":   "内容",
		"note_type": "task_state",
	})
	_, _ = tool.Execute(ctx, map[string]interface{}{
		"action":    "create",
		"title":     "阻塞项",
		"content":   "内容",
		"note_type": "blocker",
	})

	// 按类型过滤
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action":    "list",
		"note_type": "blocker",
	})

	if err != nil {
		t.Fatalf("列出笔记失败: %v", err)
	}
	if !strings.Contains(result, "共 1 条") {
		t.Errorf("期望只有 1 条 blocker 类型笔记，得到: %s", result)
	}
	if !strings.Contains(result, "阻塞项") {
		t.Errorf("期望包含阻塞项笔记，得到: %s", result)
	}
}

func TestNoteTool_ListWithLimit(t *testing.T) {
	tool, _ := setupNoteTool(t)
	ctx := context.Background()

	// 创建 5 条笔记
	for i := 0; i < 5; i++ {
		_, _ = tool.Execute(ctx, map[string]interface{}{
			"action":  "create",
			"title":   "限制测试",
			"content": "内容",
		})
	}

	// 限制返回 2 条
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action": "list",
		"limit":  2,
	})

	if err != nil {
		t.Fatalf("列出笔记失败: %v", err)
	}
	if !strings.Contains(result, "共 2 条") {
		t.Errorf("期望限制为 2 条，得到: %s", result)
	}
}

func TestNoteTool_Search(t *testing.T) {
	tool, _ := setupNoteTool(t)
	ctx := context.Background()

	// 创建笔记
	_, _ = tool.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"title":   "搜索目标",
		"content": "这里包含关键词 milestone",
		"tags":    []interface{}{"important"},
	})
	_, _ = tool.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"title":   "其他笔记",
		"content": "无关内容",
	})

	// 搜索
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action": "search",
		"query":  "milestone",
	})

	if err != nil {
		t.Fatalf("搜索失败: %v", err)
	}
	if !strings.Contains(result, "共 1 条") {
		t.Errorf("期望找到 1 条匹配，得到: %s", result)
	}
	if !strings.Contains(result, "搜索目标") {
		t.Errorf("期望包含搜索目标，得到: %s", result)
	}
}

func TestNoteTool_SearchNoResults(t *testing.T) {
	tool, _ := setupNoteTool(t)
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"action": "search",
		"query":  "不存在的关键词",
	})

	if err != nil {
		t.Fatalf("搜索失败: %v", err)
	}
	if !strings.Contains(result, "未找到匹配") {
		t.Errorf("期望未找到消息，得到: %s", result)
	}
}

func TestNoteTool_Summary(t *testing.T) {
	tool, _ := setupNoteTool(t)
	ctx := context.Background()

	// 创建不同类型的笔记
	_, _ = tool.Execute(ctx, map[string]interface{}{
		"action":    "create",
		"title":     "任务1",
		"content":   "内容",
		"note_type": "task_state",
	})
	_, _ = tool.Execute(ctx, map[string]interface{}{
		"action":    "create",
		"title":     "任务2",
		"content":   "内容",
		"note_type": "task_state",
	})
	_, _ = tool.Execute(ctx, map[string]interface{}{
		"action":    "create",
		"title":     "阻塞",
		"content":   "内容",
		"note_type": "blocker",
	})

	result, err := tool.Execute(ctx, map[string]interface{}{
		"action": "summary",
	})

	if err != nil {
		t.Fatalf("获取摘要失败: %v", err)
	}
	if !strings.Contains(result, "总笔记数: 3") {
		t.Errorf("期望总数为 3，得到: %s", result)
	}
	if !strings.Contains(result, "task_state: 2") {
		t.Errorf("期望 task_state 为 2，得到: %s", result)
	}
	if !strings.Contains(result, "blocker: 1") {
		t.Errorf("期望 blocker 为 1，得到: %s", result)
	}
}

func TestNoteTool_Validate(t *testing.T) {
	tool, _ := setupNoteTool(t)

	// 缺少 action
	err := tool.Validate(map[string]interface{}{})
	if err == nil {
		t.Error("缺少 action 应该返回错误")
	}

	// create 缺少 title
	err = tool.Validate(map[string]interface{}{
		"action":  "create",
		"content": "内容",
	})
	if err == nil {
		t.Error("create 缺少 title 应该返回错误")
	}

	// read 缺少 note_id
	err = tool.Validate(map[string]interface{}{
		"action": "read",
	})
	if err == nil {
		t.Error("read 缺少 note_id 应该返回错误")
	}

	// search 缺少 query
	err = tool.Validate(map[string]interface{}{
		"action": "search",
	})
	if err == nil {
		t.Error("search 缺少 query 应该返回错误")
	}
}

func TestNoteTool_ConcurrentAccess(t *testing.T) {
	tool, _ := setupNoteTool(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	errCh := make(chan error, 20)

	// 并发创建笔记
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := tool.Execute(ctx, map[string]interface{}{
				"action":  "create",
				"title":   "并发测试",
				"content": "内容",
			})
			if err != nil {
				errCh <- err
			}
		}(i)
	}

	// 并发读取
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := tool.Execute(ctx, map[string]interface{}{
				"action": "list",
			})
			if err != nil {
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("并发操作错误: %v", err)
	}
}

func TestNoteTool_UnsupportedAction(t *testing.T) {
	tool, _ := setupNoteTool(t)
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]interface{}{
		"action": "invalid_action",
	})

	if err == nil {
		t.Error("不支持的操作应该返回错误")
	}
	if !strings.Contains(err.Error(), "不支持的操作") {
		t.Errorf("错误消息应该包含 '不支持的操作'，得到: %v", err)
	}
}

// extractNoteID 从创建结果中提取笔记 ID
func extractNoteID(result string) string {
	// 格式: "ID: note_20250101_120000_1"
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ID: ") {
			return strings.TrimPrefix(line, "ID: ")
		}
	}
	return ""
}
