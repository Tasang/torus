/**
 * torus_plugin.cpp — Native C++ Manticore token filter implementing the
 * Torus Thai/CJK tokenizer.  Zero external dependencies; dictionary
 * embedded via words_th_data.h (generated with xxd).
 *
 * Build:
 *   make          # from this directory; generates words_th_data.h then compiles
 *   make plugin   # same
 *
 * Spec: TORUS_SPECS.md (Tasang/torus)  – all section references below match
 * that document.
 */

#include <algorithm>
#include <array>
#include <cctype>
#include <cstdint>
#include <cstring>
#include <string>
#include <unordered_map>
#include <vector>

extern "C" {
#include "/usr/include/manticore/sphinxudf.h"
}

#include "words_th_data.h"   // defines words_th_txt[] and words_th_txt_len

// ═══════════════════════════════════════════════════════════════════════════
// §1  UTF-8 utilities
// ═══════════════════════════════════════════════════════════════════════════

static uint32_t utf8_next(const char*& p, const char* end) {
    if (p >= end) return 0;
    auto b = [](const char* q) { return (uint32_t)(uint8_t)*q; };
    uint32_t c = b(p++);
    if (c < 0x80) return c;
    if ((c & 0xE0) == 0xC0 && p < end)
        return ((c & 0x1F) << 6) | (b(p++) & 0x3F);
    if ((c & 0xF0) == 0xE0 && p + 1 < end) {
        uint32_t r = ((c & 0x0F) << 12) | ((b(p) & 0x3F) << 6) | (b(p+1) & 0x3F);
        p += 2; return r;
    }
    if ((c & 0xF8) == 0xF0 && p + 2 < end) {
        uint32_t r = ((c & 0x07) << 18) | ((b(p) & 0x3F) << 12) |
                     ((b(p+1) & 0x3F) << 6) | (b(p+2) & 0x3F);
        p += 3; return r;
    }
    return 0xFFFD;
}

static std::vector<uint32_t> to_codepoints(const char* s, int len) {
    std::vector<uint32_t> v;
    const char* end = s + len;
    while (s < end) { uint32_t c = utf8_next(s, end); if (!c) break; v.push_back(c); }
    return v;
}

static void encode_cp(std::string& out, uint32_t cp) {
    if (cp < 0x80)   { out += (char)cp; return; }
    if (cp < 0x800)  { out += (char)(0xC0|(cp>>6)); out += (char)(0x80|(cp&0x3F)); return; }
    if (cp < 0x10000){ out += (char)(0xE0|(cp>>12)); out += (char)(0x80|((cp>>6)&0x3F)); out += (char)(0x80|(cp&0x3F)); return; }
    out += (char)(0xF0|(cp>>18)); out += (char)(0x80|((cp>>12)&0x3F));
    out += (char)(0x80|((cp>>6)&0x3F)); out += (char)(0x80|(cp&0x3F));
}

static std::string span(const std::vector<uint32_t>& v, int a, int b) {
    std::string s; for (int i = a; i < b; i++) encode_cp(s, v[i]); return s;
}

// ═══════════════════════════════════════════════════════════════════════════
// §2  Thai character classification (spec §4)
// ═══════════════════════════════════════════════════════════════════════════

enum CC : uint8_t { ccOther=0, ccConsonant=1, ccLeading=2, ccMiddle=3, ccTone=4, ccSpecial=5 };

