#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../../
LIBC="$ROOT/libc-gen/c/"
BINARYEN="./tools/binaryen/bin/"
WASI_SDK="./tools/wasi-sdk/bin/"

trap 'rm -f alloc alloc.wasm' EXIT

for ALLOC in bump sbrk tlsf; do
	mkdir -p "$ALLOC"

	"$WASI_SDK/clang" --target=wasm32 -ffreestanding -nostdlib -std=c23 -g0 -Oz \
		-Wall -Wextra -Wno-unused-parameter -Wno-unused-function \
		-o alloc "$LIBC/malloc_$ALLOC.c" -I"$LIBC" \
		-mexec-model=reactor \
		-mmutable-globals -mmultivalue \
		-mnontrapping-fptoint -msign-ext \
		-mreference-types -mbulk-memory \
		-mextended-const -mtail-call \
		-mwide-arithmetic \
		-Wl,--no-entry \
		-Wl,--stack-first \
		-Wl,--import-undefined \
		-Wl,--export=free \
		-Wl,--export=malloc \
		-Wl,--export=realloc \
		-Wl,--export=memalign

	"$BINARYEN/wasm-opt" -g alloc -o alloc.wasm \
		--gufa-optimizing --generate-global-effects \
		--low-memory-unused --converge -O4 \
		--enable-mutable-globals --enable-multivalue \
		--enable-nontrapping-float-to-int --enable-sign-ext \
		--enable-reference-types --enable-bulk-memory \
		--enable-extended-const --enable-tail-call \
		--enable-wide-arithmetic \
		--strip --strip-producers

	go run "$ROOT" -unsafe -pkg "$ALLOC" -o "$ALLOC/$ALLOC.go" alloc.wasm
done
