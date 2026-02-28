package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed fac.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "fac.0.wasm")
}
