package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed return_call_indirect.0.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(host{t}), data, "return_call_indirect.0.wasm")
}

type host struct{ *testing.T }

func (h host) Xprint_i32_f32(v0 int32, v1 float32) {
	h.Log(v0, v1)
}
