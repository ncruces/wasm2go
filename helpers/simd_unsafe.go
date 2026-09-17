//go:build ignore

package helpers

import (
	"encoding/binary"
	"math/bits"
	"unsafe"
)

//go:nosplit
func load128(mem []byte, addr uint64) v128 {
	b := (*[16]byte)(mem[addr:])
	if !unalignedOK {
		return v128{
			binary.LittleEndian.Uint64(b[:8]),
			binary.LittleEndian.Uint64(b[8:])}
	}
	val := *(*v128)(unsafe.Pointer(b))
	if big {
		val.hi = bits.ReverseBytes64(val.hi)
		val.lo = bits.ReverseBytes64(val.lo)
	}
	return val
}

//go:nosplit
func store128(mem []byte, addr uint64, val v128) {
	b := (*[16]byte)(mem[addr:])
	if !unalignedOK {
		binary.LittleEndian.PutUint64(b[:8], val.lo)
		binary.LittleEndian.PutUint64(b[8:], val.hi)
		return
	}
	if big {
		val.hi = bits.ReverseBytes64(val.hi)
		val.lo = bits.ReverseBytes64(val.lo)
	}
	*(*v128)(unsafe.Pointer(b)) = val
}
