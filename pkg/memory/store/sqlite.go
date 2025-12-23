package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteDocumentStore SQLite 文档存储
//
// 基于 SQLite 的持久化文档存储，适用于生产环境。
type SQLiteDocumentStore struct {
	db *sql.DB
}

// NewSQLiteDocumentStore 创建 SQLite 文档存储
func NewSQLiteDocumentStore(dbPath string) (*SQLiteDocumentStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &SQLiteDocumentStore{db: db}

	// 初始化表结构
	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	return store, nil
}

// initSchema 初始化表结构
func (s *SQLiteDocumentStore) initSchema() error {
	// 创建文档表（通用表，使用 collection 字段区分）
	query := `
	CREATE TABLE IF NOT EXISTS documents (
		id TEXT NOT NULL,
		collection TEXT NOT NULL,
		content TEXT,
		metadata TEXT,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		PRIMARY KEY (collection, id)
	);
	CREATE INDEX IF NOT EXISTS idx_documents_collection ON documents(collection);
	CREATE INDEX IF NOT EXISTS idx_documents_created_at ON documents(collection, created_at);
	`

	_, err := s.db.Exec(query)
	return err
}

// Put 存储文档
func (s *SQLiteDocumentStore) Put(ctx context.Context, collection string, id string, doc Document) error {
	metadata, err := json.Marshal(doc.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	now := time.Now().UnixMilli()
	createdAt := now
	if !doc.CreatedAt.IsZero() {
		createdAt = doc.CreatedAt.UnixMilli()
	}

	query := `
	INSERT INTO documents (id, collection, content, metadata, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?)
	ON CONFLICT(collection, id) DO UPDATE SET
		content = excluded.content,
		metadata = excluded.metadata,
		updated_at = excluded.updated_at
	`

	_, err = s.db.ExecContext(ctx, query, id, collection, doc.Content, string(metadata), createdAt, now)
	return err
}

// Get 获取文档
func (s *SQLiteDocumentStore) Get(ctx context.Context, collection string, id string) (*Document, error) {
	query := `SELECT id, content, metadata, created_at, updated_at FROM documents WHERE collection = ? AND id = ?`

	var doc Document
	var metadataStr string
	var createdAt, updatedAt int64

	err := s.db.QueryRowContext(ctx, query, collection, id).Scan(
		&doc.ID, &doc.Content, &metadataStr, &createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &doc.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	doc.CreatedAt = time.UnixMilli(createdAt)
	doc.UpdatedAt = time.UnixMilli(updatedAt)

	return &doc, nil
}

// Delete 删除文档
func (s *SQLiteDocumentStore) Delete(ctx context.Context, collection string, id string) error {
	query := `DELETE FROM documents WHERE collection = ? AND id = ?`
	result, err := s.db.ExecContext(ctx, query, collection, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// Query 条件查询
func (s *SQLiteDocumentStore) Query(ctx context.Context, collection string, filter Filter, opts ...QueryOption) ([]Document, error) {
	options := &queryOptions{limit: 100}
	for _, opt := range opts {
		opt(options)
	}

	whereClause, args := s.buildWhereClause(filter)
	args = append([]interface{}{collection}, args...)

	query := fmt.Sprintf(
		"SELECT id, content, metadata, created_at, updated_at FROM documents WHERE collection = ? %s",
		whereClause,
	)

	// 排序
	if options.orderBy != "" {
		order := "ASC"
		if options.desc {
			order = "DESC"
		}
		query += fmt.Sprintf(" ORDER BY %s %s", s.sanitizeColumn(options.orderBy), order)
	}

	// 分页
	if options.limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", options.limit)
	}
	if options.offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", options.offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Document
	for rows.Next() {
		var doc Document
		var metadataStr string
		var createdAt, updatedAt int64

		if err := rows.Scan(&doc.ID, &doc.Content, &metadataStr, &createdAt, &updatedAt); err != nil {
			return nil, err
		}

		if metadataStr != "" {
			if err := json.Unmarshal([]byte(metadataStr), &doc.Metadata); err != nil {
				continue // 跳过无效记录
			}
		}

		doc.CreatedAt = time.UnixMilli(createdAt)
		doc.UpdatedAt = time.UnixMilli(updatedAt)
		results = append(results, doc)
	}

	return results, rows.Err()
}

// Count 统计数量
func (s *SQLiteDocumentStore) Count(ctx context.Context, collection string, filter Filter) (int, error) {
	whereClause, args := s.buildWhereClause(filter)
	args = append([]interface{}{collection}, args...)

	query := fmt.Sprintf("SELECT COUNT(*) FROM documents WHERE collection = ? %s", whereClause)

	var count int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

// Clear 清空集合
func (s *SQLiteDocumentStore) Clear(ctx context.Context, collection string) error {
	query := `DELETE FROM documents WHERE collection = ?`
	_, err := s.db.ExecContext(ctx, query, collection)
	return err
}

// Close 关闭连接
func (s *SQLiteDocumentStore) Close() error {
	return s.db.Close()
}

// buildWhereClause 构建 WHERE 子句
func (s *SQLiteDocumentStore) buildWhereClause(filter Filter) (string, []interface{}) {
	if filter.Field == "" && len(filter.And) == 0 && len(filter.Or) == 0 {
		return "", nil
	}

	var conditions []string
	var args []interface{}

	// 处理 And 条件
	if len(filter.And) > 0 {
		var andClauses []string
		for _, f := range filter.And {
			clause, clauseArgs := s.buildWhereClause(f)
			if clause != "" {
				andClauses = append(andClauses, strings.TrimPrefix(clause, " AND "))
				args = append(args, clauseArgs...)
			}
		}
		if len(andClauses) > 0 {
			return " AND (" + strings.Join(andClauses, " AND ") + ")", args
		}
		return "", nil
	}

	// 处理 Or 条件
	if len(filter.Or) > 0 {
		var orClauses []string
		for _, f := range filter.Or {
			clause, clauseArgs := s.buildWhereClause(f)
			if clause != "" {
				orClauses = append(orClauses, strings.TrimPrefix(clause, " AND "))
				args = append(args, clauseArgs...)
			}
		}
		if len(orClauses) > 0 {
			return " AND (" + strings.Join(orClauses, " OR ") + ")", args
		}
		return "", nil
	}

	// 处理单个条件
	column := s.getColumnName(filter.Field)
	switch filter.Op {
	case "eq", "":
		conditions = append(conditions, fmt.Sprintf("%s = ?", column))
		args = append(args, filter.Value)
	case "ne":
		conditions = append(conditions, fmt.Sprintf("%s != ?", column))
		args = append(args, filter.Value)
	case "gt":
		conditions = append(conditions, fmt.Sprintf("%s > ?", column))
		args = append(args, filter.Value)
	case "gte":
		conditions = append(conditions, fmt.Sprintf("%s >= ?", column))
		args = append(args, filter.Value)
	case "lt":
		conditions = append(conditions, fmt.Sprintf("%s < ?", column))
		args = append(args, filter.Value)
	case "lte":
		conditions = append(conditions, fmt.Sprintf("%s <= ?", column))
		args = append(args, filter.Value)
	case "contains":
		conditions = append(conditions, fmt.Sprintf("%s LIKE ?", column))
		args = append(args, "%"+fmt.Sprintf("%v", filter.Value)+"%")
	case "in":
		if arr, ok := filter.Value.([]interface{}); ok {
			placeholders := make([]string, len(arr))
			for i, v := range arr {
				placeholders[i] = "?"
				args = append(args, v)
			}
			conditions = append(conditions, fmt.Sprintf("%s IN (%s)", column, strings.Join(placeholders, ",")))
		}
	}

	if len(conditions) > 0 {
		return " AND " + strings.Join(conditions, " AND "), args
	}

	return "", nil
}

// getColumnName 获取列名（支持 metadata 字段）
func (s *SQLiteDocumentStore) getColumnName(field string) string {
	switch field {
	case "id", "content", "created_at", "updated_at":
		return field
	default:
		// 使用 JSON 提取函数访问 metadata 字段
		return fmt.Sprintf("json_extract(metadata, '$.%s')", field)
	}
}

// sanitizeColumn 清理列名防止 SQL 注入
func (s *SQLiteDocumentStore) sanitizeColumn(col string) string {
	// 只允许基本列名
	allowed := map[string]bool{
		"id": true, "content": true, "created_at": true, "updated_at": true,
	}
	if allowed[col] {
		return col
	}
	return "id" // 默认按 id 排序
}

// Compile-time interface check
var _ DocumentStore = (*SQLiteDocumentStore)(nil)
