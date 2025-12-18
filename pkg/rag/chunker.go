package rag

import (
	"strings"
	"unicode"
)

// DocumentChunker 文档分块器接口
type DocumentChunker interface {
	// Chunk 将文档分割成块
	Chunk(doc Document) []DocumentChunk
}

// RecursiveCharacterChunker 递归字符分块器
//
// 使用分隔符列表递归分割文本，直到块大小在限制范围内。
type RecursiveCharacterChunker struct {
	ChunkSize      int      // 目标块大小
	ChunkOverlap   int      // 块之间的重叠大小
	Separators     []string // 分隔符列表（按优先级）
	LengthFunction func(string) int // 长度计算函数
}

// NewRecursiveCharacterChunker 创建递归字符分块器
func NewRecursiveCharacterChunker(chunkSize, overlap int) *RecursiveCharacterChunker {
	return &RecursiveCharacterChunker{
		ChunkSize:    chunkSize,
		ChunkOverlap: overlap,
		Separators: []string{
			"\n\n",  // 段落
			"\n",    // 行
			". ",    // 句子
			"! ",    // 句子
			"? ",    // 句子
			"; ",    // 分句
			", ",    // 短语
			" ",     // 单词
			"",      // 字符
		},
		LengthFunction: func(s string) int { return len(s) },
	}
}

// Chunk 将文档分割成块
func (c *RecursiveCharacterChunker) Chunk(doc Document) []DocumentChunk {
	chunks := c.splitText(doc.Content, c.Separators)

	result := make([]DocumentChunk, len(chunks))
	offset := 0

	for i, content := range chunks {
		endOffset := offset + len(content)
		result[i] = DocumentChunk{
			ID:          generateChunkID(doc.ID, i),
			DocumentID:  doc.ID,
			Content:     content,
			Index:       i,
			StartOffset: offset,
			EndOffset:   endOffset,
			Metadata:    doc.Metadata,
		}

		// 计算下一块的起始位置（考虑重叠）
		if c.ChunkOverlap < len(content) {
			offset = endOffset - c.ChunkOverlap
		} else {
			offset = endOffset
		}
	}

	return result
}

// splitText 递归分割文本
func (c *RecursiveCharacterChunker) splitText(text string, separators []string) []string {
	var result []string

	// 基本情况：文本足够小
	if c.LengthFunction(text) <= c.ChunkSize {
		if strings.TrimSpace(text) != "" {
			result = append(result, text)
		}
		return result
	}

	// 没有更多分隔符，强制按字符分割
	if len(separators) == 0 {
		return c.splitByLength(text)
	}

	separator := separators[0]
	remainingSeparators := separators[1:]

	// 使用当前分隔符分割
	var splits []string
	if separator == "" {
		// 按字符分割
		splits = c.splitByLength(text)
	} else {
		splits = strings.Split(text, separator)
	}

	// 合并和递归处理
	var currentChunk strings.Builder

	for i, split := range splits {
		splitWithSep := split
		if i < len(splits)-1 && separator != "" {
			splitWithSep += separator
		}

		potentialLength := c.LengthFunction(currentChunk.String() + splitWithSep)

		if potentialLength > c.ChunkSize && currentChunk.Len() > 0 {
			// 当前块已满，保存并开始新块
			chunkText := strings.TrimSpace(currentChunk.String())
			if chunkText != "" {
				result = append(result, chunkText)
			}
			currentChunk.Reset()

			// 添加重叠
			if c.ChunkOverlap > 0 && len(result) > 0 {
				lastChunk := result[len(result)-1]
				overlap := getOverlap(lastChunk, c.ChunkOverlap)
				currentChunk.WriteString(overlap)
			}
		}

		// 如果单个片段超过限制，递归分割
		if c.LengthFunction(splitWithSep) > c.ChunkSize {
			// 先保存当前块
			if currentChunk.Len() > 0 {
				chunkText := strings.TrimSpace(currentChunk.String())
				if chunkText != "" {
					result = append(result, chunkText)
				}
				currentChunk.Reset()
			}
			// 递归分割
			subChunks := c.splitText(splitWithSep, remainingSeparators)
			result = append(result, subChunks...)
		} else {
			currentChunk.WriteString(splitWithSep)
		}
	}

	// 保存最后一个块
	if currentChunk.Len() > 0 {
		chunkText := strings.TrimSpace(currentChunk.String())
		if chunkText != "" {
			result = append(result, chunkText)
		}
	}

	return result
}

