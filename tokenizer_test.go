package torus

import (
	"reflect"
	"testing"
)

func TestTokenizer_SimpleWords(t *testing.T) {
	tok := New()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single word",
			input:    "hello",
			expected: []string{"hello"},
		},
		{
			name:     "multiple words",
			input:    "hello world",
			expected: []string{"hello", "world"},
		},
		{
			name:     "with punctuation",
			input:    "hello, world!",
			expected: []string{"hello", "world"},
		},
		{
			name:     "uppercase to lowercase",
			input:    "Hello World",
			expected: []string{"hello", "world"},
		},
		{
			name:     "numbers",
			input:    "test123 456",
			expected: []string{"test123", "456"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "only punctuation",
			input:    ".,!?;:",
			expected: []string{},
		},
		{
			name:     "mixed punctuation and words",
			input:    "hello...world!!!test",
			expected: []string{"hello", "world", "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tok.TokenizeToStrings(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("TokenizeToStrings(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTokenizer_Thai(t *testing.T) {
	tok := New()

	tests := []struct {
		name     string
		input    string
		minCount int // minimum expected token count
	}{
		{
			name:     "thai place names",
			input:    "กรุงเทพ",
			minCount: 1,
		},
		{
			name:     "thai with english",
			input:    "hello กรุงเทพ world",
			minCount: 3,
		},
		{
			name:     "thai dictionary word",
			input:    "โตเกียว",
			minCount: 1, // Should match as single word from dictionary
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tok.TokenizeToStrings(tt.input)
			if len(result) < tt.minCount {
				t.Errorf("TokenizeToStrings(%q) returned %d tokens, want at least %d: %v",
					tt.input, len(result), tt.minCount, result)
			}
		})
	}
}

func TestTokenizer_CJK(t *testing.T) {
	tok := New()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "chinese characters",
			input:    "你好",
			expected: []string{"你", "好"},
		},
		{
			name:     "chinese with space",
			input:    "你好 世界",
			expected: []string{"你", "好", "世", "界"},
		},
		{
			name:     "japanese hiragana",
			input:    "あいう",
			expected: []string{"あ", "い", "う"},
		},
		{
			name:     "japanese katakana",
			input:    "アイウ",
			expected: []string{"ア", "イ", "ウ"},
		},
		{
			name:     "korean hangul",
			input:    "한글",
			expected: []string{"한", "글"},
		},
		{
			name:     "mixed cjk and english",
			input:    "hello你好world",
			expected: []string{"hello", "你", "好", "world"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tok.TokenizeToStrings(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("TokenizeToStrings(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTokenizer_MixedLanguages(t *testing.T) {
	tok := New()

	tests := []struct {
		name     string
		input    string
		minCount int
	}{
		{
			name:     "english thai chinese",
			input:    "hello สวัสดี 你好",
			minCount: 3,
		},
		{
			name:     "all mixed together",
			input:    "test中文ทดสอบend",
			minCount: 4, // test, 中, 文, thai..., end
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tok.TokenizeToStrings(tt.input)
			if len(result) < tt.minCount {
				t.Errorf("TokenizeToStrings(%q) returned %d tokens, want at least %d: %v",
					tt.input, len(result), tt.minCount, result)
			}
		})
	}
}

func TestTokenizer_TokenPositions(t *testing.T) {
	tok := New()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple positions",
			input: "hello world",
		},
		{
			name:  "with punctuation",
			input: "hello, world!",
		},
		{
			name:  "cjk positions",
			input: "你好world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := tok.Tokenize(tt.input)

			// Verify positions are sequential
			for i, token := range tokens {
				if token.Position != i {
					t.Errorf("Token %d has position %d, want %d", i, token.Position, i)
				}

				// Verify byte offsets are valid
				if token.Start < 0 || token.End > len(tt.input) || token.Start >= token.End {
					t.Errorf("Invalid byte offsets for token %q: start=%d, end=%d, input len=%d",
						token.Text, token.Start, token.End, len(tt.input))
				}
			}
		})
	}
}

func TestTokenizer_NoCaseChange(t *testing.T) {
	tok := New(WithLowercase(false))

	input := "Hello World"
	expected := []string{"Hello", "World"}

	result := tok.TokenizeToStrings(input)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("TokenizeToStrings(%q) = %v, want %v", input, result, expected)
	}
}

func TestTokenizer_NormalizeToken(t *testing.T) {
	tok := New()

	tests := []struct {
		input    string
		expected string
	}{
		{"Hello", "hello"},
		{"WORLD", "world"},
		{"Test123", "test123"},
		{"你好", "你好"}, // CJK doesn't change
	}

	for _, tt := range tests {
		result := tok.NormalizeToken(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizeToken(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestTokenizer_Parallel(t *testing.T) {
	tok := New()

	texts := []string{
		"hello world",
		"你好世界",
		"สวัสดีครับ",
		"test 123",
		"mixed 中文 ไทย",
	}

	results := tok.TokenizeToStringsParallel(texts)

	if len(results) != len(texts) {
		t.Errorf("TokenizeToStringsParallel returned %d results, want %d", len(results), len(texts))
	}

	// Verify each result is non-empty for non-empty input
	for i, result := range results {
		if len(result) == 0 {
			t.Errorf("TokenizeToStringsParallel result[%d] is empty for input %q", i, texts[i])
		}
	}
}

func TestTrie_Basic(t *testing.T) {
	trie := NewTrie()

	// Insert words
	trie.Insert("hello")
	trie.Insert("help")
	trie.Insert("world")

	// Test Contains
	if !trie.Contains("hello") {
		t.Error("Trie should contain 'hello'")
	}
	if !trie.Contains("help") {
		t.Error("Trie should contain 'help'")
	}
	if trie.Contains("hell") {
		t.Error("Trie should not contain 'hell' (not inserted)")
	}

	// Test HasPrefix
	if !trie.HasPrefix("hel") {
		t.Error("Trie should have prefix 'hel'")
	}
	if trie.HasPrefix("xyz") {
		t.Error("Trie should not have prefix 'xyz'")
	}
}

func TestTrie_LongestMatch(t *testing.T) {
	trie := NewTrie()

	trie.Insert("go")
	trie.Insert("golang")
	trie.Insert("gopher")

	tests := []struct {
		input    string
		expected int
	}{
		{"golang is great", 6}, // "golang"
		{"go home", 2},         // "go"
		{"gopher mask", 6},     // "gopher"
		{"python", 0},          // no match
	}

	for _, tt := range tests {
		runes := []rune(tt.input)
		result := trie.LongestMatch(runes)
		if result != tt.expected {
			t.Errorf("LongestMatch(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestThaiSegmenter_Basic(t *testing.T) {
	seg := NewThaiSegmenter()

	// Test dictionary is loaded
	if seg.trie.MaxWordLength() == 0 {
		t.Error("Thai dictionary should be loaded")
	}

	// Test segmentation produces output
	result := seg.SegmentToTokens("โตเกียว")
	if len(result) == 0 {
		t.Error("Segmentation should produce at least one token")
	}
}

func TestIsThaiChar(t *testing.T) {
	tests := []struct {
		char     rune
		expected bool
	}{
		{'ก', true},  // Thai consonant
		{'า', true},  // Thai vowel
		{'่', true},  // Thai tone mark
		{'a', false}, // Latin
		{'你', false}, // Chinese
	}

	for _, tt := range tests {
		result := IsThaiChar(tt.char)
		if result != tt.expected {
			t.Errorf("IsThaiChar(%q) = %v, want %v", tt.char, result, tt.expected)
		}
	}
}

func TestIsCJKChar(t *testing.T) {
	tests := []struct {
		char     rune
		expected bool
	}{
		{'你', true},  // Chinese
		{'好', true},  // Chinese
		{'a', false}, // Latin
		{'ก', false}, // Thai
	}

	for _, tt := range tests {
		result := IsCJKChar(tt.char)
		if result != tt.expected {
			t.Errorf("IsCJKChar(%q) = %v, want %v", tt.char, result, tt.expected)
		}
	}
}

func TestIsJapaneseKana(t *testing.T) {
	tests := []struct {
		char     rune
		expected bool
	}{
		{'あ', true},  // Hiragana
		{'ア', true},  // Katakana
		{'a', false}, // Latin
		{'你', false}, // Chinese (not kana)
	}

	for _, tt := range tests {
		result := IsJapaneseKana(tt.char)
		if result != tt.expected {
			t.Errorf("IsJapaneseKana(%q) = %v, want %v", tt.char, result, tt.expected)
		}
	}
}

func TestIsKoreanHangul(t *testing.T) {
	tests := []struct {
		char     rune
		expected bool
	}{
		{'한', true},  // Hangul
		{'글', true},  // Hangul
		{'a', false}, // Latin
	}

	for _, tt := range tests {
		result := IsKoreanHangul(tt.char)
		if result != tt.expected {
			t.Errorf("IsKoreanHangul(%q) = %v, want %v", tt.char, result, tt.expected)
		}
	}
}

func BenchmarkTokenizer_English(b *testing.B) {
	tok := New()
	text := "The quick brown fox jumps over the lazy dog"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.TokenizeToStrings(text)
	}
}

func BenchmarkTokenizer_Thai(b *testing.B) {
	tok := New()
	text := "โตเกียว กรุงเทพ นิวยอร์ก ซิดนีย์"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.TokenizeToStrings(text)
	}
}

func BenchmarkTokenizer_CJK(b *testing.B) {
	tok := New()
	text := "你好世界这是一个测试"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.TokenizeToStrings(text)
	}
}

func BenchmarkTokenizer_Mixed(b *testing.B) {
	tok := New()
	text := "hello world 你好世界 สวัสดี test123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.TokenizeToStrings(text)
	}
}

func BenchmarkTrie_LongestMatch(b *testing.B) {
	trie := NewTrie()
	// Load some words
	words := []string{"hello", "world", "test", "testing", "tested", "golang", "go", "gopher"}
	for _, w := range words {
		trie.Insert(w)
	}

	text := []rune("testing is fun")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.LongestMatch(text)
	}
}