// 128-entry table for 0x0E00–0x0E7F (spec §4.3)
static const CC thaiCC[128] = {
    ccOther,                                                              // 0x00
    ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant, // 01-08
    ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant, // 09-10
    ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant, // 11-18
    ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant, // 19-20
    ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant, // 21-28
    ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant,ccConsonant,                        // 29-2E
    ccSpecial,                                                            // 2F ฯ
    ccMiddle,ccMiddle,ccMiddle,ccMiddle,                                  // 30-33 ะ ั า ำ
    ccMiddle,ccMiddle,ccMiddle,ccMiddle,ccMiddle,ccMiddle,                // 34-39 ิ ี ึ ื ุ ู
    ccMiddle,                                                             // 3A ฺ
    ccOther,ccOther,ccOther,ccOther,ccOther,                              // 3B-3F
    ccLeading,ccLeading,ccLeading,ccLeading,ccLeading,                   // 40-44 เ แ โ ใ ไ
    ccMiddle,                                                             // 45 ๅ
    ccSpecial,                                                            // 46 ๆ
    ccTone,ccTone,ccTone,ccTone,ccTone,ccTone,                           // 47-4C ็ ่ ้ ๊ ๋ ์
    ccMiddle,                                                             // 4D ํ
    ccOther,ccOther,                                                      // 4E-4F
    ccOther,ccOther,ccOther,ccOther,ccOther,ccOther,ccOther,ccOther,ccOther,ccOther, // 50-59
    ccOther,ccOther,                                                      // 5A-5B
    ccOther,ccOther,ccOther,ccOther,ccOther,ccOther,ccOther,ccOther,      // 5C-63
    ccOther,ccOther,ccOther,ccOther,ccOther,ccOther,ccOther,ccOther,      // 64-6B
    ccOther,ccOther,ccOther,ccOther,ccOther,ccOther,ccOther,ccOther,      // 6C-73
    ccOther,ccOther,ccOther,ccOther,ccOther,ccOther,ccOther,ccOther,      // 74-7B
    ccOther,ccOther,ccOther,ccOther,                                      // 7C-7F
};

static inline CC classify(uint32_t r) {
    return (r >= 0x0E00 && r <= 0x0E7F) ? thaiCC[r - 0x0E00] : ccOther;
}

static inline bool isThai(uint32_t r)         { return r >= 0x0E00 && r <= 0x0E7F; }
static inline bool isConsonant(uint32_t r)    { return r >= 0x0E01 && r <= 0x0E2E; }
static inline bool isLeading(uint32_t r)      { return r >= 0x0E40 && r <= 0x0E44; }
static inline bool isMiddle(uint32_t r)       { return classify(r) == ccMiddle; }
static inline bool isTone(uint32_t r)         { return r >= 0x0E47 && r <= 0x0E4C; }

static inline bool isCombining(uint32_t r) {
    // Following vowels and tone marks that can't start a token (spec §4.1)
    switch (r) {
        case 0x0E30: case 0x0E32: case 0x0E34: case 0x0E35: case 0x0E36: case 0x0E37:
        case 0x0E38: case 0x0E39: case 0x0E45: case 0x0E47: case 0x0E4C: case 0x0E46:
        case 0x0E4D: case 0x0E3A: case 0x0E48: case 0x0E49: case 0x0E4A: case 0x0E4B:
            return true;
        default: return false;
    }
}

static inline bool canStartToken(uint32_t r) { return !isCombining(r); }

static inline bool isSpace(uint32_t r) {
    return r == 0x20 || r == 0x09 || r == 0x0A || r == 0x0D || r == 0x00A0;
}

// ═══════════════════════════════════════════════════════════════════════════
// §3  CJK detection (spec §11)
// ═══════════════════════════════════════════════════════════════════════════

static inline bool isCJKLike(uint32_t r) {
    return (r >= 0x4E00 && r <= 0x9FFF)   || (r >= 0x3400 && r <= 0x4DBF)
        || (r >= 0x20000 && r <= 0x2A6DF) || (r >= 0x3040 && r <= 0x309F)
        || (r >= 0x30A0 && r <= 0x30FF)   || (r >= 0xAC00 && r <= 0xD7AF)
        || (r >= 0x3130 && r <= 0x318F)   || (r >= 0xF900 && r <= 0xFAFF)
        || (r >= 0x3100 && r <= 0x31BF);
}

