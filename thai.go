package torus

import (
	"bufio"
	"embed"
	"strings"
	"unicode"
)

//go:embed data/words_th.txt
var thaiDictFS embed.FS

// Thai Unicode character range
const (
	thaiStart = 0x0E00
	thaiEnd   = 0x0E7F
)

// Thai vowels that cannot start a word (must follow a consonant)
var thaiFollowingVowels = map[rune]bool{
	'ะ': true, // Sara A
	'า': true, // Sara Aa
	'ิ': true, // Sara I
	'ี': true, // Sara Ii
	'ึ': true, // Sara Ue
	'ื': true, // Sara Uee
	'ุ': true, // Sara U
	'ู': true, // Sara Uu
	'ๅ': true, // Lakkhangyao
	'็': true, // Maitaikhu
	'์': true, // Thanthakhat (silent mark)
	'ๆ': true, // Maiyamok (repetition)
	'ํ': true, // Nikhahit (anusvara)
	'ฺ': true, // Phinthu
}

// Thai tone marks (cannot start a word)
var thaiToneMarks = map[rune]bool{
	'่': true, // Mai Ek
	'้': true, // Mai Tho
	'๊': true, // Mai Tri
	'๋': true, // Mai Chattawa
}

// ThaiSegmenter handles Thai text segmentation using longest match algorithm.
type ThaiSegmenter struct {
	trie *Trie
}

// NewThaiSegmenter creates a new Thai segmenter with the embedded dictionary.
func NewThaiSegmenter() *ThaiSegmenter {
	s := &ThaiSegmenter{
		trie: NewTrie(),
	}
	s.loadDictionary()
	return s
}

// loadDictionary loads the Thai dictionary from embedded file.
func (s *ThaiSegmenter) loadDictionary() {
	file, err := thaiDictFS.Open("data/words_th.txt")
	if err != nil {
		s.loadFallbackDictionary()
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word != "" {
			s.trie.Insert(word)
		}
	}

	if s.trie.maxLen == 0 {
		s.loadFallbackDictionary()
	}
}

// loadFallbackDictionary loads a minimal built-in dictionary as fallback.
func (s *ThaiSegmenter) loadFallbackDictionary() {
	// Minimal fallback dictionary with common Thai words
	fallback := []string{
		"การ", "ที่", "และ", "ใน", "มี", "เป็น", "ได้", "จะ", "ว่า", "ของ",
		"ไม่", "ให้", "นี้", "จาก", "ก็", "กับ", "แล้ว", "เมื่อ", "ถ้า", "ยัง",
	}
	for _, word := range fallback {
		s.trie.Insert(word)
	}
}

// IsThaiChar returns true if the rune is a Thai character.
func IsThaiChar(r rune) bool {
	return r >= thaiStart && r <= thaiEnd
}

// IsThaiCombiningMark returns true if the rune is a Thai combining mark (vowel or tone).
func IsThaiCombiningMark(r rune) bool {
	return thaiFollowingVowels[r] || thaiToneMarks[r] || unicode.IsMark(r)
}

// CanStartThaiToken returns true if the rune can start a Thai token.
func CanStartThaiToken(r rune) bool {
	return !thaiFollowingVowels[r] && !thaiToneMarks[r]
}

// Segment segments Thai text into words using longest match algorithm.
func (s *ThaiSegmenter) Segment(text string) []string {
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return nil
	}

	var segments []string
	i := 0

	for i < n {
		// Skip whitespace
		if unicode.IsSpace(runes[i]) {
			i++
			continue
		}

		// Check if current position is Thai text
		if !IsThaiChar(runes[i]) {
			// Collect non-Thai sequence
			start := i
			for i < n && !IsThaiChar(runes[i]) && !unicode.IsSpace(runes[i]) {
				i++
			}
			segments = append(segments, string(runes[start:i]))
			continue
		}

		// Try longest match from current position
		remaining := runes[i:]
		matchLen := s.trie.LongestMatch(remaining)

		if matchLen > 0 {
			// Found a dictionary word
			segments = append(segments, string(runes[i:i+matchLen]))
			i += matchLen
		} else {
			// No match - take a single character cluster (base + combining marks)
			clusterEnd := i + 1
			for clusterEnd < n && IsThaiCombiningMark(runes[clusterEnd]) {
				clusterEnd++
			}
			segments = append(segments, string(runes[i:clusterEnd]))
			i = clusterEnd
		}
	}

	return segments
}

// SegmentToTokens segments Thai text and handles orphan vowels by merging them
// with the previous token.
func (s *ThaiSegmenter) SegmentToTokens(text string) []string {
	segments := s.Segment(text)
	if len(segments) == 0 {
		return nil
	}

	var tokens []string
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}

		runes := []rune(seg)
		if len(runes) > 0 && !CanStartThaiToken(runes[0]) {
			// Cannot start a token - merge with previous
			if len(tokens) > 0 {
				tokens[len(tokens)-1] += seg
				continue
			}
		}
		tokens = append(tokens, seg)
	}

	return tokens
}

// ContainsThaiChar checks if text contains any Thai characters.
func ContainsThaiChar(text string) bool {
	for _, r := range text {
		if IsThaiChar(r) {
			return true
		}
	}
	return false
}
