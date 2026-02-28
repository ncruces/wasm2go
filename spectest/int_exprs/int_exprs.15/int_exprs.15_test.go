package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed int_exprs.15.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "int_exprs.15.wasm")
}
