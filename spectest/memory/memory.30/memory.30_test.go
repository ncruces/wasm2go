package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed memory.30.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "memory.30.wasm")
}
