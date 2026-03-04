package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed traps.2.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "traps.2.wasm")
}
