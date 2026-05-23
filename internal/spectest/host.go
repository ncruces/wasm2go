package spectest

import "testing"

type Host struct{ *testing.T }

func (h Host) Xprint_i32_f32(v0 int32, v1 float32) {
	h.Log(v0, v1)
}
