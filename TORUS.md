# TORUS -- Golang-Based Thai Language Tokenizer Module

TORUS is a Golang module designed specifically for Thai language tokenization using simple algoritm and small dictionary. It it still based on "LongestMatch" using Trie, but without word recombination except for certain hard logic case.

The project is inspired by Mapkha, but completely rewritten by Claude Code using Opus 4.5

## Key Concepts

- Golang library, can be included as a core part of other Golang projects
- Geared for speed and atomic result (reducing false negative)
- Auto recombination logic should be kept to absolute valid cases (just orphanage vowels)
- Added CJK (character splitting) to cover most other languages

## Key Benefits Comparing to others (PyThaiNLP or Mapkha)

- Lightening fast with compiled code and benefit from Goroutine for parallel processing
- Advanced word recombination may result in false negative match for tokenized FTS
- Can be embedded into other Golang project natively, delivering optimal performance
- Handling CJK as well as Thai & English

