package torus

// Thai character classification for atomic segmentation.
// Atomic segmentation splits Thai text into the smallest possible units
// that can exist independently based on Thai orthographic rules.
//
// Optimized using finite state automaton with precomputed transitions.

// Character classes for the automaton
type charClass uint8

const (
	ccOther      charClass = iota // Non-Thai
	ccConsonant                   // ก-ฮ
	ccLeading                     // เ แ โ ไ ใ
	ccMiddle                      // ะ า ิ ี ึ ื ุ ู ํ ๅ ฺ
	ccTone                        // ็ ่ ้ ๊ ๋ ์
	ccSpecial                     // ฯ ๆ
	ccNumClasses                  // sentinel
)

// Automaton states
type atomState uint8

const (
	stStart    atomState = iota // Initial/between atoms
	stLeading                   // After leading vowel
	stConsBase                  // After consonant (base)
	stMiddle                    // After middle vowel
	stTone                      // After tone mark
	stNumStates                 // sentinel
)

// Actions
type atomAction uint8

const (
	actContinue   atomAction = iota // Continue building current atom
	actEmit                         // Emit current atom, start new
	actEmitSingle                   // Emit single char atom
	actAttach                       // Attach to previous atom
)

// Transition entry: next state + action
type transition struct {
	next   atomState
	action atomAction
}

// Precomputed transition table [state][charClass] -> transition
var atomTransitions = [stNumStates][ccNumClasses]transition{
	// stStart: initial state
	stStart: {
		ccOther:     {stStart, actEmit},       // Non-Thai: emit as-is
		ccConsonant: {stConsBase, actContinue}, // Start consonant atom
		ccLeading:   {stLeading, actContinue},  // Start leading vowel atom
		ccMiddle:    {stStart, actAttach},      // Orphan: attach to prev
		ccTone:      {stStart, actAttach},      // Orphan: attach to prev
		ccSpecial:   {stStart, actEmitSingle},  // Single char atom
	},
	// stLeading: after leading vowel, expecting consonant
	stLeading: {
		ccOther:     {stStart, actEmit},        // Emit leading alone, handle other
		ccConsonant: {stConsBase, actContinue}, // Leading + consonant
		ccLeading:   {stLeading, actEmit},      // Emit, start new leading
		ccMiddle:    {stStart, actEmit},        // Emit leading, attach middle
		ccTone:      {stStart, actEmit},        // Emit leading, attach tone
		ccSpecial:   {stStart, actEmit},        // Emit leading, emit special
	},
	// stConsBase: after consonant
	stConsBase: {
		ccOther:     {stStart, actEmit},        // Emit atom
		ccConsonant: {stConsBase, actEmit},     // Emit, start new consonant
		ccLeading:   {stLeading, actEmit},      // Emit, start leading
		ccMiddle:    {stMiddle, actContinue},   // Add middle vowel
		ccTone:      {stTone, actContinue},     // Add tone mark
		ccSpecial:   {stStart, actEmit},        // Emit, emit special
	},
	// stMiddle: after middle vowel
	stMiddle: {
		ccOther:     {stStart, actEmit},        // Emit atom
		ccConsonant: {stConsBase, actEmit},     // Could be final C in leading pattern - handle specially
		ccLeading:   {stLeading, actEmit},      // Emit, start leading
		ccMiddle:    {stMiddle, actContinue},   // Additional middle (าะ, ือ)
		ccTone:      {stTone, actContinue},     // Add tone
		ccSpecial:   {stStart, actEmit},        // Emit, emit special
	},
	// stTone: after tone mark
	stTone: {
		ccOther:     {stStart, actEmit},        // Emit atom
		ccConsonant: {stConsBase, actEmit},     // Emit, start new consonant
		ccLeading:   {stLeading, actEmit},      // Emit, start leading
		ccMiddle:    {stStart, actEmit},        // Emit (rare case)
		ccTone:      {stStart, actEmit},        // Emit (shouldn't happen)
		ccSpecial:   {stStart, actEmit},        // Emit, emit special
	},
}

// Precomputed character class lookup table for Thai range (0x0E00-0x0E7F)
// Index = rune - 0x0E00
var thaiCharClass = [128]charClass{
	// 0x0E00: Thai character ฀ (reserved)
	ccOther,
	// 0x0E01-0x0E2E: Consonants ก-ฮ (46 chars)
	ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, // 01-08
	ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, // 09-10
	ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, // 11-18
	ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, // 19-20
	ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, // 21-28
	ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, ccConsonant, // 29-2E
	// 0x0E2F: ฯ (Paiyannoi)
	ccSpecial,
	// 0x0E30: ะ (Sara A)
	ccMiddle,
	// 0x0E31: ั (Mai Han-Akat) - above vowel, treat as middle
	ccMiddle,
	// 0x0E32-0x0E33: า ำ
	ccMiddle, ccMiddle,
	// 0x0E34-0x0E39: ิ ี ึ ื ุ ู
	ccMiddle, ccMiddle, ccMiddle, ccMiddle, ccMiddle, ccMiddle,
	// 0x0E3A: ฺ (Phinthu)
	ccMiddle,
	// 0x0E3B-0x0E3F: reserved/currency
	ccOther, ccOther, ccOther, ccOther, ccOther,
	// 0x0E40-0x0E44: เ แ โ ใ ไ (leading vowels)
	ccLeading, ccLeading, ccLeading, ccLeading, ccLeading,
	// 0x0E45: ๅ (Lakkhangyao)
	ccMiddle,
	// 0x0E46: ๆ (Maiyamok)
	ccSpecial,
	// 0x0E47-0x0E4C: ็ ่ ้ ๊ ๋ ์ (tone marks)
	ccTone, ccTone, ccTone, ccTone, ccTone, ccTone,
	// 0x0E4D: ํ (Nikhahit)
	ccMiddle,
	// 0x0E4E: ๎ (Yamakkan)
	ccOther,
	// 0x0E4F: ๏ (Fongman)
	ccOther,
	// 0x0E50-0x0E59: Thai digits ๐-๙
	ccOther, ccOther, ccOther, ccOther, ccOther, ccOther, ccOther, ccOther, ccOther, ccOther,
	// 0x0E5A-0x0E5B: ๚ ๛
	ccOther, ccOther,
	// 0x0E5C-0x0E7F: reserved
	ccOther, ccOther, ccOther, ccOther, ccOther, ccOther, ccOther, ccOther,
	ccOther, ccOther, ccOther, ccOther, ccOther, ccOther, ccOther, ccOther,
	ccOther, ccOther, ccOther, ccOther, ccOther, ccOther, ccOther, ccOther,
	ccOther, ccOther, ccOther, ccOther, ccOther, ccOther, ccOther, ccOther,
	ccOther, ccOther, ccOther, ccOther,
}

