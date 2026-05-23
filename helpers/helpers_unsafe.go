//go:build ignore

package helpers

import (
	"encoding/binary"
	"math/bits"
	"unsafe"
)

// Faster memory access, using unsafe.

//go:nosplit
func load16(b []byte) uint16 {
	if !unalignedOK {
		return binary.LittleEndian.Uint16(b)
	}
	v := *(*uint16)(unsafe.Pointer((*[2]byte)(b)))
	if big {
		return bits.ReverseBytes16(v)
	}
	return v
}

//go:nosplit
func store16(b []byte, v uint16) {
	if !unalignedOK {
		binary.LittleEndian.PutUint16(b, v)
		return
	}
	if big {
		v = bits.ReverseBytes16(v)
	}
	*(*uint16)(unsafe.Pointer((*[2]byte)(b))) = v
}

//go:nosplit
func load32(b []byte) uint32 {
	if !unalignedOK {
		return binary.LittleEndian.Uint32(b)
	}
	v := *(*uint32)(unsafe.Pointer((*[4]byte)(b)))
	if big {
		return bits.ReverseBytes32(v)
	}
	return v
}

//go:nosplit
func store32(b []byte, v uint32) {
	if !unalignedOK {
		binary.LittleEndian.PutUint32(b, v)
		return
	}
	if big {
		v = bits.ReverseBytes32(v)
	}
	*(*uint32)(unsafe.Pointer((*[4]byte)(b))) = v
}

//go:nosplit
func load64(b []byte) uint64 {
	if !unalignedOK {
		return binary.LittleEndian.Uint64(b)
	}
	v := *(*uint64)(unsafe.Pointer((*[8]byte)(b)))
	if big {
		return bits.ReverseBytes64(v)
	}
	return v
}

//go:nosplit
func store64(b []byte, v uint64) {
	if !unalignedOK {
		binary.LittleEndian.PutUint64(b, v)
		return
	}
	if big {
		v = bits.ReverseBytes64(v)
	}
	*(*uint64)(unsafe.Pointer((*[8]byte)(b))) = v
}
