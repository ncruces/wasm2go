//go:build ignore

/*
wasm32-wasip1-clang -g -O0 \
  -ffreestanding -nostdlib \
  testdata/dwarfline/dwarfline.c \
  -o testdata/dwarfline/dwarfline.wasm \
  -ffile-prefix-map=$(pwd)/= \
  -fdebug-compilation-dir=. \
  -Wl,--no-entry \
  -Wl,--import-undefined \
  -Wl,--export=simple_call \
  -Wl,--export=expr_call \
  -Wl,--export=chain_call \
  -Wl,--export=branch_call \
  -Wl,--export=loop_call \
  -Wl,--export=multi_call
*/

extern void env_sink(int);
extern int env_source(void);

void simple_call(int a) {
	env_sink(a);
}

void expr_call(int a, int b) {
	env_sink(a + b * 2 - 1);
}

void chain_call(void) {
	env_sink(env_source());
}

void branch_call(int c) {
	if (c) {
		env_sink(1);
	} else {
		env_sink(0);
	}
}

void loop_call(int n) {
	for (int i = 0; i < n; i++) {
		env_sink(i);
	}
}

void multi_call(int a, int b) {
	env_sink(a);
	env_sink(b);
	env_sink(a + b);
}
