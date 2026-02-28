package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed call_indirect.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "call_indirect.0.wasm")
}
