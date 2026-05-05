package helpers

import (
	"fmt"
	"math"
	"runtime"
	"strings"
	"testing"
)

func Test_i32(t *testing.T) {
	// Must compile, may panic.
	defer func() { recover() }()
	_ = math.MinInt32 / i32(-1)
	_ = uint32(i32(-1))
	_ = int32(1) / i32(0)
}

func Test_i64(t *testing.T) {
	// Must compile, may panic.
	defer func() { recover() }()
	_ = math.MinInt64 / i64(-1)
	_ = uint64(i64(-1))
	_ = int64(1) / i64(0)
}

func Test_f32_const(t *testing.T) {
	if strings.HasPrefix(runtime.GOARCH, "mips") {
		t.SkipNow()
	}
	t1 := math.Float32frombits(0x7fa00000)
	t2 := t1 * f32(1)
	t3 := math.Float32bits(t2)
	if t3&0x7fc00000 != 0x7fc00000 {
		t.Errorf("%x", t3)
	}
}

func Test_f64_const(t *testing.T) {
	if strings.HasPrefix(runtime.GOARCH, "mips") {
		t.SkipNow()
	}
	t1 := math.Float64frombits(0x7ff4000000000000)
	t2 := t1 * f64(1)
	t3 := math.Float64bits(t2)
	if t3&0x7ff8000000000000 != 0x7ff8000000000000 {
		t.Errorf("%x", t3)
	}
}

func Test_i32_div_s(t *testing.T) {
	tests := []struct {
		x, y int32
		r    int32
		p    bool
	}{
		{10, 2, 5, false},
		{-10, 2, -5, false},
		{math.MinInt32, 1, math.MinInt32, false},
		{math.MinInt32, -1, 0, true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d,%d", tt.x, tt.y), func(t *testing.T) {
			if tt.p {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("i32_div_s(%d, %d) did not panic", tt.x, tt.y)
					}
				}()
			}
			got := i32_div_s(tt.x, tt.y)
			if got != tt.r {
				t.Errorf("i32_div_s(%d, %d) = %v, want %v", tt.x, tt.y, got, tt.r)
			}
		})
	}
}

func Test_i64_div_s(t *testing.T) {
	tests := []struct {
		x, y int64
		r    int64
		p    bool
	}{
		{10, 2, 5, false},
		{-10, 2, -5, false},
		{math.MinInt64, 1, math.MinInt64, false},
		{math.MinInt64, -1, 0, true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d,%d", tt.x, tt.y), func(t *testing.T) {
			if tt.p {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("i64_div_s(%d, %d) did not panic", tt.x, tt.y)
					}
				}()
			}
			got := i64_div_s(tt.x, tt.y)
			if got != tt.r {
				t.Errorf("i64_div_s(%d, %d) = %v, want %v", tt.x, tt.y, got, tt.r)
			}
		})
	}
}

func Test_i32_neg_s(t *testing.T) {
	tests := []struct {
		x int32
		r int32
		p bool
	}{
		{1, -1, false},
		{math.MinInt32, math.MinInt32, true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.x), func(t *testing.T) {
			if tt.p {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("i32_neg_s(%d) did not panic", tt.x)
					}
				}()
			}
			got := i32_neg_s(tt.x)
			if got != tt.r {
				t.Errorf("i32_neg_s(%d) = %v, want %v", tt.x, got, tt.r)
			}
		})
	}
}

func Test_i64_neg_s(t *testing.T) {
	tests := []struct {
		x int64
		r int64
		p bool
	}{
		{1, -1, false},
		{math.MinInt64, math.MinInt64, true},
		{math.MinInt32, -math.MinInt32, false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.x), func(t *testing.T) {
			if tt.p {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("i64_neg_s(%d) did not panic", tt.x)
					}
				}()
			}
			got := i64_neg_s(tt.x)
			if got != tt.r {
				t.Errorf("i64_neg_s(%d) = %v, want %v", tt.x, got, tt.r)
			}
		})
	}
}

