package helpers

import (
	"math/bits"
	"runtime"
	"sync/atomic"
	"unsafe"
)

//go:nosplit
func atomic_load32(b []byte) uint32 {
	ptr := (*uint32)(unsafe.Pointer((*[4]byte)(b)))
	v := atomic.LoadUint32(ptr)
	if little {
		return v
	}
	return bits.ReverseBytes32(v)
}

//go:nosplit
func atomic_load64(b []byte) uint64 {
	ptr := (*uint64)(unsafe.Pointer((*[8]byte)(b)))
	v := atomic.LoadUint64(ptr)
	if little {
		return v
	}
	return bits.ReverseBytes64(v)
}

//go:nosplit
func atomic_load8[T uint32 | uint64](mem []byte, addr T) uint8 {
	_ = mem[addr] // bounds check
	ptr := (*uint32)(unsafe.Pointer(&mem[addr&^3]))
	shift := (uint32(addr) & 3) * 8
	v := atomic.LoadUint32(ptr)
	if big {
		v = bits.ReverseBytes32(v)
	}
	return uint8(v >> shift)
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

func atomic_store8[T uint32 | uint64](mem []byte, addr T, v uint32) {
	_ = mem[addr] // bounds check
	ptr := (*uint32)(unsafe.Pointer(&mem[addr&^3]))
	shift := (uint32(addr) & 3) * 8

	v = (v & 0xFF) << shift
	mask := ^(uint32(0xFF) << shift)
	if big {
		v = bits.ReverseBytes32(v)
		mask = bits.ReverseBytes32(mask)
	}

	for {
		old := atomic.LoadUint32(ptr)
		new := (old & mask) | v
		if atomic.CompareAndSwapUint32(ptr, old, new) {
			return
		}
	}
}

func atomic_cmpxchg32(b []byte, old, new uint32) uint32 {
	ptr := (*uint32)(unsafe.Pointer((*[4]byte)(b)))
	exp := old
	if big {
		exp = bits.ReverseBytes32(old)
		new = bits.ReverseBytes32(new)
	}
	for {
		if old := atomic.LoadUint32(ptr); old != exp {
			if little {
				return old
			}
			return bits.ReverseBytes32(old)
		}
		if atomic.CompareAndSwapUint32(ptr, exp, new) {
			return old
		}
	}
}

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
