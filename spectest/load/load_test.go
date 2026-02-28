package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed load.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "load.0.wasm")
}