// splitByLength 按长度分割
func (c *RecursiveCharacterChunker) splitByLength(text string) []string {
	var result []string
	runes := []rune(text)

	for i := 0; i < len(runes); i += c.ChunkSize - c.ChunkOverlap {
		end := i + c.ChunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunk := string(runes[i:end])
		if strings.TrimSpace(chunk) != "" {
			result = append(result, chunk)
		}
		if end == len(runes) {
			break
		}
	}

	return result
}

// getOverlap 获取重叠部分
func getOverlap(text string, overlapSize int) string {
	runes := []rune(text)
	if len(runes) <= overlapSize {
		return text
	}

	// 尝试在单词边界截断
	start := len(runes) - overlapSize
	overlap := string(runes[start:])

	// 找到第一个单词边界
	for i, r := range overlap {
		if unicode.IsSpace(r) {
			return strings.TrimSpace(overlap[i:])
		}
	}

	return overlap
}

// SentenceChunker 句子分块器
type SentenceChunker struct {
	MaxChunkSize int
	MinChunkSize int
}

// NewSentenceChunker 创建句子分块器
func NewSentenceChunker(maxSize, minSize int) *SentenceChunker {
	return &SentenceChunker{
		MaxChunkSize: maxSize,
		MinChunkSize: minSize,
	}
}

// Chunk 按句子分割文档
func (c *SentenceChunker) Chunk(doc Document) []DocumentChunk {
	sentences := splitSentences(doc.Content)

	var chunks []DocumentChunk
	var currentContent strings.Builder
	var startOffset int
	offset := 0

	for _, sentence := range sentences {
		potentialLength := currentContent.Len() + len(sentence)

		// 如果添加这个句子会超过最大限制
		if potentialLength > c.MaxChunkSize && currentContent.Len() >= c.MinChunkSize {
			// 保存当前块
			chunks = append(chunks, DocumentChunk{
				ID:          generateChunkID(doc.ID, len(chunks)),
				DocumentID:  doc.ID,
				Content:     strings.TrimSpace(currentContent.String()),
				Index:       len(chunks),
				StartOffset: startOffset,
				EndOffset:   offset,
				Metadata:    doc.Metadata,
			})
			currentContent.Reset()
			startOffset = offset
		}

		currentContent.WriteString(sentence)
		offset += len(sentence)
	}

	// 保存最后一个块
	if currentContent.Len() > 0 {
		chunks = append(chunks, DocumentChunk{
			ID:          generateChunkID(doc.ID, len(chunks)),
			DocumentID:  doc.ID,
			Content:     strings.TrimSpace(currentContent.String()),
			Index:       len(chunks),
			StartOffset: startOffset,
			EndOffset:   offset,
			Metadata:    doc.Metadata,
		})
	}

	return chunks
}

// splitSentences 简单的句子分割
func splitSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	for i, r := range text {
		current.WriteRune(r)

		// 检查是否是句子结束
		if r == '.' || r == '!' || r == '?' || r == '。' || r == '！' || r == '？' {
			// 检查下一个字符是否是空白或结束
			if i+1 >= len(text) || unicode.IsSpace(rune(text[i+1])) {
				sentences = append(sentences, current.String())
				current.Reset()
			}
		}
	}

	// 添加剩余内容
	if current.Len() > 0 {
		sentences = append(sentences, current.String())
	}

	return sentences
}

// compile-time interface check
var _ DocumentChunker = (*RecursiveCharacterChunker)(nil)
var _ DocumentChunker = (*SentenceChunker)(nil)
