package spectest

import "testing"

// https://github.com/WebAssembly/spec/tree/main/interpreter#spectest-host-module

var (
	global_i32 int32   = 666
	global_i64 int64   = 666
	global_f32 float32 = 666.6
	global_f64 float64 = 666.6

	table []any = make([]any, 10, 20)
)

type Host struct{ *testing.T }

func (h Host) Xglobal_i32() *int32 { return &global_i32 }

func (h Host) Xglobal_i64() *int64 { return &global_i64 }

func (h Host) Xglobal_f32() *float32 { return &global_f32 }

func (h Host) Xglobal_f64() *float64 { return &global_f64 }

func (h Host) Xtable() *[]any { return &table }

func (h Host) Xprint() { h.Log() }

func (h Host) Xprint_i32(v0 int32)   { h.Log(v0) }
func (h Host) Xprint_i64(v0 int64)   { h.Log(v0) }
func (h Host) Xprint_f32(v0 float32) { h.Log(v0) }
func (h Host) Xprint_f64(v0 float64) { h.Log(v0) }

func (h Host) Xprint_i32_f32(v0 int32, v1 float32) {
	h.Log(v0, v1)
}

func (h Host) Xprint_f64_f64(v0 int64, v1 float64) {
	h.Log(v0, v1)
}
