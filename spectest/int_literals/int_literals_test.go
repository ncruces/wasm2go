package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed int_literals.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "int_literals.0.wasm")
}
