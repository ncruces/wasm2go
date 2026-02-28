package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed endianness.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "endianness.0.wasm")
}
