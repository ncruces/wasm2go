package libc

import "encoding/binary"

var memory []byte

type (
	sptr_t int32   // int32  or int64
	uptr_t uint32  // uint32 or uint64
	cptr_t uintptr // uint32 or uint64
)

func load16(mem []byte, addr cptr_t) uint16 {
	return binary.LittleEndian.Uint16(mem[addr:])
}

func store16(mem []byte, addr cptr_t, val uint16) {
	binary.LittleEndian.PutUint16(mem[addr:], val)
}

func load32(mem []byte, addr cptr_t) uint32 {
	return binary.LittleEndian.Uint32(mem[addr:])
}

func store32(mem []byte, addr cptr_t, val uint32) {
	binary.LittleEndian.PutUint32(mem[addr:], val)
}

func load64(mem []byte, addr cptr_t) uint64 {
	return binary.LittleEndian.Uint64(mem[addr:])
}

func store64(mem []byte, addr cptr_t, val uint64) {
	binary.LittleEndian.PutUint64(mem[addr:], val)
}
