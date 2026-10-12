// Compare fixed-width windows with the original little-endian accessors.
// Full-slice capacity and integer overflow must preserve Wasm memory traps.
// https://go.dev/ref/spec#Slice_expressions

package helpers

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"testing"
)

func caught(f func()) (trapped bool) {
	defer func() {
		if recover() != nil {
			trapped = true
		}
	}()
	f()
	return
}
func readFor[T uint32 | uint64](b []byte, a T, n int) uint64 {
	switch n {
	case 2:
		return uint64(load16(b, a))
	case 4:
		return uint64(load32(b, a))
	default:
		return load64(b, a)
	}
}
func writeFor[T uint32 | uint64](b []byte, a T, n int, v uint64) {
	switch n {
	case 2:
		store16(b, a, uint16(v))
	case 4:
		store32(b, a, uint32(v))
	default:
		store64(b, a, v)
	}
}
func referenceRead(b []byte, a uint64, n int) uint64 {
	switch n {
	case 2:
		return uint64(binary.LittleEndian.Uint16(b[a:]))
	case 4:
		return uint64(binary.LittleEndian.Uint32(b[a:]))
	default:
		return binary.LittleEndian.Uint64(b[a:])
	}
}
func referenceWrite(b []byte, a uint64, n int, v uint64) {
	switch n {
	case 2:
		binary.LittleEndian.PutUint16(b[a:], uint16(v))
	case 4:
		binary.LittleEndian.PutUint32(b[a:], uint32(v))
	default:
		binary.LittleEndian.PutUint64(b[a:], v)
	}
}
func checkAddress[T uint32 | uint64](t *testing.T, backing []byte, length, n int, a T) {
	t.Helper()
	want := append([]byte(nil), backing...)
	got := append([]byte(nil), backing...)
	ws, gs := want[:length], got[:length]
	var w, g uint64
	wt := caught(func() { w = referenceRead(ws, uint64(a), n) })
	gt := caught(func() { g = readFor(gs, a, n) })
	if wt != gt || (!wt && w != g) {
		t.Fatalf("read width=%d len=%d cap=%d addr=%x traps=%v/%v values=%x/%x", n, length, cap(gs), a, wt, gt, w, g)
	}
	value := uint64(0x123456789abcdef0)
	wt = caught(func() { referenceWrite(ws, uint64(a), n, value) })
	gt = caught(func() { writeFor(gs, a, n, value) })
	if wt != gt || !bytes.Equal(want, got) {
		t.Fatalf("write width=%d len=%d cap=%d addr=%x traps=%v/%v", n, length, cap(gs), a, wt, gt)
	}
	if gt && !bytes.Equal(got, backing) {
		t.Fatal("trapping write mutated backing storage")
	}
}
func checkType[T uint32 | uint64](t *testing.T, extreme []T) {
	t.Helper()
	for _, capacity := range []int{0, 1, 2, 3, 4, 7, 8, 9, 16, 31, 64, 127, 256} {
		backing := make([]byte, capacity)
		for i := range backing {
			backing[i] = byte(i*37 + 11)
		}
		for length := 0; length <= capacity; length++ {
			for _, n := range []int{2, 4, 8} {
				for a := 0; a <= length+8; a++ {
					checkAddress(t, backing, length, n, T(a))
				}
				for _, a := range extreme {
					checkAddress(t, backing, length, n, a)
				}
			}
		}
	}
}
func TestMemoryWindows32(t *testing.T) {
	checkType(t, []uint32{math.MaxUint32, math.MaxUint32 - 1, math.MaxUint32 - 7, 1 << 31})
}
func TestMemoryWindows64(t *testing.T) {
	checkType(t, []uint64{math.MaxUint64, math.MaxUint64 - 1, math.MaxUint64 - 7, 1 << 63, 1 << 32, (1 << 32) + 1})
}
func TestMemoryGrowth(t *testing.T) {
	for _, n := range []int{2, 4, 8} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			b := make([]byte, 65536)
			a := uint64(len(b))
			if !caught(func() { readFor(b, a, n) }) {
				t.Fatal("access beyond initial memory accepted")
			}
			if memory_grow(&b, 1, 4) != 1 {
				t.Fatal("grow")
			}
			writeFor(b, a, n, 12345)
			if readFor(b, a, n) != 12345 {
				t.Fatal("new page")
			}
			if !caught(func() { writeFor(b, uint64(len(b)-n+1), n, 42) }) {
				t.Fatal("end-of-memory write accepted")
			}
		})
	}
}
