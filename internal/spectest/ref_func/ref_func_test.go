package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/internal/spectest"
	ref_func_0 "github.com/ncruces/wasm2go/internal/spectest/ref_func/ref_func.0"
	ref_func_1 "github.com/ncruces/wasm2go/internal/spectest/ref_func/ref_func.1"
)

func Test(t *testing.T) {
	spectest.TestModule(t, func() any {
		return ref_func_1.New(ref_func_0.New())
	}, "ref_func.json", "ref_func.1.wasm")
}
