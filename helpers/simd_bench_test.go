package helpers

import (
	"encoding/binary"
	"testing"
)

// Microbenchmarks motivating the two-word v128 representation. Each op is
// measured both as the helper in simd.go (words) and as the same op on a
// [16]byte with a per-byte loop (bytes), which is how the helpers were first
// written. The difference is dominated by representation, not lane math:
// a [16]byte is stack-assigned by the Go ABI and not SSA-able, so every call
// and every intermediate goes through memory; a struct of two uint64 travels
// in registers. See the README for the effect on whole programs.

type bytes128 [16]byte

func bytesLoad(b []byte) bytes128 { return bytes128(b) }

func bytesStore(b []byte, v bytes128) { *(*bytes128)(b) = v }

func bytesAnd(a, b bytes128) (r bytes128) {
	for i := range r {
		r[i] = a[i] & b[i]
	}
	return
}

func bytesAdd8(a, b bytes128) (r bytes128) {
	for i := range r {
		r[i] = a[i] + b[i]
	}
	return
}

func bytesEq8(a, b bytes128) (r bytes128) {
	for i := range r {
		if a[i] == b[i] {
			r[i] = 0xff
		}
	}
	return
}

func bytesSplat8(x int32) (r bytes128) {
	for i := range r {
		r[i] = byte(x)
	}
	return
}

func bytesBitmask8(v bytes128) int32 {
	const msb = 0x8080808080808080
	const mul = 0x0102040810204080
	lo := (binary.LittleEndian.Uint64(v[0:8]) & msb) >> 7 * mul >> 56
	hi := (binary.LittleEndian.Uint64(v[8:16]) & msb) >> 7 * mul >> 56
	return int32(hi<<8 | lo)
}

var (
	sinkB  bytes128
	sinkV  v128
	sinkI  int32
	benchA = bytes128{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	benchV = load128(benchA[:])
	benchM = make([]byte, 4096)
)

func BenchmarkV128(b *testing.B) {
	b.Run("and/bytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sinkB = bytesAnd(benchA, sinkB)
		}
	})
	b.Run("and/words", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sinkV = v128_and(benchV, sinkV)
		}
	})
	b.Run("i8x16.add/bytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sinkB = bytesAdd8(benchA, sinkB)
		}
	})
	b.Run("i8x16.add/words", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sinkV = i8x16_add(benchV, sinkV)
		}
	})
	b.Run("i8x16.eq/bytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sinkB = bytesEq8(benchA, sinkB)
		}
	})
	b.Run("i8x16.eq/words", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sinkV = i8x16_eq(benchV, sinkV)
		}
	})
	b.Run("i8x16.bitmask/bytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sinkI += bytesBitmask8(sinkB)
			sinkB[0]++
		}
	})
	b.Run("i8x16.bitmask/words", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sinkI += i8x16_bitmask(sinkV)
			sinkV.lo++
		}
	})
	// The inner step of a SIMD JSON scanner: load 16 bytes, compare against
	// a splatted byte, mask, and extract a bitmask.
	b.Run("scan16/bytes", func(b *testing.B) {
		b.SetBytes(16)
		q := bytesSplat8('"')
		for i := 0; i < b.N; i++ {
			v := bytesLoad(benchM[i&4095&^15:])
			sinkI += bytesBitmask8(bytesAnd(bytesEq8(v, q), sinkB))
		}
	})
	b.Run("scan16/words", func(b *testing.B) {
		b.SetBytes(16)
		q := i8x16_splat('"')
		for i := 0; i < b.N; i++ {
			v := load128(benchM[i&4095&^15:])
			sinkI += i8x16_bitmask(v128_and(i8x16_eq(v, q), sinkV))
		}
	})
	// Copying 16 bytes through a v128, as compilers emit for memcpy.
	b.Run("copy16/bytes", func(b *testing.B) {
		b.SetBytes(16)
		for i := 0; i < b.N; i++ {
			bytesStore(benchM[(i+1)&4095&^15:], bytesLoad(benchM[i&4095&^15:]))
		}
	})
	b.Run("copy16/words", func(b *testing.B) {
		b.SetBytes(16)
		for i := 0; i < b.N; i++ {
			store128(benchM[(i+1)&4095&^15:], load128(benchM[i&4095&^15:]))
		}
	})
}
