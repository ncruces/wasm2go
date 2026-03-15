package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
	elem39 "github.com/ncruces/wasm2go/spectest/elem/elem.39"
	elem40 "github.com/ncruces/wasm2go/spectest/elem/elem.40"
)

//go:embed elem.json
var data []byte

func Test(t *testing.T) {
	mod := New()
	spectest.Test(t, mod, data, "elem.38.wasm")
	elem39.New(mod)
	spectest.Test(t, mod, data, "elem.39.wasm")
	elem40.New(mod)
	spectest.Test(t, mod, data, "elem.40.wasm")
}
