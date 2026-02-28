package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed i64.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "i64.0.wasm")
}
