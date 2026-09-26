#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../../../
LIBC="$ROOT/libc-gen/c/"
BINARYEN="$ROOT/libc-gen/tools/binaryen/bin/"
WASI_SDK="$ROOT/libc-gen/tools/wasi-sdk/bin/"

trap 'rm -f sjlj sjlj.wasm' EXIT

"$WASI_SDK/clang" --target=wasm32 -ffreestanding -nostdlib -std=c23 -g0 -Oz \
	-Wall -Wextra -Wno-unused-parameter -Wno-unused-function \
	-o sjlj main.c "$LIBC/libc.c" "$LIBC/setjmp_em.c" -I"$LIBC" \
	-mllvm -enable-emscripten-sjlj \
	-mexec-model=reactor \
	-mmutable-globals -mmultivalue \
	-mnontrapping-fptoint -msign-ext \
	-mreference-types -mbulk-memory \
	-mextended-const -mtail-call \
	-mwide-arithmetic \
	-Wl,--no-entry \
	-Wl,--stack-first \
	-Wl,--export-table \
	-Wl,--import-undefined \
	-Wl,--export=__stack_pointer \
	-Wl,--export=test

"$BINARYEN/wasm-opt" -g sjlj -o sjlj.wasm \
	--gufa-optimizing --generate-global-effects \
	--low-memory-unused --converge -O4 \
	--enable-mutable-globals --enable-multivalue \
	--enable-nontrapping-float-to-int --enable-sign-ext \
	--enable-reference-types --enable-bulk-memory \
	--enable-extended-const --enable-tail-call \
	--enable-wide-arithmetic \
	--strip --strip-producers

go run "$ROOT/libc-gen" -wasm sjlj.wasm -o ../libc.go
go run "$ROOT" -unsafe -provided ../libc.go -o ../sjlj.go sjlj.wasm
