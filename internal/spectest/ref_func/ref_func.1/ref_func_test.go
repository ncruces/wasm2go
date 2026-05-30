package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/internal/spectest"
	M "github.com/ncruces/wasm2go/internal/spectest/ref_func/ref_func.0"
)

//go:embed ref_func_test.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(M.New()), data, "ref_func.1.wasm")
}