// ═══════════════════════════════════════════════════════════════════════════
// §4  Trie (spec §5)  — Thai-optimised: flat 128-slot array for U+0E00–0x0E7F
// ═══════════════════════════════════════════════════════════════════════════

struct TrieNode {
    std::array<TrieNode*, 128> thai{};          // fast path: Thai block
    std::unordered_map<uint32_t, TrieNode*> rest; // rare: non-Thai in dict
    bool isEnd = false;

    TrieNode* child(uint32_t cp) const {
        if (cp >= 0x0E00 && cp <= 0x0E7F) return thai[cp - 0x0E00];
        auto it = rest.find(cp); return it != rest.end() ? it->second : nullptr;
    }
    TrieNode*& ensureChild(uint32_t cp, std::vector<TrieNode*>& pool) {
        if (cp >= 0x0E00 && cp <= 0x0E7F) {
            auto& slot = thai[cp - 0x0E00];
            if (!slot) { slot = new TrieNode(); pool.push_back(slot); }
            return slot;
        }
        auto& slot = rest[cp];
        if (!slot) { slot = new TrieNode(); pool.push_back(slot); }
        return slot;
    }
};

struct Trie {
    TrieNode* root;
    int       maxLen = 0;
    std::vector<TrieNode*> pool;   // owns all nodes

    Trie() { root = new TrieNode(); pool.push_back(root); }
    ~Trie() { for (auto* n : pool) delete n; }

    void insert(const std::vector<uint32_t>& word) {
        if (word.empty()) return;
        if ((int)word.size() > maxLen) maxLen = (int)word.size();
        TrieNode* n = root;
        for (uint32_t cp : word) n = n->ensureChild(cp, pool);
        n->isEnd = true;
    }

    int longestMatch(const std::vector<uint32_t>& v, int from) const {
        TrieNode* n = root;
        int best = 0;
        for (int i = from; i < (int)v.size(); i++) {
            n = n->child(v[i]);
            if (!n) break;
            if (n->isEnd) best = i - from + 1;
        }
        return best;
    }

    std::vector<int> allMatches(const std::vector<uint32_t>& v, int from) const {
        std::vector<int> out;
        TrieNode* n = root;
        for (int i = from; i < (int)v.size(); i++) {
            n = n->child(v[i]);
            if (!n) break;
            if (n->isEnd) out.push_back(i - from + 1);
        }
        return out;
    }
};

// ═══════════════════════════════════════════════════════════════════════════
// §5  Dictionary (spec §6) — loaded once at library init from embedded bytes
// ═══════════════════════════════════════════════════════════════════════════

static Trie* g_trie = nullptr;

static void loadDictionary() {
    g_trie = new Trie();
    const char* p   = (const char*)words_th_txt;
    const char* end = p + words_th_txt_len;
    while (p < end) {
        const char* nl = (const char*)memchr(p, '\n', end - p);
        const char* le = nl ? nl : end;
        // trim CR/spaces
        while (le > p && (*(le-1) == '\r' || *(le-1) == ' ')) le--;
        if (le > p) g_trie->insert(to_codepoints(p, (int)(le - p)));
        p = nl ? nl + 1 : end;
    }
}

// ═══════════════════════════════════════════════════════════════════════════
// §6  Dict mode segmentation (spec §7)
// ═══════════════════════════════════════════════════════════════════════════

