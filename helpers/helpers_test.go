package helpers

import (
	"fmt"
	"math"
	"testing"
)

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
		t.Run(fmt.Sprintf("%d/%d", tt.x, tt.y), func(t *testing.T) {
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
		t.Run(fmt.Sprintf("%d/%d", tt.x, tt.y), func(t *testing.T) {
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
