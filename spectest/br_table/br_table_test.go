package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed br_table.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "br_table.0.wasm")
}
