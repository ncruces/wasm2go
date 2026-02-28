package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed func_ptrs.8.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "func_ptrs.8.wasm")
}
