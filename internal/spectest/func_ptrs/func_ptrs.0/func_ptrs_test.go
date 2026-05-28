package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/internal/spectest"
)

//go:embed func_ptrs_test.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(spectest.Host{T: t}), data, "func_ptrs.0.wasm")
}
