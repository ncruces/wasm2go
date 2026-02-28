package main

import (
	"bytes"
	_ "embed"
	"math"
	"os"
	"slices"
	"testing"

	fib_test "github.com/ncruces/wasm2go/testdata/fib"
	primes_test "github.com/ncruces/wasm2go/testdata/primes"
	recursion_test "github.com/ncruces/wasm2go/testdata/recursion"
	stack_test "github.com/ncruces/wasm2go/testdata/stack"
	table_test "github.com/ncruces/wasm2go/testdata/table"
	trig_test "github.com/ncruces/wasm2go/testdata/trig"
)

func Test_generate(t *testing.T) {
	tests := []string{"fib", "memory", "primes", "recursion", "stack", "table", "trig"}

	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			path := "testdata/" + name + "/" + name

			in, err := os.Open(path + ".wasm")
			if err != nil {
				t.Fatal(err)
			}
			defer in.Close()

			var out bytes.Buffer
			err = translate(in, &out)
			if err != nil {
				t.Fatal(err)
			}

			err = os.WriteFile(path+".go", out.Bytes(), 0644)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func Test_spec(t *testing.T) {
	tests := []string{
		"i32", "i64", "f32", "f64",
		"block", "stack",
	}

	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			path := "spectest/" + name + "/" + name

			in, err := os.Open(path + ".wasm")
			if err != nil {
				t.Fatal(err)
			}
			defer in.Close()

			var out bytes.Buffer
			err = translate(in, &out)
			if err != nil {
				t.Fatal(err)
			}

			err = os.WriteFile(path+".go", out.Bytes(), 0644)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

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
