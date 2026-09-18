//go:build ignore

package helpers

import (
	"encoding/binary"
	"math/bits"
	"unsafe"
)

// Faster memory access, using unsafe.

//go:nosplit
func load16(mem []byte, addr uint64) uint16 {
	if !unalignedOK {
		return binary.LittleEndian.Uint16(mem[addr:])
	}
	_ = mem[addr+1]
	val := *(*uint16)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(mem)), uintptr(addr)))
	if big {
		return bits.ReverseBytes16(val)
	}
	return val
}

//go:nosplit
func store16(mem []byte, addr uint64, val uint16) {
	if !unalignedOK {
		binary.LittleEndian.PutUint16(mem[addr:], val)
		return
	}
	if big {
		val = bits.ReverseBytes16(val)
	}
	_ = mem[addr+1]
	*(*uint16)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(mem)), uintptr(addr))) = val
}

//go:nosplit
func load32(mem []byte, addr uint64) uint32 {
	if !unalignedOK {
		return binary.LittleEndian.Uint32(mem[addr:])
	}
	_ = mem[addr+3]
	val := *(*uint32)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(mem)), uintptr(addr)))
	if big {
		return bits.ReverseBytes32(val)
	}
	return val
}

//go:nosplit
func store32(mem []byte, addr uint64, val uint32) {
	if !unalignedOK {
		binary.LittleEndian.PutUint32(mem[addr:], val)
		return
	}
	if big {
		val = bits.ReverseBytes32(val)
	}
	_ = mem[addr+3]
	*(*uint32)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(mem)), uintptr(addr))) = val
}

//go:nosplit
func load64(mem []byte, addr uint64) uint64 {
	if !unalignedOK {
		return binary.LittleEndian.Uint64(mem[addr:])
	}
	_ = mem[addr+7]
	val := *(*uint64)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(mem)), uintptr(addr)))
	if big {
		return bits.ReverseBytes64(val)
	}
	return val
}

//go:nosplit
func store64(mem []byte, addr uint64, val uint64) {
	if !unalignedOK {
		binary.LittleEndian.PutUint64(mem[addr:], val)
		return
	}
	if big {
		val = bits.ReverseBytes64(val)
	}
	_ = mem[addr+7]
	*(*uint64)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(mem)), uintptr(addr))) = val
}
