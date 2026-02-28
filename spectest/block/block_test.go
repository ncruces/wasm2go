package wasm2go_test

import (
	_ "embed"
	"reflect"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
	wasm2go "github.com/ncruces/wasm2go/spectest/block"
)

//go:embed block.json
var data []byte

func Test(t *testing.T) {
	mod := wasm2go.New()
	spectest.Test(t, reflect.ValueOf(mod), data)
}
