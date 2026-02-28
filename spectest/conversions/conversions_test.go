package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed conversions.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "conversions.0.wasm")
}
