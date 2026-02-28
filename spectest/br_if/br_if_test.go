package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed br_if.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "br_if.0.wasm")
}
