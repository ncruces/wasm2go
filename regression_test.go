//go:build !generator

package main

import (
	_ "embed"
	"testing"

	br_table_test "github.com/ncruces/wasm2go/testdata/regression/br_table"
	select_test "github.com/ncruces/wasm2go/testdata/regression/select_effect"
	store_grow_test "github.com/ncruces/wasm2go/testdata/regression/store_grow"
)

func Test_regression_br_table(t *testing.T) {
	m := br_table_test.New()

	// Targets fall through the ladder: 0 takes +1, +10, then the early
	// exit (r == 11) past +100; 1 takes +10, +100; 2 takes +100; the
	// default target only the final +1000.
	tests := [...]struct{ x, want int32 }{
		{0, 1011}, {1, 1110}, {2, 1100}, {3, 1000}, {100, 1000}, {-1, 1000},
	}
	for _, tt := range tests {
		if got := m.Xclassify(tt.x); got != tt.want {
			t.Errorf("classify(%d) = %d, want %d", tt.x, got, tt.want)
		}
	}
}

func Test_regression_select_effect(t *testing.T) {
	m := select_test.New()

	if got := m.Xtest(0); got != 5 {
		t.Errorf("test(0) = %d, want 5", got)
	}
	if got := m.Xcounter(); got != 1 {
		t.Errorf("counter = %d after test(0), want 1 (operand must run unconditionally)", got)
	}

	if got := m.Xtest(1); got != 100 {
		t.Errorf("test(1) = %d, want 100", got)
	}
	if got := m.Xcounter(); got != 2 {
		t.Errorf("counter = %d after test(1), want 2", got)
	}
}

func Test_regression_store_grow(t *testing.T) {
	m := store_grow_test.New()

	if got := m.Xtest(); got != 12345 {
		t.Errorf("test() = %d, want 12345 (store must use memory slice after memory.grow)", got)
	}
	if got := m.Xsize(); got != 2 {
		t.Errorf("size = %d, want 2", got)
	}
}
