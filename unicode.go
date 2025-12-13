package torus

import "unicode"

// CJK Unicode ranges
const (
	// CJK Unified Ideographs
	cjkUnifiedStart = 0x4E00
	cjkUnifiedEnd   = 0x9FFF

	// CJK Unified Ideographs Extension A
	cjkExtAStart = 0x3400
	cjkExtAEnd   = 0x4DBF

	// CJK Unified Ideographs Extension B
	cjkExtBStart = 0x20000
	cjkExtBEnd   = 0x2A6DF

	// CJK Unified Ideographs Extension C
	cjkExtCStart = 0x2A700
	cjkExtCEnd   = 0x2B73F

	// CJK Unified Ideographs Extension D
	cjkExtDStart = 0x2B740
	cjkExtDEnd   = 0x2B81F

	// CJK Unified Ideographs Extension E
	cjkExtEStart = 0x2B820
	cjkExtEEnd   = 0x2CEAF

	// CJK Unified Ideographs Extension F
	cjkExtFStart = 0x2CEB0
	cjkExtFEnd   = 0x2EBEF

	// CJK Compatibility Ideographs
	cjkCompatStart = 0xF900
	cjkCompatEnd   = 0xFAFF

	// Hiragana
	hiraganaStart = 0x3040
	hiraganaEnd   = 0x309F

	// Katakana
	katakanaStart = 0x30A0
	katakanaEnd   = 0x30FF

	// Katakana Phonetic Extensions
	katakanaExtStart = 0x31F0
	katakanaExtEnd   = 0x31FF

	// Hangul Syllables (Korean)
	hangulStart = 0xAC00
	hangulEnd   = 0xD7AF

	// Hangul Jamo
	hangulJamoStart = 0x1100
	hangulJamoEnd   = 0x11FF

	// Hangul Compatibility Jamo
	hangulCompatStart = 0x3130
	hangulCompatEnd   = 0x318F

	// Bopomofo
	bopomofoStart = 0x3100
	bopomofoEnd   = 0x312F

	// Bopomofo Extended
	bopomofoExtStart = 0x31A0
	bopomofoExtEnd   = 0x31BF
)

// IsCJKChar returns true if the rune is a CJK ideograph (Chinese character).
func IsCJKChar(r rune) bool {
	return (r >= cjkUnifiedStart && r <= cjkUnifiedEnd) ||
		(r >= cjkExtAStart && r <= cjkExtAEnd) ||
		(r >= cjkExtBStart && r <= cjkExtBEnd) ||
		(r >= cjkExtCStart && r <= cjkExtCEnd) ||
		(r >= cjkExtDStart && r <= cjkExtDEnd) ||
		(r >= cjkExtEStart && r <= cjkExtEEnd) ||
		(r >= cjkExtFStart && r <= cjkExtFEnd) ||
		(r >= cjkCompatStart && r <= cjkCompatEnd)
}

// IsJapaneseKana returns true if the rune is Hiragana or Katakana.
func IsJapaneseKana(r rune) bool {
	return (r >= hiraganaStart && r <= hiraganaEnd) ||
		(r >= katakanaStart && r <= katakanaEnd) ||
		(r >= katakanaExtStart && r <= katakanaExtEnd)
}

// IsKoreanHangul returns true if the rune is a Korean Hangul character.
func IsKoreanHangul(r rune) bool {
	return (r >= hangulStart && r <= hangulEnd) ||
		(r >= hangulJamoStart && r <= hangulJamoEnd) ||
		(r >= hangulCompatStart && r <= hangulCompatEnd)
}

// IsBopomofo returns true if the rune is a Bopomofo character.
func IsBopomofo(r rune) bool {
	return (r >= bopomofoStart && r <= bopomofoEnd) ||
		(r >= bopomofoExtStart && r <= bopomofoExtEnd)
}

// IsCJKLike returns true if the rune belongs to any CJK-like script
// that should be split character by character.
func IsCJKLike(r rune) bool {
	return IsCJKChar(r) || IsJapaneseKana(r) || IsKoreanHangul(r) || IsBopomofo(r)
}

// ContainsCJK checks if text contains any CJK-like characters.
func ContainsCJK(text string) bool {
	for _, r := range text {
		if IsCJKLike(r) {
			return true
		}
	}
	return false
}

// IsWordChar returns true if the rune is a word character
// (letter, digit, mark, or underscore).
func IsWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) || r == '_'
}