func Test_i32_shl(t *testing.T) {
	tests := []struct {
		x, y int32
		want int32
	}{
		{1, 1, 2},
		{1, 32, 1},
		{1, 33, 2},
		{-1, 1, -2},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d,%d", tt.x, tt.y), func(t *testing.T) {
			if got := i32_shl(tt.x, tt.y); got != tt.want {
				t.Errorf("i32_shl(%d, %d) = %d, want %d", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func Test_i32_shr_s(t *testing.T) {
	tests := []struct {
		x, y int32
		want int32
	}{
		{2, 1, 1},
		{-2, 1, -1},
		{-1, 31, -1},
		{-1, 32, -1},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d,%d", tt.x, tt.y), func(t *testing.T) {
			if got := i32_shr_s(tt.x, tt.y); got != tt.want {
				t.Errorf("i32_shr_s(%d, %d) = %d, want %d", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func Test_i32_shr_u(t *testing.T) {
	tests := []struct {
		x, y int32
		want int32
	}{
		{2, 1, 1},
		{-2, 1, math.MaxInt32},
		{-1, 31, 1},
		{-1, 32, -1},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d,%d", tt.x, tt.y), func(t *testing.T) {
			if got := i32_shr_u(tt.x, tt.y); got != tt.want {
				t.Errorf("i32_shr_u(%d, %d) = %d, want %d", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func Test_i64_shl(t *testing.T) {
	tests := []struct {
		x, y int64
		want int64
	}{
		{1, 1, 2},
		{1, 64, 1},
		{1, 65, 2},
		{-1, 1, -2},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d,%d", tt.x, tt.y), func(t *testing.T) {
			if got := i64_shl(tt.x, tt.y); got != tt.want {
				t.Errorf("i64_shl(%d, %d) = %d, want %d", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func Test_i64_shr_s(t *testing.T) {
	tests := []struct {
		x, y int64
		want int64
	}{
		{2, 1, 1},
		{-2, 1, -1},
		{-1, 63, -1},
		{-1, 64, -1},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d,%d", tt.x, tt.y), func(t *testing.T) {
			if got := i64_shr_s(tt.x, tt.y); got != tt.want {
				t.Errorf("i64_shr_s(%d, %d) = %d, want %d", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func Test_i64_shr_u(t *testing.T) {
	tests := []struct {
		x, y int64
		want int64
	}{
		{2, 1, 1},
		{-2, 1, math.MaxInt64},
		{-1, 63, 1},
		{-1, 64, -1},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d,%d", tt.x, tt.y), func(t *testing.T) {
			if got := i64_shr_u(tt.x, tt.y); got != tt.want {
				t.Errorf("i64_shr_u(%d, %d) = %d, want %d", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func Test_i32_rotl(t *testing.T) {
	tests := []struct {
		x, y int32
		want int32
	}{
		{1, 1, 2},
		{1, 31, math.MinInt32},
		{1, 32, 1},
		{1, 33, 2},
		{-1, 1, -1},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d,%d", tt.x, tt.y), func(t *testing.T) {
			if got := i32_rotl(tt.x, tt.y); got != tt.want {
				t.Errorf("i32_rotl(%d, %d) = %d, want %d", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func Test_i32_rotr(t *testing.T) {
	tests := []struct {
		x, y int32
		want int32
	}{
		{2, 1, 1},
		{1, 1, math.MinInt32},
		{1, 32, 1},
		{1, 33, math.MinInt32},
		{-1, 1, -1},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d,%d", tt.x, tt.y), func(t *testing.T) {
			if got := i32_rotr(tt.x, tt.y); got != tt.want {
				t.Errorf("i32_rotr(%d, %d) = %d, want %d", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func Test_i64_rotl(t *testing.T) {
	tests := []struct {
		x, y int64
		want int64
	}{
		{1, 1, 2},
		{1, 63, math.MinInt64},
		{1, 64, 1},
		{1, 65, 2},
		{-1, 1, -1},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d,%d", tt.x, tt.y), func(t *testing.T) {
			if got := i64_rotl(tt.x, tt.y); got != tt.want {
				t.Errorf("i64_rotl(%d, %d) = %d, want %d", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func Test_i64_rotr(t *testing.T) {
	tests := []struct {
		x, y int64
		want int64
	}{
		{2, 1, 1},
		{1, 1, math.MinInt64},
		{1, 64, 1},
		{1, 65, math.MinInt64},
		{-1, 1, -1},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d,%d", tt.x, tt.y), func(t *testing.T) {
			if got := i64_rotr(tt.x, tt.y); got != tt.want {
				t.Errorf("i64_rotr(%d, %d) = %d, want %d", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func Test_f32_abs(t *testing.T) {
	got := f32_abs(math.Float32frombits(0xffc00000))
	if math.Float32bits(got) != 0x7fc00000 {
		t.Errorf("f32_abs(-NaN) = %f, want NaN", got)
	}
}

func Test_f32_copysign(t *testing.T) {
	got := f32_copysign(math.Float32frombits(0x7f800000), math.Float32frombits(0xffc00000))
	if math.Float32bits(got) != 0xff800000 {
		t.Errorf("f32_copysign(+Inf, -NaN) = %f, want -Inf", got)
	}
}

func Test_f32_min(t *testing.T) {
	got := f32_min(0, math.Float32frombits(0x80000000))
	if math.Float32bits(got) != 0x80000000 {
		t.Errorf("f32_max(+0, -0) = %f, want -0", got)
	}
}

func Test_f32_max(t *testing.T) {
	got := f32_max(0, math.Float32frombits(0x80000000))
	if math.Float32bits(got) != 0 {
		t.Errorf("f32_max(+0, -0) = %f, want +0", got)
	}
}

func Test_f64_min(t *testing.T) {
	got := f64_min(0, math.Float64frombits(0x8000000000000000))
	if math.Float64bits(got) != 0x8000000000000000 {
		t.Errorf("f64_max(+0, -0) = %f, want -0", got)
	}
}

func Test_f64_max(t *testing.T) {
	got := f64_max(0, math.Float64frombits(0x8000000000000000))
	if math.Float64bits(got) != 0 {
		t.Errorf("f64_max(+0, -0) = %f, want +0", got)
	}
}

func Test_i32_trunc_f64_s(t *testing.T) {
	tests := []struct {
		f float64
		i int32
		p bool
	}{
		{0, 0, false},
		{1, 1, false},
		{-1, -1, false},
		{1.5, 1, false},
		{-1.5, -1, false},
		{1000, 1000, false},
		{-1000, -1000, false},
		{math.MaxInt32, math.MaxInt32, false},
		{math.MinInt32, math.MinInt32, false},
		{math.MaxFloat64, 0, true},
		{-math.MaxFloat64, 0, true},
		{math.Inf(1), 0, true},
		{math.Inf(-1), 0, true},
		{math.NaN(), 0, true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			if tt.p {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("i32_trunc_f64_s(%f) did not panic", tt.f)
					}
				}()
			}
			got := i32_trunc_f64_s(tt.f)
			if got != tt.i {
				t.Errorf("i32_trunc_f64_s(%f) = %v, want %v", tt.f, got, tt.i)
			}
		})
	}
}

func Test_i32_trunc_f32_s(t *testing.T) {
	tests := []struct {
		f float32
		i int32
		p bool
	}{
		{0, 0, false},
		{1, 1, false},
		{-1, -1, false},
		{1.5, 1, false},
		{-1.5, -1, false},
		{1000, 1000, false},
		{-1000, -1000, false},
		{math.MaxInt32, 0, true},
		{math.MinInt32, math.MinInt32, false},
		{math.MaxFloat32, 0, true},
		{-math.MaxFloat32, 0, true},
		{float32(math.Inf(1)), 0, true},
		{float32(math.Inf(-1)), 0, true},
		{float32(math.NaN()), 0, true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			if tt.p {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("i32_trunc_f32_s(%f) did not panic", tt.f)
					}
				}()
			}
			got := i32_trunc_f32_s(tt.f)
			if got != tt.i {
				t.Errorf("i32_trunc_f32_s(%f) = %v, want %v", tt.f, got, tt.i)
			}
		})
	}
}

func Test_i32_trunc_f64_u(t *testing.T) {
	tests := []struct {
		f float64
		i int32
		p bool
	}{
		{0, 0, false},
		{1, 1, false},
		{-1, 0, true},
		{1.5, 1, false},
		{-1.5, 0, true},
		{1000, 1000, false},
		{-1000, 0, true},
		{math.MaxUint32, -1, false},
		{math.MaxFloat64, 0, true},
		{-math.MaxFloat64, 0, true},
		{math.Inf(1), 0, true},
		{math.Inf(-1), 0, true},
		{math.NaN(), 0, true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			if tt.p {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("i32_trunc_f64_u(%f) did not panic", tt.f)
					}
				}()
			}
			got := i32_trunc_f64_u(tt.f)
			if got != tt.i {
				t.Errorf("i32_trunc_f64_u(%f) = %v, want %v", tt.f, got, tt.i)
			}
		})
	}
}

func Test_i32_trunc_f32_u(t *testing.T) {
	tests := []struct {
		f float32
		i int32
		p bool
	}{
		{0, 0, false},
		{1, 1, false},
		{-1, 0, true},
		{1.5, 1, false},
		{-1.5, 0, true},
		{1000, 1000, false},
		{-1000, 0, true},
		{math.MaxUint32, 0, true},
		{math.MaxFloat32, 0, true},
		{-math.MaxFloat32, 0, true},
		{float32(math.Inf(1)), 0, true},
		{float32(math.Inf(-1)), 0, true},
		{float32(math.NaN()), 0, true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			if tt.p {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("i32_trunc_f32_u(%f) did not panic", tt.f)
					}
				}()
			}
			got := i32_trunc_f32_u(tt.f)
			if got != tt.i {
				t.Errorf("i32_trunc_f32_u(%f) = %v, want %v", tt.f, got, tt.i)
			}
		})
	}
}

func Test_i64_trunc_f64_s(t *testing.T) {
	tests := []struct {
		f float64
		i int64
		p bool
	}{
		{0, 0, false},
		{1, 1, false},
		{-1, -1, false},
		{1.5, 1, false},
		{-1.5, -1, false},
		{1000, 1000, false},
		{-1000, -1000, false},
		{float64(math.MaxInt64), 0, true},
		{float64(math.MinInt64), math.MinInt64, false},
		{math.MaxFloat64, 0, true},
		{-math.MaxFloat64, 0, true},
		{math.Inf(1), 0, true},
		{math.Inf(-1), 0, true},
		{math.NaN(), 0, true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			if tt.p {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("i64_trunc_f64_s(%f) did not panic", tt.f)
					}
				}()
			}
			got := i64_trunc_f64_s(tt.f)
			if got != tt.i {
				t.Errorf("i64_trunc_f64_s(%f) = %v, want %v", tt.f, got, tt.i)
			}
		})
	}
}

func Test_i64_trunc_f32_s(t *testing.T) {
	tests := []struct {
		f float32
		i int64
		p bool
	}{
		{0, 0, false},
		{1, 1, false},
		{-1, -1, false},
		{1.5, 1, false},
		{-1.5, -1, false},
		{1000, 1000, false},
		{-1000, -1000, false},
		{float32(math.MaxInt64), 0, true},
		{float32(math.MinInt64), math.MinInt64, false},
		{math.MaxFloat32, 0, true},
		{-math.MaxFloat32, 0, true},
		{float32(math.Inf(1)), 0, true},
		{float32(math.Inf(-1)), 0, true},
		{float32(math.NaN()), 0, true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			if tt.p {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("i64_trunc_f32_s(%f) did not panic", tt.f)
					}
				}()
			}
			got := i64_trunc_f32_s(tt.f)
			if got != tt.i {
				t.Errorf("i64_trunc_f32_s(%f) = %v, want %v", tt.f, got, tt.i)
			}
		})
	}
}

func Test_i64_trunc_f64_u(t *testing.T) {
	tests := []struct {
		f float64
		i int64
		p bool
	}{
		{0, 0, false},
		{1, 1, false},
		{-1, 0, true},
		{1.5, 1, false},
		{-1.5, 0, true},
		{1000, 1000, false},
		{-1000, 0, true},
		{float64(math.MaxUint64), 0, true},
		{math.MaxFloat64, 0, true},
		{-math.MaxFloat64, 0, true},
		{math.Inf(1), 0, true},
		{math.Inf(-1), 0, true},
		{math.NaN(), 0, true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			if tt.p {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("i64_trunc_f64_u(%f) did not panic", tt.f)
					}
				}()
			}
			got := i64_trunc_f64_u(tt.f)
			if got != tt.i {
				t.Errorf("i64_trunc_f64_u(%f) = %v, want %v", tt.f, got, tt.i)
			}
		})
	}
}

func Test_i64_trunc_f32_u(t *testing.T) {
	tests := []struct {
		f float32
		i int64
		p bool
	}{
		{0, 0, false},
		{1, 1, false},
		{-1, 0, true},
		{1.5, 1, false},
		{-1.5, 0, true},
		{1000, 1000, false},
		{-1000, 0, true},
		{float32(math.MaxUint64), 0, true},
		{math.MaxFloat32, 0, true},
		{-math.MaxFloat32, 0, true},
		{float32(math.Inf(1)), 0, true},
		{float32(math.Inf(-1)), 0, true},
		{float32(math.NaN()), 0, true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			if tt.p {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("i64_trunc_f32_u(%f) did not panic", tt.f)
					}
				}()
			}
			got := i64_trunc_f32_u(tt.f)
			if got != tt.i {
				t.Errorf("i64_trunc_f32_u(%f) = %v, want %v", tt.f, got, tt.i)
			}
		})
	}
}

func Test_i32_trunc_sat_f64_s(t *testing.T) {
	tests := []struct {
		f float64
		i int32
	}{
		{0, 0},
		{1, 1},
		{-1, -1},
		{1.5, 1},
		{-1.5, -1},
		{1000, 1000},
		{-1000, -1000},
		{math.MaxInt32, math.MaxInt32},
		{math.MinInt32, math.MinInt32},
		{math.MaxFloat64, math.MaxInt32},
		{-math.MaxFloat64, math.MinInt32},
		{math.Inf(1), math.MaxInt32},
		{math.Inf(-1), math.MinInt32},
		{math.NaN(), 0},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			got := i32_trunc_sat_f64_s(tt.f)
			if got != tt.i {
				t.Errorf("i32_trunc_sat_f64_s(%f) = %v, want %v", tt.f, got, tt.i)
			}
		})
	}
}

func Test_i32_trunc_sat_f32_s(t *testing.T) {
	tests := []struct {
		f float32
		i int32
	}{
		{0, 0},
		{1, 1},
		{-1, -1},
		{1.5, 1},
		{-1.5, -1},
		{1000, 1000},
		{-1000, -1000},
		{math.MaxInt32, math.MaxInt32},
		{math.MinInt32, math.MinInt32},
		{math.MaxFloat32, math.MaxInt32},
		{-math.MaxFloat32, math.MinInt32},
		{float32(math.Inf(1)), math.MaxInt32},
		{float32(math.Inf(-1)), math.MinInt32},
		{float32(math.NaN()), 0},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			got := i32_trunc_sat_f32_s(tt.f)
			if got != tt.i {
				t.Errorf("i32_trunc_sat_f32_s(%f) = %v, want %v", tt.f, got, tt.i)
			}
		})
	}
}

func Test_i32_trunc_sat_f64_u(t *testing.T) {
	tests := []struct {
		f float64
		i int32
	}{
		{0, 0},
		{1, 1},
		{-1, 0},
		{1.5, 1},
		{-1.5, 0},
		{1000, 1000},
		{-1000, 0},
		{math.MaxUint32, -1},
		{math.MaxFloat64, -1},
		{-math.MaxFloat64, 0},
		{math.Inf(1), -1},
		{math.Inf(-1), 0},
		{math.NaN(), 0},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			got := i32_trunc_sat_f64_u(tt.f)
			if got != tt.i {
				t.Errorf("i32_trunc_sat_f64_u(%f) = %v, want %v", tt.f, got, tt.i)
			}
		})
	}
}

func Test_i32_trunc_sat_f32_u(t *testing.T) {
	tests := []struct {
		f float32
		i int32
	}{
		{0, 0},
		{1, 1},
		{-1, 0},
		{1.5, 1},
		{-1.5, 0},
		{1000, 1000},
		{-1000, 0},
		{math.MaxUint32, -1},
		{math.MaxFloat32, -1},
		{-math.MaxFloat32, 0},
		{float32(math.Inf(1)), -1},
		{float32(math.Inf(-1)), 0},
		{float32(math.NaN()), 0},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			got := i32_trunc_sat_f32_u(tt.f)
			if got != tt.i {
				t.Errorf("i32_trunc_sat_f32_u(%f) = %v, want %v", tt.f, got, tt.i)
			}
		})
	}
}

func Test_i64_trunc_sat_f64_s(t *testing.T) {
	tests := []struct {
		f float64
		i int64
	}{
		{0, 0},
		{1, 1},
		{-1, -1},
		{1.5, 1},
		{-1.5, -1},
		{1000, 1000},
		{-1000, -1000},
		{math.MaxInt64, math.MaxInt64},
		{math.MinInt64, math.MinInt64},
		{math.MaxFloat64, math.MaxInt64},
		{-math.MaxFloat64, math.MinInt64},
		{math.Inf(1), math.MaxInt64},
		{math.Inf(-1), math.MinInt64},
		{math.NaN(), 0},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			got := i64_trunc_sat_f64_s(tt.f)
			if got != tt.i {
				t.Errorf("i64_trunc_sat_f64_s(%f) = %v, want %v", tt.f, got, tt.i)
			}
		})
	}
}

func Test_i64_trunc_sat_f32_s(t *testing.T) {
	tests := []struct {
		f float32
		i int64
	}{
		{0, 0},
		{1, 1},
		{-1, -1},
		{1.5, 1},
		{-1.5, -1},
		{1000, 1000},
		{-1000, -1000},
		{math.MaxInt64, math.MaxInt64},
		{math.MinInt64, math.MinInt64},
		{math.MaxFloat32, math.MaxInt64},
		{-math.MaxFloat32, math.MinInt64},
		{float32(math.Inf(1)), math.MaxInt64},
		{float32(math.Inf(-1)), math.MinInt64},
		{float32(math.NaN()), 0},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			got := i64_trunc_sat_f32_s(tt.f)
			if got != tt.i {
				t.Errorf("i64_trunc_sat_f32_s(%f) = %v, want %v", tt.f, got, tt.i)
			}
		})
	}
}

func Test_i64_trunc_sat_f64_u(t *testing.T) {
	tests := []struct {
		f float64
		i int64
	}{
		{0, 0},
		{1, 1},
		{-1, 0},
		{1.5, 1},
		{-1.5, 0},
		{1000, 1000},
		{-1000, 0},
		{math.MaxUint64, -1},
		{math.MaxFloat64, -1},
		{-math.MaxFloat64, 0},
		{math.Inf(1), -1},
		{math.Inf(-1), 0},
		{math.NaN(), 0},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			got := i64_trunc_sat_f64_u(tt.f)
			if got != tt.i {
				t.Errorf("i64_trunc_sat_f64_u(%f) = %v, want %v", tt.f, got, tt.i)
			}
		})
	}
}

func Test_i64_trunc_sat_f32_u(t *testing.T) {
	tests := []struct {
		f float32
		i int64
	}{
		{0, 0},
		{1, 1},
		{-1, 0},
		{1.5, 1},
		{-1.5, 0},
		{1000, 1000},
		{-1000, 0},
		{math.MaxUint64, -1},
		{math.MaxFloat32, -1},
		{-math.MaxFloat32, 0},
		{float32(math.Inf(1)), -1},
		{float32(math.Inf(-1)), 0},
		{float32(math.NaN()), 0},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			got := i64_trunc_sat_f32_u(tt.f)
			if got != tt.i {
				t.Errorf("i64_trunc_sat_f32_u(%f) = %v, want %v", tt.f, got, tt.i)
			}
		})
	}
}

