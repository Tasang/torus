# TORUS

TORUS is a Go module for Thai-language tokenization, built around a Trie-based
longest-match algorithm over a small dictionary rather than heavier
statistical/ML segmentation. It also handles CJK and Latin/other-script text,
so it can serve as a general-purpose multilingual tokenizer in mixed content.

The project is inspired by [Mapkha](https://github.com/rkcosmos/deepcut) but
was rewritten from scratch as a lightweight, dependency-free Go library.

## Key concepts

- Pure Go library — importable directly into other Go projects, no external
  runtime dependencies.
- Optimized for speed and "atomic" results (few false negatives) over
  maximal recombination.
- Word recombination is intentionally limited to unambiguous cases (e.g.
  orphaned vowels), instead of aggressive re-merging.
- CJK text is split character-by-character; Latin/other scripts use standard
  Unicode word-boundary rules.

## Tokenization modes

| Mode | Description |
|------|-------------|
| `ModeDict` (default) | Trie-based longest-match dictionary lookup |
| `ModeAtomic` | FSA-based splitting into smallest valid Thai orthographic units |
| `ModeCombined` | Dictionary segmentation with an atomic-validity guarantee (backtracks non-atomic tokens) |

## Repository layout

| Path | Description |
|------|-------------|
| `tokenizer.go` | Core `TorusTokenizer` API — modes, options, `Tokenize`/`TokenizeToStrings`, parallel variants |
| `thai.go` | Thai dictionary-based segmenter (Trie longest match) |
| `atomic.go` | FSA-based atomic Thai unit segmenter |
| `trie.go` | Trie data structure used for dictionary lookups |
| `unicode.go` | Character classification helpers (CJK, Hangul, Kana, Bopomofo, word chars) |
| `data/words_th.txt` | Thai word dictionary used to build the Trie |
| `cmd/torus-test/` | CLI for tokenizing stdin text, useful for manual testing |
| `plugin/` | Native C++ token filter plugin for [Manticore Search](https://manticoresearch.com/), embedding the same dictionary and reimplementing the tokenizer logic in C++ |
| `TORUS_SPECS.md` | Exhaustive, language-agnostic implementation spec (data structures, algorithms, test vectors) for porting Torus to other languages/runtimes |
| `TORUS.md` | Short project overview and rationale |

## Installation

```bash
go get github.com/Tasang/torus
```

## Usage

```go
package main

import (
    "fmt"

    "github.com/Tasang/torus"
)

func main() {
    tok := torus.New()
    tokens := tok.Tokenize("สวัสดีครับ")
    for _, t := range tokens {
        fmt.Println(t.Text)
    }
}
```

Selecting a mode:

```go
tok := torus.New(torus.WithMode(torus.ModeAtomic))
tokens := tok.TokenizeToStrings("เสียงเพลง")
// ["เสีย", "ง", "เพ", "ล", "ง"]
```

## CLI

`cmd/torus-test` reads lines from stdin and prints tokenized output:

```bash
go run ./cmd/torus-test -m dict      # dict mode (default)
go run ./cmd/torus-test -m atomic    # atomic mode
go run ./cmd/torus-test -m combined  # combined mode
go run ./cmd/torus-test -v           # verbose: show byte positions
go run ./cmd/torus-test -1           # one token per line
```

## Manticore plugin

`plugin/torus_plugin.cpp` is a standalone C++ reimplementation of the
tokenizer as a native Manticore Search token filter, with the dictionary
embedded at build time. Build with:

```bash
cd plugin
make        # generates words_th_data.h and compiles tok_filter.so
make test   # builds and runs test_plugin
```

## Testing

```bash
go test ./...
```

## Why not full recombination?

Aggressive word recombination (as used by some Thai NLP libraries) can merge
tokens in ways that reduce recall for full-text search. TORUS favors smaller,
more atomic tokens to minimize false negatives when used for FTS indexing,
while still compiling to native code and using goroutines for parallel
processing.