static std::vector<std::string> dictSegment(const std::vector<uint32_t>& cv) {
    int n = (int)cv.size();
    std::vector<std::string> raw;
    for (int i = 0; i < n; ) {
        if (isSpace(cv[i])) { i++; continue; }
        if (!isThai(cv[i])) {
            if (isCJKLike(cv[i])) {
                raw.push_back(span(cv, i, i+1)); i++; continue;
            }
            int s = i;
            while (i < n && !isThai(cv[i]) && !isSpace(cv[i]) && !isCJKLike(cv[i])) i++;
            if (i > s) raw.push_back(span(cv, s, i));
            continue;
        }
        int m = g_trie->longestMatch(cv, i);
        if (m > 0) { raw.push_back(span(cv, i, i+m)); i += m; }
        else {
            int e = i+1; while (e < n && isCombining(cv[e])) e++;
            raw.push_back(span(cv, i, e)); i = e;
        }
    }
    // orphan merge
    std::vector<std::string> out;
    for (auto& seg : raw) {
        if (seg.empty()) continue;
        auto rs = to_codepoints(seg.c_str(), (int)seg.size());
        if (!rs.empty() && !canStartToken(rs[0]) && !out.empty())
            out.back() += seg;
        else
            out.push_back(std::move(seg));
    }
    return out;
}

// ═══════════════════════════════════════════════════════════════════════════
// §7  Atomic FSA (spec §8)
// ═══════════════════════════════════════════════════════════════════════════

enum AState : uint8_t { stStart=0, stLeading=1, stConsBase=2, stMiddle=3, stTone=4 };
enum AAction: uint8_t { actContinue=0, actEmit=1, actEmitSingle=2, actAttach=3 };
struct Trans { AState next; AAction action; };

// Spec §8.3 transition table (verbatim)
static const Trans atomT[5][6] = {
    {{stStart,actEmit},{stConsBase,actContinue},{stLeading,actContinue},{stStart,actAttach},{stStart,actAttach},{stStart,actEmitSingle}},
    {{stStart,actEmit},{stConsBase,actContinue},{stLeading,actEmit},   {stStart,actEmit},  {stStart,actEmit},  {stStart,actEmit}},
    {{stStart,actEmit},{stConsBase,actEmit},    {stLeading,actEmit},   {stMiddle,actContinue},{stTone,actContinue},{stStart,actEmit}},
    {{stStart,actEmit},{stConsBase,actEmit},    {stLeading,actEmit},   {stMiddle,actContinue},{stTone,actContinue},{stStart,actEmit}},
    {{stStart,actEmit},{stConsBase,actEmit},    {stLeading,actEmit},   {stStart,actEmit},  {stStart,actEmit},  {stStart,actEmit}},
};

static std::vector<std::string> atomicSegment(const std::vector<uint32_t>& cv) {
    int n = (int)cv.size();
    std::vector<std::string> out;
    AState state = stStart;
    int atomStart = 0;
    bool hadLeading = false, hadMai = false;

    for (int i = 0; i < n; i++) {
        uint32_t r = cv[i];
        CC cls = classify(r);

        if (cls == ccOther) {
            if (i > atomStart) out.push_back(span(cv, atomStart, i));
            if (isCJKLike(r)) {
                out.push_back(span(cv, i, i+1));
            } else {
                int s = i;
                while (i < n && classify(cv[i]) == ccOther && !isCJKLike(cv[i])) i++;
                out.push_back(span(cv, s, i)); i--;
            }
            state = stStart; atomStart = i+1; hadLeading = hadMai = false;
            continue;
        }

        const Trans& tr = atomT[state][cls];
        switch (tr.action) {
        case actContinue:
            if (state == stStart) { atomStart = i; hadLeading = (cls==ccLeading); hadMai = false; }
            if (r == 0x0E31) hadMai = true;
            break;
        case actEmit:
            if (i > atomStart) {
                if (state==stMiddle && cls==ccConsonant && hadLeading) {
                    out.push_back(span(cv,atomStart,i+1));
                    atomStart=i+1; state=stStart; hadLeading=hadMai=false; continue;
                }
                if ((state==stMiddle||state==stTone) && cls==ccConsonant && hadMai) {
                    out.push_back(span(cv,atomStart,i+1));
                    atomStart=i+1; state=stStart; hadLeading=hadMai=false; continue;
                }
                out.push_back(span(cv,atomStart,i));
            }
            atomStart = i; hadLeading = (cls==ccLeading); hadMai = false;
            break;
        case actEmitSingle:
            if (i > atomStart) out.push_back(span(cv,atomStart,i));
            out.push_back(span(cv,i,i+1));
            atomStart=i+1; hadLeading=hadMai=false;
            break;
        case actAttach:
            if (!out.empty()) out.back() += span(cv,i,i+1);
            else out.push_back(span(cv,i,i+1));
            atomStart=i+1; hadLeading=hadMai=false;
            break;
        }
        state = tr.next;
    }
    if (atomStart < n) out.push_back(span(cv, atomStart, n));
    return out;
}

