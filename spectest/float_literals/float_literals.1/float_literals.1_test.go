package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed float_literals.1.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "float_literals.1.wasm")
}
