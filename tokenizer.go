// Package torus provides Thai language tokenization using Trie-based longest match algorithm.
//
// TORUS is designed specifically for Thai language tokenization with support for:
//   - Thai text segmentation using dictionary-based longest match
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
package torus

import (
	"strings"
	"unicode"
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

// TorusTokenizer implements the Tokenizer interface with Thai and CJK support.
type TorusTokenizer struct {
	thaiSegmenter *ThaiSegmenter
	lowercase     bool
}

// Option is a function that configures the tokenizer.
type Option func(*TorusTokenizer)

// WithLowercase sets whether to lowercase tokens (default: true).
func WithLowercase(lowercase bool) Option {
	return func(t *TorusTokenizer) {
		t.lowercase = lowercase
	}
}

// New creates a new TorusTokenizer with default settings.
func New(opts ...Option) *TorusTokenizer {
	t := &TorusTokenizer{
		thaiSegmenter: NewThaiSegmenter(),
		lowercase:     true,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
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
			// Thai text - use segmenter
			segmentEnd := i
			for segmentEnd < n && (IsThaiChar(runes[segmentEnd]) || unicode.IsMark(runes[segmentEnd])) {
				segmentEnd++
			}

			thaiText := string(runes[i:segmentEnd])
			thaiTokens := t.thaiSegmenter.SegmentToTokens(thaiText)

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
