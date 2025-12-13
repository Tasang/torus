// Package torus provides Thai language tokenization using Trie-based longest match algorithm.
package torus

// TrieNode represents a node in the Trie data structure.
type TrieNode struct {
	children map[rune]*TrieNode
	isEnd    bool
}

// Trie is a prefix tree for efficient dictionary lookup.
type Trie struct {
	root   *TrieNode
	maxLen int // Maximum word length in runes
}

// NewTrie creates a new empty Trie.
func NewTrie() *Trie {
	return &Trie{
		root:   &TrieNode{children: make(map[rune]*TrieNode)},
		maxLen: 0,
	}
}

// Insert adds a word to the Trie.
func (t *Trie) Insert(word string) {
	if word == "" {
		return
	}

	runes := []rune(word)
	if len(runes) > t.maxLen {
		t.maxLen = len(runes)
	}

	node := t.root
	for _, r := range runes {
		if node.children == nil {
			node.children = make(map[rune]*TrieNode)
		}
		if _, exists := node.children[r]; !exists {
			node.children[r] = &TrieNode{children: make(map[rune]*TrieNode)}
		}
		node = node.children[r]
	}
	node.isEnd = true
}

// Contains checks if a word exists in the Trie.
func (t *Trie) Contains(word string) bool {
	node := t.root
	for _, r := range word {
		if node.children == nil {
			return false
		}
		next, exists := node.children[r]
		if !exists {
			return false
		}
		node = next
	}
	return node.isEnd
}

// HasPrefix checks if any word in the Trie starts with the given prefix.
func (t *Trie) HasPrefix(prefix string) bool {
	node := t.root
	for _, r := range prefix {
		if node.children == nil {
			return false
		}
		next, exists := node.children[r]
		if !exists {
			return false
		}
		node = next
	}
	return true
}

// LongestMatch finds the longest word in the Trie that matches the beginning of the input.
// Returns the length (in runes) of the longest match, or 0 if no match found.
func (t *Trie) LongestMatch(runes []rune) int {
	node := t.root
	longestMatch := 0

	for i, r := range runes {
		if node.children == nil {
			break
		}
		next, exists := node.children[r]
		if !exists {
			break
		}
		node = next
		if node.isEnd {
			longestMatch = i + 1
		}
	}

	return longestMatch
}

// MaxWordLength returns the maximum word length in the Trie.
func (t *Trie) MaxWordLength() int {
	return t.maxLen
}
