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

func TestTrie_AllMatches(t *testing.T) {
	trie := NewTrie()

	trie.Insert("go")
	trie.Insert("golang")
	trie.Insert("gopher")

	tests := []struct {
		input    string
		expected []int
	}{
		{"golang is great", []int{2, 6}}, // "go" and "golang"
		{"go home", []int{2}},             // "go" only
		{"gopher mask", []int{2, 6}},      // "go" and "gopher"
		{"python", nil},                   // no match
	}

	for _, tt := range tests {
		runes := []rune(tt.input)
		result := trie.AllMatches(runes)
		if !reflect.DeepEqual(result, tt.expected) {
			t.Errorf("AllMatches(%q) = %v, want %v", tt.input, result, tt.expected)
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

// ============================================================
// Atomic Segmentation Tests
// ============================================================

func TestAtomicSegmenter_Basic(t *testing.T) {
	seg := NewAtomicSegmenter()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "example from spec",
			input:    "เสียงเพลงบทนี้ดูมีความเพราะ",
			expected: []string{"เสีย", "ง", "เพ", "ล", "ง", "บ", "ท", "นี้", "ดู", "มี", "ค", "วา", "ม", "เพ", "ราะ"},
		},
		{
			name:     "leading vowel with middle vowel and consonant",
			input:    "เสีย",
			expected: []string{"เสีย"},
		},
		{
			name:     "leading vowel only",
			input:    "เพ",
			expected: []string{"เพ"},
		},
		{
			name:     "consonant with middle vowel",
			input:    "ดู",
			expected: []string{"ดู"},
		},
		{
			name:     "consonant with middle vowel and tone",
			input:    "นี้",
			expected: []string{"นี้"},
		},
		{
			name:     "single consonant",
			input:    "ก",
			expected: []string{"ก"},
		},
		{
			name:     "special atoms",
			input:    "ฯๆ",
			expected: []string{"ฯ", "ๆ"},
		},
		{
			name:     "consonant with aa-short vowel",
			input:    "ราะ",
			expected: []string{"ราะ"},
		},
		{
			name:     "multiple atoms",
			input:    "กาบ",
			expected: []string{"กา", "บ"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := seg.SegmentToTokens(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("SegmentToTokens(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTokenizer_ModeAtomic(t *testing.T) {
	tok := New(WithMode(ModeAtomic))

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "atomic thai segmentation",
			input:    "เสียงเพลง",
			expected: []string{"เสีย", "ง", "เพ", "ล", "ง"},
		},
		{
			name:     "mixed with english",
			input:    "hello เสีย world",
			expected: []string{"hello", "เสีย", "world"},
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

func TestTokenizer_ModeCombined(t *testing.T) {
	tok := New(WithMode(ModeCombined))
	tokDict := New(WithMode(ModeDict))

	// Combined mode keeps valid Dict tokens as-is
	// Only merges and atomizes when Dict produces invalid tokens

	// Test with valid Dict tokens - should match Dict output
	result := tok.TokenizeToStrings("โตเกียว")
	dictResult := tokDict.TokenizeToStrings("โตเกียว")

	if len(result) == 0 {
		t.Error("Combined mode should produce tokens")
	}

	// Should match Dict when tokens are valid
	if !reflect.DeepEqual(result, dictResult) {
		t.Errorf("Combined should match Dict for valid tokens: got %v, want %v", result, dictResult)
	}

	// Test with name - should also match Dict
	result2 := tok.TokenizeToStrings("ประยุทธ์ จันทร์โอชา")
	dictResult2 := tokDict.TokenizeToStrings("ประยุทธ์ จันทร์โอชา")

	if !reflect.DeepEqual(result2, dictResult2) {
		t.Errorf("Combined should match Dict: got %v, want %v", result2, dictResult2)
	}
}

func TestTokenizer_ModeDefault(t *testing.T) {
	tok := New()

	if tok.Mode() != ModeDict {
		t.Errorf("Default mode should be ModeDict, got %v", tok.Mode())
	}
}

func TestAtomicSegmenter_LeadingVowels(t *testing.T) {
	seg := NewAtomicSegmenter()

	// All leading vowels should attach to following consonant
	leadingVowels := []rune{'เ', 'แ', 'โ', 'ไ', 'ใ'}

	for _, v := range leadingVowels {
		input := string(v) + "ก" // leading vowel + consonant
		result := seg.SegmentToTokens(input)
		if len(result) != 1 {
			t.Errorf("Leading vowel %c + consonant should be single atom, got %v", v, result)
		}
	}
}

func TestAtomicSegmenter_ToneMarks(t *testing.T) {
	seg := NewAtomicSegmenter()

	tests := []struct {
		input    string
		expected []string
	}{
		{"ก่", []string{"ก่"}},   // Mai Ek
		{"ก้", []string{"ก้"}},   // Mai Tho
		{"ก๊", []string{"ก๊"}},   // Mai Tri
		{"ก๋", []string{"ก๋"}},   // Mai Chattawa
		{"ก์", []string{"ก์"}},   // Karun (silent mark)
		{"กี่", []string{"กี่"}}, // Middle vowel + tone
	}

	for _, tt := range tests {
		result := seg.SegmentToTokens(tt.input)
		if !reflect.DeepEqual(result, tt.expected) {
			t.Errorf("SegmentToTokens(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestAtomicSegmenter_MiddleVowelCombinations(t *testing.T) {
	seg := NewAtomicSegmenter()

	tests := []struct {
		input    string
		expected []string
	}{
		{"กะ", []string{"กะ"}},     // Short A
		{"กา", []string{"กา"}},     // Long A
		{"กาะ", []string{"กาะ"}},   // Combined vowel (aa + short marker)
		{"กือ", []string{"กื", "อ"}}, // ื is vowel, อ is consonant (atomically separate)
		{"กี", []string{"กี"}},     // Long I
		{"กู", []string{"กู"}},     // Long U
	}

	for _, tt := range tests {
		result := seg.SegmentToTokens(tt.input)
		if !reflect.DeepEqual(result, tt.expected) {
			t.Errorf("SegmentToTokens(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func BenchmarkTokenizer_Atomic(b *testing.B) {
	tok := New(WithMode(ModeAtomic))
	text := "เสียงเพลงบทนี้ดูมีความเพราะ"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.TokenizeToStrings(text)
	}
}

func BenchmarkTokenizer_Combined(b *testing.B) {
	tok := New(WithMode(ModeCombined))
	text := "โตเกียว กรุงเทพ นิวยอร์ก"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.TokenizeToStrings(text)
	}
}

// Direct segmenter benchmarks (without tokenizer overhead)
func BenchmarkSegmenter_Dict(b *testing.B) {
	seg := NewThaiSegmenter()
	text := "เสียงเพลงบทนี้ดูมีความเพราะ"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		seg.SegmentToTokens(text)
	}
}

func BenchmarkSegmenter_Atomic(b *testing.B) {
	seg := NewAtomicSegmenter()
	text := "เสียงเพลงบทนี้ดูมีความเพราะ"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		seg.SegmentToTokens(text)
	}
}

// Long Thai text for benchmarking (4k+ characters)
var longThaiText = `กรุงเทพมหานครเป็นเมืองหลวงและนครที่มีประชากรมากที่สุดของประเทศไทย เป็นศูนย์กลางการปกครอง การศึกษา การคมนาคมขนส่ง การเงินการธนาคาร การพาณิชย์ การสื่อสาร และความเจริญของประเทศ เป็นเมืองที่มีชื่อยาวที่สุดในโลก ตั้งอยู่บนสามเหลี่ยมปากแม่น้ำเจ้าพระยา มีแม่น้ำเจ้าพระยาไหลผ่านและแบ่งเมืองออกเป็นสองฝั่ง คือ ฝั่งพระนครและฝั่งธนบุรี กรุงเทพมหานครมีพื้นที่ทั้งหมด 1,568.737 ตารางกิโลเมตร มีประชากรตามทะเบียนราษฎรกว่า 5 ล้านคน แต่ประชากรที่อาศัยอยู่จริงคาดว่ามีสูงถึง 10 ล้านคน ทำให้เป็นเมืองที่มีประชากรหนาแน่นมากเป็นอันดับต้นๆ ของโลก
ประวัติศาสตร์ของกรุงเทพมหานครเริ่มต้นเมื่อพระบาทสมเด็จพระพุทธยอดฟ้าจุฬาโลกมหาราช ทรงสถาปนาเป็นราชธานีแห่งใหม่ของอาณาจักรสยามเมื่อปี พ.ศ. 2325 หลังจากกรุงธนบุรีเสียแก่พม่า พระองค์ทรงย้ายเมืองหลวงจากฝั่งธนบุรีมายังฝั่งตะวันออกของแม่น้ำเจ้าพระยา และทรงสร้างพระบรมมหาราชวังเป็นที่ประทับ ตลอดจนวัดพระศรีรัตนศาสดารามหรือวัดพระแก้วเป็นวัดประจำพระราชวัง
ปัจจุบันกรุงเทพมหานครเป็นมหานครระดับโลก เป็นศูนย์กลางทางเศรษฐกิจของภูมิภาคอินโดจีน และเป็นหนึ่งในจุดหมายปลายทางท่องเที่ยวที่ได้รับความนิยมมากที่สุดในโลก มีสถานที่ท่องเที่ยวมากมาย ทั้งวัดวาอาราม พระราชวัง พิพิธภัณฑ์ ห้างสรรพสินค้า ตลาดนัด และแหล่งบันเทิงต่างๆ อาหารไทยก็เป็นที่รู้จักไปทั่วโลก โดยเฉพาะต้มยำกุ้ง ผัดไทย แกงเขียวหวาน และข้าวผัด
การคมนาคมในกรุงเทพมหานครมีหลากหลายรูปแบบ ทั้งรถยนต์ส่วนตัว รถโดยสารประจำทาง รถไฟฟ้าบีทีเอส รถไฟฟ้าใต้ดิน รถไฟฟ้าแอร์พอร์ตลิงก์ เรือด่วนเจ้าพระยา และแท็กซี่ อย่างไรก็ตาม ปัญหาการจราจรติดขัดยังคงเป็นปัญหาใหญ่ของเมือง โดยเฉพาะในชั่วโมงเร่งด่วน
กรุงเทพมหานครแบ่งการปกครองออกเป็น 50 เขต และ 180 แขวง มีผู้ว่าราชการกรุงเทพมหานครเป็นผู้บริหารสูงสุด ซึ่งมาจากการเลือกตั้งโดยตรงของประชาชน การศึกษาในกรุงเทพมหานครมีสถาบันการศึกษาทุกระดับ ตั้งแต่ระดับอนุบาลจนถึงระดับอุดมศึกษา มีมหาวิทยาลัยชั้นนำของประเทศหลายแห่ง เช่น จุฬาลงกรณ์มหาวิทยาลัย มหาวิทยาลัยธรรมศาสตร์ มหาวิทยาลัยเกษตรศาสตร์ และมหาวิทยาลัยมหิดล
สภาพอากาศของกรุงเทพมหานครเป็นแบบร้อนชื้น มีสามฤดู คือ ฤดูร้อน ฤดูฝน และฤดูหนาว อุณหภูมิเฉลี่ยตลอดปีประมาณ 28 องศาเซลเซียส ฤดูร้อนอุณหภูมิอาจสูงถึง 40 องศาเซลเซียส ส่วนฤดูหนาวอุณหภูมิอาจลดลงถึง 15 องศาเซลเซียส ปริมาณน้ำฝนเฉลี่ยประมาณ 1,500 มิลลิเมตรต่อปี
เศรษฐกิจของกรุงเทพมหานครมีขนาดใหญ่ที่สุดในประเทศ คิดเป็นสัดส่วนประมาณร้อยละ 44 ของผลิตภัณฑ์มวลรวมภายในประเทศ ภาคบริการเป็นภาคเศรษฐกิจที่ใหญ่ที่สุด โดยเฉพาะการท่องเที่ยว การเงินการธนาคาร และการค้าปลีก นอกจากนี้ยังมีภาคอุตสาหกรรมการผลิตที่สำคัญ เช่น อุตสาหกรรมยานยนต์ อิเล็กทรอนิกส์ และอาหาร`

func BenchmarkSegmenter_Dict_Long(b *testing.B) {
	seg := NewThaiSegmenter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		seg.SegmentToTokens(longThaiText)
	}
}

func BenchmarkSegmenter_Atomic_Long(b *testing.B) {
	seg := NewAtomicSegmenter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		seg.SegmentToTokens(longThaiText)
	}
}

func BenchmarkTokenizer_Dict_Long(b *testing.B) {
	tok := New(WithMode(ModeDict))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.TokenizeToStrings(longThaiText)
	}
}

func BenchmarkTokenizer_Atomic_Long(b *testing.B) {
	tok := New(WithMode(ModeAtomic))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.TokenizeToStrings(longThaiText)
	}
}

func BenchmarkTokenizer_Combined_Long(b *testing.B) {
	tok := New(WithMode(ModeCombined))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.TokenizeToStrings(longThaiText)
	}
}

func TestAtomicSegmenter_MaiHanAkatWithTone(t *testing.T) {
	seg := NewAtomicSegmenter()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "ชั่น - consonant + mai han-akat + tone + final consonant",
			input:    "ชั่น",
			expected: []string{"ชั่น"},
		},
		{
			name:     "นั้น - consonant + mai han-akat + tone + final consonant",
			input:    "นั้น",
			expected: []string{"นั้น"},
		},
		{
			name:     "จัน - consonant + mai han-akat + final consonant (no tone)",
			input:    "จัน",
			expected: []string{"จัน"},
		},
		{
			name:     "นั่ง - consonant + mai han-akat + tone + final consonant",
			input:    "นั่ง",
			expected: []string{"นั่ง"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := seg.SegmentToTokens(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("SegmentToTokens(%q) = %v, want %v", tt.input, result, tt.expected)
				// Debug: print runes
				t.Logf("Input runes: %U", []rune(tt.input))
			}
		})
	}
}

func TestCombinedMode_MaiHanAkatWithTone(t *testing.T) {
	tokCombined := New(WithMode(ModeCombined))

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "เมชั่น - mai han-akat + tone with final consonant",
			input:    "เมชั่น",
			expected: []string{"เม", "ชั่น"},
		},
		{
			name:     "นั้น - mai han-akat + tone with final consonant",
			input:    "นั้น",
			expected: []string{"นั้น"},
		},
		{
			name:     "ดิจิตัล - Dict splits mai han-akat separately",
			input:    "ดิจิตัล",
			expected: []string{"ดิ", "จิ", "ตัล"},
		},
		{
			name:     "พิธานั้น - Dict splits mai han-akat+tone separately",
			input:    "พิธานั้น",
			expected: []string{"พิ", "ธา", "นั้น"},
		},
		{
			name:     "ทรานส์ฟอร์เมชั่น - loanword with ชั่น",
			input:    "ทรานส์ฟอร์เมชั่น",
			expected: []string{"ทรานส์", "ฟ", "อร์", "เม", "ชั่น"},
		},
		{
			name:     "นายกันตพล - backtrack from นายก to นาย for valid boundary",
			input:    "นายกันตพล",
			expected: []string{"นาย", "กัน", "ต", "พล"},
		},
		{
			name:     "นายกฤษฎา - coverage backtrack: นาย+กฤษฎา over นายก+ฤษฎา",
			input:    "นายกฤษฎา",
			expected: []string{"นาย", "กฤษฎา"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokCombined.TokenizeToStrings(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("TokenizeToStrings(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

