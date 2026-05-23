package libc

import "encoding/binary"

type ptr int32
type uptr uint32

var memory []byte

func load16(b []byte) uint16 { return binary.LittleEndian.Uint16(b) }
func load32(b []byte) uint32 { return binary.LittleEndian.Uint32(b) }
func load64(b []byte) uint64 { return binary.LittleEndian.Uint64(b) }

func store16(b []byte, v uint16) { binary.LittleEndian.PutUint16(b, v) }
func store32(b []byte, v uint32) { binary.LittleEndian.PutUint32(b, v) }
func store64(b []byte, v uint64) { binary.LittleEndian.PutUint64(b, v) }