func Test_i64_add_wide(t *testing.T) {
	tests := []struct {
		x, y int64
	}{
		{0, 0},
		{1, 1},
		{0, 1},
		{-1, 1},
		{1, -1},
		{-1, -1},
		{math.MinInt64, 1},
		{math.MaxInt64, -1},
		{math.MaxInt64, math.MaxInt64},
		{math.MinInt64, math.MinInt64},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%x+%x", tt.x, tt.y), func(t *testing.T) {
			lo1, hi1 := i64_add_wide(tt.x, tt.y)
			lo2, hi2 := i64_add128(tt.x, 0, tt.y, 0)
			if lo1 != lo2 || hi1 != hi2 {
				t.Errorf("i64_add_wide(%x, %x) = (%x, %x), want (%x, %x)", tt.x, tt.y, lo1, hi1, lo2, hi2)
			}
		})
	}
}

func Test_i64_sub_wide(t *testing.T) {
	tests := []struct {
		x, y int64
	}{
		{0, 0},
		{1, 1},
		{0, 1},
		{-1, 1},
		{1, -1},
		{-1, -1},
		{math.MinInt64, 1},
		{math.MaxInt64, -1},
		{math.MaxInt64, math.MaxInt64},
		{math.MinInt64, math.MinInt64},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%x-%x", tt.x, tt.y), func(t *testing.T) {
			lo1, hi1 := i64_sub_wide(tt.x, tt.y)
			lo2, hi2 := i64_sub128(tt.x, 0, tt.y, 0)
			if lo1 != lo2 || hi1 != hi2 {
				t.Errorf("i64_sub_wide(%x, %x) = (%x, %x), want (%x, %x)", tt.x, tt.y, lo1, hi1, lo2, hi2)
			}
		})
	}
}

