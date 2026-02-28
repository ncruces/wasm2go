package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed align.107.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "align.107.wasm")
}
