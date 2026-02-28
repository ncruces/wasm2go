package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed float_exprs.68.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "float_exprs.68.wasm")
}
