package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed bulk.3.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "bulk.3.wasm")
}
