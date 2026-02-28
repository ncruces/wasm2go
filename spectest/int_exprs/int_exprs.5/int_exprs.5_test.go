package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed int_exprs.5.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "int_exprs.5.wasm")
}
