package main

import (
	_ "embed"
	"io"
	"strings"
	"testing"
)

//go:embed fib.wasm
var fib string

func Test_translate(t *testing.T) {
	err := translate("fib", strings.NewReader(fib), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
}
