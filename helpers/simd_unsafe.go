//go:build ignore

package helpers

import (
	"encoding/binary"
	"math/bits"
	"unsafe"
)

//go:nosplit
func load128[T uint32 | uint64](mem []byte, addr T) v128 {
	if !unalignedOK {
		b := (*[16]byte)(mem[addr:])
		return v128{
			binary.LittleEndian.Uint64(b[:8]),
			binary.LittleEndian.Uint64(b[8:])}
	}
	_ = mem[uint64(addr)+15]
	val := *(*v128)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(mem)), uintptr(addr)))
	if big {
		val.hi = bits.ReverseBytes64(val.hi)
		val.lo = bits.ReverseBytes64(val.lo)
	}
	return val
}

//go:nosplit
func store128[T uint32 | uint64](mem []byte, addr T, val v128) {
	if !unalignedOK {
		b := (*[16]byte)(mem[addr:])
		binary.LittleEndian.PutUint64(b[:8], val.lo)
		binary.LittleEndian.PutUint64(b[8:], val.hi)
		return
	}
	if big {
		val.hi = bits.ReverseBytes64(val.hi)
		val.lo = bits.ReverseBytes64(val.lo)
	}
	_ = mem[uint64(addr)+15]
	*(*v128)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(mem)), uintptr(addr))) = val
}
