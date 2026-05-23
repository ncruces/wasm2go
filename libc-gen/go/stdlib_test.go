package libc

import (
	"math"
	"testing"
)

func Test_strtod(t *testing.T) {
	memory = make([]byte, 1024)

	tests := []struct {
		input  string
		want   float64
		offset ptr
	}{
		{"123.45", 123.45, 6},
		{"  -98.76xyz", -98.76, 8},
		{"abc", 0, 0},
		{"1e3", 1000, 3},
		{"NaN", math.NaN(), 3},
		{"Infinity", math.Inf(1), 8},
		{"  +Inf", math.Inf(1), 6},
		{" -Inf", math.Inf(-1), 5},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			start := ptr(16)
			endptr := ptr(4)

			writeString(start, tc.input)
			store32(memory[uptr(endptr):], 0)

			got := strtod(start, endptr)
			checkFloat(t, got, tc.want)

			end := ptr(load32(memory[uptr(endptr):]))
			if tc.offset != end-start {
				t.Errorf("got %d, want %v endptr offset", end-start, tc.offset)
			}
		})
	}
}

func Test_strtol(t *testing.T) {
	memory = make([]byte, 1024)

	tests := []struct {
		input  string
		base   int32
		want   int32
		offset ptr
	}{
		{"12345", 10, 12345, 5},
		{"  -42abc", 10, -42, 5},
		{"0x1a", 0, 26, 4},
		{"1A", 16, 26, 2},
		{"Z", 36, 35, 1},
		{"abc", 10, 0, 0},
		{"  +99", 10, 99, 5},
		{"2147483647", 10, 2147483647, 10},
		{"-2147483648", 10, -2147483648, 11},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			start := ptr(16)
			endptr := ptr(4)

			writeString(start, tc.input)
			store32(memory[uptr(endptr):], 0)

			got := strtol(start, endptr, tc.base)
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}

			end := ptr(load32(memory[uptr(endptr):]))
			if tc.offset != end-start {
				t.Errorf("got %d, want %v endptr offset", end-start, tc.offset)
			}
		})
	}
}

func Test_strtoul(t *testing.T) {
	memory = make([]byte, 1024)

	tests := []struct {
		input  string
		base   int32
		want   uint32
		offset ptr
	}{
		{"12345", 10, 12345, 5},
		{"  -42abc", 10, 4294967254, 5},
		{"0x1a", 0, 26, 4},
		{"1A", 16, 26, 2},
		{"Z", 36, 35, 1},
		{"abc", 10, 0, 0},
		{"  +99", 10, 99, 5},
		{"-1", 10, 4294967295, 2},
		{"-", 10, 0, 0},
		{"-+1", 10, 0, 0},
		{"4294967295", 10, 4294967295, 10},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			start := ptr(16)
			endptr := ptr(4)

			writeString(start, tc.input)
			store32(memory[uptr(endptr):], 0)

			got := strtoul(start, endptr, tc.base)
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}

			end := ptr(load32(memory[uptr(endptr):]))
			if tc.offset != end-start {
				t.Errorf("got %d, want %v endptr offset", end-start, tc.offset)
			}
		})
	}
}
