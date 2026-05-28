package spectest

import "testing"

var (
	global_i32 int32 = 666
	global_i64 int64 = 666
)

type Host struct{ *testing.T }

func (h Host) Xglobal_i32() *int32 { return &global_i32 }

func (h Host) Xglobal_i64() *int64 { return &global_i64 }

func (h Host) Xprint_i32(v0 int32) {
	h.Log(v0)
}

func (h Host) Xprint_i32_f32(v0 int32, v1 float32) {
	h.Log(v0, v1)
}