func Test_i64_mul_wide_u(t *testing.T) {
	tests := []struct {
		x, y   int64
		lo, hi uint64
	}{
		{0, 0, 0, 0},
		{1, 1, 1, 0},
		{0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFE00000001, 0},
		{-1, -1, 1, 0xFFFFFFFFFFFFFFFE},
		{-1, 1, 0xFFFFFFFFFFFFFFFF, 0},
		{1, -1, 0xFFFFFFFFFFFFFFFF, 0},
		{math.MaxInt64, 2, 0xFFFFFFFFFFFFFFFE, 0},
		{math.MinInt64, 2, 0, 1},
		{math.MaxInt64, math.MaxInt64, 1, 0x3FFFFFFFFFFFFFFF},
		{math.MinInt64, math.MinInt64, 0, 0x4000000000000000},
		{math.MaxInt64, math.MinInt64, 0x8000000000000000, 0x3FFFFFFFFFFFFFFF},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%x*%x", tt.x, tt.y), func(t *testing.T) {
			lo, hi := i64_mul_wide_u(tt.x, tt.y)
			if uint64(lo) != tt.lo || uint64(hi) != tt.hi {
				t.Errorf("i64_mul_wide_u(%x, %x) = (%x, %x), want (%x, %x)", tt.x, tt.y, uint64(lo), uint64(hi), tt.lo, tt.hi)
			}
		})
	}
}

func Test_i64_mul_wide_s(t *testing.T) {
	tests := []struct {
		x, y   int64
		lo, hi uint64
	}{
		{0, 0, 0, 0},
		{1, 1, 1, 0},
		{0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFE00000001, 0},
		{-1, -1, 1, 0},
		{-1, 1, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		{1, -1, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		{math.MaxInt64, 2, 0xFFFFFFFFFFFFFFFE, 0},
		{math.MinInt64, 2, 0, 0xFFFFFFFFFFFFFFFF},
		{math.MaxInt64, math.MaxInt64, 1, 0x3FFFFFFFFFFFFFFF},
		{math.MinInt64, math.MinInt64, 0, 0x4000000000000000},
		{math.MaxInt64, math.MinInt64, 0x8000000000000000, 0xC000000000000000},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%x*%x", tt.x, tt.y), func(t *testing.T) {
			lo, hi := i64_mul_wide_s(tt.x, tt.y)
			if uint64(lo) != tt.lo || uint64(hi) != tt.hi {
				t.Errorf("i64_mul_wide_s(%x, %x) = (%x, %x), want (%x, %x)", tt.x, tt.y, uint64(lo), uint64(hi), tt.lo, tt.hi)
			}
		})
	}
}
