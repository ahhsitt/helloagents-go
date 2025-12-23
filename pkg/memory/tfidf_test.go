package memory

import (
	"testing"
)

func TestNewTFIDFVectorizer(t *testing.T) {
	v := NewTFIDFVectorizer()
	if v == nil {
		t.Fatal("expected non-nil vectorizer")
	}
	if v.VocabularySize() != 0 {
		t.Errorf("expected empty vocabulary, got %d", v.VocabularySize())
	}
}

func TestTokenize(t *testing.T) {
	v := NewTFIDFVectorizer()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"english", "hello world", []string{"hello", "world"}},
		{"chinese", "你好世界", []string{"你", "好", "世", "界"}},
		{"mixed", "hello 世界", []string{"hello", "世", "界"}},
		{"with numbers", "test123", []string{"test123"}},
		{"with punctuation", "hello, world!", []string{"hello", "world"}},
		{"uppercase", "Hello World", []string{"hello", "world"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := v.tokenize(tt.input)
			if len(tokens) != len(tt.expected) {
				t.Errorf("expected %d tokens, got %d: %v", len(tt.expected), len(tokens), tokens)
				return
			}
			for i, token := range tokens {
				if token != tt.expected[i] {
					t.Errorf("token %d: expected %q, got %q", i, tt.expected[i], token)
				}
			}
		})
	}
}

func TestFit(t *testing.T) {
	v := NewTFIDFVectorizer()
	docs := []string{
		"the cat sat on the mat",
		"the dog ran in the park",
		"the bird flew in the sky",
	}

	v.Fit(docs)

	if v.VocabularySize() == 0 {
		t.Error("expected non-empty vocabulary")
	}

	// Check that common words are in vocabulary
	for _, word := range []string{"the", "cat", "dog", "bird"} {
		if _, ok := v.vocabulary[word]; !ok {
			t.Errorf("expected word %q in vocabulary", word)
		}
	}
}

func TestTransform(t *testing.T) {
	v := NewTFIDFVectorizer()
	docs := []string{
		"the cat sat on the mat",
		"the dog ran in the park",
	}

	v.Fit(docs)

	vector := v.Transform("the cat")
	if vector == nil {
		t.Fatal("expected non-nil vector")
	}
	if len(vector) != v.VocabularySize() {
		t.Errorf("expected vector length %d, got %d", v.VocabularySize(), len(vector))
	}

	// Vector should be normalized (L2 norm ≈ 1)
	var norm float32
	for _, val := range vector {
		norm += val * val
	}
	if norm > 0 && (norm < 0.99 || norm > 1.01) {
		t.Errorf("expected normalized vector, got norm %f", norm)
	}
}

func TestFitTransform(t *testing.T) {
	v := NewTFIDFVectorizer()
	docs := []string{
		"the cat sat on the mat",
		"the dog ran in the park",
		"the bird flew in the sky",
	}

	vectors := v.FitTransform(docs)

	if len(vectors) != len(docs) {
		t.Errorf("expected %d vectors, got %d", len(docs), len(vectors))
	}

	for i, vec := range vectors {
		if len(vec) != v.VocabularySize() {
			t.Errorf("vector %d: expected length %d, got %d", i, v.VocabularySize(), len(vec))
		}
	}
}

func TestCosineSimilarity(t *testing.T) {
	v := NewTFIDFVectorizer()

	tests := []struct {
		name     string
		vec1     []float32
		vec2     []float32
		expected float32
	}{
		{"identical", []float32{1, 0, 0}, []float32{1, 0, 0}, 1.0},
		{"orthogonal", []float32{1, 0, 0}, []float32{0, 1, 0}, 0.0},
		{"opposite", []float32{1, 0, 0}, []float32{-1, 0, 0}, -1.0},
		{"empty", []float32{}, []float32{}, 0.0},
		{"different length", []float32{1, 0}, []float32{1, 0, 0}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim := v.CosineSimilarity(tt.vec1, tt.vec2)
			if (sim-tt.expected) > 0.01 || (sim-tt.expected) < -0.01 {
				t.Errorf("expected similarity %f, got %f", tt.expected, sim)
			}
		})
	}
}

func TestSearchSimilar(t *testing.T) {
	v := NewTFIDFVectorizer()
	docs := []string{
		"the cat sat on the mat",
		"the dog ran in the park",
		"the bird flew in the sky",
		"cats and dogs are pets",
	}

	v.FitTransform(docs)

	results := v.SearchSimilar("cat pet", 2)
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Results should be sorted by score
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Error("expected results sorted by score descending")
		}
	}
}

func TestAddDocument(t *testing.T) {
	v := NewTFIDFVectorizer()
	docs := []string{
		"the cat sat on the mat",
		"the dog ran in the park",
	}

	v.FitTransform(docs)
	initialCount := v.DocumentCount()

	idx := v.AddDocument("a new document about cats")
	if idx != initialCount {
		t.Errorf("expected index %d, got %d", initialCount, idx)
	}

	if v.DocumentCount() != initialCount+1 {
		t.Errorf("expected %d documents, got %d", initialCount+1, v.DocumentCount())
	}
}

func TestClear(t *testing.T) {
	v := NewTFIDFVectorizer()
	docs := []string{"hello world", "foo bar"}
	v.FitTransform(docs)

	v.Clear()

	if v.VocabularySize() != 0 {
		t.Errorf("expected empty vocabulary after clear, got %d", v.VocabularySize())
	}
	if v.DocumentCount() != 0 {
		t.Errorf("expected 0 documents after clear, got %d", v.DocumentCount())
	}
}

func TestEmptyVectorizer(t *testing.T) {
	v := NewTFIDFVectorizer()

	// Transform on empty vectorizer
	vector := v.Transform("hello world")
	if vector != nil {
		t.Error("expected nil vector from empty vectorizer")
	}

	// Search on empty vectorizer
	results := v.SearchSimilar("query", 10)
	if results != nil {
		t.Error("expected nil results from empty vectorizer")
	}
}

func TestChineseText(t *testing.T) {
	v := NewTFIDFVectorizer()
	docs := []string{
		"今天天气很好",
		"明天会下雨",
		"天气预报说晴天",
	}

	v.FitTransform(docs)

	// Should find documents about weather
	results := v.SearchSimilar("天气", 2)
	if len(results) == 0 {
		t.Error("expected results for Chinese query")
	}
}
