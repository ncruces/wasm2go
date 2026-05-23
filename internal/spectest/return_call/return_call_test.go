package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/internal/spectest"
)

//go:embed return_call.0.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(spectest.Host{T: t}), data, "return_call.0.wasm")
}
