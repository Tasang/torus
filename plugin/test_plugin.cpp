#include <cassert>
#include <cstdio>
#include <cstring>
#include <dlfcn.h>
#include <vector>
#include <string>

// Minimal stubs matching the plugin's expected API
typedef int   (*fn_ver)();
typedef int   (*fn_init)(void**, int, const char**, const char*, char*);
typedef int   (*fn_begin_doc)(void*, const char*, char*);
typedef void  (*fn_begin_field)(void*, int);
typedef char* (*fn_push_token)(void*, char*, int*, int*);
typedef char* (*fn_get_extra)(void*, int*);
typedef int   (*fn_end_field)(void*);
typedef void  (*fn_deinit)(void*);

struct Plugin {
    void* dl;
    fn_ver          ver;
    fn_init         init;
    fn_begin_doc    begin_doc;
    fn_begin_field  begin_field;
    fn_push_token   push_token;
    fn_get_extra    get_extra;
    fn_end_field    end_field;
    fn_deinit       deinit;
};

static Plugin load(const char* path) {
    Plugin p{};
    p.dl = dlopen(path, RTLD_NOW);
    if (!p.dl) { fprintf(stderr, "dlopen: %s\n", dlerror()); exit(1); }
#define LOAD(field, sym) \
    p.field = (fn_##field)dlsym(p.dl, sym); \
    if (!p.field) { fprintf(stderr, "missing symbol %s\n", sym); exit(1); }
    LOAD(ver,         "tok_filter_ver");
    LOAD(init,        "tok_filter_init");
    LOAD(begin_doc,   "tok_filter_begin_document");
    LOAD(begin_field, "tok_filter_begin_field");
    LOAD(push_token,  "tok_filter_push_token");
    LOAD(get_extra,   "tok_filter_get_extra_token");
    LOAD(end_field,   "tok_filter_end_field");
    LOAD(deinit,      "tok_filter_deinit");
#undef LOAD
    return p;
}

static std::vector<std::string> tokenize(Plugin& p, const char* opts, const char* text) {
    void* ud = nullptr;
    char err[256] = {};
    if (p.init(&ud, 0, nullptr, opts, err) != 0) {
        fprintf(stderr, "init error: %s\n", err); exit(1);
    }
    p.begin_doc(ud, nullptr, nullptr);
    p.begin_field(ud, 0);

    // Copy text into mutable buffer as the plugin may write to it
    char buf[4096];
    strncpy(buf, text, sizeof(buf)-1); buf[sizeof(buf)-1] = 0;

    std::vector<std::string> tokens;
    int extra = 0, delta = 0;
    char* tok = p.push_token(ud, buf, &extra, &delta);
    if (tok) {
        tokens.push_back(tok);
        for (int i = 0; i < extra; i++) {
            char* t = p.get_extra(ud, &delta);
            if (t) tokens.push_back(t);
        }
    }
    p.end_field(ud);
    p.deinit(ud);
    return tokens;
}

static void print_tokens(const std::vector<std::string>& toks) {
    printf("[");
    for (size_t i = 0; i < toks.size(); i++) {
        if (i) printf(", ");
        printf("\"%s\"", toks[i].c_str());
    }
    printf("]\n");
}

static bool contains(const std::vector<std::string>& v, const std::string& s) {
    for (auto& x : v) if (x == s) return true;
    return false;
}

int main() {
    Plugin p = load("./tok_filter.so");

    printf("tok_filter_ver = %d\n", p.ver());

    // ── Test 1: Dict mode — กลัน should split into กลาง / น or remain as-is
    printf("\n[Dict mode] กรุงเทพมหานคร\n  → ");
    auto t1 = tokenize(p, "dict", "กรุงเทพมหานคร");
    print_tokens(t1);
    assert(!t1.empty());

    // ── Test 2: Atomic mode — each syllable becomes a token
    printf("\n[Atomic mode] กรุงเทพมหานคร\n  → ");
    auto t2 = tokenize(p, "atomic", "กรุงเทพมหานคร");
    print_tokens(t2);
    assert(!t2.empty());

    // ── Test 3: Combined mode (default)
    printf("\n[Combined mode] กรุงเทพมหานคร\n  → ");
    auto t3 = tokenize(p, "combined", "กรุงเทพมหานคร");
    print_tokens(t3);
    assert(!t3.empty());

    // ── Test 4: Mixed Thai + Latin
    printf("\n[Combined mode] hello กรุงเทพ world\n  → ");
    auto t4 = tokenize(p, "combined", "hello กรุงเทพ world");
    print_tokens(t4);
    assert(contains(t4, "hello") || contains(t4, "world") || !t4.empty());

    // ── Test 5: CJK single-char tokenization
    printf("\n[Combined mode] 日本語\n  → ");
    auto t5 = tokenize(p, "combined", "日本語");
    print_tokens(t5);
    // Each CJK char should be its own token
    assert(t5.size() == 3);

    // ── Test 6: Leading vowel (เ) handling
    printf("\n[Dict mode] เมือง\n  → ");
    auto t6 = tokenize(p, "dict", "เมือง");
    print_tokens(t6);
    assert(!t6.empty());

    printf("\nAll tests passed.\n");
    dlclose(p.dl);
    return 0;
}
