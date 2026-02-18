package main

import (
	_ "embed"
	"os"
	"slices"
	"testing"

	"github.com/ncruces/wasm2go/testdata/fib"
	"github.com/ncruces/wasm2go/testdata/primes"
)

func Test_generate(t *testing.T) {
	tests := []string{"fib", "primes"}

	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			path := "testdata/" + name + "/" + name

			in, err := os.Open(path + ".wasm")
			if err != nil {
				t.Fatal(err)
			}
			defer in.Close()

			out, err := os.Create(path + ".go")
			if err != nil {
				t.Fatal(err)
			}
			defer out.Close()

			err = translate(name, in, out)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func Test_fib(t *testing.T) {
	want := []int64{0, 1, 1, 2, 3, 5, 8, 13, 21}

	var m fib.Module

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

	var m primes.Module

	var got []int32
	for i := range want {
		got = append(got, m.Xis_prime(int32(i)))
	}

	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
