package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed return.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "return.0.wasm")
}