// classifyThai returns the character class for a rune.
// Uses lookup table for Thai range, O(1) with no branching.
func classifyThai(r rune) charClass {
	if r >= 0x0E00 && r <= 0x0E7F {
		return thaiCharClass[r-0x0E00]
	}
	return ccOther
}

// AtomicSegmenter segments Thai text into atomic units using FSA.
type AtomicSegmenter struct{}

// NewAtomicSegmenter creates a new atomic segmenter.
func NewAtomicSegmenter() *AtomicSegmenter {
	return &AtomicSegmenter{}
}

// Mai Han-Akat (U+0E31) - special vowel that forms clusters with final consonant
const maiHanAkat = 0x0E31

// Segment splits Thai text into atomic units using finite state automaton.
func (s *AtomicSegmenter) Segment(text string) []string {
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return nil
	}

	// Pre-allocate result
	atoms := make([]string, 0, (n+1)/2)

	state := stStart
	atomStart := 0
	hadLeading := false    // Track if current atom started with leading vowel
	hadMaiHanAkat := false // Track if current atom has ั

	for i := 0; i < n; i++ {
		r := runes[i]
		class := classifyThai(r)

		// Handle non-Thai sequences specially
		if class == ccOther {
			// Emit current atom if any
			if i > atomStart {
				atoms = append(atoms, string(runes[atomStart:i]))
			}
			// Collect non-Thai sequence
			start := i
			for i < n && classifyThai(runes[i]) == ccOther {
				i++
			}
			atoms = append(atoms, string(runes[start:i]))
			i-- // Will be incremented by loop
			state = stStart
			atomStart = i + 1
			hadLeading = false
			hadMaiHanAkat = false
			continue
		}

		trans := atomTransitions[state][class]

		switch trans.action {
		case actContinue:
			if state == stStart {
				atomStart = i
				hadLeading = (class == ccLeading)
				hadMaiHanAkat = false
			}
			// Track Mai Han-Akat
			if r == maiHanAkat {
				hadMaiHanAkat = true
			}

		case actEmit:
			// Emit current atom
			if i > atomStart {
				// Special case 1: final consonant in leading vowel pattern
				// Special case 2: {consonant} ั [{tone}] {consonant} pattern
				if state == stMiddle && class == ccConsonant && hadLeading {
					// Include this consonant in current atom
					atoms = append(atoms, string(runes[atomStart:i+1]))
					atomStart = i + 1
					state = stStart
					hadLeading = false
					hadMaiHanAkat = false
					continue
				}
				if (state == stMiddle || state == stTone) && class == ccConsonant && hadMaiHanAkat {
					// Pattern: C + ั + [tone] + C -> single atom (e.g., จัน, นั่ง)
					atoms = append(atoms, string(runes[atomStart:i+1]))
					atomStart = i + 1
					state = stStart
					hadLeading = false
					hadMaiHanAkat = false
					continue
				}
				atoms = append(atoms, string(runes[atomStart:i]))
			}
			atomStart = i
			hadLeading = (class == ccLeading)
			hadMaiHanAkat = false

		case actEmitSingle:
			// Emit current atom if any
			if i > atomStart {
				atoms = append(atoms, string(runes[atomStart:i]))
			}
			// Emit single char
			atoms = append(atoms, string(r))
			atomStart = i + 1
			hadLeading = false
			hadMaiHanAkat = false

		case actAttach:
			// Attach to previous atom
			if len(atoms) > 0 {
				atoms[len(atoms)-1] += string(r)
			} else {
				atoms = append(atoms, string(r))
			}
			atomStart = i + 1
			hadLeading = false
			hadMaiHanAkat = false
		}

		state = trans.next
	}

	// Emit final atom
	if atomStart < n {
		atoms = append(atoms, string(runes[atomStart:]))
	}

	return atoms
}

// SegmentToTokens segments Thai text into atomic tokens.
func (s *AtomicSegmenter) SegmentToTokens(text string) []string {
	return s.Segment(text)
}

// Legacy functions for compatibility
func isThaiConsonant(r rune) bool {
	return r >= 0x0E01 && r <= 0x0E2E
}

func isThaiLeadingVowel(r rune) bool {
	return r >= 0x0E40 && r <= 0x0E44
}

func isThaiMiddleVowel(r rune) bool {
	return classifyThai(r) == ccMiddle
}

func isThaiToneMark(r rune) bool {
	return r >= 0x0E47 && r <= 0x0E4C
}

func isThaiSpecialAtom(r rune) bool {
	return r == 0x0E2F || r == 0x0E46
}
