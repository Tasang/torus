package torus

// minSplitPart is the shortest part a productive-morpheme split may produce.
// Without it, 2-rune dictionary fragments hijack the split: กรรมการ ("committee",
// "board director" — core tracked vocabulary) breaks at กร|รม|การ instead of
// กรรม|การ, because กร is a dictionary entry that clears the frequency filter.
// Requiring 3 runes on both sides forces the split to land on the morpheme
// boundary that was intended.
const minSplitPart = 3

func (t *TorusTokenizer) splitProductiveMorphemes(tokens []string) []string {
	trie := t.thaiSegmenter.trie
	out := make([]string, 0, len(tokens)+8)
	var split func(s string, depth int)
	split = func(s string, depth int) {
		r := []rune(s)
		if len(r) < 2*minSplitPart || depth > 4 {
			out = append(out, s)
			return
		}
		for k := minSplitPart; k <= len(r)-minSplitPart; k++ {
			a, b := string(r[:k]), string(r[k:])
			if !trie.Contains(a) || !trie.Contains(b) {
				continue
			}
			if productiveMorphemes[a] || productiveMorphemes[b] {
				split(a, depth+1)
				split(b, depth+1)
				return
			}
		}
		out = append(out, s)
	}
	for _, tok := range tokens {
		split(tok, 0)
	}
	return out
}
