package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed table_grow.0.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "table_grow.0.wasm")
}
