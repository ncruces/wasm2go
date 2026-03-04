package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed loop.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "loop.0.wasm")
}
