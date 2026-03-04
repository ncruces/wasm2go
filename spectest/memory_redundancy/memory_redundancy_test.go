package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed memory_redundancy.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "memory_redundancy.0.wasm")
}
