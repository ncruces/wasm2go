package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed table_size.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "table_size.0.wasm")
}
