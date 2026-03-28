package main

import (
	_ "embed"
	"encoding/binary"
	"math"
	"math/rand"
	"slices"
	"testing"

	fib_test "github.com/ncruces/wasm2go/testdata/fib"
	loops_test "github.com/ncruces/wasm2go/testdata/loops"
	primes_test "github.com/ncruces/wasm2go/testdata/primes"
	recursion_test "github.com/ncruces/wasm2go/testdata/recursion"
	stack_test "github.com/ncruces/wasm2go/testdata/stack"
	table_test "github.com/ncruces/wasm2go/testdata/table"
	trig_test "github.com/ncruces/wasm2go/testdata/trig"
)

func Test_fib(t *testing.T) {
	want := []int64{0, 1, 1, 2, 3, 5, 8, 13, 21}

	var m fib_test.Module

	var got []int64
	for i := range want {
		got = append(got, m.Xfibonacci(int64(i)))
	}

	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_primes(t *testing.T) {
	want := []int32{
		0, // 0
		0, // 1
		1, // 2
		1, // 3
		0, // 4
		1, // 5
		0, // 6
		1, // 7
		0, // 8
		0, // 9
		0, // 10
		1, // 11
		0, // 12
		1, // 13
		0, // 14
		0, // 15
		0, // 16
		1, // 17
		0, // 18
		1, // 19
	}

	var m primes_test.Module

	var got []int32
	for i := range want {
		got = append(got, m.Xis_prime(int32(i)))
	}

	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_recursive_factorial(t *testing.T) {
	want := []int32{1, 1, 2, 6, 24, 120, 720, 5040, 40320, 362880, 3628800}

	var m recursion_test.Module

	var got []int32
	for i := range want {
		got = append(got, m.Xfactorial(int32(i)))
	}

	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_recursive_evenodd(t *testing.T) {
	var m recursion_test.Module

	for i := range 100 {
		even := m.Xis_even(int32(i))
		odd := m.Xis_odd(int32(i))
		if i%2 == 0 {
			if even == 0 && odd != 0 {
				t.Errorf("i: %d, even: %d, odd: %d", i, even, odd)
			}
		} else {
			if even != 0 && odd == 0 {
				t.Errorf("i: %d, even: %d, odd: %d", i, even, odd)
			}
		}
	}
}

func Test_stack(t *testing.T) {
	var m stack_test.Module

	if got := m.Xstack_func_call(); got != (91 - 23) {
		t.Errorf("got %d, want %d", got, 91-23)
	}

	if got1, got2 := m.Xtee_for_two(5, 3); got1 != 13 || got2 != 8 {
		t.Errorf("got %d, %d, want %d, %d", got1, got2, 13, 8)
	}
}

func Test_table(t *testing.T) {
	m := table_test.New(tableEnv{})

	if got := m.Xtimes2(5); got != 2*5 {
		t.Errorf("got %d, want %d", got, 2*5)
	}

	if got := m.Xtimes3(5); got != 3*5 {
		t.Errorf("got %d, want %d", got, 3*5)
	}
}

type tableEnv struct{}

func (t tableEnv) Xjstimes3(v0 int32) int32 {
	return v0 * 3
}

func Test_trig(t *testing.T) {
	want := []float32{
		float32(math.Sin(0)),
		float32(math.Sin(1)),
		float32(math.Sin(2)),
		float32(math.Sin(3)),
		float32(math.Sin(4)),
		float32(math.Sin(5)),
		float32(math.Sin(6)),
		float32(math.Sin(7)),
	}

	var m trig_test.Module

	var got []float32
	for i := range want {
		got = append(got, float32(m.Xsin(float64(i))))
	}

	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_loops(t *testing.T) {
	m := loops_test.New(loopsEnv{})
	mem := *m.Memory.Slice()

	var want int
	const count = 50
	const start = 128
	// Initialize data in memory.
	for i := range count {
		binary.LittleEndian.PutUint32(mem[start+i*4:], uint32(i*2-1))
		want += i*2 - 1
	}

	if got := m.Xadd_all(int32(start), count); got != int32(want) {
		t.Errorf("got %v, want %v", got, want)
	}

	for range 100 {
		// Make sure rand_multiple_of_10 always returns a multiple of 10.
		if got := m.Xrand_multiple_of_10(); got%10 != 0 {
			t.Errorf("got %v, want multiple of 10", got)
		}
	}

	if got := m.Xfirst_power_over_limit(2, 1000); got != 1024 {
		t.Errorf("got %v, want %v", got, 1024)
	}
	if got := m.Xfirst_power_over_limit(2, 16); got != 32 {
		t.Errorf("got %v, want %v", got, 32)
	}
	if got := m.Xfirst_power_over_limit(2, 0); got != 1 {
		t.Errorf("got %v, want %v", got, 1)
	}
	if got := m.Xfirst_power_over_limit(3, 25); got != 27 {
		t.Errorf("got %v, want %v", got, 27)
	}
	if got := m.Xfirst_power_over_limit(25, 10000); got != 15625 {
		t.Errorf("got %v, want %v", got, 15625)
	}
}

type loopsEnv struct {
	mem [65536]byte
}

func (e loopsEnv) Xbuffer() loops_test.Memory {
	return &e
}

func (loopsEnv) Xlog_i32(v0 int32) {}

func (loopsEnv) Xrand_i32() int32 { return rand.Int31n(10000) }

func (e *loopsEnv) Slice() *[]byte { b := e.mem[:]; return &b }

func (e *loopsEnv) Grow(delta, max int64) int64 { return -1 }
