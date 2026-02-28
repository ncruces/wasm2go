package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed f32.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "f32.0.wasm")
}
