package helpers

import (
	"math"
	"strconv"
	"testing"
)

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
		t.Run(strconv.FormatFloat(tt.f, 'f', -1, 64), func(t *testing.T) {
			got := i64_trunc_sat_f64_s(tt.f)
			if got != tt.i {
				t.Errorf("i64_trunc_sat_f64_s(%f) = %v, want %v", tt.f, got, tt.i)
			}
		})
	}
}
