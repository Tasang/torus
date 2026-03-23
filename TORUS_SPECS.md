# TORUS Tokenization Library - Complete Implementation Specification

> **Target**: Implementation with Claude Sonnet-4.6 in ANY programming language with ZERO external dependencies.
>
> This document provides every detail needed to implement a fully compatible Torus tokenizer from scratch. All algorithms, data structures, constants, lookup tables, transition tables, and test vectors are specified exhaustively.

---

## Table of Contents

1. [Overview](#1-overview)
2. [Architecture](#2-architecture)
3. [Data Structures](#3-data-structures)
4. [Character Classification](#4-character-classification)
5. [Trie (Prefix Tree)](#5-trie-prefix-tree)
6. [Dictionary](#6-dictionary)
7. [Dict Mode - Longest Match Segmentation](#7-dict-mode---longest-match-segmentation)
8. [Atomic Mode - FSA Segmentation](#8-atomic-mode---fsa-segmentation)
9. [Combined Mode - Dict + Atomic Validation](#9-combined-mode---dict--atomic-validation)
10. [Main Tokenizer](#10-main-tokenizer)
11. [CJK and Multilingual Support](#11-cjk-and-multilingual-support)
12. [Parallel Processing](#12-parallel-processing)
13. [Test Vectors](#13-test-vectors)
14. [Implementation Checklist](#14-implementation-checklist)

---

## 1. Overview

**TORUS** is a multilingual tokenizer with specialized support for Thai language segmentation. It handles:

- **Thai text**: Dictionary-based and/or atomic (orthographic unit) segmentation
- **CJK text** (Chinese/Japanese/Korean): Character-by-character splitting
- **Latin/other scripts**: Unicode word-boundary-based tokenization
- **Mixed text**: Seamlessly switches strategy per character

**Three tokenization modes**:

| Mode | Enum Value | Description |
|------|-----------|-------------|
| `ModeDict` | 0 (default) | Trie-based longest-match dictionary lookup |
| `ModeAtomic` | 1 | FSA-based splitting into smallest valid Thai orthographic units |
| `ModeCombined` | 2 | Dict segmentation with atomic validity guarantee |

**Configuration options**:
- `lowercase` (default: `true`) - Whether to lowercase output tokens
- `mode` (default: `ModeDict`) - Tokenization mode

---

## 2. Architecture

```
TorusTokenizer
  |
  |-- mode: Mode (Dict|Atomic|Combined)
  |-- lowercase: bool
  |-- thaiSegmenter: ThaiSegmenter
  |     |-- trie: Trie (loaded from dictionary file)
  |-- atomicSegmenter: AtomicSegmenter (stateless, uses global FSA tables)
  |
  |-- Tokenize(text) -> []Token        // Full tokenization with byte offsets
  |-- TokenizeToStrings(text) -> []str  // Token strings only
  |-- NormalizeToken(str) -> str        // Lowercasing
  |-- TokenizeParallel([]text) -> [][]Token
  |-- TokenizeToStringsParallel([]text) -> [][]str
```

**Token structure**:
```
Token {
    Text:     string  // Normalized token text (lowercased if enabled)
    Start:    int     // Byte offset in original text (inclusive)
    End:      int     // Byte offset in original text (exclusive)
    Position: int     // 0-indexed sequential token position
}
```

---

## 3. Data Structures

### 3.1 TrieNode

```
TrieNode {
    children: Map<char, TrieNode>  // Child nodes keyed by character (rune/codepoint)
    isEnd:    bool                 // True if this node marks the end of a complete word
}
```

### 3.2 Trie

```
Trie {
    root:   TrieNode  // Root node (never null, children map initialized)
    maxLen: int        // Maximum word length in characters (updated on insert)
}
```

### 3.3 ThaiSegmenter

```
ThaiSegmenter {
    trie: Trie  // Dictionary trie loaded from words_th.txt
}
```

### 3.4 AtomicSegmenter

```
AtomicSegmenter {}  // Stateless - all data is in global lookup tables
```

---

## 4. Character Classification

### 4.1 Thai Unicode Block: U+0E00 - U+0E7F (128 codepoints)

#### Helper Functions

```
IsThaiChar(r) -> bool:
    return r >= 0x0E00 && r <= 0x0E7F

isThaiConsonant(r) -> bool:
    return r >= 0x0E01 && r <= 0x0E2E

isThaiLeadingVowel(r) -> bool:
    return r >= 0x0E40 && r <= 0x0E44

isThaiMiddleVowel(r) -> bool:
    return classifyThai(r) == ccMiddle   // Uses lookup table

isThaiToneMark(r) -> bool:
    return r >= 0x0E47 && r <= 0x0E4C

isThaiSpecialAtom(r) -> bool:
    return r == 0x0E2F || r == 0x0E46
```

#### Thai Following Vowels (cannot start a token in Dict mode)

These characters cannot start a Thai word and must follow a consonant:

```
0x0E30  ะ   Sara A (short A)
0x0E32  า   Sara Aa (long A)
0x0E34  ิ   Sara I
0x0E35  ี   Sara Ii
0x0E36  ึ   Sara Ue
0x0E37  ื   Sara Uee
0x0E38  ุ   Sara U
0x0E39  ู   Sara Uu
0x0E45  ๅ   Lakkhangyao
0x0E47  ็   Maitaikhu (short mark)
0x0E4C  ์   Thanthakhat (silent mark)
0x0E46  ๆ   Maiyamok (repetition)
0x0E4D  ํ   Nikhahit (anusvara)
0x0E3A  ฺ   Phinthu
```

#### Thai Tone Marks (also cannot start a token)

```
0x0E48  ่   Mai Ek (falling tone)
0x0E49  ้   Mai Tho (high tone)
0x0E4A  ๊   Mai Tri (rising tone)
0x0E4B  ๋   Mai Chattawa (extra high tone)
```

#### CanStartThaiToken

```
CanStartThaiToken(r) -> bool:
    return r is NOT in thaiFollowingVowels AND r is NOT a thaiToneMark
```

#### IsThaiCombiningMark

```
IsThaiCombiningMark(r) -> bool:
    return r is in thaiFollowingVowels
        OR r is a thaiToneMark
        OR r is a Unicode Mark (category M)
```

### 4.2 Character Classes for Atomic FSA

Enum `charClass` (uint8):

| Value | Name | Description | Codepoints |
|-------|------|-------------|------------|
| 0 | `ccOther` | Non-Thai or unclassified | See table below |
| 1 | `ccConsonant` | Thai consonants | U+0E01 - U+0E2E (46 chars) |
| 2 | `ccLeading` | Leading vowels | U+0E40 - U+0E44 (5 chars: เ แ โ ใ ไ) |
| 3 | `ccMiddle` | Middle/following vowels | See table below |
| 4 | `ccTone` | Tone marks | U+0E47 - U+0E4C (6 chars: ็ ่ ้ ๊ ๋ ์) |
| 5 | `ccSpecial` | Special single-char atoms | U+0E2F (ฯ), U+0E46 (ๆ) |

### 4.3 Complete Character Class Lookup Table

This is a **128-entry array** indexed by `(codepoint - 0x0E00)`. Each entry is a `charClass` value.

```
Index  Codepoint  Char  Class
-----  ---------  ----  -----------
0x00   U+0E00     ฀     ccOther (0)      // Reserved
0x01   U+0E01     ก     ccConsonant (1)
0x02   U+0E02     ข     ccConsonant (1)
0x03   U+0E03     ฃ     ccConsonant (1)
0x04   U+0E04     ค     ccConsonant (1)
0x05   U+0E05     ฅ     ccConsonant (1)
0x06   U+0E06     ฆ     ccConsonant (1)
0x07   U+0E07     ง     ccConsonant (1)
0x08   U+0E08     จ     ccConsonant (1)
0x09   U+0E09     ฉ     ccConsonant (1)
0x0A   U+0E0A     ช     ccConsonant (1)
0x0B   U+0E0B     ซ     ccConsonant (1)
0x0C   U+0E0C     ฌ     ccConsonant (1)
0x0D   U+0E0D     ญ     ccConsonant (1)
0x0E   U+0E0E     ฎ     ccConsonant (1)
0x0F   U+0E0F     ฏ     ccConsonant (1)
0x10   U+0E10     ฐ     ccConsonant (1)
0x11   U+0E11     ฑ     ccConsonant (1)
0x12   U+0E12     ฒ     ccConsonant (1)
0x13   U+0E13     ณ     ccConsonant (1)
0x14   U+0E14     ด     ccConsonant (1)
0x15   U+0E15     ต     ccConsonant (1)
0x16   U+0E16     ถ     ccConsonant (1)
0x17   U+0E17     ท     ccConsonant (1)
0x18   U+0E18     ธ     ccConsonant (1)
0x19   U+0E19     น     ccConsonant (1)
0x1A   U+0E1A     บ     ccConsonant (1)
0x1B   U+0E1B     ป     ccConsonant (1)
0x1C   U+0E1C     ผ     ccConsonant (1)
0x1D   U+0E1D     ฝ     ccConsonant (1)
0x1E   U+0E1E     พ     ccConsonant (1)
0x1F   U+0E1F     ฟ     ccConsonant (1)
0x20   U+0E20     ภ     ccConsonant (1)
0x21   U+0E21     ม     ccConsonant (1)
0x22   U+0E22     ย     ccConsonant (1)
0x23   U+0E23     ร     ccConsonant (1)
0x24   U+0E24     ฤ     ccConsonant (1)
0x25   U+0E25     ล     ccConsonant (1)
0x26   U+0E26     ฦ     ccConsonant (1)
0x27   U+0E27     ว     ccConsonant (1)
0x28   U+0E28     ศ     ccConsonant (1)
0x29   U+0E29     ษ     ccConsonant (1)
0x2A   U+0E2A     ส     ccConsonant (1)
0x2B   U+0E2B     ห     ccConsonant (1)
0x2C   U+0E2C     ฬ     ccConsonant (1)
0x2D   U+0E2D     อ     ccConsonant (1)
0x2E   U+0E2E     ฮ     ccConsonant (1)
0x2F   U+0E2F     ฯ     ccSpecial (5)    // Paiyannoi
0x30   U+0E30     ะ     ccMiddle (3)     // Sara A
0x31   U+0E31     ั     ccMiddle (3)     // Mai Han-Akat
0x32   U+0E32     า     ccMiddle (3)     // Sara Aa
0x33   U+0E33     ำ     ccMiddle (3)     // Sara Am
0x34   U+0E34     ิ     ccMiddle (3)     // Sara I
0x35   U+0E35     ี     ccMiddle (3)     // Sara Ii
0x36   U+0E36     ึ     ccMiddle (3)     // Sara Ue
0x37   U+0E37     ื     ccMiddle (3)     // Sara Uee
0x38   U+0E38     ุ     ccMiddle (3)     // Sara U
0x39   U+0E39     ู     ccMiddle (3)     // Sara Uu
0x3A   U+0E3A     ฺ     ccMiddle (3)     // Phinthu
0x3B   U+0E3B     ​     ccOther (0)      // Reserved
0x3C   U+0E3C     ​     ccOther (0)      // Reserved
0x3D   U+0E3D     ​     ccOther (0)      // Reserved
0x3E   U+0E3E     ​     ccOther (0)      // Reserved
0x3F   U+0E3F     ฿     ccOther (0)      // Thai Baht currency
0x40   U+0E40     เ     ccLeading (2)
0x41   U+0E41     แ     ccLeading (2)
0x42   U+0E42     โ     ccLeading (2)
0x43   U+0E43     ใ     ccLeading (2)
0x44   U+0E44     ไ     ccLeading (2)
0x45   U+0E45     ๅ     ccMiddle (3)     // Lakkhangyao
0x46   U+0E46     ๆ     ccSpecial (5)    // Maiyamok
0x47   U+0E47     ็     ccTone (4)       // Maitaikhu
0x48   U+0E48     ่     ccTone (4)       // Mai Ek
0x49   U+0E49     ้     ccTone (4)       // Mai Tho
0x4A   U+0E4A     ๊     ccTone (4)       // Mai Tri
0x4B   U+0E4B     ๋     ccTone (4)       // Mai Chattawa
0x4C   U+0E4C     ์     ccTone (4)       // Thanthakhat
0x4D   U+0E4D     ํ     ccMiddle (3)     // Nikhahit
0x4E   U+0E4E     ๎     ccOther (0)      // Yamakkan
0x4F   U+0E4F     ๏     ccOther (0)      // Fongman
0x50   U+0E50     ๐     ccOther (0)      // Thai digit 0
0x51   U+0E51     ๑     ccOther (0)      // Thai digit 1
0x52   U+0E52     ๒     ccOther (0)      // Thai digit 2
0x53   U+0E53     ๓     ccOther (0)      // Thai digit 3
0x54   U+0E54     ๔     ccOther (0)      // Thai digit 4
0x55   U+0E55     ๕     ccOther (0)      // Thai digit 5
0x56   U+0E56     ๖     ccOther (0)      // Thai digit 6
0x57   U+0E57     ๗     ccOther (0)      // Thai digit 7
0x58   U+0E58     ๘     ccOther (0)      // Thai digit 8
0x59   U+0E59     ๙     ccOther (0)      // Thai digit 9
0x5A   U+0E5A     ๚     ccOther (0)
0x5B   U+0E5B     ๛     ccOther (0)
0x5C-0x7F         ...   ccOther (0)      // All reserved (36 entries)
```

**As a flat array** (for direct copy-paste into implementations):

```
thaiCharClass[128] = {
  0,                                          // 0x00: reserved
  1, 1, 1, 1, 1, 1, 1, 1,                    // 0x01-0x08: consonants
  1, 1, 1, 1, 1, 1, 1, 1,                    // 0x09-0x10: consonants
  1, 1, 1, 1, 1, 1, 1, 1,                    // 0x11-0x18: consonants
  1, 1, 1, 1, 1, 1, 1, 1,                    // 0x19-0x20: consonants
  1, 1, 1, 1, 1, 1, 1, 1,                    // 0x21-0x28: consonants
  1, 1, 1, 1, 1, 1,                          // 0x29-0x2E: consonants
  5,                                          // 0x2F: ฯ special
  3,                                          // 0x30: ะ middle
  3,                                          // 0x31: ั middle (Mai Han-Akat)
  3, 3,                                       // 0x32-0x33: า ำ middle
  3, 3, 3, 3, 3, 3,                          // 0x34-0x39: ิ ี ึ ื ุ ู middle
  3,                                          // 0x3A: ฺ middle (Phinthu)
  0, 0, 0, 0, 0,                             // 0x3B-0x3F: reserved/currency
  2, 2, 2, 2, 2,                             // 0x40-0x44: เ แ โ ใ ไ leading
  3,                                          // 0x45: ๅ middle (Lakkhangyao)
  5,                                          // 0x46: ๆ special
  4, 4, 4, 4, 4, 4,                          // 0x47-0x4C: ็ ่ ้ ๊ ๋ ์ tone
  3,                                          // 0x4D: ํ middle (Nikhahit)
  0, 0,                                       // 0x4E-0x4F: other
  0, 0, 0, 0, 0, 0, 0, 0, 0, 0,             // 0x50-0x59: Thai digits
  0, 0,                                       // 0x5A-0x5B: other
  0, 0, 0, 0, 0, 0, 0, 0,                    // 0x5C-0x63: reserved
  0, 0, 0, 0, 0, 0, 0, 0,                    // 0x64-0x6B: reserved
  0, 0, 0, 0, 0, 0, 0, 0,                    // 0x6C-0x73: reserved
  0, 0, 0, 0, 0, 0, 0, 0,                    // 0x74-0x7B: reserved
  0, 0, 0, 0                                  // 0x7C-0x7F: reserved
}
```

**Classification function**:
```
classifyThai(r) -> charClass:
    if r >= 0x0E00 AND r <= 0x0E7F:
        return thaiCharClass[r - 0x0E00]
    return ccOther
```

---

## 5. Trie (Prefix Tree)

### 5.1 Operations

#### Insert

```
Trie.Insert(word):
    if word is empty: return

    runes = toCodepoints(word)
    if len(runes) > this.maxLen:
        this.maxLen = len(runes)

    node = this.root
    for each r in runes:
        if r not in node.children:
            node.children[r] = new TrieNode(children={}, isEnd=false)
        node = node.children[r]
    node.isEnd = true
```

#### Contains

```
Trie.Contains(word) -> bool:
    node = this.root
    for each r in word:
        if node.children is empty: return false
        if r not in node.children: return false
        node = node.children[r]
    return node.isEnd
```

#### HasPrefix

```
Trie.HasPrefix(prefix) -> bool:
    node = this.root
    for each r in prefix:
        if node.children is empty: return false
        if r not in node.children: return false
        node = node.children[r]
    return true
```

#### LongestMatch

```
Trie.LongestMatch(runes) -> int:
    node = this.root
    longestMatch = 0

    for i = 0; i < len(runes); i++:
        if node.children is empty: break
        r = runes[i]
        if r not in node.children: break
        node = node.children[r]
        if node.isEnd:
            longestMatch = i + 1

    return longestMatch
```

#### AllMatches

Returns all dictionary word lengths matching from the start of runes, in ascending order.
Used by Combined mode for boundary-aware backtracking.

```
Trie.AllMatches(runes) -> []int:
    matches = []
    node = this.root

    for i = 0; i < len(runes); i++:
        if node.children is empty: break
        r = runes[i]
        if r not in node.children: break
        node = node.children[r]
        if node.isEnd:
            matches.append(i + 1)

    return matches
```

**Example**: If trie contains "go" and "golang", then `AllMatches("golang is great")` returns `[2, 6]`.

---

## 6. Dictionary

### 6.1 Format

Plain text file `words_th.txt`:
- One Thai word per line
- UTF-8 encoded
- 24,238 words total
- Leading/trailing whitespace trimmed on load
- Empty lines skipped

### 6.2 Loading

```
ThaiSegmenter.loadDictionary():
    // Open and read words_th.txt (embedded or from filesystem)
    for each line in file:
        word = trim(line)
        if word is not empty:
            this.trie.Insert(word)

    // If dictionary failed to load (maxLen == 0), use fallback
    if this.trie.maxLen == 0:
        this.loadFallbackDictionary()
```

### 6.3 Fallback Dictionary

If the main dictionary file cannot be loaded, use these 20 common Thai words:

```
การ, ที่, และ, ใน, มี, เป็น, ได้, จะ, ว่า, ของ,
ไม่, ให้, นี้, จาก, ก็, กับ, แล้ว, เมื่อ, ถ้า, ยัง
```

---

## 7. Dict Mode - Longest Match Segmentation

### 7.1 Segment (Raw)

```
ThaiSegmenter.Segment(text) -> []string:
    runes = toCodepoints(text)
    n = len(runes)
    if n == 0: return null

    segments = []
    i = 0

    while i < n:
        // Skip whitespace
        if isWhitespace(runes[i]):
            i++
            continue

        // Non-Thai: collect sequence
        if NOT IsThaiChar(runes[i]):
            start = i
            while i < n AND NOT IsThaiChar(runes[i]) AND NOT isWhitespace(runes[i]):
                i++
            segments.append(string(runes[start:i]))
            continue

        // Thai: try longest match
        matchLen = this.trie.LongestMatch(runes[i:])

        if matchLen > 0:
            // Dictionary match found
            segments.append(string(runes[i : i+matchLen]))
            i += matchLen
        else:
            // No match: take single character + combining marks
            clusterEnd = i + 1
            while clusterEnd < n AND IsThaiCombiningMark(runes[clusterEnd]):
                clusterEnd++
            segments.append(string(runes[i:clusterEnd]))
            i = clusterEnd

    return segments
```

### 7.2 SegmentToTokens (with orphan merging)

```
ThaiSegmenter.SegmentToTokens(text) -> []string:
    segments = this.Segment(text)
    if segments is empty: return null

    tokens = []
    for each seg in segments:
        seg = trim(seg)
        if seg is empty: continue

        runes = toCodepoints(seg)
        if len(runes) > 0 AND NOT CanStartThaiToken(runes[0]):
            // Cannot start a token - merge with previous
            if len(tokens) > 0:
                tokens[last] += seg
                continue

        tokens.append(seg)

    return tokens
```

---

## 8. Atomic Mode - FSA Segmentation

### 8.1 FSA States

| Value | Name | Description |
|-------|------|-------------|
| 0 | `stStart` | Initial state / between atoms |
| 1 | `stLeading` | After leading vowel, expecting consonant |
| 2 | `stConsBase` | After consonant (base character) |
| 3 | `stMiddle` | After middle vowel |
| 4 | `stTone` | After tone mark |

### 8.2 FSA Actions

| Value | Name | Description |
|-------|------|-------------|
| 0 | `actContinue` | Continue building current atom |
| 1 | `actEmit` | Emit current atom, start new one at current char |
| 2 | `actEmitSingle` | Emit current atom + emit current char as standalone atom |
| 3 | `actAttach` | Attach current char to previous atom (orphan handling) |

### 8.3 Transition Table

Each entry is `{nextState, action}`. Table is indexed by `[currentState][charClass]`.

```
atomTransitions[5][6] = {

    // stStart (0):
    {
        {stStart,    actEmit},        // ccOther:     emit as-is
        {stConsBase, actContinue},     // ccConsonant: start consonant atom
        {stLeading,  actContinue},     // ccLeading:   start leading vowel atom
        {stStart,    actAttach},       // ccMiddle:    orphan - attach to prev
        {stStart,    actAttach},       // ccTone:      orphan - attach to prev
        {stStart,    actEmitSingle},   // ccSpecial:   single char atom
    },

    // stLeading (1):
    {
        {stStart,    actEmit},         // ccOther:     emit leading alone
        {stConsBase, actContinue},     // ccConsonant: leading + consonant
        {stLeading,  actEmit},         // ccLeading:   emit, start new leading
        {stStart,    actEmit},         // ccMiddle:    emit leading, attach middle
        {stStart,    actEmit},         // ccTone:      emit leading, attach tone
        {stStart,    actEmit},         // ccSpecial:   emit leading, emit special
    },

    // stConsBase (2):
    {
        {stStart,    actEmit},         // ccOther:     emit atom
        {stConsBase, actEmit},         // ccConsonant: emit, start new consonant
        {stLeading,  actEmit},         // ccLeading:   emit, start leading
        {stMiddle,   actContinue},     // ccMiddle:    add middle vowel
        {stTone,     actContinue},     // ccTone:      add tone mark
        {stStart,    actEmit},         // ccSpecial:   emit, emit special
    },

    // stMiddle (3):
    {
        {stStart,    actEmit},         // ccOther:     emit atom
        {stConsBase, actEmit},         // ccConsonant: emit (special cases below)
        {stLeading,  actEmit},         // ccLeading:   emit, start leading
        {stMiddle,   actContinue},     // ccMiddle:    additional middle (าะ combo)
        {stTone,     actContinue},     // ccTone:      add tone
        {stStart,    actEmit},         // ccSpecial:   emit, emit special
    },

    // stTone (4):
    {
        {stStart,    actEmit},         // ccOther:     emit atom
        {stConsBase, actEmit},         // ccConsonant: emit (special cases below)
        {stLeading,  actEmit},         // ccLeading:   emit, start leading
        {stStart,    actEmit},         // ccMiddle:    emit (rare)
        {stStart,    actEmit},         // ccTone:      emit (rare)
        {stStart,    actEmit},         // ccSpecial:   emit, emit special
    },
}
```

### 8.4 FSA Algorithm

**Constant**:
```
maiHanAkat = 0x0E31  // ั
```

**Algorithm**:

```
AtomicSegmenter.Segment(text) -> []string:
    runes = toCodepoints(text)
    n = len(runes)
    if n == 0: return null

    atoms = []  // pre-allocate ~(n+1)/2 capacity
    state = stStart
    atomStart = 0
    hadLeading = false
    hadMaiHanAkat = false

    for i = 0; i < n; i++:
        r = runes[i]
        class = classifyThai(r)

        // === Handle non-Thai sequences specially ===
        if class == ccOther:
            // Emit current atom if any
            if i > atomStart:
                atoms.append(string(runes[atomStart:i]))
            // Collect entire non-Thai sequence
            start = i
            while i < n AND classifyThai(runes[i]) == ccOther:
                i++
            atoms.append(string(runes[start:i]))
            i--  // Will be incremented by loop
            state = stStart
            atomStart = i + 1
            hadLeading = false
            hadMaiHanAkat = false
            continue

        // === Look up transition ===
        trans = atomTransitions[state][class]

        switch trans.action:

        case actContinue:
            if state == stStart:
                atomStart = i
                hadLeading = (class == ccLeading)
                hadMaiHanAkat = false
            // Track Mai Han-Akat
            if r == maiHanAkat:
                hadMaiHanAkat = true

        case actEmit:
            if i > atomStart:
                // SPECIAL CASE 1: Leading vowel pattern final consonant
                // Pattern: {leading} + {consonant} + {middle} + {consonant}
                // When in stMiddle, receiving ccConsonant, and hadLeading:
                //   Include this consonant in current atom
                if state == stMiddle AND class == ccConsonant AND hadLeading:
                    atoms.append(string(runes[atomStart : i+1]))
                    atomStart = i + 1
                    state = stStart
                    hadLeading = false
                    hadMaiHanAkat = false
                    continue

                // SPECIAL CASE 2: Mai Han-Akat with final consonant
                // Pattern: {C} + ั + [{tone}] + {C}
                // When in stMiddle/stTone, receiving ccConsonant, and hadMaiHanAkat:
                //   Include this consonant in current atom
                if (state == stMiddle OR state == stTone) AND class == ccConsonant AND hadMaiHanAkat:
                    atoms.append(string(runes[atomStart : i+1]))
                    atomStart = i + 1
                    state = stStart
                    hadLeading = false
                    hadMaiHanAkat = false
                    continue

                // Normal emit
                atoms.append(string(runes[atomStart:i]))

            atomStart = i
            hadLeading = (class == ccLeading)
            hadMaiHanAkat = false

        case actEmitSingle:
            // Emit current atom if any
            if i > atomStart:
                atoms.append(string(runes[atomStart:i]))
            // Emit single character
            atoms.append(string(r))
            atomStart = i + 1
            hadLeading = false
            hadMaiHanAkat = false

        case actAttach:
            // Attach to previous atom
            if len(atoms) > 0:
                atoms[last] += string(r)
            else:
                atoms.append(string(r))
            atomStart = i + 1
            hadLeading = false
            hadMaiHanAkat = false

        // Update state
        state = trans.next

    // Emit final atom
    if atomStart < n:
        atoms.append(string(runes[atomStart:]))

    return atoms
```

---

## 9. Combined Mode - segmentWithBacktrack

Combined mode uses `segmentWithBacktrack` as its primary segmenter. This function
replaces greedy longest-match with **coverage-aware backtracking**: at each position
it finds all dictionary matches and selects using two criteria (in priority order):

1. **Boundary validity**: the character after the match must not be a middle vowel
   (which would indicate the match consumed a consonant belonging to the next syllable).
   Tone marks are NOT considered invalid — they modify the preceding character and are
   absorbed as combining marks.

2. **Remainder coverage**: the text after the match (past any absorbed combining marks)
   should start with a dictionary word. If the longest match leaves uncovered text but
   a shorter match yields coverage, the shorter match wins.

### 9.1 Examples

**Boundary backtracking** — "นายกันตพล":
1. `AllMatches` returns `[3, 4]` (for "นาย" and "นายก")
2. Try "นายก"(4): next char ั is middle vowel → **invalid boundary** → skip
3. Try "นาย"(3): next char ก is consonant → valid; remainder "กันตพล" has dict match "กัน" → **coverage** → choose
4. Result: `["นาย", "กัน", "ต", "พล"]`

**Coverage backtracking** — "นายกฤษฎา":
1. `AllMatches` returns `[3, 4]` (for "นาย" and "นายก")
2. Try "นายก"(4): next char ฤ is consonant → valid boundary; remainder "ฤษฎา" has NO dict match → no coverage → save as fallback
3. Try "นาย"(3): next char ก is consonant → valid; remainder "กฤษฎา" has dict match → **coverage** → choose
4. Result: `["นาย", "กฤษฎา"]`

**Combining mark absorption** — "อร์เม...":
1. `AllMatches` returns `[2]` for "อร"
2. Try "อร"(2): next char ์ is tone mark → valid boundary (tone marks are NOT rejected)
3. After match, absorb trailing combining marks: "อร" + "์" → "อร์"
4. Continue from เ...

### 9.2 Helper Functions

#### needsMergeWithNext

Returns `true` if a token is incomplete and needs to absorb following tokens.
Used in Step 3 (forward merge) of `segmentWithBacktrack`.

```
needsMergeWithNext(tok) -> bool:
    if tok is empty: return false
    runes = toCodepoints(tok)

    // 1. Check if entire token is just leading vowel(s)
    allLeading = true
    for each r in runes:
        if NOT isThaiLeadingVowel(r):
            allLeading = false
            break
    if allLeading: return true

    // 2. Check if ends with a leading vowel
    if isThaiLeadingVowel(runes[last]):
        return true

    // 3. Check for Mai Han-Akat (ั) without final consonant
    n = len(runes)
    for i = 0; i < n; i++:
        if runes[i] == 0x0E31:  // ั
            hasFollowingConsonant = false
            for j = i + 1; j < n; j++:
                if isThaiConsonant(runes[j]):
                    hasFollowingConsonant = true
                    break
                if NOT isThaiToneMark(runes[j]):
                    break  // Hit non-tone, non-consonant
            if NOT hasFollowingConsonant:
                return true

    return false
```

#### needsMergeWithPrev

Returns `true` if a token starts with something that can't begin an atom.

```
needsMergeWithPrev(tok) -> bool:
    if tok is empty: return false
    first = toCodepoints(tok)[0]

    if isThaiMiddleVowel(first) OR isThaiToneMark(first):
        return true
    return false
```

### 9.3 segmentWithBacktrack Algorithm

```
segmentWithBacktrack(text) -> []string:
    runes = toCodepoints(text)
    n = len(runes)
    if n == 0: return null

    // ── Step 1: Dict with coverage-aware backtracking ──
    rawSegments = []
    i = 0

    while i < n:
        if isWhitespace(runes[i]):
            i++; continue

        if NOT IsThaiChar(runes[i]):
            start = i
            while i < n AND NOT IsThaiChar(runes[i]) AND NOT isWhitespace(runes[i]):
                i++
            rawSegments.append(string(runes[start:i]))
            continue

        matches = trie.AllMatches(runes[i:])

        chosen = 0
        fallback = 0    // valid boundary but no remainder coverage

        for j = len(matches) - 1; j >= 0; j--:
            mLen = matches[j]
            end = i + mLen

            if end >= n:
                chosen = mLen; break

            // Check 1: boundary validity
            // Only middle vowels invalidate — tone marks are absorbed later
            if isThaiMiddleVowel(runes[end]):
                continue

            // Skip past combining marks to find the true remainder start
            effectiveEnd = end
            while effectiveEnd < n AND IsThaiCombiningMark(runes[effectiveEnd]):
                effectiveEnd++

            if effectiveEnd >= n:
                chosen = mLen; break

            // Check 2: remainder coverage
            remainderMatches = trie.AllMatches(runes[effectiveEnd:])
            if len(remainderMatches) > 0:
                chosen = mLen; break          // best: valid + coverage

            if fallback == 0:
                fallback = mLen               // save first valid-boundary match

        if chosen == 0:
            chosen = fallback

        if chosen > 0:
            // Absorb trailing combining marks (tone marks, etc.)
            end = i + chosen
            while end < n AND IsThaiCombiningMark(runes[end]):
                end++
            rawSegments.append(string(runes[i:end]))
            i = end
        else:
            // No dict match — single char + combining marks
            clusterEnd = i + 1
            while clusterEnd < n AND IsThaiCombiningMark(runes[clusterEnd]):
                clusterEnd++
            rawSegments.append(string(runes[i:clusterEnd]))
            i = clusterEnd

    // ── Step 2: Orphan merging ──
    tokens = []
    for each seg in rawSegments:
        if seg is empty: continue
        rs = toCodepoints(seg)
        if len(rs) > 0 AND (isThaiMiddleVowel(rs[0]) OR isThaiToneMark(rs[0])) AND len(tokens) > 0:
            tokens[last] += seg
            continue
        tokens.append(seg)

    // ── Step 3: Forward merge + atomic fallback ──
    result = []
    j = 0
    while j < len(tokens):
        tok = tokens[j]
        if needsMergeWithNext(tok) AND j + 1 < len(tokens):
            merged = tok
            j++
            while j < len(tokens):
                merged += tokens[j]
                j++
                if NOT needsMergeWithNext(merged): break
            atoms = atomicSegmenter.SegmentToTokens(merged)
            result = result + atoms
            continue
        result.append(tok)
        j++

    return result
```

### 9.4 segmentThai dispatch (Combined mode)

```
segmentThai(text) -> []string:
    switch mode:
        case ModeCombined:
            return segmentWithBacktrack(text)
        ...
```

---

## 10. Main Tokenizer

### 10.1 IsWordChar

```
IsWordChar(r) -> bool:
    return isLetter(r) OR isDigit(r) OR isMark(r) OR r == '_'
```

Where `isLetter`, `isDigit`, `isMark` use Unicode categories:
- `isLetter`: Unicode General Category L (Lu, Ll, Lt, Lm, Lo)
- `isDigit`: Unicode General Category Nd
- `isMark`: Unicode General Category M (Mn, Mc, Me)

### 10.2 Tokenize Algorithm

```
Tokenize(text) -> []Token:
    tokens = []
    runes = toCodepoints(text)
    n = len(runes)

    // Build byte offset map: rune_index -> byte_offset
    byteOffsets = array[n + 1]
    bytePos = 0
    for i = 0; i < n; i++:
        byteOffsets[i] = bytePos
        bytePos += utf8ByteLength(runes[i])
    byteOffsets[n] = bytePos

    position = 0
    i = 0

    while i < n:
        // Skip non-word characters
        if NOT IsWordChar(runes[i]):
            i++
            continue

        startRune = i

        // === THAI TEXT ===
        if IsThaiChar(runes[i]):
            // Collect contiguous Thai + Unicode marks
            segmentEnd = i
            while segmentEnd < n AND (IsThaiChar(runes[segmentEnd]) OR isMark(runes[segmentEnd])):
                segmentEnd++

            thaiText = string(runes[i:segmentEnd])
            thaiTokens = this.segmentThai(thaiText)

            // Build byte offset map for Thai segment
            thaiRunes = toCodepoints(thaiText)
            thaiByteOffsets = array[len(thaiRunes) + 1]
            thaiBytePos = 0
            for j = 0; j < len(thaiRunes); j++:
                thaiByteOffsets[j] = thaiBytePos
                thaiBytePos += utf8ByteLength(thaiRunes[j])
            thaiByteOffsets[len(thaiRunes)] = thaiBytePos

            thaiRunePos = 0
            for each tok in thaiTokens:
                tokRunes = toCodepoints(tok)
                tokLen = len(tokRunes)

                tokenText = tok
                if this.lowercase:
                    tokenText = toLower(tok)

                startByte = byteOffsets[startRune] + thaiByteOffsets[thaiRunePos]
                endByte = byteOffsets[startRune] + thaiByteOffsets[thaiRunePos + tokLen]

                tokens.append(Token{
                    Text:     tokenText,
                    Start:    startByte,
                    End:      endByte,
                    Position: position,
                })
                position++
                thaiRunePos += tokLen

            i = segmentEnd

        // === CJK TEXT ===
        else if IsCJKLike(runes[i]):
            while i < n AND IsCJKLike(runes[i]):
                tokenText = string(runes[i])
                if this.lowercase:
                    tokenText = toLower(tokenText)

                tokens.append(Token{
                    Text:     tokenText,
                    Start:    byteOffsets[i],
                    End:      byteOffsets[i + 1],
                    Position: position,
                })
                position++
                i++

        // === OTHER TEXT (Latin, etc.) ===
        else:
            while i < n AND IsWordChar(runes[i]) AND NOT IsThaiChar(runes[i]) AND NOT IsCJKLike(runes[i]):
                i++

            tokenText = string(runes[startRune:i])
            if this.lowercase:
                tokenText = toLower(tokenText)

            tokens.append(Token{
                Text:     tokenText,
                Start:    byteOffsets[startRune],
                End:      byteOffsets[i],
                Position: position,
            })
            position++

    return tokens
```

### 10.3 TokenizeToStrings

```
TokenizeToStrings(text) -> []string:
    tokens = this.Tokenize(text)
    result = array[len(tokens)]
    for i, tok in tokens:
        result[i] = tok.Text
    return result
```

### 10.4 NormalizeToken

```
NormalizeToken(token) -> string:
    if this.lowercase:
        return toLower(token)
    return token
```

### 10.5 segmentThai (mode dispatch)

```
segmentThai(text) -> []string:
    switch this.mode:
        case ModeAtomic:
            return this.atomicSegmenter.SegmentToTokens(text)
        case ModeCombined:
            return [Combined Mode Algorithm from Section 9.2]
        default (ModeDict):
            return this.thaiSegmenter.SegmentToTokens(text)
```

---

## 11. CJK and Multilingual Support

### 11.1 CJK Unicode Ranges

```
IsCJKChar(r) -> bool:
    return (r >= 0x4E00  AND r <= 0x9FFF)    // CJK Unified Ideographs
        OR (r >= 0x3400  AND r <= 0x4DBF)    // Extension A
        OR (r >= 0x20000 AND r <= 0x2A6DF)   // Extension B
        OR (r >= 0x2A700 AND r <= 0x2B73F)   // Extension C
        OR (r >= 0x2B740 AND r <= 0x2B81F)   // Extension D
        OR (r >= 0x2B820 AND r <= 0x2CEAF)   // Extension E
        OR (r >= 0x2CEB0 AND r <= 0x2EBEF)   // Extension F
        OR (r >= 0xF900  AND r <= 0xFAFF)    // Compatibility Ideographs
```

### 11.2 Japanese Kana

```
IsJapaneseKana(r) -> bool:
    return (r >= 0x3040 AND r <= 0x309F)     // Hiragana
        OR (r >= 0x30A0 AND r <= 0x30FF)     // Katakana
        OR (r >= 0x31F0 AND r <= 0x31FF)     // Katakana Phonetic Extensions
```

### 11.3 Korean Hangul

```
IsKoreanHangul(r) -> bool:
    return (r >= 0xAC00 AND r <= 0xD7AF)     // Hangul Syllables
        OR (r >= 0x1100 AND r <= 0x11FF)     // Hangul Jamo
        OR (r >= 0x3130 AND r <= 0x318F)     // Hangul Compatibility Jamo
```

### 11.4 Bopomofo

```
IsBopomofo(r) -> bool:
    return (r >= 0x3100 AND r <= 0x312F)     // Bopomofo
        OR (r >= 0x31A0 AND r <= 0x31BF)     // Bopomofo Extended
```

### 11.5 IsCJKLike (composite)

```
IsCJKLike(r) -> bool:
    return IsCJKChar(r) OR IsJapaneseKana(r) OR IsKoreanHangul(r) OR IsBopomofo(r)
```

### 11.6 ContainsCJK

```
ContainsCJK(text) -> bool:
    for each r in text:
        if IsCJKLike(r): return true
    return false
```

### 11.7 ContainsThaiChar

```
ContainsThaiChar(text) -> bool:
    for each r in text:
        if IsThaiChar(r): return true
    return false
```

---

## 12. Parallel Processing

### 12.1 Parallelization Threshold

```
PARALLEL_THRESHOLD = 4
```

### 12.2 TokenizeParallel

```
TokenizeParallel(texts) -> [][]Token:
    results = array[len(texts)]

    if len(texts) <= PARALLEL_THRESHOLD:
        // Process sequentially
        for i, text in texts:
            results[i] = this.Tokenize(text)
        return results

    // Process in parallel (one thread/goroutine per text)
    for each (i, text) in texts:
        spawn:
            results[i] = this.Tokenize(text)

    waitForAll()
    return results
```

### 12.3 TokenizeToStringsParallel

Same pattern as TokenizeParallel but calls `TokenizeToStrings` instead.

---

## 13. Test Vectors

All test cases below MUST pass for a conforming implementation.

### 13.1 Simple English (ModeDict, lowercase=true)

| Input | Expected Output |
|-------|-----------------|
| `"hello"` | `["hello"]` |
| `"hello world"` | `["hello", "world"]` |
| `"hello, world!"` | `["hello", "world"]` |
| `"Hello World"` | `["hello", "world"]` |
| `"test123 456"` | `["test123", "456"]` |
| `""` | `[]` (empty) |
| `".,!?;:"` | `[]` (empty) |
| `"hello...world!!!test"` | `["hello", "world", "test"]` |

### 13.2 Lowercase Option (lowercase=false)

| Input | Expected Output |
|-------|-----------------|
| `"Hello World"` | `["Hello", "World"]` |

### 13.3 NormalizeToken (lowercase=true)

| Input | Expected Output |
|-------|-----------------|
| `"Hello"` | `"hello"` |
| `"WORLD"` | `"world"` |
| `"Test123"` | `"test123"` |
| `"你好"` | `"你好"` |

### 13.4 CJK Tokenization

| Input | Expected Output |
|-------|-----------------|
| `"你好"` | `["你", "好"]` |
| `"你好 世界"` | `["你", "好", "世", "界"]` |
| `"あいう"` | `["あ", "い", "う"]` |
| `"アイウ"` | `["ア", "イ", "ウ"]` |
| `"한글"` | `["한", "글"]` |
| `"hello你好world"` | `["hello", "你", "好", "world"]` |

### 13.5 Token Positions

For all inputs, tokens must have:
- Sequential `Position` values starting from 0
- Valid byte offsets: `0 <= Start < End <= len(input)`
- `text[Start:End]` when lowercased equals `Token.Text`

### 13.6 Atomic Segmentation

| Input | Expected Output |
|-------|-----------------|
| `"เสียงเพลงบทนี้ดูมีความเพราะ"` | `["เสีย", "ง", "เพ", "ล", "ง", "บ", "ท", "นี้", "ดู", "มี", "ค", "วา", "ม", "เพ", "ราะ"]` |
| `"เสีย"` | `["เสีย"]` |
| `"เพ"` | `["เพ"]` |
| `"ดู"` | `["ดู"]` |
| `"นี้"` | `["นี้"]` |
| `"ก"` | `["ก"]` |
| `"ฯๆ"` | `["ฯ", "ๆ"]` |
| `"ราะ"` | `["ราะ"]` |
| `"กาบ"` | `["กา", "บ"]` |

### 13.7 Leading Vowels (Atomic)

All five leading vowels followed by consonant ก must produce single atom:

| Input | Expected Output |
|-------|-----------------|
| `"เก"` | `["เก"]` |
| `"แก"` | `["แก"]` |
| `"โก"` | `["โก"]` |
| `"ไก"` | `["ไก"]` |
| `"ใก"` | `["ใก"]` |

### 13.8 Tone Marks (Atomic)

| Input | Expected Output |
|-------|-----------------|
| `"ก่"` | `["ก่"]` |
| `"ก้"` | `["ก้"]` |
| `"ก๊"` | `["ก๊"]` |
| `"ก๋"` | `["ก๋"]` |
| `"ก์"` | `["ก์"]` |
| `"กี่"` | `["กี่"]` |

### 13.9 Middle Vowel Combinations (Atomic)

| Input | Expected Output |
|-------|-----------------|
| `"กะ"` | `["กะ"]` |
| `"กา"` | `["กา"]` |
| `"กาะ"` | `["กาะ"]` |
| `"กือ"` | `["กื", "อ"]` |
| `"กี"` | `["กี"]` |
| `"กู"` | `["กู"]` |

### 13.10 Mai Han-Akat (ั) Patterns (Atomic)

| Input | Expected Output | Pattern |
|-------|-----------------|---------|
| `"จัน"` | `["จัน"]` | C + ั + C |
| `"นั้น"` | `["นั้น"]` | C + ั + tone + C |
| `"นั่ง"` | `["นั่ง"]` | C + ั + tone + C |
| `"ชั่น"` | `["ชั่น"]` | C + ั + tone + C |

### 13.11 Atomic Mode via Tokenizer

| Input | Expected Output |
|-------|-----------------|
| `"เสียงเพลง"` | `["เสีย", "ง", "เพ", "ล", "ง"]` |
| `"hello เสีย world"` | `["hello", "เสีย", "world"]` |

### 13.12 Combined Mode

| Input | Expected Output |
|-------|-----------------|
| `"เมชั่น"` | `["เม", "ชั่น"]` |
| `"นั้น"` | `["นั้น"]` |
| `"ดิจิตัล"` | `["ดิ", "จิ", "ตัล"]` |
| `"พิธานั้น"` | `["พิ", "ธา", "นั้น"]` |
| `"ทรานส์ฟอร์เมชั่น"` | `["ทรานส์", "ฟ", "อร์", "เม", "ชั่น"]` |
| `"นายกันตพล"` | `["นาย", "กัน", "ต", "พล"]` |
| `"นายกฤษฎา"` | `["นาย", "กฤษฎา"]` |

**Boundary backtracking**: `"นายกันตพล"` — `นายก`(4) is longest match but next char
`ั` is a middle vowel (invalid boundary). Backtracks to `นาย`(3), remainder `กัน` has
dict coverage.

**Coverage backtracking**: `"นายกฤษฎา"` — `นายก`(4) has valid boundary (next char `ฤ`
is consonant) but remainder `ฤษฎา` has NO dict coverage. Backtracks to `นาย`(3),
remainder `กฤษฎา` IS a dict word.

### 13.13 Combined Mode Must Match Dict for Valid Tokens

For `"โตเกียว"`:
- Combined mode output must exactly equal Dict mode output

For `"ประยุทธ์ จันทร์โอชา"`:
- Combined mode output must exactly equal Dict mode output

### 13.14 Default Mode

```
New() -> tokenizer with Mode() == ModeDict (0)
```

### 13.15 Character Classification

| Char | IsThaiChar | IsCJKChar | IsJapaneseKana | IsKoreanHangul |
|------|-----------|-----------|----------------|----------------|
| `ก` (0x0E01) | true | false | false | false |
| `า` (0x0E32) | true | - | - | - |
| `่` (0x0E48) | true | - | - | - |
| `a` | false | false | false | false |
| `你` (0x4F60) | false | true | false | false |
| `好` (0x597D) | false | true | false | false |
| `あ` (0x3042) | false | false | true | false |
| `ア` (0x30A2) | false | false | true | false |
| `한` (0xD55C) | false | false | false | true |
| `글` (0xAE00) | false | false | false | true |

---

## 14. Implementation Checklist

Use this checklist to verify your implementation is complete:

### Data Structures
- [ ] `TrieNode` with children map and isEnd flag
- [ ] `Trie` with root, maxLen, Insert, Contains, HasPrefix, LongestMatch, AllMatches
- [ ] `Token` struct with Text, Start, End, Position
- [ ] `TorusTokenizer` with mode, lowercase, both segmenters

### Character Classification
- [ ] 128-entry `thaiCharClass` lookup table (exact values from Section 4.3)
- [ ] `classifyThai()` function
- [ ] `IsThaiChar()`, `isThaiConsonant()`, `isThaiLeadingVowel()`
- [ ] `isThaiMiddleVowel()`, `isThaiToneMark()`, `isThaiSpecialAtom()`
- [ ] `CanStartThaiToken()`, `IsThaiCombiningMark()`
- [ ] `IsWordChar()` using Unicode categories
- [ ] All CJK/Kana/Hangul/Bopomofo range checks

### Dictionary
- [ ] Load `words_th.txt` (24,238 words, one per line)
- [ ] Fallback dictionary (20 words)
- [ ] Trie populated from dictionary

### Dict Mode
- [ ] `ThaiSegmenter.Segment()` - raw longest match with whitespace/non-Thai handling
- [ ] `ThaiSegmenter.SegmentToTokens()` - orphan vowel/tone merging

### Atomic Mode
- [ ] 5x6 FSA transition table (exact values from Section 8.3)
- [ ] `AtomicSegmenter.Segment()` with all four actions
- [ ] Special case: leading vowel pattern final consonant (hadLeading)
- [ ] Special case: Mai Han-Akat with final consonant (hadMaiHanAkat)
- [ ] Non-Thai sequence collection within FSA loop

### Combined Mode
- [ ] `segmentWithBacktrack()` - the primary Combined mode segmenter:
  - [ ] Step 1: Dict matching with boundary validity + coverage-aware backtracking
  - [ ] Boundary check: only middle vowels invalidate (NOT tone marks)
  - [ ] Coverage check: remainder (past combining marks) must start with dict word
  - [ ] Combining mark absorption after match selection
  - [ ] Step 2: Orphan merging (middle vowels/tones that can't start tokens)
  - [ ] Step 3: Forward merge (needsMergeWithNext) + atomic fallback
- [ ] `needsMergeWithNext()` - leading vowel check, mai han-akat check
- [ ] `needsMergeWithPrev()` - middle vowel/tone start check

### Main Tokenizer
- [ ] `Tokenize()` with byte offset calculation
- [ ] Thai text collection (Thai chars + Unicode marks)
- [ ] CJK character-by-character splitting
- [ ] Latin/other word character collection
- [ ] Non-word character skipping
- [ ] Lowercase normalization
- [ ] `TokenizeToStrings()`, `NormalizeToken()`

### Parallel Processing
- [ ] Threshold of 4 for parallel vs sequential
- [ ] `TokenizeParallel()`, `TokenizeToStringsParallel()`

### All Test Vectors from Section 13 pass
- [ ] Simple English (8 cases)
- [ ] Lowercase option (1 case)
- [ ] NormalizeToken (4 cases)
- [ ] CJK (6 cases)
- [ ] Token positions (3 cases)
- [ ] Atomic basic (9 cases)
- [ ] Leading vowels (5 cases)
- [ ] Tone marks (6 cases)
- [ ] Middle vowel combinations (6 cases)
- [ ] Mai Han-Akat (4 cases)
- [ ] Atomic via tokenizer (2 cases)
- [ ] Combined mode (7 cases)
- [ ] Combined matches Dict (2 cases)
- [ ] Default mode (1 case)
- [ ] Character classification (10+ cases)