// ═══════════════════════════════════════════════════════════════════════════
// §8  Combined mode (spec §9)
// ═══════════════════════════════════════════════════════════════════════════

static bool needsMerge(const std::string& tok) {
    if (tok.empty()) return false;
    auto rs = to_codepoints(tok.c_str(), (int)tok.size());
    if (rs.empty()) return false;
    bool allLead = true;
    for (auto r : rs) if (!isLeading(r)) { allLead = false; break; }
    if (allLead) return true;
    if (isLeading(rs.back())) return true;
    int n = (int)rs.size();
    for (int i = 0; i < n; i++) {
        if (rs[i] == 0x0E31) {
            bool found = false;
            for (int j=i+1; j<n; j++) {
                if (isConsonant(rs[j])) { found=true; break; }
                if (!isTone(rs[j])) break;
            }
            if (!found) return true;
        }
    }
    return false;
}

static std::vector<std::string> combinedSegment(const std::vector<uint32_t>& cv) {
    int n = (int)cv.size();
    std::vector<std::string> raw;

    // Step 1: dict with backtracking
    for (int i = 0; i < n; ) {
        if (isSpace(cv[i])) { i++; continue; }
        if (!isThai(cv[i])) {
            if (isCJKLike(cv[i])) {
                raw.push_back(span(cv, i, i+1)); i++; continue;
            }
            int s = i;
            while (i < n && !isThai(cv[i]) && !isSpace(cv[i]) && !isCJKLike(cv[i])) i++;
            if (i > s) raw.push_back(span(cv, s, i));
            continue;
        }
        auto matches = g_trie->allMatches(cv, i);
        int chosen = 0, fallback = 0;
        for (int j=(int)matches.size()-1; j>=0; j--) {
            int mlen = matches[j], end = i+mlen;
            if (end >= n) { chosen = mlen; break; }
            if (isMiddle(cv[end])) continue;
            int eff = end;
            while (eff < n && isCombining(cv[eff])) eff++;
            if (eff >= n) { chosen = mlen; break; }
            if (!g_trie->allMatches(cv, eff).empty()) { chosen = mlen; break; }
            if (!fallback) fallback = mlen;
        }
        if (!chosen) chosen = fallback;
        if (chosen > 0) {
            int end = i+chosen;
            while (end < n && isCombining(cv[end])) end++;
            raw.push_back(span(cv, i, end)); i = end;
        } else {
            int e = i+1; while (e < n && isCombining(cv[e])) e++;
            raw.push_back(span(cv, i, e)); i = e;
        }
    }

    // Step 2: orphan merge
    std::vector<std::string> toks;
    for (auto& seg : raw) {
        if (seg.empty()) continue;
        auto rs = to_codepoints(seg.c_str(), (int)seg.size());
        if (!rs.empty() && (isMiddle(rs[0])||isTone(rs[0])) && !toks.empty())
            toks.back() += seg;
        else
            toks.push_back(std::move(seg));
    }

    // Step 3: forward merge + atomic fallback
    std::vector<std::string> out;
    for (int j = 0; j < (int)toks.size(); ) {
        if (needsMerge(toks[j]) && j+1 < (int)toks.size()) {
            std::string merged = toks[j++];
            while (j < (int)toks.size()) {
                merged += toks[j++];
                if (!needsMerge(merged)) break;
            }
            auto sub = to_codepoints(merged.c_str(), (int)merged.size());
            for (auto& a : atomicSegment(sub)) out.push_back(std::move(a));
        } else {
            out.push_back(std::move(toks[j++]));
        }
    }
    return out;
}

