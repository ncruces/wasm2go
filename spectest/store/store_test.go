package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed store.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "store.0.wasm")
}
