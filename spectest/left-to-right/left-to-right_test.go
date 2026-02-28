package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed left-to-right.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "left-to-right.0.wasm")
}
