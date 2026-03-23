// Package torus provides Thai language tokenization using Trie-based longest match algorithm.
//
// TORUS is designed specifically for Thai language tokenization with support for:
//   - Thai text segmentation using dictionary-based longest match (Dict mode)
//   - Atomic segmentation into smallest valid Thai units (Atomic mode)
//   - Combined mode: dictionary-based with atomic guarantee
//   - CJK character-by-character splitting (Chinese, Japanese, Korean)
//   - Standard Unicode word boundary detection for other languages
//   - Proper handling of Thai combining marks (vowels, tone marks)
//
// Basic usage:
//
//	tok := torus.New()
//	tokens := tok.Tokenize("สวัสดีครับ")
//	for _, t := range tokens {
//	    fmt.Println(t.Text)
//	}
//
// Using different modes:
//
//	tok := torus.New(torus.WithMode(torus.ModeAtomic))
//	tokens := tok.TokenizeToStrings("เสียงเพลง")
//	// Returns: ["เสีย", "ง", "เพ", "ล", "ง"]
package torus

import (
	"strings"
	"unicode"
)

// Mode represents the tokenization mode for Thai text.
type Mode int

const (
	// ModeDict uses dictionary-based longest match segmentation.
	ModeDict Mode = iota

	// ModeAtomic splits Thai text into smallest valid atomic units.
	ModeAtomic

	// ModeCombined uses dictionary-based segmentation but guarantees
	// each token is also a valid atomic unit (splits non-atomic dict tokens).
	ModeCombined
)

// Token represents a tokenized segment of text.
type Token struct {
	Text     string // Normalized token text (lowercase)
	Start    int    // Byte offset in original text
	End      int    // Byte offset end (exclusive)
	Position int    // Token position (0-indexed)
}

// Tokenizer defines the interface for text tokenization.
type Tokenizer interface {
	// Tokenize splits text into tokens with position information.
	Tokenize(text string) []Token

	// TokenizeToStrings splits text into token strings only.
	TokenizeToStrings(text string) []string

	// NormalizeToken normalizes a token string (lowercase).
	NormalizeToken(token string) string
}

// ThaiSegmenterInterface defines the interface for Thai segmenters.
type ThaiSegmenterInterface interface {
	SegmentToTokens(text string) []string
}

// TorusTokenizer implements the Tokenizer interface with Thai and CJK support.
type TorusTokenizer struct {
	mode            Mode
	thaiSegmenter   *ThaiSegmenter
	atomicSegmenter *AtomicSegmenter
	lowercase       bool
}

// Option is a function that configures the tokenizer.
type Option func(*TorusTokenizer)

// WithLowercase sets whether to lowercase tokens (default: true).
func WithLowercase(lowercase bool) Option {
	return func(t *TorusTokenizer) {
		t.lowercase = lowercase
	}
}

// WithMode sets the tokenization mode (default: ModeDict).
func WithMode(mode Mode) Option {
	return func(t *TorusTokenizer) {
		t.mode = mode
	}
}

