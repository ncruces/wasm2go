package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed local_set.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "local_set.0.wasm")
}
