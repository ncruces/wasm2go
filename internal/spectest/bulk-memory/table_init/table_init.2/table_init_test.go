package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/internal/spectest"
	a "github.com/ncruces/wasm2go/internal/spectest/bulk-memory/table_init/table_init.0"
)

func Test(t *testing.T) {
	spectest.TestModule(t, func() any {
		return New(a.New())
	}, "../table_init.json", "table_init.2.wasm")
}
