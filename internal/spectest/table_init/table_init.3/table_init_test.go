package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/internal/spectest"
	a "github.com/ncruces/wasm2go/internal/spectest/table_init/table_init.0"
)

//go:embed table_init_test.json
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(a.New()), data, "table_init.3.wasm")
}