// ═══════════════════════════════════════════════════════════════════════════
// §9  Mode dispatch
// ═══════════════════════════════════════════════════════════════════════════

static std::vector<std::string> segmentThai(const std::vector<uint32_t>& cv, int mode) {
    if (mode == 1) return atomicSegment(cv);
    if (mode == 2) return combinedSegment(cv);
    return dictSegment(cv);
}

// ═══════════════════════════════════════════════════════════════════════════
// §10  Plugin context
// ═══════════════════════════════════════════════════════════════════════════

struct Ctx {
    int   mode      = 2;
    int   pendingIdx = 0;
    std::vector<std::string> pending;
    char  buf[2048];
};

static const char* emit(Ctx* ctx, const std::string& tok, int* delta, int posInc) {
    size_t n = std::min(tok.size(), sizeof(ctx->buf) - 1);
    memcpy(ctx->buf, tok.c_str(), n);
    ctx->buf[n] = 0;
    if (delta) *delta = posInc;
    return ctx->buf;
}

// ═══════════════════════════════════════════════════════════════════════════
// §11  Manticore plugin interface (Manticore 25 / sphinxplugin.h API)
// ═══════════════════════════════════════════════════════════════════════════

extern "C" {

__attribute__((constructor)) static void plugin_load()   { loadDictionary(); }
__attribute__((destructor))  static void plugin_unload() { delete g_trie; g_trie = nullptr; }

int tok_filter_ver() { return SPH_UDF_VERSION; }

int tok_filter_init(void** ud, int, const char**, const char* opts, char* err) {
    if (!g_trie) { if (err) strncpy(err, "torus: dict not loaded", SPH_UDF_ERROR_LEN-1); return 1; }
    auto* ctx = new Ctx();
    if (opts && *opts) {
        if (!strcmp(opts,"dict"))    ctx->mode = 0;
        else if (!strcmp(opts,"atomic"))   ctx->mode = 1;
        else if (!strcmp(opts,"combined")) ctx->mode = 2;
    }
    *ud = ctx; return 0;
}

int  tok_filter_begin_document(void* ud, const char*, char*) {
    auto* c = (Ctx*)ud; if (c) { c->pending.clear(); c->pendingIdx = 0; } return 0;
}
void tok_filter_begin_field(void* ud, int) {
    auto* c = (Ctx*)ud; if (c) { c->pending.clear(); c->pendingIdx = 0; }
}

char* tok_filter_push_token(void* ud, char* token, int* extra, int* delta) {
    auto* ctx = (Ctx*)ud;
    if (!ctx || !token) { if (extra) *extra = 0; return nullptr; }

    auto cv = to_codepoints(token, (int)strlen(token));
    ctx->pending = segmentThai(cv, ctx->mode);
    ctx->pending.erase(
        std::remove_if(ctx->pending.begin(), ctx->pending.end(),
                       [](const std::string& s){ return s.empty(); }),
        ctx->pending.end());

    if (ctx->pending.empty()) { if (extra) *extra = 0; return nullptr; }

    if (extra) *extra = (int)ctx->pending.size() - 1;
    ctx->pendingIdx = 1;
    return (char*)emit(ctx, ctx->pending[0], delta, 1);
}

char* tok_filter_get_extra_token(void* ud, int* delta) {
    auto* ctx = (Ctx*)ud;
    if (!ctx || ctx->pendingIdx >= (int)ctx->pending.size()) return nullptr;
    return (char*)emit(ctx, ctx->pending[ctx->pendingIdx++], delta, 1);
}

int  tok_filter_end_field(void*) { return 0; }
void tok_filter_deinit(void* ud) { delete (Ctx*)ud; }

} // extern "C"