// New creates a new TorusTokenizer with default settings.
func New(opts ...Option) *TorusTokenizer {
	t := &TorusTokenizer{
		mode:            ModeDict,
		thaiSegmenter:   NewThaiSegmenter(),
		atomicSegmenter: NewAtomicSegmenter(),
		lowercase:       true,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Mode returns the current tokenization mode.
func (t *TorusTokenizer) Mode() Mode {
	return t.mode
}

// segmentThai segments Thai text based on the current mode.
func (t *TorusTokenizer) segmentThai(text string) []string {
	switch t.mode {
	case ModeAtomic:
		return t.atomicSegmenter.SegmentToTokens(text)

	case ModeCombined:
		// Use dictionary segmentation with coverage-aware backtracking.
		// This handles all merge cases (orphan vowels, mai han-akat, leading vowels)
		// and also backtracks greedy matches when a shorter match yields better
		// dictionary coverage of the remainder.
		return t.segmentWithBacktrack(text)

	default: // ModeDict
		return t.thaiSegmenter.SegmentToTokens(text)
	}
}

// needsMergeWithNext returns true if token ends in a state requiring continuation.
// e.g., standalone leading vowel (เ แ โ ไ ใ) or ends with leading vowel.
// Also handles mai han-akat (ั) which requires a final consonant.
func needsMergeWithNext(tok string) bool {
	if tok == "" {
		return false
	}
	runes := []rune(tok)

	// Check if entire token is just leading vowel(s)
	allLeading := true
	for _, r := range runes {
		if !isThaiLeadingVowel(r) {
			allLeading = false
			break
		}
	}
	if allLeading {
		return true
	}

	// Check if token ends with a leading vowel (needs consonant after)
	lastRune := runes[len(runes)-1]
	if isThaiLeadingVowel(lastRune) {
		return true
	}

	// Check if token has mai han-akat (ั) without a final consonant
	// Pattern: C + ั + [tone] needs a final consonant
	// Valid: C + ั + [tone] + C (e.g., นั้น, ชั่น)
	// Invalid: C + ั + [tone] (e.g., ชั่) - needs merge
	n := len(runes)
	for i := 0; i < n; i++ {
		if runes[i] == 0x0E31 { // mai han-akat (ั)
			// Check if there's a consonant after mai han-akat (possibly after tone marks)
			hasFollowingConsonant := false
			for j := i + 1; j < n; j++ {
				if isThaiConsonant(runes[j]) {
					hasFollowingConsonant = true
					break
				}
				// Skip tone marks
				if !isThaiToneMark(runes[j]) {
					break
				}
			}
			if !hasFollowingConsonant {
				return true
			}
		}
	}

	return false
}

// needsMergeWithPrev returns true if token starts with something that can't start an atom.
// e.g., standalone middle vowel or tone mark.
func needsMergeWithPrev(tok string) bool {
	if tok == "" {
		return false
	}
	runes := []rune(tok)
	first := runes[0]

	// Middle vowels and tone marks can't start a valid atom
	if isThaiMiddleVowel(first) || isThaiToneMark(first) {
		return true
	}

	return false
}

// segmentWithBacktrack segments Thai text using dictionary with coverage-aware
// backtracking. It is the primary segmenter for Combined mode.
//
// At each position it finds all dictionary matches and selects using two criteria
// (in priority order):
//  1. Boundary validity: the character after the match must not be an orphan
//     vowel or tone mark.
//  2. Remainder coverage: the text after the match should start with a
//     dictionary word (i.e., no single-char fallback gap).
//
// If the longest match fails criterion 2, shorter matches are tried. This solves
// greedy ambiguity: e.g., "นายกฤษฎา" picks "นาย"(3) over "นายก"(4) because
// the remainder "กฤษฎา" has dict coverage while "ฤษฎา" does not.
//
// After dict matching, orphan merging and forward merging (with atomic fallback)
// handle any remaining edge cases.
func (t *TorusTokenizer) segmentWithBacktrack(text string) []string {
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return nil
	}

	// Step 1: Dict segmentation with coverage-aware backtracking
	var rawSegments []string
	i := 0

	for i < n {
		if unicode.IsSpace(runes[i]) {
			i++
			continue
		}

		if !IsThaiChar(runes[i]) {
			start := i
			for i < n && !IsThaiChar(runes[i]) && !unicode.IsSpace(runes[i]) {
				i++
			}
			rawSegments = append(rawSegments, string(runes[start:i]))
			continue
		}

		// Try dict matches with boundary + coverage validation
		matches := t.thaiSegmenter.trie.AllMatches(runes[i:])

		chosen := 0
		fallback := 0 // valid boundary but no dict coverage on remainder

		// Try from longest to shortest
		for j := len(matches) - 1; j >= 0; j-- {
			mLen := matches[j]
			end := i + mLen
			if end >= n {
				chosen = mLen
				break
			}

			// Check 1: boundary validity
			// Middle vowels (ะ า ิ ี ึ ื ุ ู ั ำ etc.) after a match indicate
			// the match consumed a consonant that belongs to the next syllable.
			// Tone marks (็ ่ ้ ๊ ๋ ์) modify the preceding character and will
			// be absorbed as combining marks — they don't invalidate the boundary.
			if isThaiMiddleVowel(runes[end]) {
				continue // invalid boundary, try shorter
			}

			// Skip past any trailing combining marks (tones, etc.) that will
			// be absorbed into this match, to check the true remainder.
			effectiveEnd := end
			for effectiveEnd < n && IsThaiCombiningMark(runes[effectiveEnd]) {
				effectiveEnd++
			}

			if effectiveEnd >= n {
				chosen = mLen
				break
			}

			// Check 2: does remainder (after combining marks) start with a dict word?
			remainderMatches := t.thaiSegmenter.trie.AllMatches(runes[effectiveEnd:])
			if len(remainderMatches) > 0 {
				chosen = mLen // best: valid boundary + remainder coverage
				break
			}

			// Valid boundary but no remainder coverage — save as fallback
			if fallback == 0 {
				fallback = mLen
			}
		}

		if chosen == 0 {
			chosen = fallback // use valid-boundary-only match
		}

		if chosen > 0 {
			// Extend match to absorb any trailing combining marks (tone marks,
			// following vowels) that can't start their own token. This mirrors
			// Dict mode's orphan merging — e.g., "อร" + "์" → "อร์".
			end := i + chosen
			for end < n && IsThaiCombiningMark(runes[end]) {
				end++
			}
			rawSegments = append(rawSegments, string(runes[i:end]))
			i = end
		} else {
			// No valid dict match - take single char + combining marks
			clusterEnd := i + 1
			for clusterEnd < n && IsThaiCombiningMark(runes[clusterEnd]) {
				clusterEnd++
			}
			rawSegments = append(rawSegments, string(runes[i:clusterEnd]))
			i = clusterEnd
		}
	}

	// Step 2: Orphan merging - merge tokens that can't start independently
	var tokens []string
	for _, seg := range rawSegments {
		if seg == "" {
			continue
		}
		rs := []rune(seg)
		if len(rs) > 0 && (isThaiMiddleVowel(rs[0]) || isThaiToneMark(rs[0])) && len(tokens) > 0 {
			tokens[len(tokens)-1] += seg
			continue
		}
		tokens = append(tokens, seg)
	}

	// Step 3: Forward merge for incomplete tokens (mai han-akat, leading vowels),
	// atomize merged portions as final fallback
	var result []string
	j := 0
	for j < len(tokens) {
		tok := tokens[j]
		if needsMergeWithNext(tok) && j+1 < len(tokens) {
			merged := tok
			j++
			for j < len(tokens) {
				merged += tokens[j]
				j++
				if !needsMergeWithNext(merged) {
					break
				}
			}
			atoms := t.atomicSegmenter.SegmentToTokens(merged)
			result = append(result, atoms...)
			continue
		}
		result = append(result, tok)
		j++
	}

	return result
}

// Tokenize splits text into tokens with position information.
func (t *TorusTokenizer) Tokenize(text string) []Token {
	var tokens []Token
	runes := []rune(text)
	n := len(runes)

	// Build byte offset map: rune index -> byte offset
	byteOffsets := make([]int, n+1)
	bytePos := 0
	for i, r := range runes {
		byteOffsets[i] = bytePos
		bytePos += len(string(r))
	}
	byteOffsets[n] = bytePos

	position := 0
	i := 0

	for i < n {
		// Skip non-word characters
		if !IsWordChar(runes[i]) {
			i++
			continue
		}

		startRune := i

		// Check what kind of text we're dealing with
		if IsThaiChar(runes[i]) {
			// Thai text - use appropriate segmenter based on mode
			segmentEnd := i
			for segmentEnd < n && (IsThaiChar(runes[segmentEnd]) || unicode.IsMark(runes[segmentEnd])) {
				segmentEnd++
			}

			thaiText := string(runes[i:segmentEnd])
			thaiTokens := t.segmentThai(thaiText)

			// Calculate byte offsets for Thai tokens
			thaiRunes := []rune(thaiText)
			thaiByteOffsets := make([]int, len(thaiRunes)+1)
			thaiBytePos := 0
			for j, r := range thaiRunes {
				thaiByteOffsets[j] = thaiBytePos
				thaiBytePos += len(string(r))
			}
			thaiByteOffsets[len(thaiRunes)] = thaiBytePos

			thaiRunePos := 0
			for _, tok := range thaiTokens {
				tokRunes := []rune(tok)
				tokLen := len(tokRunes)

				tokenText := tok
				if t.lowercase {
					tokenText = strings.ToLower(tok)
				}

				startByte := byteOffsets[startRune] + thaiByteOffsets[thaiRunePos]
				endByte := byteOffsets[startRune] + thaiByteOffsets[thaiRunePos+tokLen]

				tokens = append(tokens, Token{
					Text:     tokenText,
					Start:    startByte,
					End:      endByte,
					Position: position,
				})
				position++
				thaiRunePos += tokLen
			}

			i = segmentEnd
		} else if IsCJKLike(runes[i]) {
			// CJK text - split character by character
			for i < n && IsCJKLike(runes[i]) {
				tokenText := string(runes[i])
				if t.lowercase {
					tokenText = strings.ToLower(tokenText)
				}

				tokens = append(tokens, Token{
					Text:     tokenText,
					Start:    byteOffsets[i],
					End:      byteOffsets[i+1],
					Position: position,
				})
				position++
				i++
			}
		} else {
			// Regular text - collect word characters
			for i < n && IsWordChar(runes[i]) && !IsThaiChar(runes[i]) && !IsCJKLike(runes[i]) {
				i++
			}

			tokenText := string(runes[startRune:i])
			if t.lowercase {
				tokenText = strings.ToLower(tokenText)
			}

			tokens = append(tokens, Token{
				Text:     tokenText,
				Start:    byteOffsets[startRune],
				End:      byteOffsets[i],
				Position: position,
			})
			position++
		}
	}

	return tokens
}

// TokenizeToStrings splits text into token strings only.
func (t *TorusTokenizer) TokenizeToStrings(text string) []string {
	tokens := t.Tokenize(text)
	result := make([]string, len(tokens))
	for i, tok := range tokens {
		result[i] = tok.Text
	}
	return result
}

// NormalizeToken normalizes a token string.
func (t *TorusTokenizer) NormalizeToken(token string) string {
	if t.lowercase {
		return strings.ToLower(token)
	}
	return token
}

// TokenizeParallel tokenizes multiple texts in parallel using goroutines.
// Returns a slice of token slices in the same order as the input texts.
func (t *TorusTokenizer) TokenizeParallel(texts []string) [][]Token {
	results := make([][]Token, len(texts))

	// For small batches, process sequentially
	if len(texts) <= 4 {
		for i, text := range texts {
			results[i] = t.Tokenize(text)
		}
		return results
	}

	// Process in parallel
	done := make(chan struct{})
	for i, text := range texts {
		go func(idx int, txt string) {
			results[idx] = t.Tokenize(txt)
			done <- struct{}{}
		}(i, text)
	}

	// Wait for all goroutines to complete
	for range texts {
		<-done
	}

	return results
}

// TokenizeToStringsParallel tokenizes multiple texts in parallel and returns strings only.
func (t *TorusTokenizer) TokenizeToStringsParallel(texts []string) [][]string {
	results := make([][]string, len(texts))

	// For small batches, process sequentially
	if len(texts) <= 4 {
		for i, text := range texts {
			results[i] = t.TokenizeToStrings(text)
		}
		return results
	}

	// Process in parallel
	done := make(chan struct{})
	for i, text := range texts {
		go func(idx int, txt string) {
			results[idx] = t.TokenizeToStrings(txt)
			done <- struct{}{}
		}(i, text)
	}

	// Wait for all goroutines to complete
	for range texts {
		<-done
	}

	return results
}
