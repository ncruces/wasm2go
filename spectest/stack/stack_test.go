package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed stack.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "stack.0.wasm")
}
