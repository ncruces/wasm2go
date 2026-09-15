//go:build !generator

package main

import (
	_ "embed"
	"testing"

	oob_trap_test "github.com/ncruces/wasm2go/testdata/regression/oob_trap"
	provided_helper_test "github.com/ncruces/wasm2go/testdata/regression/provided_helper"
	select_test "github.com/ncruces/wasm2go/testdata/regression/select_effect"
	store_grow_test "github.com/ncruces/wasm2go/testdata/regression/store_grow"
)

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

func Test_regression_oob_trap(t *testing.T) {
	m := oob_trap_test.New()

	mustTrap := func(name string, f func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Errorf("%s: expected out-of-bounds trap", name)
			}
		}()
		f()
	}

	// In-bounds at the very edge of the 64 KiB memory.
	m.Xst32(65532, -1)
	if got := m.Xld32(65532); got != -1 {
		t.Errorf("ld32(65532) = %d, want -1", got)
	}
	if got := m.Xld16(65534); got != 0xffff {
		t.Errorf("ld16(65534) = %d, want 65535", got)
	}

	// First byte past the end, and accesses straddling the end.
	mustTrap("ld32 first past end", func() { m.Xld32(65536) })
	mustTrap("ld32 straddling end", func() { m.Xld32(65533) })
	mustTrap("ld16 straddling end", func() { m.Xld16(65535) })
	mustTrap("ld64 straddling end", func() { m.Xld64(65529) })
	mustTrap("st32 straddling end", func() { m.Xst32(65533, 0) })
	mustTrap("st64 first past end", func() { m.Xst64(65536, 0) })

	// v128 accesses at the edge and straddling it.
	m.Xst128(65520, 0x0102030405060708)
	if got := m.Xld128(65520); got != 0x0102030405060708 {
		t.Errorf("ld128(65520) = %#x after st128, want 0x0102030405060708", got)
	}
	mustTrap("st128 straddling end", func() { m.Xst128(65528, 0) })
	mustTrap("ld128 straddling end", func() { m.Xld128(65521) })

	// Huge static offset: effective address up to 2^33-2 must trap, not wrap.
	mustTrap("static offset 0xffffffff", func() { m.Xld32o(0) })
	mustTrap("static offset at max address", func() { m.Xld32o(-1) })

	// After growth, formerly out-of-bounds addresses become valid,
	// and the trap boundary moves to the new end.
	if got := m.Xgrow(1); got != 1 {
		t.Fatalf("grow(1) = %d, want 1", got)
	}
	m.Xst32(65536, 42)
	if got := m.Xld32(65536); got != 42 {
		t.Errorf("ld32(65536) = %d after grow, want 42", got)
	}
	mustTrap("ld32 past grown end", func() { m.Xld32(131073) })

	// The grown slice has spare capacity; 16-byte accesses must still trap
	// at the memory's length, not its capacity.
	m.Xst128(131056, 42)
	if got := m.Xld128(131056); got != 42 {
		t.Errorf("ld128(131056) = %d after grow, want 42", got)
	}
	mustTrap("st128 past grown end", func() { m.Xst128(131064, 0) })
	mustTrap("ld128 past grown end", func() { m.Xld128(131064) })
}

func Test_regression_provided_helper(t *testing.T) {
	m := provided_helper_test.New()

	if got := m.Xtest(); got != 0x0807060504030201 {
		t.Errorf("test() = %#x, want 0x0807060504030201 (provided import must be able to call load64)", got)
	}
}
