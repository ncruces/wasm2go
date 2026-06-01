package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/internal/spectest"
)

//go:embed global_test.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(spectest.Host{T: t}), data, "global.0.wasm")
}
