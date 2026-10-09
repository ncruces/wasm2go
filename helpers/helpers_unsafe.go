//go:build ignore

package helpers

import (
	"encoding/binary"
	"math/bits"
	"unsafe"
)

// Faster memory access, using unsafe.
//
// Any usage of unsafe to dereference an address is preceeded by
// a bounds check on the highest referenced byte.
//
// When called with uint32 addresses, this is trivially safe.
// However, uint64 addresses present the risk of overflow.
//
// As such, these functions assume that adding a small constant to
// an uint64 addresse cannot cause it to overflow.
//
// To ensure this is the case, callers must wrap arbitrary int64
// addresses with the p64 helper before adding any constant offset.

//go:nosplit
func load16[T uint32 | uint64](mem []byte, addr T) uint16 {
	if !unalignedOK {
		return binary.LittleEndian.Uint16(mem[addr:])
	}
	_ = mem[uint64(addr)+1]
	val := *(*uint16)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(mem)), uintptr(addr)))
	if big {
		return bits.ReverseBytes16(val)
	}
	return val
}

//go:nosplit
func store16[T uint32 | uint64](mem []byte, addr T, val uint16) {
	if !unalignedOK {
		binary.LittleEndian.PutUint16(mem[addr:], val)
		return
	}
	if big {
		val = bits.ReverseBytes16(val)
	}
	_ = mem[uint64(addr)+1]
	*(*uint16)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(mem)), uintptr(addr))) = val
}

//go:nosplit
func load32[T uint32 | uint64](mem []byte, addr T) uint32 {
	if !unalignedOK {
		return binary.LittleEndian.Uint32(mem[addr:])
	}
	_ = mem[uint64(addr)+3]
	val := *(*uint32)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(mem)), uintptr(addr)))
	if big {
		return bits.ReverseBytes32(val)
	}
	return val
}

//go:nosplit
func store32[T uint32 | uint64](mem []byte, addr T, val uint32) {
	if !unalignedOK {
		binary.LittleEndian.PutUint32(mem[addr:], val)
		return
	}
	if big {
		val = bits.ReverseBytes32(val)
	}
	_ = mem[uint64(addr)+3]
	*(*uint32)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(mem)), uintptr(addr))) = val
}

//go:nosplit
func load64[T uint32 | uint64](mem []byte, addr T) uint64 {
	if !unalignedOK {
		return binary.LittleEndian.Uint64(mem[addr:])
	}
	_ = mem[uint64(addr)+7]
	val := *(*uint64)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(mem)), uintptr(addr)))
	if big {
		return bits.ReverseBytes64(val)
	}
	return val
}

//go:nosplit
func store64[T uint32 | uint64](mem []byte, addr T, val uint64) {
	if !unalignedOK {
		binary.LittleEndian.PutUint64(mem[addr:], val)
		return
	}
	if big {
		val = bits.ReverseBytes64(val)
	}
	_ = mem[uint64(addr)+7]
	*(*uint64)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(mem)), uintptr(addr))) = val
}
