//go:build !generator

package main

import (
	_ "embed"
	"testing"

	bulk_grow_test "github.com/ncruces/wasm2go/testdata/regression/bulk_grow"
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
}

func Test_regression_bulk_grow(t *testing.T) {
	mustTrap := func(name string, f func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Errorf("%s: expected out-of-bounds trap", name)
			}
		}()
		f()
	}

	m := bulk_grow_test.New()

	// Before any growth the backing slices are exact, so these already trap.
	mustTrap("memory.fill past end", func() { m.Xfill(65536, 0xAA, 16) })
	mustTrap("table.fill past end", func() { m.Xtfill(1, 1) })

	// One page of growth: len is 128 KiB, but append leaves cap > len.
	if got := m.Xgrow(1); got != 1 {
		t.Fatalf("grow(1) = %d, want 1", got)
	}
	mem := *m.Xmemory().Slice()
	t.Logf("after memory.grow: len=%d cap=%d", len(mem), cap(mem))

	const end = 2 << 16
	mustTrap("memory.fill past grown end", func() { m.Xfill(end, 0xAA, 16) })
	mustTrap("memory.copy src past grown end", func() { m.Xcopy(0, end, 16) })
	mustTrap("memory.copy dst past grown end", func() { m.Xcopy(end, 0, 16) })
	mustTrap("memory.init past grown end", func() { m.Xinit(end, 0, 4) })

	// If the fill and copy above did not trap, bytes written past the end
	// were just read back into bounds.
	if got := m.Xld8(0); got != 0 {
		t.Errorf("ld8(0) = %#x, want 0: bytes written past the end were read back", got)
	}

	// Same for tables: 1 -> 2 -> 3 elements leaves cap 4.
	m.Xtgrow(1)
	if got := m.Xtgrow(1); got != 2 {
		t.Fatalf("table.grow(1) = %d, want 2", got)
	}
	mustTrap("table.fill past grown end", func() { m.Xtfill(3, 1) })
}

func Test_regression_provided_helper(t *testing.T) {
	m := provided_helper_test.New()

	if got := m.Xtest(); got != 0x0807060504030201 {
		t.Errorf("test() = %#x, want 0x0807060504030201 (provided import must be able to call load64)", got)
	}
}
