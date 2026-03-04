package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed memory_trap.1.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "memory_trap.1.wasm")
}
