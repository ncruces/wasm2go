package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed f64.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "f64.0.wasm")
}
