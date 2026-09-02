PLUGIN   = tok_filter.so
DICT_HDR = words_th_data.h
DICT_SRC = ../data/words_th.txt
MORPH_HDR = morphemes_data.h
MORPH_SRC = ../morphemes.go

.PHONY: all plugin test clean

all: plugin

test: $(PLUGIN) test_plugin
	./test_plugin

test_plugin: test_plugin.cpp
	g++ -std=c++17 -O0 -o test_plugin test_plugin.cpp -ldl

plugin: $(PLUGIN)

$(PLUGIN): torus_plugin.cpp $(DICT_HDR) $(MORPH_HDR)
	g++ -std=c++17 -O2 -fPIC -shared -o $(PLUGIN) torus_plugin.cpp

# Generated from the Go source so the two implementations cannot drift.
$(MORPH_HDR): $(MORPH_SRC)
	@echo '// Generated from $(MORPH_SRC) by make -- do not edit.' > $@
	@echo 'static const char* const torus_morphemes[] = {' >> $@
	@sed -n 's/^[[:space:]]*"\([^"]*\)":[[:space:]]*true,.*/\t"\1",/p' $< >> $@
	@echo '};' >> $@
	@echo 'static const unsigned int torus_morphemes_len =' >> $@
	@echo '    sizeof(torus_morphemes) / sizeof(torus_morphemes[0]);' >> $@

$(DICT_HDR): $(DICT_SRC)
	xxd -i $< \
	  | sed 's/unsigned char [a-zA-Z0-9_]*\[\]/static const unsigned char words_th_txt[]/' \
	  | sed 's/unsigned int [a-zA-Z0-9_]*_len/static const unsigned int words_th_txt_len/' \
	  > $@

clean:
	rm -f $(PLUGIN) $(DICT_HDR) $(MORPH_HDR) test_plugin
