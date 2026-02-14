package main

import (
	"encoding/binary"
	"io"
)

func readLEB128(r io.ByteReader) (uint64, error) {
	return binary.ReadUvarint(r)
}
