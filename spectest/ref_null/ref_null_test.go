package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed ref_null.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "ref_null.0.wasm")
}
