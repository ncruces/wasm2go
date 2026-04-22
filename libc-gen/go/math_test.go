package libc

import (
	"math"
	"testing"
)

var zero float64

func Test_lgamma(t *testing.T) {
	got := lgamma(2.5)
	want, _ := math.Lgamma(2.5)
	checkFloat(t, got, want)
}

func Test_lgamma_r(t *testing.T) {
	memory = make([]byte, 1024)
	sptr := ptr(4)

	got := lgamma_r(2.5, sptr)
	want, sign := math.Lgamma(2.5)

	checkFloat(t, got, want)

	gotSign := int32(load32(memory[uptr(sptr):]))
	if gotSign != int32(sign) {
		t.Errorf("want sign %v, got %v", sign, gotSign)
	}
}

func Test_frexp(t *testing.T) {
	memory = make([]byte, 1024)
	eptr := ptr(8)

	got := frexp(16.0, eptr)
	want, exp := math.Frexp(16.0)

	checkFloat(t, got, want)

	gotExp := int32(load32(memory[uptr(eptr):]))
	if gotExp != int32(exp) {
		t.Errorf("want exp %v, got %v", exp, gotExp)
	}
}

func Test_modf(t *testing.T) {
	memory = make([]byte, 1024)
	iptr := ptr(16)

	tests := []struct {
		name  string
		x     float64
		wantI float64
		wantF float64
	}{
		{"positive", 3.14, 3.0, 0.14000000000000012},
		{"negative", -2.71, -2.0, -0.7100000000000004},
		{"inf", math.Inf(1), math.Inf(1), 0.0},
		{"-inf", math.Inf(-1), math.Inf(-1), math.Copysign(0, -1)},
		{"zero", 0.0, 0.0, 0.0},
		{"-zero", math.Copysign(0, -1), math.Copysign(0, -1), math.Copysign(0, -1)},
		{"nan", math.NaN(), math.NaN(), math.NaN()},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store64(memory[uptr(iptr):], 0)
			gotF := modf(tc.x, iptr)
			gotI := math.Float64frombits(load64(memory[uptr(iptr):]))

			checkFloat(t, gotI, tc.wantI)
			checkFloat(t, gotF, tc.wantF)
		})
	}
}

func Test_fmax(t *testing.T) {
	tests := []struct {
		x, y float64
		want float64
	}{
		{1, 2, 2},
		{2, 1, 2},
		{-1, 1, 1},
		{math.NaN(), 1, 1},
		{1, math.NaN(), 1},
		{math.NaN(), math.NaN(), math.NaN()},
		{0.0, -zero, 0.0},
		{-zero, 0.0, 0.0},
	}

	for _, tc := range tests {
		got := fmax(tc.x, tc.y)
		checkFloat(t, got, tc.want)
	}
}

func Test_fmin(t *testing.T) {
	tests := []struct {
		x, y float64
		want float64
	}{
		{1, 2, 1},
		{2, 1, 1},
		{-1, 1, -1},
		{math.NaN(), 1, 1},
		{1, math.NaN(), 1},
		{math.NaN(), math.NaN(), math.NaN()},
		{-zero, 0.0, -zero},
		{0.0, -zero, -zero},
	}

	for _, tc := range tests {
		got := fmin(tc.x, tc.y)
		checkFloat(t, got, tc.want)
	}
}

// checkFloat checks floating point equality exactly, handling NaNs and signed zeros.
func checkFloat(t *testing.T, got, want float64) {
	t.Helper()
	if math.Float64bits(got) != math.Float64bits(want) &&
		math.IsNaN(got) != math.IsNaN(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
