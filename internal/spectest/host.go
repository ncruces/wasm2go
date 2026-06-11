package spectest

import (
	"math"
	"testing"
)

// https://github.com/WebAssembly/spec/tree/main/interpreter#spectest-host-module

var (
	global_i32 int32   = 666
	global_i64 int64   = 666
	global_f32 float32 = 666.6
	global_f64 float64 = 666.6

	table  []any  = make([]any, 10, 20)
	memory []byte = make([]byte, 65536)
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

func (h Host) Xmemory() Memory {
	return h
}

func (h Host) Slice() *[]byte {
	return &memory
}

func (h Host) Grow(delta, max int64) int64 {
	buf := memory
	len := int64(len(buf))
	old := len >> 16
	if delta == 0 {
		return old
	}
	new := old + delta
	add := new<<16 - len
	max = min(max, int64(math.MaxInt)>>16)
	if new > max || new < old || add < 0 {
		return -1
	}
	memory = append(buf, make([]byte, add)...)
	return old
}

type Memory = interface {
	Slice() *[]byte
	Grow(delta, max int64) int64
}
