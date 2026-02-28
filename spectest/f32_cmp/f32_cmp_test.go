package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed f32_cmp.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "f32_cmp.0.wasm")
}
