package helpers

import (
	"math/bits"
	"runtime"
	"sync/atomic"
	"unsafe"
)

// Use nosplit only on functions with no loops.

//go:nosplit
func atomic_load32(b []byte) uint32 {
	ptr := (*uint32)(unsafe.Pointer((*[4]byte)(b)))
	v := atomic.LoadUint32(ptr)
	if big {
		return bits.ReverseBytes32(v)
	}
	return v
}

//go:nosplit
func atomic_load64(b []byte) uint64 {
	ptr := (*uint64)(unsafe.Pointer((*[8]byte)(b)))
	v := atomic.LoadUint64(ptr)
	if big {
		return bits.ReverseBytes64(v)
	}
	return v
}

//go:nosplit
func atomic_store32(b []byte, v uint32) {
	ptr := (*uint32)(unsafe.Pointer((*[4]byte)(b)))
	if big {
		v = bits.ReverseBytes32(v)
	}
	atomic.StoreUint32(ptr, v)
}

//go:nosplit
func atomic_store64(b []byte, v uint64) {
	ptr := (*uint64)(unsafe.Pointer((*[8]byte)(b)))
	if big {
		v = bits.ReverseBytes64(v)
	}
	atomic.StoreUint64(ptr, v)
}

//go:nosplit
func atomic_xchg32(b []byte, v uint32) uint32 {
	ptr := (*uint32)(unsafe.Pointer((*[4]byte)(b)))
	if big {
		v = bits.ReverseBytes32(v)
	}
	old := atomic.SwapUint32(ptr, v)
	if big {
		return bits.ReverseBytes32(old)
	}
	return old
}

func atomic_cmpxchg32(b []byte, old, new uint32) uint32 {
	ptr := (*uint32)(unsafe.Pointer((*[4]byte)(b)))
	exp := old
	if big {
		exp = bits.ReverseBytes32(old)
		new = bits.ReverseBytes32(new)
	}
	for {
		if atomic.CompareAndSwapUint32(ptr, exp, new) {
			return old
		}
		if cur := atomic.LoadUint32(ptr); cur != exp {
			if big {
				return bits.ReverseBytes32(cur)
			}
			return cur
		}
	}
}

func atomic_add32(b []byte, v uint32) uint32 {
	ptr := (*uint32)(unsafe.Pointer((*[4]byte)(b)))
	if little {
		return atomic.AddUint32(ptr, +v) - v
	}
	for {
		cur := atomic.LoadUint32(ptr)
		old := bits.ReverseBytes32(cur)
		if atomic.CompareAndSwapUint32(ptr, cur, bits.ReverseBytes32(old+v)) {
			return old
		}
	}
}

func atomic_sub32(b []byte, v uint32) uint32 {
	ptr := (*uint32)(unsafe.Pointer((*[4]byte)(b)))
	if little {
		return atomic.AddUint32(ptr, -v) + v
	}
	for {
		cur := atomic.LoadUint32(ptr)
		old := bits.ReverseBytes32(cur)
		if atomic.CompareAndSwapUint32(ptr, cur, bits.ReverseBytes32(old-v)) {
			return old
		}
	}
}

//go:nosplit
func atomic_load8[T uint32 | int64](mem []byte, addr T) uint8 {
	ptr := (*uint32)(unsafe.Pointer(&mem[addr&^3]))
	shift := (uint32(addr) & 3) * 8
	_ = mem[addr] // bounds check

	v := atomic.LoadUint32(ptr)
	if big {
		v = bits.ReverseBytes32(v)
	}
	return uint8(v >> shift)
}

func atomic_store8[T uint32 | int64](mem []byte, addr T, v uint8) {
	ptr := (*uint32)(unsafe.Pointer(&mem[addr&^3]))
	shift := (uint32(addr) & 3) * 8
	_ = mem[addr] // bounds check

	new8 := uint32(v) << shift
	mask := uint32(255) << shift
	if big {
		new8 = bits.ReverseBytes32(new8)
		mask = bits.ReverseBytes32(mask)
	}

	for {
		cur := atomic.LoadUint32(ptr)
		if atomic.CompareAndSwapUint32(ptr, cur, (cur&^mask)|new8) {
			return
		}
	}
}

func atomic_xchg8[T uint32 | int64](mem []byte, addr T, v uint8) uint8 {
	ptr := (*uint32)(unsafe.Pointer(&mem[addr&^3]))
	shift := (uint32(addr) & 3) * 8
	_ = mem[addr] // bounds check

	new8 := uint32(v) << shift
	mask := uint32(255) << shift
	if big {
		new8 = bits.ReverseBytes32(new8)
		mask = bits.ReverseBytes32(mask)
	}

	for {
		cur := atomic.LoadUint32(ptr)
		if atomic.CompareAndSwapUint32(ptr, cur, (cur&^mask)|new8) {
			if big {
				cur = bits.ReverseBytes32(cur)
			}
			return uint8(cur >> shift)
		}
	}
}

func atomic_cmpxchg8[T uint32 | int64](mem []byte, addr T, old, new uint8) uint8 {
	ptr := (*uint32)(unsafe.Pointer(&mem[addr&^3]))
	shift := (uint32(addr) & 3) * 8
	_ = mem[addr] // bounds check

	exp8 := uint32(old) << shift
	new8 := uint32(new) << shift
	mask := uint32(255) << shift
	if big {
		exp8 = bits.ReverseBytes32(exp8)
		new8 = bits.ReverseBytes32(new8)
		mask = bits.ReverseBytes32(mask)
	}

	for {
		cur := atomic.LoadUint32(ptr)
		if cur&mask != exp8 {
			if big {
				cur = bits.ReverseBytes32(cur)
			}
			return uint8(cur >> shift)
		}
		if atomic.CompareAndSwapUint32(ptr, cur, (cur&^mask)|new8) {
			return old
		}
	}
}

// Compiler error if endianess is unknown.
var _ = map[bool]struct{}{big: {}, little: {}}

const (
	big = false ||
		runtime.GOARCH == "ppc64" || runtime.GOARCH == "s390x" ||
		runtime.GOARCH == "mips" || runtime.GOARCH == "mips64"

	little = false ||
		runtime.GOARCH == "386" || runtime.GOARCH == "amd64" ||
		runtime.GOARCH == "arm" || runtime.GOARCH == "arm64" ||
		runtime.GOARCH == "riscv64" || runtime.GOARCH == "wasm" ||
		runtime.GOARCH == "ppc64le" || runtime.GOARCH == "loong64" ||
		runtime.GOARCH == "mipsle" || runtime.GOARCH == "mips64le"
)
