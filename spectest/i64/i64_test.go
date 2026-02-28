package wasm2go_test

import (
	_ "embed"
	"reflect"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
	wasm2go "github.com/ncruces/wasm2go/spectest/i64"
)

//go:embed i64.json
var data []byte

func Test(t *testing.T) {
	var mod wasm2go.Module
	spectest.Test(t, reflect.ValueOf(&mod), data)
}
