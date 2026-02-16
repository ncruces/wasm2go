package main

import (
	"errors"
	"io"
)

const maxLEB128Len64 = 10

var errOverflow = errors.New("leb128 overflows a 64-bit integer")

func readLEB128(r io.ByteReader) (uint64, error) {
	var x uint64
	var s uint
	for i := range maxLEB128Len64 {
		b, err := r.ReadByte()
		if err != nil {
			return x, err
		}
		x |= uint64(b&0x7f) << s
		s += 7
		if b <= 0x7f {
			if i < maxLEB128Len64-1 || b <= 1 {
				return x, nil
			}
		}
	}
	return x, errOverflow
}

func readSignedLEB128(r io.ByteReader) (int64, error) {
	var x int64
	var s uint
	for i := range maxLEB128Len64 {
		b, err := r.ReadByte()
		if err != nil {
			return x, err
		}
		x |= int64(b&0x7f) << s
		s += 7
		if b <= 0x7f {
			if b >= 0x40 {
				x |= -1 << s
			}
			if i < maxLEB128Len64-1 || b == 0 || b == 0x7f {
				return x, nil
			}
		}
	}
	return x, errOverflow
}
