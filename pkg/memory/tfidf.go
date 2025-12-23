package memory

import (
	"math"
	"sort"
	"strings"
	"sync"
	"unicode"
)

// TFIDFVectorizer TF-IDF 向量化器
//
// 用于工作记忆的本地语义检索，无需外部 API。
type TFIDFVectorizer struct {
	vocabulary map[string]int // 词汇表：词 -> 索引
	idf        []float32      // 逆文档频率
	documents  [][]float32    // 已向量化的文档
	docCount   int            // 文档数量
	mu         sync.RWMutex
}

// NewTFIDFVectorizer 创建 TF-IDF 向量化器
func NewTFIDFVectorizer() *TFIDFVectorizer {
	return &TFIDFVectorizer{
		vocabulary: make(map[string]int),
		idf:        make([]float32, 0),
		documents:  make([][]float32, 0),
	}
}

// tokenize 分词
//
// 支持英文空格分词和中文字符分词。
func (v *TFIDFVectorizer) tokenize(text string) []string {
	text = strings.ToLower(text)
	var tokens []string
	var currentWord strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			// 中文字符单独成词
			if unicode.Is(unicode.Han, r) {
				if currentWord.Len() > 0 {
					tokens = append(tokens, currentWord.String())
					currentWord.Reset()
				}
				tokens = append(tokens, string(r))
			} else {
				currentWord.WriteRune(r)
			}
		} else {
			if currentWord.Len() > 0 {
				tokens = append(tokens, currentWord.String())
				currentWord.Reset()
			}
		}
	}

	if currentWord.Len() > 0 {
		tokens = append(tokens, currentWord.String())
	}

	return tokens
}

// Fit 训练向量化器
//
// 根据文档集合构建词汇表和计算 IDF。
func (v *TFIDFVectorizer) Fit(documents []string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	// 统计词频
	wordDocCount := make(map[string]int)
	allWords := make(map[string]struct{})

	for _, doc := range documents {
		tokens := v.tokenize(doc)
		seen := make(map[string]struct{})
		for _, token := range tokens {
			allWords[token] = struct{}{}
			if _, ok := seen[token]; !ok {
				wordDocCount[token]++
				seen[token] = struct{}{}
			}
		}
	}

	// 构建词汇表（按字母顺序排序以保证一致性）
	words := make([]string, 0, len(allWords))
	for word := range allWords {
		words = append(words, word)
	}
	sort.Strings(words)

	v.vocabulary = make(map[string]int, len(words))
	for i, word := range words {
		v.vocabulary[word] = i
	}

	// 计算 IDF
	v.idf = make([]float32, len(words))
	n := float64(len(documents))
	for word, idx := range v.vocabulary {
		df := float64(wordDocCount[word])
		v.idf[idx] = float32(math.Log(n/df) + 1.0)
	}

	v.docCount = len(documents)
}

// Transform 将文本转换为 TF-IDF 向量
func (v *TFIDFVectorizer) Transform(text string) []float32 {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if len(v.vocabulary) == 0 {
		return nil
	}

	tokens := v.tokenize(text)
	if len(tokens) == 0 {
		return make([]float32, len(v.vocabulary))
	}

	// 计算 TF
	tf := make(map[string]int)
	for _, token := range tokens {
		tf[token]++
	}

	// 计算 TF-IDF 向量
	vector := make([]float32, len(v.vocabulary))
	for word, count := range tf {
		if idx, ok := v.vocabulary[word]; ok {
			// TF = log(1 + count)
			tfValue := float32(math.Log(1 + float64(count)))
			vector[idx] = tfValue * v.idf[idx]
		}
	}

	// L2 归一化
	v.normalize(vector)

	return vector
}

// FitTransform 训练并转换
func (v *TFIDFVectorizer) FitTransform(documents []string) [][]float32 {
	v.Fit(documents)

	v.mu.Lock()
	defer v.mu.Unlock()

	v.documents = make([][]float32, len(documents))
	for i, doc := range documents {
		// 使用内部方法避免死锁
		v.documents[i] = v.transformInternal(doc)
	}

	return v.documents
}

// transformInternal 内部转换方法（调用者需持有锁）
func (v *TFIDFVectorizer) transformInternal(text string) []float32 {
	if len(v.vocabulary) == 0 {
		return nil
	}

	tokens := v.tokenize(text)
	if len(tokens) == 0 {
		return make([]float32, len(v.vocabulary))
	}

	tf := make(map[string]int)
	for _, token := range tokens {
		tf[token]++
	}

	vector := make([]float32, len(v.vocabulary))
	for word, count := range tf {
		if idx, ok := v.vocabulary[word]; ok {
			tfValue := float32(math.Log(1 + float64(count)))
			vector[idx] = tfValue * v.idf[idx]
		}
	}

	v.normalize(vector)
	return vector
}

// normalize L2 归一化
func (v *TFIDFVectorizer) normalize(vector []float32) {
	var norm float32
	for _, val := range vector {
		norm += val * val
	}
	if norm > 0 {
		norm = float32(math.Sqrt(float64(norm)))
		for i := range vector {
			vector[i] /= norm
		}
	}
}

// CosineSimilarity 计算余弦相似度
func (v *TFIDFVectorizer) CosineSimilarity(vec1, vec2 []float32) float32 {
	if len(vec1) != len(vec2) || len(vec1) == 0 {
		return 0
	}

	var dot float32
	for i := range vec1 {
		dot += vec1[i] * vec2[i]
	}

	// 向量已归一化，所以余弦相似度就是点积
	return dot
}

// AddDocument 增量添加文档
//
// 注意：增量添加不会更新 IDF，建议在有大量新文档时重新调用 Fit。
func (v *TFIDFVectorizer) AddDocument(text string) int {
	v.mu.Lock()
	defer v.mu.Unlock()

	vector := v.transformInternal(text)
	v.documents = append(v.documents, vector)
	v.docCount++

	return len(v.documents) - 1
}

// SearchSimilar 搜索相似文档
func (v *TFIDFVectorizer) SearchSimilar(query string, topK int) []SimilarityResult {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if len(v.documents) == 0 {
		return nil
	}

	queryVector := v.transformInternal(query)
	if queryVector == nil {
		return nil
	}

	// 计算所有文档的相似度
	results := make([]SimilarityResult, len(v.documents))
	for i, docVector := range v.documents {
		results[i] = SimilarityResult{
			Index: i,
			Score: v.CosineSimilarity(queryVector, docVector),
		}
	}

	// 按相似度排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// 返回 topK
	if topK > 0 && topK < len(results) {
		return results[:topK]
	}
	return results
}

// SimilarityResult 相似度结果
type SimilarityResult struct {
	Index int     // 文档索引
	Score float32 // 相似度分数
}

// VocabularySize 返回词汇表大小
func (v *TFIDFVectorizer) VocabularySize() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.vocabulary)
}

// DocumentCount 返回文档数量
func (v *TFIDFVectorizer) DocumentCount() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.documents)
}

// Clear 清空向量化器
func (v *TFIDFVectorizer) Clear() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.vocabulary = make(map[string]int)
	v.idf = make([]float32, 0)
	v.documents = make([][]float32, 0)
	v.docCount = 0
}
