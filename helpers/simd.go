// Portable helpers for the Wasm fixed-width SIMD (v128) instructions.
//
// A v128 is two little-endian 64-bit words: lane i of width w occupies bits
// [w*i, w*i+w) of lo when w*i < 64, and of hi otherwise. Two uint64 fields
// are register-assigned by the Go ABI and SSA-able, unlike a [16]byte, so
// vectors stay in registers across chained ops and helper calls are cheap.
//
// Integer lane math is SWAR (SIMD within a register): a handful of 64-bit
// steps applied to both words. Add/sub isolate the carry out of each lane
// with a top-bit mask; equality uses zero-lane detection; unsigned compares
// use the borrow trick and signed ones bias the operands; shifts mask the
// bits that would cross lanes; min/max/saturation select through compare
// masks. Ops with no useful SWAR form (multiplies, float lane math,
// conversions, shuffles, narrows) are unrolled per lane.
//
// Every helper is self-contained: wasm2go's resolveHelpers copies referenced
// helpers by name without transitive resolution, so helpers never call each
// other or use package-level declarations. The file was produced by a
// generator script and is now the source of truth.

package helpers

import (
	"encoding/binary"
	"math"
)

// The Wasm v128 type as two little-endian 64-bit words: lane i of width w
// occupies bits [w*i, w*i+w) of lo for w*i < 64, of hi otherwise. Structs of two
// integer fields are register-assigned by the Go ABI and SSA-able, unlike [16]byte.
type v128 struct{ lo, hi uint64 }

//go:nosplit
func load128(b []byte) v128 {
	_ = b[15]
	return v128{binary.LittleEndian.Uint64(b), binary.LittleEndian.Uint64(b[8:])}
}

//go:nosplit
func store128(b []byte, v v128) {
	_ = b[15]
	binary.LittleEndian.PutUint64(b, v.lo)
	binary.LittleEndian.PutUint64(b[8:], v.hi)
}

//go:nosplit
func v128_load32_zero(x uint32) v128 {
	return v128{uint64(x), 0}
}

//go:nosplit
func v128_load64_zero(x uint64) v128 {
	return v128{x, 0}
}

//go:nosplit
func i8x16_splat(x int32) v128 {
	w := uint64(uint8(x)) * 0x0101010101010101
	return v128{w, w}
}

//go:nosplit
func i16x8_splat(x int32) v128 {
	w := uint64(uint16(x)) * 0x0001000100010001
	return v128{w, w}
}

//go:nosplit
func i32x4_splat(x int32) v128 {
	w := uint64(uint32(x)) * 0x0000000100000001
	return v128{w, w}
}

//go:nosplit
func i64x2_splat(x int64) v128 {
	w := uint64(x)
	return v128{w, w}
}

//go:nosplit
func f32x4_splat(x float32) v128 {
	w := uint64(math.Float32bits(x)) * 0x0000000100000001
	return v128{w, w}
}

//go:nosplit
func f64x2_splat(x float64) v128 {
	w := math.Float64bits(x)
	return v128{w, w}
}

//go:nosplit
func i8x16_extract_lane_s(v v128, l int) int32 {
	w := v.lo
	if l >= 8 {
		w = v.hi
	}
	return int32(int8(w >> (8 * uint(l&7))))
}

//go:nosplit
func i8x16_extract_lane_u(v v128, l int) int32 {
	w := v.lo
	if l >= 8 {
		w = v.hi
	}
	return int32(uint8(w >> (8 * uint(l&7))))
}

//go:nosplit
func i16x8_extract_lane_s(v v128, l int) int32 {
	w := v.lo
	if l >= 4 {
		w = v.hi
	}
	return int32(int16(w >> (16 * uint(l&3))))
}

//go:nosplit
func i16x8_extract_lane_u(v v128, l int) int32 {
	w := v.lo
	if l >= 4 {
		w = v.hi
	}
	return int32(uint16(w >> (16 * uint(l&3))))
}

//go:nosplit
func i32x4_extract_lane(v v128, l int) int32 {
	w := v.lo
	if l >= 2 {
		w = v.hi
	}
	return int32(w >> (32 * uint(l&1)))
}

//go:nosplit
func i64x2_extract_lane(v v128, l int) int64 {
	if l == 0 {
		return int64(v.lo)
	}
	return int64(v.hi)
}

//go:nosplit
func f32x4_extract_lane(v v128, l int) float32 {
	w := v.lo
	if l >= 2 {
		w = v.hi
	}
	return math.Float32frombits(uint32(w >> (32 * uint(l&1))))
}

//go:nosplit
func f64x2_extract_lane(v v128, l int) float64 {
	if l == 0 {
		return math.Float64frombits(v.lo)
	}
	return math.Float64frombits(v.hi)
}

//go:nosplit
func i8x16_replace_lane(v v128, l int, x int32) v128 {
	sh := 8 * uint(l&7)
	m := uint64(0x00000000000000ff) << sh
	b := uint64(uint8(x)) << sh
	if l < 8 {
		return v128{v.lo&^m | b, v.hi}
	}
	return v128{v.lo, v.hi&^m | b}
}

//go:nosplit
func i16x8_replace_lane(v v128, l int, x int32) v128 {
	sh := 16 * uint(l&3)
	m := uint64(0x000000000000ffff) << sh
	b := uint64(uint16(x)) << sh
	if l < 4 {
		return v128{v.lo&^m | b, v.hi}
	}
	return v128{v.lo, v.hi&^m | b}
}

//go:nosplit
func i32x4_replace_lane(v v128, l int, x int32) v128 {
	sh := 32 * uint(l&1)
	m := uint64(0x00000000ffffffff) << sh
	b := uint64(uint32(x)) << sh
	if l < 2 {
		return v128{v.lo&^m | b, v.hi}
	}
	return v128{v.lo, v.hi&^m | b}
}

//go:nosplit
func f32x4_replace_lane(v v128, l int, x float32) v128 {
	sh := 32 * uint(l&1)
	m := uint64(0x00000000ffffffff) << sh
	b := uint64(math.Float32bits(x)) << sh
	if l < 2 {
		return v128{v.lo&^m | b, v.hi}
	}
	return v128{v.lo, v.hi&^m | b}
}

//go:nosplit
func i64x2_replace_lane(v v128, l int, x int64) v128 {
	if l == 0 {
		return v128{uint64(x), v.hi}
	}
	return v128{v.lo, uint64(x)}
}

//go:nosplit
func f64x2_replace_lane(v v128, l int, x float64) v128 {
	if l == 0 {
		return v128{math.Float64bits(x), v.hi}
	}
	return v128{v.lo, math.Float64bits(x)}
}

//go:nosplit
func v128_any_true(v v128) int32 {
	if v.lo|v.hi != 0 {
		return 1
	}
	return 0
}

//go:nosplit
func i8x16_bitmask(v v128) int32 {
	const msb = 0x8080808080808080
	const mul = 0x0102040810204080
	lo := (v.lo & msb) >> 7 * mul >> 56
	hi := (v.hi & msb) >> 7 * mul >> 56
	return int32(hi<<8 | lo)
}

//go:nosplit
func i8x16_shuffle(a, b, m v128) v128 {
	var t [32]byte
	binary.LittleEndian.PutUint64(t[0:], a.lo)
	binary.LittleEndian.PutUint64(t[8:], a.hi)
	binary.LittleEndian.PutUint64(t[16:], b.lo)
	binary.LittleEndian.PutUint64(t[24:], b.hi)
	var r [16]byte
	for i := 0; i < 8; i++ {
		r[i] = t[byte(m.lo>>(8*uint(i)))&31]
		r[8+i] = t[byte(m.hi>>(8*uint(i)))&31]
	}
	return v128{binary.LittleEndian.Uint64(r[0:]), binary.LittleEndian.Uint64(r[8:])}
}

//go:nosplit
func i8x16_swizzle(a, s v128) v128 {
	var t [16]byte
	binary.LittleEndian.PutUint64(t[0:], a.lo)
	binary.LittleEndian.PutUint64(t[8:], a.hi)
	var r [16]byte
	for i := 0; i < 8; i++ {
		if j := byte(s.lo >> (8 * uint(i))); j < 16 {
			r[i] = t[j]
		}
		if j := byte(s.hi >> (8 * uint(i))); j < 16 {
			r[8+i] = t[j]
		}
	}
	return v128{binary.LittleEndian.Uint64(r[0:]), binary.LittleEndian.Uint64(r[8:])}
}

//go:nosplit
func i8x16_all_true(v v128) int32 {
	x0, x1 := v.lo, v.hi
	z0 := ^(((x0 & 0x7f7f7f7f7f7f7f7f) + 0x7f7f7f7f7f7f7f7f) | x0 | 0x7f7f7f7f7f7f7f7f) & 0x8080808080808080
	z1 := ^(((x1 & 0x7f7f7f7f7f7f7f7f) + 0x7f7f7f7f7f7f7f7f) | x1 | 0x7f7f7f7f7f7f7f7f) & 0x8080808080808080
	if z0|z1 != 0 {
		return 0
	}
	return 1
}

//go:nosplit
func i16x8_all_true(v v128) int32 {
	x0, x1 := v.lo, v.hi
	z0 := ^(((x0 & 0x7fff7fff7fff7fff) + 0x7fff7fff7fff7fff) | x0 | 0x7fff7fff7fff7fff) & 0x8000800080008000
	z1 := ^(((x1 & 0x7fff7fff7fff7fff) + 0x7fff7fff7fff7fff) | x1 | 0x7fff7fff7fff7fff) & 0x8000800080008000
	if z0|z1 != 0 {
		return 0
	}
	return 1
}

//go:nosplit
func i32x4_all_true(v v128) int32 {
	x0, x1 := v.lo, v.hi
	z0 := ^(((x0 & 0x7fffffff7fffffff) + 0x7fffffff7fffffff) | x0 | 0x7fffffff7fffffff) & 0x8000000080000000
	z1 := ^(((x1 & 0x7fffffff7fffffff) + 0x7fffffff7fffffff) | x1 | 0x7fffffff7fffffff) & 0x8000000080000000
	if z0|z1 != 0 {
		return 0
	}
	return 1
}

//go:nosplit
func i64x2_all_true(v v128) int32 {
	x0, x1 := v.lo, v.hi
	if x0 == 0 || x1 == 0 {
		return 0
	}
	return 1
}

//go:nosplit
func v128_and(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 & y0
	t1_1 := x1 & y1
	return v128{t1_0, t1_1}
}

//go:nosplit
func v128_or(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 | y0
	t1_1 := x1 | y1
	return v128{t1_0, t1_1}
}

//go:nosplit
func v128_xor(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 ^ y0
	t1_1 := x1 ^ y1
	return v128{t1_0, t1_1}
}

//go:nosplit
func v128_andnot(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 &^ y0
	t1_1 := x1 &^ y1
	return v128{t1_0, t1_1}
}

//go:nosplit
func v128_not(a v128) v128 {
	x0, x1 := a.lo, a.hi
	t1_0 := ^x0
	t1_1 := ^x1
	return v128{t1_0, t1_1}
}

//go:nosplit
func v128_bitselect(a, b, c v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	z0, z1 := c.lo, c.hi
	return v128{(x0 & z0) | (y0 &^ z0), (x1 & z1) | (y1 &^ z1)}
}

//go:nosplit
func i8x16_add(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ((x0 & 0x7f7f7f7f7f7f7f7f) + (y0 & 0x7f7f7f7f7f7f7f7f)) ^ ((x0 ^ y0) & 0x8080808080808080)
	t1_1 := ((x1 & 0x7f7f7f7f7f7f7f7f) + (y1 & 0x7f7f7f7f7f7f7f7f)) ^ ((x1 ^ y1) & 0x8080808080808080)
	return v128{t1_0, t1_1}
}

//go:nosplit
func i8x16_sub(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ((x0 | 0x8080808080808080) - (y0 & 0x7f7f7f7f7f7f7f7f)) ^ ((x0 ^ ^y0) & 0x8080808080808080)
	t1_1 := ((x1 | 0x8080808080808080) - (y1 & 0x7f7f7f7f7f7f7f7f)) ^ ((x1 ^ ^y1) & 0x8080808080808080)
	return v128{t1_0, t1_1}
}

//go:nosplit
func i8x16_neg(a v128) v128 {
	x0, x1 := a.lo, a.hi
	t1_0 := (0x8080808080808080 - (x0 & 0x7f7f7f7f7f7f7f7f)) ^ (^x0 & 0x8080808080808080)
	t1_1 := (0x8080808080808080 - (x1 & 0x7f7f7f7f7f7f7f7f)) ^ (^x1 & 0x8080808080808080)
	return v128{t1_0, t1_1}
}

//go:nosplit
func i8x16_abs(a v128) v128 {
	x0, x1 := a.lo, a.hi
	t1_0 := ((x0 & 0x8080808080808080) >> 7) * 0x00000000000000ff
	t2_0 := x0 ^ t1_0
	t3_0 := ((t2_0 | 0x8080808080808080) - (t1_0 & 0x7f7f7f7f7f7f7f7f)) ^ ((t2_0 ^ ^t1_0) & 0x8080808080808080)
	t1_1 := ((x1 & 0x8080808080808080) >> 7) * 0x00000000000000ff
	t2_1 := x1 ^ t1_1
	t3_1 := ((t2_1 | 0x8080808080808080) - (t1_1 & 0x7f7f7f7f7f7f7f7f)) ^ ((t2_1 ^ ^t1_1) & 0x8080808080808080)
	return v128{t3_0, t3_1}
}

//go:nosplit
func i8x16_eq(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 ^ y0
	t2_0 := ^(((t1_0 & 0x7f7f7f7f7f7f7f7f) + 0x7f7f7f7f7f7f7f7f) | t1_0 | 0x7f7f7f7f7f7f7f7f) & 0x8080808080808080
	t3_0 := (t2_0 >> 7) * 0x00000000000000ff
	t1_1 := x1 ^ y1
	t2_1 := ^(((t1_1 & 0x7f7f7f7f7f7f7f7f) + 0x7f7f7f7f7f7f7f7f) | t1_1 | 0x7f7f7f7f7f7f7f7f) & 0x8080808080808080
	t3_1 := (t2_1 >> 7) * 0x00000000000000ff
	return v128{t3_0, t3_1}
}

//go:nosplit
func i8x16_ne(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 ^ y0
	t2_0 := ^(((t1_0 & 0x7f7f7f7f7f7f7f7f) + 0x7f7f7f7f7f7f7f7f) | t1_0 | 0x7f7f7f7f7f7f7f7f) & 0x8080808080808080
	t3_0 := (t2_0 >> 7) * 0x00000000000000ff
	t4_0 := ^t3_0
	t1_1 := x1 ^ y1
	t2_1 := ^(((t1_1 & 0x7f7f7f7f7f7f7f7f) + 0x7f7f7f7f7f7f7f7f) | t1_1 | 0x7f7f7f7f7f7f7f7f) & 0x8080808080808080
	t3_1 := (t2_1 >> 7) * 0x00000000000000ff
	t4_1 := ^t3_1
	return v128{t4_0, t4_1}
}

//go:nosplit
func i8x16_lt_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 ^ 0x8080808080808080
	t2_0 := y0 ^ 0x8080808080808080
	t3_0 := ^((t1_0 | 0x8080808080808080) - (t2_0 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t4_0 := ((^t1_0 & t2_0) | (^(t1_0 ^ t2_0) & t3_0)) & 0x8080808080808080
	t5_0 := (t4_0 >> 7) * 0x00000000000000ff
	t1_1 := x1 ^ 0x8080808080808080
	t2_1 := y1 ^ 0x8080808080808080
	t3_1 := ^((t1_1 | 0x8080808080808080) - (t2_1 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t4_1 := ((^t1_1 & t2_1) | (^(t1_1 ^ t2_1) & t3_1)) & 0x8080808080808080
	t5_1 := (t4_1 >> 7) * 0x00000000000000ff
	return v128{t5_0, t5_1}
}

//go:nosplit
func i8x16_lt_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ^((x0 | 0x8080808080808080) - (y0 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t2_0 := ((^x0 & y0) | (^(x0 ^ y0) & t1_0)) & 0x8080808080808080
	t3_0 := (t2_0 >> 7) * 0x00000000000000ff
	t1_1 := ^((x1 | 0x8080808080808080) - (y1 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t2_1 := ((^x1 & y1) | (^(x1 ^ y1) & t1_1)) & 0x8080808080808080
	t3_1 := (t2_1 >> 7) * 0x00000000000000ff
	return v128{t3_0, t3_1}
}

//go:nosplit
func i8x16_gt_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := y0 ^ 0x8080808080808080
	t2_0 := x0 ^ 0x8080808080808080
	t3_0 := ^((t1_0 | 0x8080808080808080) - (t2_0 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t4_0 := ((^t1_0 & t2_0) | (^(t1_0 ^ t2_0) & t3_0)) & 0x8080808080808080
	t5_0 := (t4_0 >> 7) * 0x00000000000000ff
	t1_1 := y1 ^ 0x8080808080808080
	t2_1 := x1 ^ 0x8080808080808080
	t3_1 := ^((t1_1 | 0x8080808080808080) - (t2_1 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t4_1 := ((^t1_1 & t2_1) | (^(t1_1 ^ t2_1) & t3_1)) & 0x8080808080808080
	t5_1 := (t4_1 >> 7) * 0x00000000000000ff
	return v128{t5_0, t5_1}
}

//go:nosplit
func i8x16_gt_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ^((y0 | 0x8080808080808080) - (x0 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t2_0 := ((^y0 & x0) | (^(y0 ^ x0) & t1_0)) & 0x8080808080808080
	t3_0 := (t2_0 >> 7) * 0x00000000000000ff
	t1_1 := ^((y1 | 0x8080808080808080) - (x1 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t2_1 := ((^y1 & x1) | (^(y1 ^ x1) & t1_1)) & 0x8080808080808080
	t3_1 := (t2_1 >> 7) * 0x00000000000000ff
	return v128{t3_0, t3_1}
}

//go:nosplit
func i8x16_le_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := y0 ^ 0x8080808080808080
	t2_0 := x0 ^ 0x8080808080808080
	t3_0 := ^((t1_0 | 0x8080808080808080) - (t2_0 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t4_0 := ((^t1_0 & t2_0) | (^(t1_0 ^ t2_0) & t3_0)) & 0x8080808080808080
	t5_0 := (t4_0 >> 7) * 0x00000000000000ff
	t6_0 := ^t5_0
	t1_1 := y1 ^ 0x8080808080808080
	t2_1 := x1 ^ 0x8080808080808080
	t3_1 := ^((t1_1 | 0x8080808080808080) - (t2_1 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t4_1 := ((^t1_1 & t2_1) | (^(t1_1 ^ t2_1) & t3_1)) & 0x8080808080808080
	t5_1 := (t4_1 >> 7) * 0x00000000000000ff
	t6_1 := ^t5_1
	return v128{t6_0, t6_1}
}

//go:nosplit
func i8x16_le_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ^((y0 | 0x8080808080808080) - (x0 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t2_0 := ((^y0 & x0) | (^(y0 ^ x0) & t1_0)) & 0x8080808080808080
	t3_0 := (t2_0 >> 7) * 0x00000000000000ff
	t4_0 := ^t3_0
	t1_1 := ^((y1 | 0x8080808080808080) - (x1 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t2_1 := ((^y1 & x1) | (^(y1 ^ x1) & t1_1)) & 0x8080808080808080
	t3_1 := (t2_1 >> 7) * 0x00000000000000ff
	t4_1 := ^t3_1
	return v128{t4_0, t4_1}
}

//go:nosplit
func i8x16_ge_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 ^ 0x8080808080808080
	t2_0 := y0 ^ 0x8080808080808080
	t3_0 := ^((t1_0 | 0x8080808080808080) - (t2_0 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t4_0 := ((^t1_0 & t2_0) | (^(t1_0 ^ t2_0) & t3_0)) & 0x8080808080808080
	t5_0 := (t4_0 >> 7) * 0x00000000000000ff
	t6_0 := ^t5_0
	t1_1 := x1 ^ 0x8080808080808080
	t2_1 := y1 ^ 0x8080808080808080
	t3_1 := ^((t1_1 | 0x8080808080808080) - (t2_1 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t4_1 := ((^t1_1 & t2_1) | (^(t1_1 ^ t2_1) & t3_1)) & 0x8080808080808080
	t5_1 := (t4_1 >> 7) * 0x00000000000000ff
	t6_1 := ^t5_1
	return v128{t6_0, t6_1}
}

//go:nosplit
func i8x16_ge_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ^((x0 | 0x8080808080808080) - (y0 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t2_0 := ((^x0 & y0) | (^(x0 ^ y0) & t1_0)) & 0x8080808080808080
	t3_0 := (t2_0 >> 7) * 0x00000000000000ff
	t4_0 := ^t3_0
	t1_1 := ^((x1 | 0x8080808080808080) - (y1 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t2_1 := ((^x1 & y1) | (^(x1 ^ y1) & t1_1)) & 0x8080808080808080
	t3_1 := (t2_1 >> 7) * 0x00000000000000ff
	t4_1 := ^t3_1
	return v128{t4_0, t4_1}
}

//go:nosplit
func i8x16_min_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 ^ 0x8080808080808080
	t2_0 := y0 ^ 0x8080808080808080
	t3_0 := ^((t1_0 | 0x8080808080808080) - (t2_0 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t4_0 := ((^t1_0 & t2_0) | (^(t1_0 ^ t2_0) & t3_0)) & 0x8080808080808080
	t5_0 := (t4_0 >> 7) * 0x00000000000000ff
	t6_0 := (x0 & t5_0) | (y0 &^ t5_0)
	t1_1 := x1 ^ 0x8080808080808080
	t2_1 := y1 ^ 0x8080808080808080
	t3_1 := ^((t1_1 | 0x8080808080808080) - (t2_1 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t4_1 := ((^t1_1 & t2_1) | (^(t1_1 ^ t2_1) & t3_1)) & 0x8080808080808080
	t5_1 := (t4_1 >> 7) * 0x00000000000000ff
	t6_1 := (x1 & t5_1) | (y1 &^ t5_1)
	return v128{t6_0, t6_1}
}

//go:nosplit
func i8x16_min_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ^((x0 | 0x8080808080808080) - (y0 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t2_0 := ((^x0 & y0) | (^(x0 ^ y0) & t1_0)) & 0x8080808080808080
	t3_0 := (t2_0 >> 7) * 0x00000000000000ff
	t4_0 := (x0 & t3_0) | (y0 &^ t3_0)
	t1_1 := ^((x1 | 0x8080808080808080) - (y1 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t2_1 := ((^x1 & y1) | (^(x1 ^ y1) & t1_1)) & 0x8080808080808080
	t3_1 := (t2_1 >> 7) * 0x00000000000000ff
	t4_1 := (x1 & t3_1) | (y1 &^ t3_1)
	return v128{t4_0, t4_1}
}

//go:nosplit
func i8x16_max_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 ^ 0x8080808080808080
	t2_0 := y0 ^ 0x8080808080808080
	t3_0 := ^((t1_0 | 0x8080808080808080) - (t2_0 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t4_0 := ((^t1_0 & t2_0) | (^(t1_0 ^ t2_0) & t3_0)) & 0x8080808080808080
	t5_0 := (t4_0 >> 7) * 0x00000000000000ff
	t6_0 := (y0 & t5_0) | (x0 &^ t5_0)
	t1_1 := x1 ^ 0x8080808080808080
	t2_1 := y1 ^ 0x8080808080808080
	t3_1 := ^((t1_1 | 0x8080808080808080) - (t2_1 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t4_1 := ((^t1_1 & t2_1) | (^(t1_1 ^ t2_1) & t3_1)) & 0x8080808080808080
	t5_1 := (t4_1 >> 7) * 0x00000000000000ff
	t6_1 := (y1 & t5_1) | (x1 &^ t5_1)
	return v128{t6_0, t6_1}
}

//go:nosplit
func i8x16_max_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ^((x0 | 0x8080808080808080) - (y0 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t2_0 := ((^x0 & y0) | (^(x0 ^ y0) & t1_0)) & 0x8080808080808080
	t3_0 := (t2_0 >> 7) * 0x00000000000000ff
	t4_0 := (y0 & t3_0) | (x0 &^ t3_0)
	t1_1 := ^((x1 | 0x8080808080808080) - (y1 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t2_1 := ((^x1 & y1) | (^(x1 ^ y1) & t1_1)) & 0x8080808080808080
	t3_1 := (t2_1 >> 7) * 0x00000000000000ff
	t4_1 := (y1 & t3_1) | (x1 &^ t3_1)
	return v128{t4_0, t4_1}
}

//go:nosplit
func i8x16_shl(a v128, y int32) v128 {
	x0, x1 := a.lo, a.hi
	s := uint(y) & 7
	t1_0 := (uint64(0x00000000000000ff) >> s) * 0x0101010101010101
	t2_0 := (x0 & t1_0) << s
	t1_1 := (uint64(0x00000000000000ff) >> s) * 0x0101010101010101
	t2_1 := (x1 & t1_1) << s
	return v128{t2_0, t2_1}
}

//go:nosplit
func i8x16_shr_u(a v128, y int32) v128 {
	x0, x1 := a.lo, a.hi
	s := uint(y) & 7
	t1_0 := (uint64(0x00000000000000ff) >> s) * 0x0101010101010101
	t2_0 := (x0 >> s) & t1_0
	t1_1 := (uint64(0x00000000000000ff) >> s) * 0x0101010101010101
	t2_1 := (x1 >> s) & t1_1
	return v128{t2_0, t2_1}
}

//go:nosplit
func i8x16_shr_s(a v128, y int32) v128 {
	x0, x1 := a.lo, a.hi
	s := uint(y) & 7
	t1_0 := x0 ^ 0x8080808080808080
	t2_0 := (uint64(0x00000000000000ff) >> s) * 0x0101010101010101
	t3_0 := (t1_0 >> s) & t2_0
	t4_0 := (uint64(0x0000000000000080) >> s) * 0x0101010101010101
	t5_0 := ((t3_0 | 0x8080808080808080) - (t4_0 & 0x7f7f7f7f7f7f7f7f)) ^ ((t3_0 ^ ^t4_0) & 0x8080808080808080)
	t1_1 := x1 ^ 0x8080808080808080
	t2_1 := (uint64(0x00000000000000ff) >> s) * 0x0101010101010101
	t3_1 := (t1_1 >> s) & t2_1
	t4_1 := (uint64(0x0000000000000080) >> s) * 0x0101010101010101
	t5_1 := ((t3_1 | 0x8080808080808080) - (t4_1 & 0x7f7f7f7f7f7f7f7f)) ^ ((t3_1 ^ ^t4_1) & 0x8080808080808080)
	return v128{t5_0, t5_1}
}

//go:nosplit
func i8x16_add_sat_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ((x0 & 0x7f7f7f7f7f7f7f7f) + (y0 & 0x7f7f7f7f7f7f7f7f)) ^ ((x0 ^ y0) & 0x8080808080808080)
	t2_0 := ^((t1_0 | 0x8080808080808080) - (x0 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t3_0 := ((^t1_0 & x0) | (^(t1_0 ^ x0) & t2_0)) & 0x8080808080808080
	t4_0 := (t3_0 >> 7) * 0x00000000000000ff
	t5_0 := t1_0 | t4_0
	t1_1 := ((x1 & 0x7f7f7f7f7f7f7f7f) + (y1 & 0x7f7f7f7f7f7f7f7f)) ^ ((x1 ^ y1) & 0x8080808080808080)
	t2_1 := ^((t1_1 | 0x8080808080808080) - (x1 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t3_1 := ((^t1_1 & x1) | (^(t1_1 ^ x1) & t2_1)) & 0x8080808080808080
	t4_1 := (t3_1 >> 7) * 0x00000000000000ff
	t5_1 := t1_1 | t4_1
	return v128{t5_0, t5_1}
}

//go:nosplit
func i8x16_sub_sat_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ^((x0 | 0x8080808080808080) - (y0 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t2_0 := ((^x0 & y0) | (^(x0 ^ y0) & t1_0)) & 0x8080808080808080
	t3_0 := (t2_0 >> 7) * 0x00000000000000ff
	t4_0 := ((x0 | 0x8080808080808080) - (y0 & 0x7f7f7f7f7f7f7f7f)) ^ ((x0 ^ ^y0) & 0x8080808080808080)
	t5_0 := t4_0 &^ t3_0
	t1_1 := ^((x1 | 0x8080808080808080) - (y1 & 0x7f7f7f7f7f7f7f7f)) & 0x8080808080808080
	t2_1 := ((^x1 & y1) | (^(x1 ^ y1) & t1_1)) & 0x8080808080808080
	t3_1 := (t2_1 >> 7) * 0x00000000000000ff
	t4_1 := ((x1 | 0x8080808080808080) - (y1 & 0x7f7f7f7f7f7f7f7f)) ^ ((x1 ^ ^y1) & 0x8080808080808080)
	t5_1 := t4_1 &^ t3_1
	return v128{t5_0, t5_1}
}

//go:nosplit
func i8x16_avgr_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := (x0 | y0) - (((x0 ^ y0) >> 1) & 0x7f7f7f7f7f7f7f7f)
	t1_1 := (x1 | y1) - (((x1 ^ y1) >> 1) & 0x7f7f7f7f7f7f7f7f)
	return v128{t1_0, t1_1}
}

//go:nosplit
func i16x8_add(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ((x0 & 0x7fff7fff7fff7fff) + (y0 & 0x7fff7fff7fff7fff)) ^ ((x0 ^ y0) & 0x8000800080008000)
	t1_1 := ((x1 & 0x7fff7fff7fff7fff) + (y1 & 0x7fff7fff7fff7fff)) ^ ((x1 ^ y1) & 0x8000800080008000)
	return v128{t1_0, t1_1}
}

//go:nosplit
func i16x8_sub(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ((x0 | 0x8000800080008000) - (y0 & 0x7fff7fff7fff7fff)) ^ ((x0 ^ ^y0) & 0x8000800080008000)
	t1_1 := ((x1 | 0x8000800080008000) - (y1 & 0x7fff7fff7fff7fff)) ^ ((x1 ^ ^y1) & 0x8000800080008000)
	return v128{t1_0, t1_1}
}

//go:nosplit
func i16x8_neg(a v128) v128 {
	x0, x1 := a.lo, a.hi
	t1_0 := (0x8000800080008000 - (x0 & 0x7fff7fff7fff7fff)) ^ (^x0 & 0x8000800080008000)
	t1_1 := (0x8000800080008000 - (x1 & 0x7fff7fff7fff7fff)) ^ (^x1 & 0x8000800080008000)
	return v128{t1_0, t1_1}
}

//go:nosplit
func i16x8_abs(a v128) v128 {
	x0, x1 := a.lo, a.hi
	t1_0 := ((x0 & 0x8000800080008000) >> 15) * 0x000000000000ffff
	t2_0 := x0 ^ t1_0
	t3_0 := ((t2_0 | 0x8000800080008000) - (t1_0 & 0x7fff7fff7fff7fff)) ^ ((t2_0 ^ ^t1_0) & 0x8000800080008000)
	t1_1 := ((x1 & 0x8000800080008000) >> 15) * 0x000000000000ffff
	t2_1 := x1 ^ t1_1
	t3_1 := ((t2_1 | 0x8000800080008000) - (t1_1 & 0x7fff7fff7fff7fff)) ^ ((t2_1 ^ ^t1_1) & 0x8000800080008000)
	return v128{t3_0, t3_1}
}

//go:nosplit
func i16x8_eq(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 ^ y0
	t2_0 := ^(((t1_0 & 0x7fff7fff7fff7fff) + 0x7fff7fff7fff7fff) | t1_0 | 0x7fff7fff7fff7fff) & 0x8000800080008000
	t3_0 := (t2_0 >> 15) * 0x000000000000ffff
	t1_1 := x1 ^ y1
	t2_1 := ^(((t1_1 & 0x7fff7fff7fff7fff) + 0x7fff7fff7fff7fff) | t1_1 | 0x7fff7fff7fff7fff) & 0x8000800080008000
	t3_1 := (t2_1 >> 15) * 0x000000000000ffff
	return v128{t3_0, t3_1}
}

//go:nosplit
func i16x8_ne(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 ^ y0
	t2_0 := ^(((t1_0 & 0x7fff7fff7fff7fff) + 0x7fff7fff7fff7fff) | t1_0 | 0x7fff7fff7fff7fff) & 0x8000800080008000
	t3_0 := (t2_0 >> 15) * 0x000000000000ffff
	t4_0 := ^t3_0
	t1_1 := x1 ^ y1
	t2_1 := ^(((t1_1 & 0x7fff7fff7fff7fff) + 0x7fff7fff7fff7fff) | t1_1 | 0x7fff7fff7fff7fff) & 0x8000800080008000
	t3_1 := (t2_1 >> 15) * 0x000000000000ffff
	t4_1 := ^t3_1
	return v128{t4_0, t4_1}
}

//go:nosplit
func i16x8_lt_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 ^ 0x8000800080008000
	t2_0 := y0 ^ 0x8000800080008000
	t3_0 := ^((t1_0 | 0x8000800080008000) - (t2_0 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t4_0 := ((^t1_0 & t2_0) | (^(t1_0 ^ t2_0) & t3_0)) & 0x8000800080008000
	t5_0 := (t4_0 >> 15) * 0x000000000000ffff
	t1_1 := x1 ^ 0x8000800080008000
	t2_1 := y1 ^ 0x8000800080008000
	t3_1 := ^((t1_1 | 0x8000800080008000) - (t2_1 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t4_1 := ((^t1_1 & t2_1) | (^(t1_1 ^ t2_1) & t3_1)) & 0x8000800080008000
	t5_1 := (t4_1 >> 15) * 0x000000000000ffff
	return v128{t5_0, t5_1}
}

//go:nosplit
func i16x8_lt_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ^((x0 | 0x8000800080008000) - (y0 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t2_0 := ((^x0 & y0) | (^(x0 ^ y0) & t1_0)) & 0x8000800080008000
	t3_0 := (t2_0 >> 15) * 0x000000000000ffff
	t1_1 := ^((x1 | 0x8000800080008000) - (y1 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t2_1 := ((^x1 & y1) | (^(x1 ^ y1) & t1_1)) & 0x8000800080008000
	t3_1 := (t2_1 >> 15) * 0x000000000000ffff
	return v128{t3_0, t3_1}
}

//go:nosplit
func i16x8_gt_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := y0 ^ 0x8000800080008000
	t2_0 := x0 ^ 0x8000800080008000
	t3_0 := ^((t1_0 | 0x8000800080008000) - (t2_0 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t4_0 := ((^t1_0 & t2_0) | (^(t1_0 ^ t2_0) & t3_0)) & 0x8000800080008000
	t5_0 := (t4_0 >> 15) * 0x000000000000ffff
	t1_1 := y1 ^ 0x8000800080008000
	t2_1 := x1 ^ 0x8000800080008000
	t3_1 := ^((t1_1 | 0x8000800080008000) - (t2_1 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t4_1 := ((^t1_1 & t2_1) | (^(t1_1 ^ t2_1) & t3_1)) & 0x8000800080008000
	t5_1 := (t4_1 >> 15) * 0x000000000000ffff
	return v128{t5_0, t5_1}
}

//go:nosplit
func i16x8_gt_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ^((y0 | 0x8000800080008000) - (x0 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t2_0 := ((^y0 & x0) | (^(y0 ^ x0) & t1_0)) & 0x8000800080008000
	t3_0 := (t2_0 >> 15) * 0x000000000000ffff
	t1_1 := ^((y1 | 0x8000800080008000) - (x1 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t2_1 := ((^y1 & x1) | (^(y1 ^ x1) & t1_1)) & 0x8000800080008000
	t3_1 := (t2_1 >> 15) * 0x000000000000ffff
	return v128{t3_0, t3_1}
}

//go:nosplit
func i16x8_le_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := y0 ^ 0x8000800080008000
	t2_0 := x0 ^ 0x8000800080008000
	t3_0 := ^((t1_0 | 0x8000800080008000) - (t2_0 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t4_0 := ((^t1_0 & t2_0) | (^(t1_0 ^ t2_0) & t3_0)) & 0x8000800080008000
	t5_0 := (t4_0 >> 15) * 0x000000000000ffff
	t6_0 := ^t5_0
	t1_1 := y1 ^ 0x8000800080008000
	t2_1 := x1 ^ 0x8000800080008000
	t3_1 := ^((t1_1 | 0x8000800080008000) - (t2_1 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t4_1 := ((^t1_1 & t2_1) | (^(t1_1 ^ t2_1) & t3_1)) & 0x8000800080008000
	t5_1 := (t4_1 >> 15) * 0x000000000000ffff
	t6_1 := ^t5_1
	return v128{t6_0, t6_1}
}

//go:nosplit
func i16x8_le_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ^((y0 | 0x8000800080008000) - (x0 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t2_0 := ((^y0 & x0) | (^(y0 ^ x0) & t1_0)) & 0x8000800080008000
	t3_0 := (t2_0 >> 15) * 0x000000000000ffff
	t4_0 := ^t3_0
	t1_1 := ^((y1 | 0x8000800080008000) - (x1 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t2_1 := ((^y1 & x1) | (^(y1 ^ x1) & t1_1)) & 0x8000800080008000
	t3_1 := (t2_1 >> 15) * 0x000000000000ffff
	t4_1 := ^t3_1
	return v128{t4_0, t4_1}
}

//go:nosplit
func i16x8_ge_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 ^ 0x8000800080008000
	t2_0 := y0 ^ 0x8000800080008000
	t3_0 := ^((t1_0 | 0x8000800080008000) - (t2_0 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t4_0 := ((^t1_0 & t2_0) | (^(t1_0 ^ t2_0) & t3_0)) & 0x8000800080008000
	t5_0 := (t4_0 >> 15) * 0x000000000000ffff
	t6_0 := ^t5_0
	t1_1 := x1 ^ 0x8000800080008000
	t2_1 := y1 ^ 0x8000800080008000
	t3_1 := ^((t1_1 | 0x8000800080008000) - (t2_1 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t4_1 := ((^t1_1 & t2_1) | (^(t1_1 ^ t2_1) & t3_1)) & 0x8000800080008000
	t5_1 := (t4_1 >> 15) * 0x000000000000ffff
	t6_1 := ^t5_1
	return v128{t6_0, t6_1}
}

//go:nosplit
func i16x8_ge_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ^((x0 | 0x8000800080008000) - (y0 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t2_0 := ((^x0 & y0) | (^(x0 ^ y0) & t1_0)) & 0x8000800080008000
	t3_0 := (t2_0 >> 15) * 0x000000000000ffff
	t4_0 := ^t3_0
	t1_1 := ^((x1 | 0x8000800080008000) - (y1 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t2_1 := ((^x1 & y1) | (^(x1 ^ y1) & t1_1)) & 0x8000800080008000
	t3_1 := (t2_1 >> 15) * 0x000000000000ffff
	t4_1 := ^t3_1
	return v128{t4_0, t4_1}
}

//go:nosplit
func i16x8_min_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 ^ 0x8000800080008000
	t2_0 := y0 ^ 0x8000800080008000
	t3_0 := ^((t1_0 | 0x8000800080008000) - (t2_0 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t4_0 := ((^t1_0 & t2_0) | (^(t1_0 ^ t2_0) & t3_0)) & 0x8000800080008000
	t5_0 := (t4_0 >> 15) * 0x000000000000ffff
	t6_0 := (x0 & t5_0) | (y0 &^ t5_0)
	t1_1 := x1 ^ 0x8000800080008000
	t2_1 := y1 ^ 0x8000800080008000
	t3_1 := ^((t1_1 | 0x8000800080008000) - (t2_1 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t4_1 := ((^t1_1 & t2_1) | (^(t1_1 ^ t2_1) & t3_1)) & 0x8000800080008000
	t5_1 := (t4_1 >> 15) * 0x000000000000ffff
	t6_1 := (x1 & t5_1) | (y1 &^ t5_1)
	return v128{t6_0, t6_1}
}

//go:nosplit
func i16x8_min_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ^((x0 | 0x8000800080008000) - (y0 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t2_0 := ((^x0 & y0) | (^(x0 ^ y0) & t1_0)) & 0x8000800080008000
	t3_0 := (t2_0 >> 15) * 0x000000000000ffff
	t4_0 := (x0 & t3_0) | (y0 &^ t3_0)
	t1_1 := ^((x1 | 0x8000800080008000) - (y1 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t2_1 := ((^x1 & y1) | (^(x1 ^ y1) & t1_1)) & 0x8000800080008000
	t3_1 := (t2_1 >> 15) * 0x000000000000ffff
	t4_1 := (x1 & t3_1) | (y1 &^ t3_1)
	return v128{t4_0, t4_1}
}

//go:nosplit
func i16x8_max_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 ^ 0x8000800080008000
	t2_0 := y0 ^ 0x8000800080008000
	t3_0 := ^((t1_0 | 0x8000800080008000) - (t2_0 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t4_0 := ((^t1_0 & t2_0) | (^(t1_0 ^ t2_0) & t3_0)) & 0x8000800080008000
	t5_0 := (t4_0 >> 15) * 0x000000000000ffff
	t6_0 := (y0 & t5_0) | (x0 &^ t5_0)
	t1_1 := x1 ^ 0x8000800080008000
	t2_1 := y1 ^ 0x8000800080008000
	t3_1 := ^((t1_1 | 0x8000800080008000) - (t2_1 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t4_1 := ((^t1_1 & t2_1) | (^(t1_1 ^ t2_1) & t3_1)) & 0x8000800080008000
	t5_1 := (t4_1 >> 15) * 0x000000000000ffff
	t6_1 := (y1 & t5_1) | (x1 &^ t5_1)
	return v128{t6_0, t6_1}
}

//go:nosplit
func i16x8_max_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ^((x0 | 0x8000800080008000) - (y0 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t2_0 := ((^x0 & y0) | (^(x0 ^ y0) & t1_0)) & 0x8000800080008000
	t3_0 := (t2_0 >> 15) * 0x000000000000ffff
	t4_0 := (y0 & t3_0) | (x0 &^ t3_0)
	t1_1 := ^((x1 | 0x8000800080008000) - (y1 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t2_1 := ((^x1 & y1) | (^(x1 ^ y1) & t1_1)) & 0x8000800080008000
	t3_1 := (t2_1 >> 15) * 0x000000000000ffff
	t4_1 := (y1 & t3_1) | (x1 &^ t3_1)
	return v128{t4_0, t4_1}
}

//go:nosplit
func i16x8_shl(a v128, y int32) v128 {
	x0, x1 := a.lo, a.hi
	s := uint(y) & 15
	t1_0 := (uint64(0x000000000000ffff) >> s) * 0x0001000100010001
	t2_0 := (x0 & t1_0) << s
	t1_1 := (uint64(0x000000000000ffff) >> s) * 0x0001000100010001
	t2_1 := (x1 & t1_1) << s
	return v128{t2_0, t2_1}
}

//go:nosplit
func i16x8_shr_u(a v128, y int32) v128 {
	x0, x1 := a.lo, a.hi
	s := uint(y) & 15
	t1_0 := (uint64(0x000000000000ffff) >> s) * 0x0001000100010001
	t2_0 := (x0 >> s) & t1_0
	t1_1 := (uint64(0x000000000000ffff) >> s) * 0x0001000100010001
	t2_1 := (x1 >> s) & t1_1
	return v128{t2_0, t2_1}
}

//go:nosplit
func i16x8_shr_s(a v128, y int32) v128 {
	x0, x1 := a.lo, a.hi
	s := uint(y) & 15
	t1_0 := x0 ^ 0x8000800080008000
	t2_0 := (uint64(0x000000000000ffff) >> s) * 0x0001000100010001
	t3_0 := (t1_0 >> s) & t2_0
	t4_0 := (uint64(0x0000000000008000) >> s) * 0x0001000100010001
	t5_0 := ((t3_0 | 0x8000800080008000) - (t4_0 & 0x7fff7fff7fff7fff)) ^ ((t3_0 ^ ^t4_0) & 0x8000800080008000)
	t1_1 := x1 ^ 0x8000800080008000
	t2_1 := (uint64(0x000000000000ffff) >> s) * 0x0001000100010001
	t3_1 := (t1_1 >> s) & t2_1
	t4_1 := (uint64(0x0000000000008000) >> s) * 0x0001000100010001
	t5_1 := ((t3_1 | 0x8000800080008000) - (t4_1 & 0x7fff7fff7fff7fff)) ^ ((t3_1 ^ ^t4_1) & 0x8000800080008000)
	return v128{t5_0, t5_1}
}

//go:nosplit
func i16x8_add_sat_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ((x0 & 0x7fff7fff7fff7fff) + (y0 & 0x7fff7fff7fff7fff)) ^ ((x0 ^ y0) & 0x8000800080008000)
	t2_0 := ^((t1_0 | 0x8000800080008000) - (x0 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t3_0 := ((^t1_0 & x0) | (^(t1_0 ^ x0) & t2_0)) & 0x8000800080008000
	t4_0 := (t3_0 >> 15) * 0x000000000000ffff
	t5_0 := t1_0 | t4_0
	t1_1 := ((x1 & 0x7fff7fff7fff7fff) + (y1 & 0x7fff7fff7fff7fff)) ^ ((x1 ^ y1) & 0x8000800080008000)
	t2_1 := ^((t1_1 | 0x8000800080008000) - (x1 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t3_1 := ((^t1_1 & x1) | (^(t1_1 ^ x1) & t2_1)) & 0x8000800080008000
	t4_1 := (t3_1 >> 15) * 0x000000000000ffff
	t5_1 := t1_1 | t4_1
	return v128{t5_0, t5_1}
}

//go:nosplit
func i16x8_sub_sat_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ^((x0 | 0x8000800080008000) - (y0 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t2_0 := ((^x0 & y0) | (^(x0 ^ y0) & t1_0)) & 0x8000800080008000
	t3_0 := (t2_0 >> 15) * 0x000000000000ffff
	t4_0 := ((x0 | 0x8000800080008000) - (y0 & 0x7fff7fff7fff7fff)) ^ ((x0 ^ ^y0) & 0x8000800080008000)
	t5_0 := t4_0 &^ t3_0
	t1_1 := ^((x1 | 0x8000800080008000) - (y1 & 0x7fff7fff7fff7fff)) & 0x8000800080008000
	t2_1 := ((^x1 & y1) | (^(x1 ^ y1) & t1_1)) & 0x8000800080008000
	t3_1 := (t2_1 >> 15) * 0x000000000000ffff
	t4_1 := ((x1 | 0x8000800080008000) - (y1 & 0x7fff7fff7fff7fff)) ^ ((x1 ^ ^y1) & 0x8000800080008000)
	t5_1 := t4_1 &^ t3_1
	return v128{t5_0, t5_1}
}

//go:nosplit
func i16x8_avgr_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := (x0 | y0) - (((x0 ^ y0) >> 1) & 0x7fff7fff7fff7fff)
	t1_1 := (x1 | y1) - (((x1 ^ y1) >> 1) & 0x7fff7fff7fff7fff)
	return v128{t1_0, t1_1}
}

//go:nosplit
func i32x4_add(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ((x0 & 0x7fffffff7fffffff) + (y0 & 0x7fffffff7fffffff)) ^ ((x0 ^ y0) & 0x8000000080000000)
	t1_1 := ((x1 & 0x7fffffff7fffffff) + (y1 & 0x7fffffff7fffffff)) ^ ((x1 ^ y1) & 0x8000000080000000)
	return v128{t1_0, t1_1}
}

//go:nosplit
func i32x4_sub(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ((x0 | 0x8000000080000000) - (y0 & 0x7fffffff7fffffff)) ^ ((x0 ^ ^y0) & 0x8000000080000000)
	t1_1 := ((x1 | 0x8000000080000000) - (y1 & 0x7fffffff7fffffff)) ^ ((x1 ^ ^y1) & 0x8000000080000000)
	return v128{t1_0, t1_1}
}

//go:nosplit
func i32x4_neg(a v128) v128 {
	x0, x1 := a.lo, a.hi
	t1_0 := (0x8000000080000000 - (x0 & 0x7fffffff7fffffff)) ^ (^x0 & 0x8000000080000000)
	t1_1 := (0x8000000080000000 - (x1 & 0x7fffffff7fffffff)) ^ (^x1 & 0x8000000080000000)
	return v128{t1_0, t1_1}
}

//go:nosplit
func i32x4_abs(a v128) v128 {
	x0, x1 := a.lo, a.hi
	t1_0 := ((x0 & 0x8000000080000000) >> 31) * 0x00000000ffffffff
	t2_0 := x0 ^ t1_0
	t3_0 := ((t2_0 | 0x8000000080000000) - (t1_0 & 0x7fffffff7fffffff)) ^ ((t2_0 ^ ^t1_0) & 0x8000000080000000)
	t1_1 := ((x1 & 0x8000000080000000) >> 31) * 0x00000000ffffffff
	t2_1 := x1 ^ t1_1
	t3_1 := ((t2_1 | 0x8000000080000000) - (t1_1 & 0x7fffffff7fffffff)) ^ ((t2_1 ^ ^t1_1) & 0x8000000080000000)
	return v128{t3_0, t3_1}
}

//go:nosplit
func i32x4_eq(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 ^ y0
	t2_0 := ^(((t1_0 & 0x7fffffff7fffffff) + 0x7fffffff7fffffff) | t1_0 | 0x7fffffff7fffffff) & 0x8000000080000000
	t3_0 := (t2_0 >> 31) * 0x00000000ffffffff
	t1_1 := x1 ^ y1
	t2_1 := ^(((t1_1 & 0x7fffffff7fffffff) + 0x7fffffff7fffffff) | t1_1 | 0x7fffffff7fffffff) & 0x8000000080000000
	t3_1 := (t2_1 >> 31) * 0x00000000ffffffff
	return v128{t3_0, t3_1}
}

//go:nosplit
func i32x4_ne(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 ^ y0
	t2_0 := ^(((t1_0 & 0x7fffffff7fffffff) + 0x7fffffff7fffffff) | t1_0 | 0x7fffffff7fffffff) & 0x8000000080000000
	t3_0 := (t2_0 >> 31) * 0x00000000ffffffff
	t4_0 := ^t3_0
	t1_1 := x1 ^ y1
	t2_1 := ^(((t1_1 & 0x7fffffff7fffffff) + 0x7fffffff7fffffff) | t1_1 | 0x7fffffff7fffffff) & 0x8000000080000000
	t3_1 := (t2_1 >> 31) * 0x00000000ffffffff
	t4_1 := ^t3_1
	return v128{t4_0, t4_1}
}

//go:nosplit
func i32x4_lt_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 ^ 0x8000000080000000
	t2_0 := y0 ^ 0x8000000080000000
	t3_0 := ^((t1_0 | 0x8000000080000000) - (t2_0 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t4_0 := ((^t1_0 & t2_0) | (^(t1_0 ^ t2_0) & t3_0)) & 0x8000000080000000
	t5_0 := (t4_0 >> 31) * 0x00000000ffffffff
	t1_1 := x1 ^ 0x8000000080000000
	t2_1 := y1 ^ 0x8000000080000000
	t3_1 := ^((t1_1 | 0x8000000080000000) - (t2_1 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t4_1 := ((^t1_1 & t2_1) | (^(t1_1 ^ t2_1) & t3_1)) & 0x8000000080000000
	t5_1 := (t4_1 >> 31) * 0x00000000ffffffff
	return v128{t5_0, t5_1}
}

//go:nosplit
func i32x4_lt_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ^((x0 | 0x8000000080000000) - (y0 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t2_0 := ((^x0 & y0) | (^(x0 ^ y0) & t1_0)) & 0x8000000080000000
	t3_0 := (t2_0 >> 31) * 0x00000000ffffffff
	t1_1 := ^((x1 | 0x8000000080000000) - (y1 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t2_1 := ((^x1 & y1) | (^(x1 ^ y1) & t1_1)) & 0x8000000080000000
	t3_1 := (t2_1 >> 31) * 0x00000000ffffffff
	return v128{t3_0, t3_1}
}

//go:nosplit
func i32x4_gt_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := y0 ^ 0x8000000080000000
	t2_0 := x0 ^ 0x8000000080000000
	t3_0 := ^((t1_0 | 0x8000000080000000) - (t2_0 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t4_0 := ((^t1_0 & t2_0) | (^(t1_0 ^ t2_0) & t3_0)) & 0x8000000080000000
	t5_0 := (t4_0 >> 31) * 0x00000000ffffffff
	t1_1 := y1 ^ 0x8000000080000000
	t2_1 := x1 ^ 0x8000000080000000
	t3_1 := ^((t1_1 | 0x8000000080000000) - (t2_1 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t4_1 := ((^t1_1 & t2_1) | (^(t1_1 ^ t2_1) & t3_1)) & 0x8000000080000000
	t5_1 := (t4_1 >> 31) * 0x00000000ffffffff
	return v128{t5_0, t5_1}
}

//go:nosplit
func i32x4_gt_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ^((y0 | 0x8000000080000000) - (x0 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t2_0 := ((^y0 & x0) | (^(y0 ^ x0) & t1_0)) & 0x8000000080000000
	t3_0 := (t2_0 >> 31) * 0x00000000ffffffff
	t1_1 := ^((y1 | 0x8000000080000000) - (x1 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t2_1 := ((^y1 & x1) | (^(y1 ^ x1) & t1_1)) & 0x8000000080000000
	t3_1 := (t2_1 >> 31) * 0x00000000ffffffff
	return v128{t3_0, t3_1}
}

//go:nosplit
func i32x4_le_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := y0 ^ 0x8000000080000000
	t2_0 := x0 ^ 0x8000000080000000
	t3_0 := ^((t1_0 | 0x8000000080000000) - (t2_0 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t4_0 := ((^t1_0 & t2_0) | (^(t1_0 ^ t2_0) & t3_0)) & 0x8000000080000000
	t5_0 := (t4_0 >> 31) * 0x00000000ffffffff
	t6_0 := ^t5_0
	t1_1 := y1 ^ 0x8000000080000000
	t2_1 := x1 ^ 0x8000000080000000
	t3_1 := ^((t1_1 | 0x8000000080000000) - (t2_1 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t4_1 := ((^t1_1 & t2_1) | (^(t1_1 ^ t2_1) & t3_1)) & 0x8000000080000000
	t5_1 := (t4_1 >> 31) * 0x00000000ffffffff
	t6_1 := ^t5_1
	return v128{t6_0, t6_1}
}

//go:nosplit
func i32x4_le_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ^((y0 | 0x8000000080000000) - (x0 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t2_0 := ((^y0 & x0) | (^(y0 ^ x0) & t1_0)) & 0x8000000080000000
	t3_0 := (t2_0 >> 31) * 0x00000000ffffffff
	t4_0 := ^t3_0
	t1_1 := ^((y1 | 0x8000000080000000) - (x1 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t2_1 := ((^y1 & x1) | (^(y1 ^ x1) & t1_1)) & 0x8000000080000000
	t3_1 := (t2_1 >> 31) * 0x00000000ffffffff
	t4_1 := ^t3_1
	return v128{t4_0, t4_1}
}

//go:nosplit
func i32x4_ge_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 ^ 0x8000000080000000
	t2_0 := y0 ^ 0x8000000080000000
	t3_0 := ^((t1_0 | 0x8000000080000000) - (t2_0 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t4_0 := ((^t1_0 & t2_0) | (^(t1_0 ^ t2_0) & t3_0)) & 0x8000000080000000
	t5_0 := (t4_0 >> 31) * 0x00000000ffffffff
	t6_0 := ^t5_0
	t1_1 := x1 ^ 0x8000000080000000
	t2_1 := y1 ^ 0x8000000080000000
	t3_1 := ^((t1_1 | 0x8000000080000000) - (t2_1 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t4_1 := ((^t1_1 & t2_1) | (^(t1_1 ^ t2_1) & t3_1)) & 0x8000000080000000
	t5_1 := (t4_1 >> 31) * 0x00000000ffffffff
	t6_1 := ^t5_1
	return v128{t6_0, t6_1}
}

//go:nosplit
func i32x4_ge_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ^((x0 | 0x8000000080000000) - (y0 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t2_0 := ((^x0 & y0) | (^(x0 ^ y0) & t1_0)) & 0x8000000080000000
	t3_0 := (t2_0 >> 31) * 0x00000000ffffffff
	t4_0 := ^t3_0
	t1_1 := ^((x1 | 0x8000000080000000) - (y1 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t2_1 := ((^x1 & y1) | (^(x1 ^ y1) & t1_1)) & 0x8000000080000000
	t3_1 := (t2_1 >> 31) * 0x00000000ffffffff
	t4_1 := ^t3_1
	return v128{t4_0, t4_1}
}

//go:nosplit
func i32x4_min_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 ^ 0x8000000080000000
	t2_0 := y0 ^ 0x8000000080000000
	t3_0 := ^((t1_0 | 0x8000000080000000) - (t2_0 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t4_0 := ((^t1_0 & t2_0) | (^(t1_0 ^ t2_0) & t3_0)) & 0x8000000080000000
	t5_0 := (t4_0 >> 31) * 0x00000000ffffffff
	t6_0 := (x0 & t5_0) | (y0 &^ t5_0)
	t1_1 := x1 ^ 0x8000000080000000
	t2_1 := y1 ^ 0x8000000080000000
	t3_1 := ^((t1_1 | 0x8000000080000000) - (t2_1 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t4_1 := ((^t1_1 & t2_1) | (^(t1_1 ^ t2_1) & t3_1)) & 0x8000000080000000
	t5_1 := (t4_1 >> 31) * 0x00000000ffffffff
	t6_1 := (x1 & t5_1) | (y1 &^ t5_1)
	return v128{t6_0, t6_1}
}

//go:nosplit
func i32x4_min_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ^((x0 | 0x8000000080000000) - (y0 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t2_0 := ((^x0 & y0) | (^(x0 ^ y0) & t1_0)) & 0x8000000080000000
	t3_0 := (t2_0 >> 31) * 0x00000000ffffffff
	t4_0 := (x0 & t3_0) | (y0 &^ t3_0)
	t1_1 := ^((x1 | 0x8000000080000000) - (y1 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t2_1 := ((^x1 & y1) | (^(x1 ^ y1) & t1_1)) & 0x8000000080000000
	t3_1 := (t2_1 >> 31) * 0x00000000ffffffff
	t4_1 := (x1 & t3_1) | (y1 &^ t3_1)
	return v128{t4_0, t4_1}
}

//go:nosplit
func i32x4_max_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := x0 ^ 0x8000000080000000
	t2_0 := y0 ^ 0x8000000080000000
	t3_0 := ^((t1_0 | 0x8000000080000000) - (t2_0 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t4_0 := ((^t1_0 & t2_0) | (^(t1_0 ^ t2_0) & t3_0)) & 0x8000000080000000
	t5_0 := (t4_0 >> 31) * 0x00000000ffffffff
	t6_0 := (y0 & t5_0) | (x0 &^ t5_0)
	t1_1 := x1 ^ 0x8000000080000000
	t2_1 := y1 ^ 0x8000000080000000
	t3_1 := ^((t1_1 | 0x8000000080000000) - (t2_1 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t4_1 := ((^t1_1 & t2_1) | (^(t1_1 ^ t2_1) & t3_1)) & 0x8000000080000000
	t5_1 := (t4_1 >> 31) * 0x00000000ffffffff
	t6_1 := (y1 & t5_1) | (x1 &^ t5_1)
	return v128{t6_0, t6_1}
}

//go:nosplit
func i32x4_max_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ^((x0 | 0x8000000080000000) - (y0 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t2_0 := ((^x0 & y0) | (^(x0 ^ y0) & t1_0)) & 0x8000000080000000
	t3_0 := (t2_0 >> 31) * 0x00000000ffffffff
	t4_0 := (y0 & t3_0) | (x0 &^ t3_0)
	t1_1 := ^((x1 | 0x8000000080000000) - (y1 & 0x7fffffff7fffffff)) & 0x8000000080000000
	t2_1 := ((^x1 & y1) | (^(x1 ^ y1) & t1_1)) & 0x8000000080000000
	t3_1 := (t2_1 >> 31) * 0x00000000ffffffff
	t4_1 := (y1 & t3_1) | (x1 &^ t3_1)
	return v128{t4_0, t4_1}
}

//go:nosplit
func i32x4_shl(a v128, y int32) v128 {
	x0, x1 := a.lo, a.hi
	s := uint(y) & 31
	t1_0 := (uint64(0x00000000ffffffff) >> s) * 0x0000000100000001
	t2_0 := (x0 & t1_0) << s
	t1_1 := (uint64(0x00000000ffffffff) >> s) * 0x0000000100000001
	t2_1 := (x1 & t1_1) << s
	return v128{t2_0, t2_1}
}

//go:nosplit
func i32x4_shr_u(a v128, y int32) v128 {
	x0, x1 := a.lo, a.hi
	s := uint(y) & 31
	t1_0 := (uint64(0x00000000ffffffff) >> s) * 0x0000000100000001
	t2_0 := (x0 >> s) & t1_0
	t1_1 := (uint64(0x00000000ffffffff) >> s) * 0x0000000100000001
	t2_1 := (x1 >> s) & t1_1
	return v128{t2_0, t2_1}
}

//go:nosplit
func i32x4_shr_s(a v128, y int32) v128 {
	x0, x1 := a.lo, a.hi
	s := uint(y) & 31
	t1_0 := x0 ^ 0x8000000080000000
	t2_0 := (uint64(0x00000000ffffffff) >> s) * 0x0000000100000001
	t3_0 := (t1_0 >> s) & t2_0
	t4_0 := (uint64(0x0000000080000000) >> s) * 0x0000000100000001
	t5_0 := ((t3_0 | 0x8000000080000000) - (t4_0 & 0x7fffffff7fffffff)) ^ ((t3_0 ^ ^t4_0) & 0x8000000080000000)
	t1_1 := x1 ^ 0x8000000080000000
	t2_1 := (uint64(0x00000000ffffffff) >> s) * 0x0000000100000001
	t3_1 := (t1_1 >> s) & t2_1
	t4_1 := (uint64(0x0000000080000000) >> s) * 0x0000000100000001
	t5_1 := ((t3_1 | 0x8000000080000000) - (t4_1 & 0x7fffffff7fffffff)) ^ ((t3_1 ^ ^t4_1) & 0x8000000080000000)
	return v128{t5_0, t5_1}
}

//go:nosplit
func i8x16_popcnt(a v128) v128 {
	x0, x1 := a.lo, a.hi
	t1_0 := x0 - ((x0 >> 1) & 0x5555555555555555)
	t2_0 := (t1_0 & 0x3333333333333333) + ((t1_0 >> 2) & 0x3333333333333333)
	t3_0 := (t2_0 + (t2_0 >> 4)) & 0x0f0f0f0f0f0f0f0f
	t1_1 := x1 - ((x1 >> 1) & 0x5555555555555555)
	t2_1 := (t1_1 & 0x3333333333333333) + ((t1_1 >> 2) & 0x3333333333333333)
	t3_1 := (t2_1 + (t2_1 >> 4)) & 0x0f0f0f0f0f0f0f0f
	return v128{t3_0, t3_1}
}

//go:nosplit
func i64x2_add(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	return v128{x0 + y0, x1 + y1}
}

//go:nosplit
func i64x2_sub(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	return v128{x0 - y0, x1 - y1}
}

//go:nosplit
func i64x2_mul(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	return v128{x0 * y0, x1 * y1}
}

//go:nosplit
func i64x2_eq(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	var r0 uint64
	if x0 == y0 {
		r0 = ^uint64(0)
	}
	var r1 uint64
	if x1 == y1 {
		r1 = ^uint64(0)
	}
	return v128{r0, r1}
}

//go:nosplit
func i64x2_ne(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	var r0 uint64
	if x0 != y0 {
		r0 = ^uint64(0)
	}
	var r1 uint64
	if x1 != y1 {
		r1 = ^uint64(0)
	}
	return v128{r0, r1}
}

//go:nosplit
func i64x2_lt_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	var r0 uint64
	if int64(x0) < int64(y0) {
		r0 = ^uint64(0)
	}
	var r1 uint64
	if int64(x1) < int64(y1) {
		r1 = ^uint64(0)
	}
	return v128{r0, r1}
}

//go:nosplit
func i64x2_gt_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	var r0 uint64
	if int64(x0) > int64(y0) {
		r0 = ^uint64(0)
	}
	var r1 uint64
	if int64(x1) > int64(y1) {
		r1 = ^uint64(0)
	}
	return v128{r0, r1}
}

//go:nosplit
func i64x2_le_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	var r0 uint64
	if int64(x0) <= int64(y0) {
		r0 = ^uint64(0)
	}
	var r1 uint64
	if int64(x1) <= int64(y1) {
		r1 = ^uint64(0)
	}
	return v128{r0, r1}
}

//go:nosplit
func i64x2_ge_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	var r0 uint64
	if int64(x0) >= int64(y0) {
		r0 = ^uint64(0)
	}
	var r1 uint64
	if int64(x1) >= int64(y1) {
		r1 = ^uint64(0)
	}
	return v128{r0, r1}
}

//go:nosplit
func i64x2_neg(a v128) v128 {
	x0, x1 := a.lo, a.hi
	return v128{-x0, -x1}
}

//go:nosplit
func i64x2_shl(a v128, y int32) v128 {
	x0, x1 := a.lo, a.hi
	s := uint(y) & 63
	return v128{x0 << s, x1 << s}
}

//go:nosplit
func i64x2_shr_u(a v128, y int32) v128 {
	x0, x1 := a.lo, a.hi
	s := uint(y) & 63
	return v128{x0 >> s, x1 >> s}
}

//go:nosplit
func i64x2_shr_s(a v128, y int32) v128 {
	x0, x1 := a.lo, a.hi
	s := uint(y) & 63
	return v128{uint64(int64(x0) >> s), uint64(int64(x1) >> s)}
}

//go:nosplit
func i16x8_extend_low_i8x16_u(a v128) v128 {
	x0 := a.lo
	t1_0 := x0 & 0xffffffff
	t2_0 := (t1_0 | (t1_0 << 16)) & 0x0000ffff0000ffff
	t3_0 := (t2_0 | (t2_0 << 8)) & 0x00ff00ff00ff00ff
	t1_1 := x0 >> 32
	t2_1 := t1_1 & 0xffffffff
	t3_1 := (t2_1 | (t2_1 << 16)) & 0x0000ffff0000ffff
	t4_1 := (t3_1 | (t3_1 << 8)) & 0x00ff00ff00ff00ff
	return v128{t3_0, t4_1}
}

//go:nosplit
func i32x4_extend_low_i16x8_u(a v128) v128 {
	x0 := a.lo
	t1_0 := x0 & 0xffffffff
	t2_0 := (t1_0 | (t1_0 << 16)) & 0x0000ffff0000ffff
	t1_1 := x0 >> 32
	t2_1 := t1_1 & 0xffffffff
	t3_1 := (t2_1 | (t2_1 << 16)) & 0x0000ffff0000ffff
	return v128{t2_0, t3_1}
}

//go:nosplit
func i64x2_extend_low_i32x4_u(a v128) v128 {
	x0 := a.lo
	t1_0 := uint64(uint32(x0))
	t1_1 := x0 >> 32
	t2_1 := uint64(uint32(t1_1))
	return v128{t1_0, t2_1}
}

//go:nosplit
func i16x8_extend_low_i8x16_s(a v128) v128 {
	x0 := a.lo
	t1_0 := x0 & 0xffffffff
	t2_0 := (t1_0 | (t1_0 << 16)) & 0x0000ffff0000ffff
	t3_0 := (t2_0 | (t2_0 << 8)) & 0x00ff00ff00ff00ff
	t4_0 := t3_0 | (((t3_0&0x0080008000800080)>>7)*0xff)<<8
	t1_1 := x0 >> 32
	t2_1 := t1_1 & 0xffffffff
	t3_1 := (t2_1 | (t2_1 << 16)) & 0x0000ffff0000ffff
	t4_1 := (t3_1 | (t3_1 << 8)) & 0x00ff00ff00ff00ff
	t5_1 := t4_1 | (((t4_1&0x0080008000800080)>>7)*0xff)<<8
	return v128{t4_0, t5_1}
}

//go:nosplit
func i32x4_extend_low_i16x8_s(a v128) v128 {
	x0 := a.lo
	t1_0 := x0 & 0xffffffff
	t2_0 := (t1_0 | (t1_0 << 16)) & 0x0000ffff0000ffff
	t3_0 := t2_0 | (((t2_0&0x0000800000008000)>>15)*0xffff)<<16
	t1_1 := x0 >> 32
	t2_1 := t1_1 & 0xffffffff
	t3_1 := (t2_1 | (t2_1 << 16)) & 0x0000ffff0000ffff
	t4_1 := t3_1 | (((t3_1&0x0000800000008000)>>15)*0xffff)<<16
	return v128{t3_0, t4_1}
}

//go:nosplit
func i64x2_extend_low_i32x4_s(a v128) v128 {
	x0 := a.lo
	t1_0 := uint64(int64(int32(x0)))
	t1_1 := x0 >> 32
	t2_1 := uint64(int64(int32(t1_1)))
	return v128{t1_0, t2_1}
}

//go:nosplit
func i16x8_extend_high_i8x16_u(a v128) v128 {
	x1 := a.hi
	t1_0 := x1 & 0xffffffff
	t2_0 := (t1_0 | (t1_0 << 16)) & 0x0000ffff0000ffff
	t3_0 := (t2_0 | (t2_0 << 8)) & 0x00ff00ff00ff00ff
	t1_1 := x1 >> 32
	t2_1 := t1_1 & 0xffffffff
	t3_1 := (t2_1 | (t2_1 << 16)) & 0x0000ffff0000ffff
	t4_1 := (t3_1 | (t3_1 << 8)) & 0x00ff00ff00ff00ff
	return v128{t3_0, t4_1}
}

//go:nosplit
func i32x4_extend_high_i16x8_u(a v128) v128 {
	x1 := a.hi
	t1_0 := x1 & 0xffffffff
	t2_0 := (t1_0 | (t1_0 << 16)) & 0x0000ffff0000ffff
	t1_1 := x1 >> 32
	t2_1 := t1_1 & 0xffffffff
	t3_1 := (t2_1 | (t2_1 << 16)) & 0x0000ffff0000ffff
	return v128{t2_0, t3_1}
}

//go:nosplit
func i64x2_extend_high_i32x4_u(a v128) v128 {
	x1 := a.hi
	t1_0 := uint64(uint32(x1))
	t1_1 := x1 >> 32
	t2_1 := uint64(uint32(t1_1))
	return v128{t1_0, t2_1}
}

//go:nosplit
func i16x8_extend_high_i8x16_s(a v128) v128 {
	x1 := a.hi
	t1_0 := x1 & 0xffffffff
	t2_0 := (t1_0 | (t1_0 << 16)) & 0x0000ffff0000ffff
	t3_0 := (t2_0 | (t2_0 << 8)) & 0x00ff00ff00ff00ff
	t4_0 := t3_0 | (((t3_0&0x0080008000800080)>>7)*0xff)<<8
	t1_1 := x1 >> 32
	t2_1 := t1_1 & 0xffffffff
	t3_1 := (t2_1 | (t2_1 << 16)) & 0x0000ffff0000ffff
	t4_1 := (t3_1 | (t3_1 << 8)) & 0x00ff00ff00ff00ff
	t5_1 := t4_1 | (((t4_1&0x0080008000800080)>>7)*0xff)<<8
	return v128{t4_0, t5_1}
}

//go:nosplit
func i32x4_extend_high_i16x8_s(a v128) v128 {
	x1 := a.hi
	t1_0 := x1 & 0xffffffff
	t2_0 := (t1_0 | (t1_0 << 16)) & 0x0000ffff0000ffff
	t3_0 := t2_0 | (((t2_0&0x0000800000008000)>>15)*0xffff)<<16
	t1_1 := x1 >> 32
	t2_1 := t1_1 & 0xffffffff
	t3_1 := (t2_1 | (t2_1 << 16)) & 0x0000ffff0000ffff
	t4_1 := t3_1 | (((t3_1&0x0000800000008000)>>15)*0xffff)<<16
	return v128{t3_0, t4_1}
}

//go:nosplit
func i64x2_extend_high_i32x4_s(a v128) v128 {
	x1 := a.hi
	t1_0 := uint64(int64(int32(x1)))
	t1_1 := x1 >> 32
	t2_1 := uint64(int64(int32(t1_1)))
	return v128{t1_0, t2_1}
}

//go:nosplit
func i8x16_narrow_i16x8_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	r0 := uint64(uint8(int8(min(max(int16(x0), -128), 127)))) | uint64(uint8(int8(min(max(int16(x0>>16), -128), 127))))<<8 | uint64(uint8(int8(min(max(int16(x0>>32), -128), 127))))<<16 | uint64(uint8(int8(min(max(int16(x0>>48), -128), 127))))<<24 | uint64(uint8(int8(min(max(int16(x1), -128), 127))))<<32 | uint64(uint8(int8(min(max(int16(x1>>16), -128), 127))))<<40 | uint64(uint8(int8(min(max(int16(x1>>32), -128), 127))))<<48 | uint64(uint8(int8(min(max(int16(x1>>48), -128), 127))))<<56
	r1 := uint64(uint8(int8(min(max(int16(y0), -128), 127)))) | uint64(uint8(int8(min(max(int16(y0>>16), -128), 127))))<<8 | uint64(uint8(int8(min(max(int16(y0>>32), -128), 127))))<<16 | uint64(uint8(int8(min(max(int16(y0>>48), -128), 127))))<<24 | uint64(uint8(int8(min(max(int16(y1), -128), 127))))<<32 | uint64(uint8(int8(min(max(int16(y1>>16), -128), 127))))<<40 | uint64(uint8(int8(min(max(int16(y1>>32), -128), 127))))<<48 | uint64(uint8(int8(min(max(int16(y1>>48), -128), 127))))<<56
	return v128{r0, r1}
}

//go:nosplit
func i8x16_narrow_i16x8_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	r0 := uint64(uint8(min(max(int16(x0), 0), 255))) | uint64(uint8(min(max(int16(x0>>16), 0), 255)))<<8 | uint64(uint8(min(max(int16(x0>>32), 0), 255)))<<16 | uint64(uint8(min(max(int16(x0>>48), 0), 255)))<<24 | uint64(uint8(min(max(int16(x1), 0), 255)))<<32 | uint64(uint8(min(max(int16(x1>>16), 0), 255)))<<40 | uint64(uint8(min(max(int16(x1>>32), 0), 255)))<<48 | uint64(uint8(min(max(int16(x1>>48), 0), 255)))<<56
	r1 := uint64(uint8(min(max(int16(y0), 0), 255))) | uint64(uint8(min(max(int16(y0>>16), 0), 255)))<<8 | uint64(uint8(min(max(int16(y0>>32), 0), 255)))<<16 | uint64(uint8(min(max(int16(y0>>48), 0), 255)))<<24 | uint64(uint8(min(max(int16(y1), 0), 255)))<<32 | uint64(uint8(min(max(int16(y1>>16), 0), 255)))<<40 | uint64(uint8(min(max(int16(y1>>32), 0), 255)))<<48 | uint64(uint8(min(max(int16(y1>>48), 0), 255)))<<56
	return v128{r0, r1}
}

//go:nosplit
func i16x8_narrow_i32x4_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	r0 := uint64(uint16(int16(min(max(int32(x0), -32768), 32767)))) | uint64(uint16(int16(min(max(int32(x0>>32), -32768), 32767))))<<16 | uint64(uint16(int16(min(max(int32(x1), -32768), 32767))))<<32 | uint64(uint16(int16(min(max(int32(x1>>32), -32768), 32767))))<<48
	r1 := uint64(uint16(int16(min(max(int32(y0), -32768), 32767)))) | uint64(uint16(int16(min(max(int32(y0>>32), -32768), 32767))))<<16 | uint64(uint16(int16(min(max(int32(y1), -32768), 32767))))<<32 | uint64(uint16(int16(min(max(int32(y1>>32), -32768), 32767))))<<48
	return v128{r0, r1}
}

//go:nosplit
func i16x8_narrow_i32x4_u(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	r0 := uint64(uint16(min(max(int32(x0), 0), 65535))) | uint64(uint16(min(max(int32(x0>>32), 0), 65535)))<<16 | uint64(uint16(min(max(int32(x1), 0), 65535)))<<32 | uint64(uint16(min(max(int32(x1>>32), 0), 65535)))<<48
	r1 := uint64(uint16(min(max(int32(y0), 0), 65535))) | uint64(uint16(min(max(int32(y0>>32), 0), 65535)))<<16 | uint64(uint16(min(max(int32(y1), 0), 65535)))<<32 | uint64(uint16(min(max(int32(y1>>32), 0), 65535)))<<48
	return v128{r0, r1}
}

//go:nosplit
func f64x2_add(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	r0 := math.Float64bits(math.Float64frombits(x0) + math.Float64frombits(y0))
	r1 := math.Float64bits(math.Float64frombits(x1) + math.Float64frombits(y1))
	return v128{r0, r1}
}

//go:nosplit
func f32x4_add(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	l0 := uint64(math.Float32bits(math.Float32frombits(uint32(x0)) + math.Float32frombits(uint32(y0))))
	h0 := uint64(math.Float32bits(math.Float32frombits(uint32(x0>>32)) + math.Float32frombits(uint32(y0>>32))))
	r0 := l0 | h0<<32
	l1 := uint64(math.Float32bits(math.Float32frombits(uint32(x1)) + math.Float32frombits(uint32(y1))))
	h1 := uint64(math.Float32bits(math.Float32frombits(uint32(x1>>32)) + math.Float32frombits(uint32(y1>>32))))
	r1 := l1 | h1<<32
	return v128{r0, r1}
}

//go:nosplit
func f64x2_sub(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	r0 := math.Float64bits(math.Float64frombits(x0) - math.Float64frombits(y0))
	r1 := math.Float64bits(math.Float64frombits(x1) - math.Float64frombits(y1))
	return v128{r0, r1}
}

//go:nosplit
func f32x4_sub(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	l0 := uint64(math.Float32bits(math.Float32frombits(uint32(x0)) - math.Float32frombits(uint32(y0))))
	h0 := uint64(math.Float32bits(math.Float32frombits(uint32(x0>>32)) - math.Float32frombits(uint32(y0>>32))))
	r0 := l0 | h0<<32
	l1 := uint64(math.Float32bits(math.Float32frombits(uint32(x1)) - math.Float32frombits(uint32(y1))))
	h1 := uint64(math.Float32bits(math.Float32frombits(uint32(x1>>32)) - math.Float32frombits(uint32(y1>>32))))
	r1 := l1 | h1<<32
	return v128{r0, r1}
}

//go:nosplit
func f64x2_mul(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	r0 := math.Float64bits(math.Float64frombits(x0) * math.Float64frombits(y0))
	r1 := math.Float64bits(math.Float64frombits(x1) * math.Float64frombits(y1))
	return v128{r0, r1}
}

//go:nosplit
func f32x4_mul(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	l0 := uint64(math.Float32bits(math.Float32frombits(uint32(x0)) * math.Float32frombits(uint32(y0))))
	h0 := uint64(math.Float32bits(math.Float32frombits(uint32(x0>>32)) * math.Float32frombits(uint32(y0>>32))))
	r0 := l0 | h0<<32
	l1 := uint64(math.Float32bits(math.Float32frombits(uint32(x1)) * math.Float32frombits(uint32(y1))))
	h1 := uint64(math.Float32bits(math.Float32frombits(uint32(x1>>32)) * math.Float32frombits(uint32(y1>>32))))
	r1 := l1 | h1<<32
	return v128{r0, r1}
}

//go:nosplit
func f64x2_div(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	r0 := math.Float64bits(math.Float64frombits(x0) / math.Float64frombits(y0))
	r1 := math.Float64bits(math.Float64frombits(x1) / math.Float64frombits(y1))
	return v128{r0, r1}
}

//go:nosplit
func f32x4_div(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	l0 := uint64(math.Float32bits(math.Float32frombits(uint32(x0)) / math.Float32frombits(uint32(y0))))
	h0 := uint64(math.Float32bits(math.Float32frombits(uint32(x0>>32)) / math.Float32frombits(uint32(y0>>32))))
	r0 := l0 | h0<<32
	l1 := uint64(math.Float32bits(math.Float32frombits(uint32(x1)) / math.Float32frombits(uint32(y1))))
	h1 := uint64(math.Float32bits(math.Float32frombits(uint32(x1>>32)) / math.Float32frombits(uint32(y1>>32))))
	r1 := l1 | h1<<32
	return v128{r0, r1}
}

//go:nosplit
func i16x8_bitmask(v v128) int32 {
	x0, x1 := v.lo, v.hi
	lo := ((x0 & 0x8000800080008000) >> 15 * 0x0001000200040008) >> 48
	hi := ((x1 & 0x8000800080008000) >> 15 * 0x0001000200040008) >> 48
	return int32(hi<<4 | lo)
}

//go:nosplit
func i32x4_bitmask(v v128) int32 {
	x0, x1 := v.lo, v.hi
	lo := x0>>31&1 | x0>>63<<1
	hi := x1>>31&1 | x1>>63<<1
	return int32(hi<<2 | lo)
}

//go:nosplit
func i64x2_bitmask(v v128) int32 {
	x0, x1 := v.lo, v.hi
	return int32(x1>>63<<1 | x0>>63)
}

//go:nosplit
func v128_load8x8_u(x uint64) v128 {
	t1_0 := x & 0xffffffff
	t2_0 := (t1_0 | (t1_0 << 16)) & 0x0000ffff0000ffff
	t3_0 := (t2_0 | (t2_0 << 8)) & 0x00ff00ff00ff00ff
	t1_1 := x >> 32
	t2_1 := t1_1 & 0xffffffff
	t3_1 := (t2_1 | (t2_1 << 16)) & 0x0000ffff0000ffff
	t4_1 := (t3_1 | (t3_1 << 8)) & 0x00ff00ff00ff00ff
	return v128{t3_0, t4_1}
}

//go:nosplit
func v128_load16x4_u(x uint64) v128 {
	t1_0 := x & 0xffffffff
	t2_0 := (t1_0 | (t1_0 << 16)) & 0x0000ffff0000ffff
	t1_1 := x >> 32
	t2_1 := t1_1 & 0xffffffff
	t3_1 := (t2_1 | (t2_1 << 16)) & 0x0000ffff0000ffff
	return v128{t2_0, t3_1}
}

//go:nosplit
func v128_load32x2_u(x uint64) v128 {
	t1_0 := uint64(uint32(x))
	t1_1 := x >> 32
	t2_1 := uint64(uint32(t1_1))
	return v128{t1_0, t2_1}
}

//go:nosplit
func v128_load8x8_s(x uint64) v128 {
	t1_0 := x & 0xffffffff
	t2_0 := (t1_0 | (t1_0 << 16)) & 0x0000ffff0000ffff
	t3_0 := (t2_0 | (t2_0 << 8)) & 0x00ff00ff00ff00ff
	t4_0 := t3_0 | (((t3_0&0x0080008000800080)>>7)*0xff)<<8
	t1_1 := x >> 32
	t2_1 := t1_1 & 0xffffffff
	t3_1 := (t2_1 | (t2_1 << 16)) & 0x0000ffff0000ffff
	t4_1 := (t3_1 | (t3_1 << 8)) & 0x00ff00ff00ff00ff
	t5_1 := t4_1 | (((t4_1&0x0080008000800080)>>7)*0xff)<<8
	return v128{t4_0, t5_1}
}

//go:nosplit
func v128_load16x4_s(x uint64) v128 {
	t1_0 := x & 0xffffffff
	t2_0 := (t1_0 | (t1_0 << 16)) & 0x0000ffff0000ffff
	t3_0 := t2_0 | (((t2_0&0x0000800000008000)>>15)*0xffff)<<16
	t1_1 := x >> 32
	t2_1 := t1_1 & 0xffffffff
	t3_1 := (t2_1 | (t2_1 << 16)) & 0x0000ffff0000ffff
	t4_1 := t3_1 | (((t3_1&0x0000800000008000)>>15)*0xffff)<<16
	return v128{t3_0, t4_1}
}

//go:nosplit
func v128_load32x2_s(x uint64) v128 {
	t1_0 := uint64(int64(int32(x)))
	t1_1 := x >> 32
	t2_1 := uint64(int64(int32(t1_1)))
	return v128{t1_0, t2_1}
}

//go:nosplit
func f32x4_eq(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := math.Float32frombits(uint32(x0))
	ly0_0 := math.Float32frombits(uint32(y0))
	var o0_0 uint32
	if lx0_0 == ly0_0 {
		o0_0 = 0xffffffff
	}
	lx0_1 := math.Float32frombits(uint32(x0 >> 32))
	ly0_1 := math.Float32frombits(uint32(y0 >> 32))
	var o0_1 uint32
	if lx0_1 == ly0_1 {
		o0_1 = 0xffffffff
	}
	r0 := uint64(o0_0) | uint64(o0_1)<<32
	lx1_0 := math.Float32frombits(uint32(x1))
	ly1_0 := math.Float32frombits(uint32(y1))
	var o1_0 uint32
	if lx1_0 == ly1_0 {
		o1_0 = 0xffffffff
	}
	lx1_1 := math.Float32frombits(uint32(x1 >> 32))
	ly1_1 := math.Float32frombits(uint32(y1 >> 32))
	var o1_1 uint32
	if lx1_1 == ly1_1 {
		o1_1 = 0xffffffff
	}
	r1 := uint64(o1_0) | uint64(o1_1)<<32
	return v128{r0, r1}
}

//go:nosplit
func f32x4_ne(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := math.Float32frombits(uint32(x0))
	ly0_0 := math.Float32frombits(uint32(y0))
	var o0_0 uint32
	if lx0_0 != ly0_0 {
		o0_0 = 0xffffffff
	}
	lx0_1 := math.Float32frombits(uint32(x0 >> 32))
	ly0_1 := math.Float32frombits(uint32(y0 >> 32))
	var o0_1 uint32
	if lx0_1 != ly0_1 {
		o0_1 = 0xffffffff
	}
	r0 := uint64(o0_0) | uint64(o0_1)<<32
	lx1_0 := math.Float32frombits(uint32(x1))
	ly1_0 := math.Float32frombits(uint32(y1))
	var o1_0 uint32
	if lx1_0 != ly1_0 {
		o1_0 = 0xffffffff
	}
	lx1_1 := math.Float32frombits(uint32(x1 >> 32))
	ly1_1 := math.Float32frombits(uint32(y1 >> 32))
	var o1_1 uint32
	if lx1_1 != ly1_1 {
		o1_1 = 0xffffffff
	}
	r1 := uint64(o1_0) | uint64(o1_1)<<32
	return v128{r0, r1}
}

//go:nosplit
func f32x4_lt(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := math.Float32frombits(uint32(x0))
	ly0_0 := math.Float32frombits(uint32(y0))
	var o0_0 uint32
	if lx0_0 < ly0_0 {
		o0_0 = 0xffffffff
	}
	lx0_1 := math.Float32frombits(uint32(x0 >> 32))
	ly0_1 := math.Float32frombits(uint32(y0 >> 32))
	var o0_1 uint32
	if lx0_1 < ly0_1 {
		o0_1 = 0xffffffff
	}
	r0 := uint64(o0_0) | uint64(o0_1)<<32
	lx1_0 := math.Float32frombits(uint32(x1))
	ly1_0 := math.Float32frombits(uint32(y1))
	var o1_0 uint32
	if lx1_0 < ly1_0 {
		o1_0 = 0xffffffff
	}
	lx1_1 := math.Float32frombits(uint32(x1 >> 32))
	ly1_1 := math.Float32frombits(uint32(y1 >> 32))
	var o1_1 uint32
	if lx1_1 < ly1_1 {
		o1_1 = 0xffffffff
	}
	r1 := uint64(o1_0) | uint64(o1_1)<<32
	return v128{r0, r1}
}

//go:nosplit
func f32x4_gt(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := math.Float32frombits(uint32(x0))
	ly0_0 := math.Float32frombits(uint32(y0))
	var o0_0 uint32
	if lx0_0 > ly0_0 {
		o0_0 = 0xffffffff
	}
	lx0_1 := math.Float32frombits(uint32(x0 >> 32))
	ly0_1 := math.Float32frombits(uint32(y0 >> 32))
	var o0_1 uint32
	if lx0_1 > ly0_1 {
		o0_1 = 0xffffffff
	}
	r0 := uint64(o0_0) | uint64(o0_1)<<32
	lx1_0 := math.Float32frombits(uint32(x1))
	ly1_0 := math.Float32frombits(uint32(y1))
	var o1_0 uint32
	if lx1_0 > ly1_0 {
		o1_0 = 0xffffffff
	}
	lx1_1 := math.Float32frombits(uint32(x1 >> 32))
	ly1_1 := math.Float32frombits(uint32(y1 >> 32))
	var o1_1 uint32
	if lx1_1 > ly1_1 {
		o1_1 = 0xffffffff
	}
	r1 := uint64(o1_0) | uint64(o1_1)<<32
	return v128{r0, r1}
}

//go:nosplit
func f32x4_le(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := math.Float32frombits(uint32(x0))
	ly0_0 := math.Float32frombits(uint32(y0))
	var o0_0 uint32
	if lx0_0 <= ly0_0 {
		o0_0 = 0xffffffff
	}
	lx0_1 := math.Float32frombits(uint32(x0 >> 32))
	ly0_1 := math.Float32frombits(uint32(y0 >> 32))
	var o0_1 uint32
	if lx0_1 <= ly0_1 {
		o0_1 = 0xffffffff
	}
	r0 := uint64(o0_0) | uint64(o0_1)<<32
	lx1_0 := math.Float32frombits(uint32(x1))
	ly1_0 := math.Float32frombits(uint32(y1))
	var o1_0 uint32
	if lx1_0 <= ly1_0 {
		o1_0 = 0xffffffff
	}
	lx1_1 := math.Float32frombits(uint32(x1 >> 32))
	ly1_1 := math.Float32frombits(uint32(y1 >> 32))
	var o1_1 uint32
	if lx1_1 <= ly1_1 {
		o1_1 = 0xffffffff
	}
	r1 := uint64(o1_0) | uint64(o1_1)<<32
	return v128{r0, r1}
}

//go:nosplit
func f32x4_ge(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := math.Float32frombits(uint32(x0))
	ly0_0 := math.Float32frombits(uint32(y0))
	var o0_0 uint32
	if lx0_0 >= ly0_0 {
		o0_0 = 0xffffffff
	}
	lx0_1 := math.Float32frombits(uint32(x0 >> 32))
	ly0_1 := math.Float32frombits(uint32(y0 >> 32))
	var o0_1 uint32
	if lx0_1 >= ly0_1 {
		o0_1 = 0xffffffff
	}
	r0 := uint64(o0_0) | uint64(o0_1)<<32
	lx1_0 := math.Float32frombits(uint32(x1))
	ly1_0 := math.Float32frombits(uint32(y1))
	var o1_0 uint32
	if lx1_0 >= ly1_0 {
		o1_0 = 0xffffffff
	}
	lx1_1 := math.Float32frombits(uint32(x1 >> 32))
	ly1_1 := math.Float32frombits(uint32(y1 >> 32))
	var o1_1 uint32
	if lx1_1 >= ly1_1 {
		o1_1 = 0xffffffff
	}
	r1 := uint64(o1_0) | uint64(o1_1)<<32
	return v128{r0, r1}
}

//go:nosplit
func f32x4_min(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := math.Float32frombits(uint32(x0))
	ly0_0 := math.Float32frombits(uint32(y0))
	var o0_0 uint32
	switch {
	case lx0_0 != lx0_0 || ly0_0 != ly0_0:
		o0_0 = 0x7fc00000
	case lx0_0 == ly0_0:
		o0_0 = math.Float32bits(lx0_0) | math.Float32bits(ly0_0)
	case lx0_0 < ly0_0:
		o0_0 = math.Float32bits(lx0_0)
	default:
		o0_0 = math.Float32bits(ly0_0)
	}
	lx0_1 := math.Float32frombits(uint32(x0 >> 32))
	ly0_1 := math.Float32frombits(uint32(y0 >> 32))
	var o0_1 uint32
	switch {
	case lx0_1 != lx0_1 || ly0_1 != ly0_1:
		o0_1 = 0x7fc00000
	case lx0_1 == ly0_1:
		o0_1 = math.Float32bits(lx0_1) | math.Float32bits(ly0_1)
	case lx0_1 < ly0_1:
		o0_1 = math.Float32bits(lx0_1)
	default:
		o0_1 = math.Float32bits(ly0_1)
	}
	r0 := uint64(o0_0) | uint64(o0_1)<<32
	lx1_0 := math.Float32frombits(uint32(x1))
	ly1_0 := math.Float32frombits(uint32(y1))
	var o1_0 uint32
	switch {
	case lx1_0 != lx1_0 || ly1_0 != ly1_0:
		o1_0 = 0x7fc00000
	case lx1_0 == ly1_0:
		o1_0 = math.Float32bits(lx1_0) | math.Float32bits(ly1_0)
	case lx1_0 < ly1_0:
		o1_0 = math.Float32bits(lx1_0)
	default:
		o1_0 = math.Float32bits(ly1_0)
	}
	lx1_1 := math.Float32frombits(uint32(x1 >> 32))
	ly1_1 := math.Float32frombits(uint32(y1 >> 32))
	var o1_1 uint32
	switch {
	case lx1_1 != lx1_1 || ly1_1 != ly1_1:
		o1_1 = 0x7fc00000
	case lx1_1 == ly1_1:
		o1_1 = math.Float32bits(lx1_1) | math.Float32bits(ly1_1)
	case lx1_1 < ly1_1:
		o1_1 = math.Float32bits(lx1_1)
	default:
		o1_1 = math.Float32bits(ly1_1)
	}
	r1 := uint64(o1_0) | uint64(o1_1)<<32
	return v128{r0, r1}
}

//go:nosplit
func f32x4_max(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := math.Float32frombits(uint32(x0))
	ly0_0 := math.Float32frombits(uint32(y0))
	var o0_0 uint32
	switch {
	case lx0_0 != lx0_0 || ly0_0 != ly0_0:
		o0_0 = 0x7fc00000
	case lx0_0 == ly0_0:
		o0_0 = math.Float32bits(lx0_0) & math.Float32bits(ly0_0)
	case lx0_0 > ly0_0:
		o0_0 = math.Float32bits(lx0_0)
	default:
		o0_0 = math.Float32bits(ly0_0)
	}
	lx0_1 := math.Float32frombits(uint32(x0 >> 32))
	ly0_1 := math.Float32frombits(uint32(y0 >> 32))
	var o0_1 uint32
	switch {
	case lx0_1 != lx0_1 || ly0_1 != ly0_1:
		o0_1 = 0x7fc00000
	case lx0_1 == ly0_1:
		o0_1 = math.Float32bits(lx0_1) & math.Float32bits(ly0_1)
	case lx0_1 > ly0_1:
		o0_1 = math.Float32bits(lx0_1)
	default:
		o0_1 = math.Float32bits(ly0_1)
	}
	r0 := uint64(o0_0) | uint64(o0_1)<<32
	lx1_0 := math.Float32frombits(uint32(x1))
	ly1_0 := math.Float32frombits(uint32(y1))
	var o1_0 uint32
	switch {
	case lx1_0 != lx1_0 || ly1_0 != ly1_0:
		o1_0 = 0x7fc00000
	case lx1_0 == ly1_0:
		o1_0 = math.Float32bits(lx1_0) & math.Float32bits(ly1_0)
	case lx1_0 > ly1_0:
		o1_0 = math.Float32bits(lx1_0)
	default:
		o1_0 = math.Float32bits(ly1_0)
	}
	lx1_1 := math.Float32frombits(uint32(x1 >> 32))
	ly1_1 := math.Float32frombits(uint32(y1 >> 32))
	var o1_1 uint32
	switch {
	case lx1_1 != lx1_1 || ly1_1 != ly1_1:
		o1_1 = 0x7fc00000
	case lx1_1 == ly1_1:
		o1_1 = math.Float32bits(lx1_1) & math.Float32bits(ly1_1)
	case lx1_1 > ly1_1:
		o1_1 = math.Float32bits(lx1_1)
	default:
		o1_1 = math.Float32bits(ly1_1)
	}
	r1 := uint64(o1_0) | uint64(o1_1)<<32
	return v128{r0, r1}
}

//go:nosplit
func f32x4_pmin(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := math.Float32frombits(uint32(x0))
	ly0_0 := math.Float32frombits(uint32(y0))
	var o0_0 uint32
	if ly0_0 < lx0_0 {
		o0_0 = math.Float32bits(ly0_0)
	} else {
		o0_0 = math.Float32bits(lx0_0)
	}
	lx0_1 := math.Float32frombits(uint32(x0 >> 32))
	ly0_1 := math.Float32frombits(uint32(y0 >> 32))
	var o0_1 uint32
	if ly0_1 < lx0_1 {
		o0_1 = math.Float32bits(ly0_1)
	} else {
		o0_1 = math.Float32bits(lx0_1)
	}
	r0 := uint64(o0_0) | uint64(o0_1)<<32
	lx1_0 := math.Float32frombits(uint32(x1))
	ly1_0 := math.Float32frombits(uint32(y1))
	var o1_0 uint32
	if ly1_0 < lx1_0 {
		o1_0 = math.Float32bits(ly1_0)
	} else {
		o1_0 = math.Float32bits(lx1_0)
	}
	lx1_1 := math.Float32frombits(uint32(x1 >> 32))
	ly1_1 := math.Float32frombits(uint32(y1 >> 32))
	var o1_1 uint32
	if ly1_1 < lx1_1 {
		o1_1 = math.Float32bits(ly1_1)
	} else {
		o1_1 = math.Float32bits(lx1_1)
	}
	r1 := uint64(o1_0) | uint64(o1_1)<<32
	return v128{r0, r1}
}

//go:nosplit
func f32x4_pmax(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := math.Float32frombits(uint32(x0))
	ly0_0 := math.Float32frombits(uint32(y0))
	var o0_0 uint32
	if lx0_0 < ly0_0 {
		o0_0 = math.Float32bits(ly0_0)
	} else {
		o0_0 = math.Float32bits(lx0_0)
	}
	lx0_1 := math.Float32frombits(uint32(x0 >> 32))
	ly0_1 := math.Float32frombits(uint32(y0 >> 32))
	var o0_1 uint32
	if lx0_1 < ly0_1 {
		o0_1 = math.Float32bits(ly0_1)
	} else {
		o0_1 = math.Float32bits(lx0_1)
	}
	r0 := uint64(o0_0) | uint64(o0_1)<<32
	lx1_0 := math.Float32frombits(uint32(x1))
	ly1_0 := math.Float32frombits(uint32(y1))
	var o1_0 uint32
	if lx1_0 < ly1_0 {
		o1_0 = math.Float32bits(ly1_0)
	} else {
		o1_0 = math.Float32bits(lx1_0)
	}
	lx1_1 := math.Float32frombits(uint32(x1 >> 32))
	ly1_1 := math.Float32frombits(uint32(y1 >> 32))
	var o1_1 uint32
	if lx1_1 < ly1_1 {
		o1_1 = math.Float32bits(ly1_1)
	} else {
		o1_1 = math.Float32bits(lx1_1)
	}
	r1 := uint64(o1_0) | uint64(o1_1)<<32
	return v128{r0, r1}
}

//go:nosplit
func f32x4_abs(a v128) v128 {
	x0, x1 := a.lo, a.hi
	t1_0 := x0 &^ 0x8000000080000000
	t1_1 := x1 &^ 0x8000000080000000
	return v128{t1_0, t1_1}
}

//go:nosplit
func f32x4_neg(a v128) v128 {
	x0, x1 := a.lo, a.hi
	t1_0 := x0 ^ 0x8000000080000000
	t1_1 := x1 ^ 0x8000000080000000
	return v128{t1_0, t1_1}
}

//go:nosplit
func f32x4_sqrt(a v128) v128 {
	x0, x1 := a.lo, a.hi
	lx0_0 := math.Float32frombits(uint32(x0))
	var o0_0 uint32
	o0_0 = math.Float32bits(float32(math.Sqrt(float64(lx0_0))))
	lx0_1 := math.Float32frombits(uint32(x0 >> 32))
	var o0_1 uint32
	o0_1 = math.Float32bits(float32(math.Sqrt(float64(lx0_1))))
	r0 := uint64(o0_0) | uint64(o0_1)<<32
	lx1_0 := math.Float32frombits(uint32(x1))
	var o1_0 uint32
	o1_0 = math.Float32bits(float32(math.Sqrt(float64(lx1_0))))
	lx1_1 := math.Float32frombits(uint32(x1 >> 32))
	var o1_1 uint32
	o1_1 = math.Float32bits(float32(math.Sqrt(float64(lx1_1))))
	r1 := uint64(o1_0) | uint64(o1_1)<<32
	return v128{r0, r1}
}

//go:nosplit
func f32x4_ceil(a v128) v128 {
	x0, x1 := a.lo, a.hi
	lx0_0 := math.Float32frombits(uint32(x0))
	var o0_0 uint32
	o0_0 = math.Float32bits(float32(math.Ceil(float64(lx0_0))))
	lx0_1 := math.Float32frombits(uint32(x0 >> 32))
	var o0_1 uint32
	o0_1 = math.Float32bits(float32(math.Ceil(float64(lx0_1))))
	r0 := uint64(o0_0) | uint64(o0_1)<<32
	lx1_0 := math.Float32frombits(uint32(x1))
	var o1_0 uint32
	o1_0 = math.Float32bits(float32(math.Ceil(float64(lx1_0))))
	lx1_1 := math.Float32frombits(uint32(x1 >> 32))
	var o1_1 uint32
	o1_1 = math.Float32bits(float32(math.Ceil(float64(lx1_1))))
	r1 := uint64(o1_0) | uint64(o1_1)<<32
	return v128{r0, r1}
}

//go:nosplit
func f32x4_floor(a v128) v128 {
	x0, x1 := a.lo, a.hi
	lx0_0 := math.Float32frombits(uint32(x0))
	var o0_0 uint32
	o0_0 = math.Float32bits(float32(math.Floor(float64(lx0_0))))
	lx0_1 := math.Float32frombits(uint32(x0 >> 32))
	var o0_1 uint32
	o0_1 = math.Float32bits(float32(math.Floor(float64(lx0_1))))
	r0 := uint64(o0_0) | uint64(o0_1)<<32
	lx1_0 := math.Float32frombits(uint32(x1))
	var o1_0 uint32
	o1_0 = math.Float32bits(float32(math.Floor(float64(lx1_0))))
	lx1_1 := math.Float32frombits(uint32(x1 >> 32))
	var o1_1 uint32
	o1_1 = math.Float32bits(float32(math.Floor(float64(lx1_1))))
	r1 := uint64(o1_0) | uint64(o1_1)<<32
	return v128{r0, r1}
}

//go:nosplit
func f32x4_trunc(a v128) v128 {
	x0, x1 := a.lo, a.hi
	lx0_0 := math.Float32frombits(uint32(x0))
	var o0_0 uint32
	o0_0 = math.Float32bits(float32(math.Trunc(float64(lx0_0))))
	lx0_1 := math.Float32frombits(uint32(x0 >> 32))
	var o0_1 uint32
	o0_1 = math.Float32bits(float32(math.Trunc(float64(lx0_1))))
	r0 := uint64(o0_0) | uint64(o0_1)<<32
	lx1_0 := math.Float32frombits(uint32(x1))
	var o1_0 uint32
	o1_0 = math.Float32bits(float32(math.Trunc(float64(lx1_0))))
	lx1_1 := math.Float32frombits(uint32(x1 >> 32))
	var o1_1 uint32
	o1_1 = math.Float32bits(float32(math.Trunc(float64(lx1_1))))
	r1 := uint64(o1_0) | uint64(o1_1)<<32
	return v128{r0, r1}
}

//go:nosplit
func f32x4_nearest(a v128) v128 {
	x0, x1 := a.lo, a.hi
	lx0_0 := math.Float32frombits(uint32(x0))
	var o0_0 uint32
	o0_0 = math.Float32bits(float32(math.RoundToEven(float64(lx0_0))))
	lx0_1 := math.Float32frombits(uint32(x0 >> 32))
	var o0_1 uint32
	o0_1 = math.Float32bits(float32(math.RoundToEven(float64(lx0_1))))
	r0 := uint64(o0_0) | uint64(o0_1)<<32
	lx1_0 := math.Float32frombits(uint32(x1))
	var o1_0 uint32
	o1_0 = math.Float32bits(float32(math.RoundToEven(float64(lx1_0))))
	lx1_1 := math.Float32frombits(uint32(x1 >> 32))
	var o1_1 uint32
	o1_1 = math.Float32bits(float32(math.RoundToEven(float64(lx1_1))))
	r1 := uint64(o1_0) | uint64(o1_1)<<32
	return v128{r0, r1}
}

//go:nosplit
func f64x2_eq(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := math.Float64frombits(x0)
	ly0_0 := math.Float64frombits(y0)
	var o0_0 uint64
	if lx0_0 == ly0_0 {
		o0_0 = 0xffffffffffffffff
	}
	r0 := uint64(o0_0)
	lx1_0 := math.Float64frombits(x1)
	ly1_0 := math.Float64frombits(y1)
	var o1_0 uint64
	if lx1_0 == ly1_0 {
		o1_0 = 0xffffffffffffffff
	}
	r1 := uint64(o1_0)
	return v128{r0, r1}
}

//go:nosplit
func f64x2_ne(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := math.Float64frombits(x0)
	ly0_0 := math.Float64frombits(y0)
	var o0_0 uint64
	if lx0_0 != ly0_0 {
		o0_0 = 0xffffffffffffffff
	}
	r0 := uint64(o0_0)
	lx1_0 := math.Float64frombits(x1)
	ly1_0 := math.Float64frombits(y1)
	var o1_0 uint64
	if lx1_0 != ly1_0 {
		o1_0 = 0xffffffffffffffff
	}
	r1 := uint64(o1_0)
	return v128{r0, r1}
}

//go:nosplit
func f64x2_lt(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := math.Float64frombits(x0)
	ly0_0 := math.Float64frombits(y0)
	var o0_0 uint64
	if lx0_0 < ly0_0 {
		o0_0 = 0xffffffffffffffff
	}
	r0 := uint64(o0_0)
	lx1_0 := math.Float64frombits(x1)
	ly1_0 := math.Float64frombits(y1)
	var o1_0 uint64
	if lx1_0 < ly1_0 {
		o1_0 = 0xffffffffffffffff
	}
	r1 := uint64(o1_0)
	return v128{r0, r1}
}

//go:nosplit
func f64x2_gt(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := math.Float64frombits(x0)
	ly0_0 := math.Float64frombits(y0)
	var o0_0 uint64
	if lx0_0 > ly0_0 {
		o0_0 = 0xffffffffffffffff
	}
	r0 := uint64(o0_0)
	lx1_0 := math.Float64frombits(x1)
	ly1_0 := math.Float64frombits(y1)
	var o1_0 uint64
	if lx1_0 > ly1_0 {
		o1_0 = 0xffffffffffffffff
	}
	r1 := uint64(o1_0)
	return v128{r0, r1}
}

//go:nosplit
func f64x2_le(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := math.Float64frombits(x0)
	ly0_0 := math.Float64frombits(y0)
	var o0_0 uint64
	if lx0_0 <= ly0_0 {
		o0_0 = 0xffffffffffffffff
	}
	r0 := uint64(o0_0)
	lx1_0 := math.Float64frombits(x1)
	ly1_0 := math.Float64frombits(y1)
	var o1_0 uint64
	if lx1_0 <= ly1_0 {
		o1_0 = 0xffffffffffffffff
	}
	r1 := uint64(o1_0)
	return v128{r0, r1}
}

//go:nosplit
func f64x2_ge(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := math.Float64frombits(x0)
	ly0_0 := math.Float64frombits(y0)
	var o0_0 uint64
	if lx0_0 >= ly0_0 {
		o0_0 = 0xffffffffffffffff
	}
	r0 := uint64(o0_0)
	lx1_0 := math.Float64frombits(x1)
	ly1_0 := math.Float64frombits(y1)
	var o1_0 uint64
	if lx1_0 >= ly1_0 {
		o1_0 = 0xffffffffffffffff
	}
	r1 := uint64(o1_0)
	return v128{r0, r1}
}

//go:nosplit
func f64x2_min(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := math.Float64frombits(x0)
	ly0_0 := math.Float64frombits(y0)
	var o0_0 uint64
	switch {
	case lx0_0 != lx0_0 || ly0_0 != ly0_0:
		o0_0 = 0x7ff8000000000000
	case lx0_0 == ly0_0:
		o0_0 = math.Float64bits(lx0_0) | math.Float64bits(ly0_0)
	case lx0_0 < ly0_0:
		o0_0 = math.Float64bits(lx0_0)
	default:
		o0_0 = math.Float64bits(ly0_0)
	}
	r0 := uint64(o0_0)
	lx1_0 := math.Float64frombits(x1)
	ly1_0 := math.Float64frombits(y1)
	var o1_0 uint64
	switch {
	case lx1_0 != lx1_0 || ly1_0 != ly1_0:
		o1_0 = 0x7ff8000000000000
	case lx1_0 == ly1_0:
		o1_0 = math.Float64bits(lx1_0) | math.Float64bits(ly1_0)
	case lx1_0 < ly1_0:
		o1_0 = math.Float64bits(lx1_0)
	default:
		o1_0 = math.Float64bits(ly1_0)
	}
	r1 := uint64(o1_0)
	return v128{r0, r1}
}

//go:nosplit
func f64x2_max(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := math.Float64frombits(x0)
	ly0_0 := math.Float64frombits(y0)
	var o0_0 uint64
	switch {
	case lx0_0 != lx0_0 || ly0_0 != ly0_0:
		o0_0 = 0x7ff8000000000000
	case lx0_0 == ly0_0:
		o0_0 = math.Float64bits(lx0_0) & math.Float64bits(ly0_0)
	case lx0_0 > ly0_0:
		o0_0 = math.Float64bits(lx0_0)
	default:
		o0_0 = math.Float64bits(ly0_0)
	}
	r0 := uint64(o0_0)
	lx1_0 := math.Float64frombits(x1)
	ly1_0 := math.Float64frombits(y1)
	var o1_0 uint64
	switch {
	case lx1_0 != lx1_0 || ly1_0 != ly1_0:
		o1_0 = 0x7ff8000000000000
	case lx1_0 == ly1_0:
		o1_0 = math.Float64bits(lx1_0) & math.Float64bits(ly1_0)
	case lx1_0 > ly1_0:
		o1_0 = math.Float64bits(lx1_0)
	default:
		o1_0 = math.Float64bits(ly1_0)
	}
	r1 := uint64(o1_0)
	return v128{r0, r1}
}

//go:nosplit
func f64x2_pmin(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := math.Float64frombits(x0)
	ly0_0 := math.Float64frombits(y0)
	var o0_0 uint64
	if ly0_0 < lx0_0 {
		o0_0 = math.Float64bits(ly0_0)
	} else {
		o0_0 = math.Float64bits(lx0_0)
	}
	r0 := uint64(o0_0)
	lx1_0 := math.Float64frombits(x1)
	ly1_0 := math.Float64frombits(y1)
	var o1_0 uint64
	if ly1_0 < lx1_0 {
		o1_0 = math.Float64bits(ly1_0)
	} else {
		o1_0 = math.Float64bits(lx1_0)
	}
	r1 := uint64(o1_0)
	return v128{r0, r1}
}

//go:nosplit
func f64x2_pmax(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := math.Float64frombits(x0)
	ly0_0 := math.Float64frombits(y0)
	var o0_0 uint64
	if lx0_0 < ly0_0 {
		o0_0 = math.Float64bits(ly0_0)
	} else {
		o0_0 = math.Float64bits(lx0_0)
	}
	r0 := uint64(o0_0)
	lx1_0 := math.Float64frombits(x1)
	ly1_0 := math.Float64frombits(y1)
	var o1_0 uint64
	if lx1_0 < ly1_0 {
		o1_0 = math.Float64bits(ly1_0)
	} else {
		o1_0 = math.Float64bits(lx1_0)
	}
	r1 := uint64(o1_0)
	return v128{r0, r1}
}

//go:nosplit
func f64x2_abs(a v128) v128 {
	x0, x1 := a.lo, a.hi
	t1_0 := x0 &^ 0x8000000000000000
	t1_1 := x1 &^ 0x8000000000000000
	return v128{t1_0, t1_1}
}

//go:nosplit
func f64x2_neg(a v128) v128 {
	x0, x1 := a.lo, a.hi
	t1_0 := x0 ^ 0x8000000000000000
	t1_1 := x1 ^ 0x8000000000000000
	return v128{t1_0, t1_1}
}

//go:nosplit
func f64x2_sqrt(a v128) v128 {
	x0, x1 := a.lo, a.hi
	lx0_0 := math.Float64frombits(x0)
	var o0_0 uint64
	o0_0 = math.Float64bits(math.Sqrt(lx0_0))
	r0 := uint64(o0_0)
	lx1_0 := math.Float64frombits(x1)
	var o1_0 uint64
	o1_0 = math.Float64bits(math.Sqrt(lx1_0))
	r1 := uint64(o1_0)
	return v128{r0, r1}
}

//go:nosplit
func f64x2_ceil(a v128) v128 {
	x0, x1 := a.lo, a.hi
	lx0_0 := math.Float64frombits(x0)
	var o0_0 uint64
	o0_0 = math.Float64bits(math.Ceil(lx0_0))
	r0 := uint64(o0_0)
	lx1_0 := math.Float64frombits(x1)
	var o1_0 uint64
	o1_0 = math.Float64bits(math.Ceil(lx1_0))
	r1 := uint64(o1_0)
	return v128{r0, r1}
}

//go:nosplit
func f64x2_floor(a v128) v128 {
	x0, x1 := a.lo, a.hi
	lx0_0 := math.Float64frombits(x0)
	var o0_0 uint64
	o0_0 = math.Float64bits(math.Floor(lx0_0))
	r0 := uint64(o0_0)
	lx1_0 := math.Float64frombits(x1)
	var o1_0 uint64
	o1_0 = math.Float64bits(math.Floor(lx1_0))
	r1 := uint64(o1_0)
	return v128{r0, r1}
}

//go:nosplit
func f64x2_trunc(a v128) v128 {
	x0, x1 := a.lo, a.hi
	lx0_0 := math.Float64frombits(x0)
	var o0_0 uint64
	o0_0 = math.Float64bits(math.Trunc(lx0_0))
	r0 := uint64(o0_0)
	lx1_0 := math.Float64frombits(x1)
	var o1_0 uint64
	o1_0 = math.Float64bits(math.Trunc(lx1_0))
	r1 := uint64(o1_0)
	return v128{r0, r1}
}

//go:nosplit
func f64x2_nearest(a v128) v128 {
	x0, x1 := a.lo, a.hi
	lx0_0 := math.Float64frombits(x0)
	var o0_0 uint64
	o0_0 = math.Float64bits(math.RoundToEven(lx0_0))
	r0 := uint64(o0_0)
	lx1_0 := math.Float64frombits(x1)
	var o1_0 uint64
	o1_0 = math.Float64bits(math.RoundToEven(lx1_0))
	r1 := uint64(o1_0)
	return v128{r0, r1}
}

//go:nosplit
func f32x4_convert_i32x4_s(a v128) v128 {
	x0, x1 := a.lo, a.hi
	lx0_0 := int32(uint32(x0))
	var o0_0 uint32
	o0_0 = math.Float32bits(float32(lx0_0))
	lx0_1 := int32(uint32(x0 >> 32))
	var o0_1 uint32
	o0_1 = math.Float32bits(float32(lx0_1))
	r0 := uint64(o0_0) | uint64(o0_1)<<32
	lx1_0 := int32(uint32(x1))
	var o1_0 uint32
	o1_0 = math.Float32bits(float32(lx1_0))
	lx1_1 := int32(uint32(x1 >> 32))
	var o1_1 uint32
	o1_1 = math.Float32bits(float32(lx1_1))
	r1 := uint64(o1_0) | uint64(o1_1)<<32
	return v128{r0, r1}
}

//go:nosplit
func f32x4_convert_i32x4_u(a v128) v128 {
	x0, x1 := a.lo, a.hi
	lx0_0 := uint32(x0)
	var o0_0 uint32
	o0_0 = math.Float32bits(float32(lx0_0))
	lx0_1 := uint32(x0 >> 32)
	var o0_1 uint32
	o0_1 = math.Float32bits(float32(lx0_1))
	r0 := uint64(o0_0) | uint64(o0_1)<<32
	lx1_0 := uint32(x1)
	var o1_0 uint32
	o1_0 = math.Float32bits(float32(lx1_0))
	lx1_1 := uint32(x1 >> 32)
	var o1_1 uint32
	o1_1 = math.Float32bits(float32(lx1_1))
	r1 := uint64(o1_0) | uint64(o1_1)<<32
	return v128{r0, r1}
}

//go:nosplit
func i32x4_trunc_sat_f32x4_s(a v128) v128 {
	x0, x1 := a.lo, a.hi
	lx0_0 := math.Float32frombits(uint32(x0))
	var o0_0 uint32
	switch {
	case lx0_0 <= math.MinInt32:
		o0_0 = 0x80000000
	case lx0_0 >= math.MaxInt32:
		o0_0 = 0x7fffffff
	case lx0_0 != lx0_0:
		o0_0 = 0
	default:
		o0_0 = uint32(int32(lx0_0))
	}
	lx0_1 := math.Float32frombits(uint32(x0 >> 32))
	var o0_1 uint32
	switch {
	case lx0_1 <= math.MinInt32:
		o0_1 = 0x80000000
	case lx0_1 >= math.MaxInt32:
		o0_1 = 0x7fffffff
	case lx0_1 != lx0_1:
		o0_1 = 0
	default:
		o0_1 = uint32(int32(lx0_1))
	}
	r0 := uint64(o0_0) | uint64(o0_1)<<32
	lx1_0 := math.Float32frombits(uint32(x1))
	var o1_0 uint32
	switch {
	case lx1_0 <= math.MinInt32:
		o1_0 = 0x80000000
	case lx1_0 >= math.MaxInt32:
		o1_0 = 0x7fffffff
	case lx1_0 != lx1_0:
		o1_0 = 0
	default:
		o1_0 = uint32(int32(lx1_0))
	}
	lx1_1 := math.Float32frombits(uint32(x1 >> 32))
	var o1_1 uint32
	switch {
	case lx1_1 <= math.MinInt32:
		o1_1 = 0x80000000
	case lx1_1 >= math.MaxInt32:
		o1_1 = 0x7fffffff
	case lx1_1 != lx1_1:
		o1_1 = 0
	default:
		o1_1 = uint32(int32(lx1_1))
	}
	r1 := uint64(o1_0) | uint64(o1_1)<<32
	return v128{r0, r1}
}

//go:nosplit
func i32x4_trunc_sat_f32x4_u(a v128) v128 {
	x0, x1 := a.lo, a.hi
	lx0_0 := math.Float32frombits(uint32(x0))
	var o0_0 uint32
	switch {
	case lx0_0 <= 0 || lx0_0 != lx0_0:
		o0_0 = 0
	case lx0_0 >= math.MaxUint32:
		o0_0 = 0xffffffff
	default:
		o0_0 = uint32(lx0_0)
	}
	lx0_1 := math.Float32frombits(uint32(x0 >> 32))
	var o0_1 uint32
	switch {
	case lx0_1 <= 0 || lx0_1 != lx0_1:
		o0_1 = 0
	case lx0_1 >= math.MaxUint32:
		o0_1 = 0xffffffff
	default:
		o0_1 = uint32(lx0_1)
	}
	r0 := uint64(o0_0) | uint64(o0_1)<<32
	lx1_0 := math.Float32frombits(uint32(x1))
	var o1_0 uint32
	switch {
	case lx1_0 <= 0 || lx1_0 != lx1_0:
		o1_0 = 0
	case lx1_0 >= math.MaxUint32:
		o1_0 = 0xffffffff
	default:
		o1_0 = uint32(lx1_0)
	}
	lx1_1 := math.Float32frombits(uint32(x1 >> 32))
	var o1_1 uint32
	switch {
	case lx1_1 <= 0 || lx1_1 != lx1_1:
		o1_1 = 0
	case lx1_1 >= math.MaxUint32:
		o1_1 = 0xffffffff
	default:
		o1_1 = uint32(lx1_1)
	}
	r1 := uint64(o1_0) | uint64(o1_1)<<32
	return v128{r0, r1}
}

//go:nosplit
func i32x4_trunc_sat_f64x2_s_zero(a v128) v128 {
	x0, x1 := a.lo, a.hi
	f0 := math.Float64frombits(x0)
	var o0 uint32
	switch {
	case f0 <= math.MinInt32:
		o0 = 0x80000000
	case f0 >= math.MaxInt32:
		o0 = 0x7fffffff
	case f0 != f0:
		o0 = 0
	default:
		o0 = uint32(int32(f0))
	}
	f1 := math.Float64frombits(x1)
	var o1 uint32
	switch {
	case f1 <= math.MinInt32:
		o1 = 0x80000000
	case f1 >= math.MaxInt32:
		o1 = 0x7fffffff
	case f1 != f1:
		o1 = 0
	default:
		o1 = uint32(int32(f1))
	}
	r0 := uint64(o0) | uint64(o1)<<32
	return v128{r0, 0}
}

//go:nosplit
func i32x4_trunc_sat_f64x2_u_zero(a v128) v128 {
	x0, x1 := a.lo, a.hi
	f0 := math.Float64frombits(x0)
	var o0 uint32
	switch {
	case f0 <= 0 || f0 != f0:
		o0 = 0
	case f0 >= math.MaxUint32:
		o0 = 0xffffffff
	default:
		o0 = uint32(f0)
	}
	f1 := math.Float64frombits(x1)
	var o1 uint32
	switch {
	case f1 <= 0 || f1 != f1:
		o1 = 0
	case f1 >= math.MaxUint32:
		o1 = 0xffffffff
	default:
		o1 = uint32(f1)
	}
	r0 := uint64(o0) | uint64(o1)<<32
	return v128{r0, 0}
}

//go:nosplit
func f32x4_demote_f64x2_zero(a v128) v128 {
	x0, x1 := a.lo, a.hi
	r0 := uint64(math.Float32bits(float32(math.Float64frombits(x0)))) | uint64(math.Float32bits(float32(math.Float64frombits(x1))))<<32
	return v128{r0, 0}
}

//go:nosplit
func f64x2_promote_low_f32x4(a v128) v128 {
	x0 := a.lo
	r0 := math.Float64bits(float64(math.Float32frombits(uint32(x0))))
	r1 := math.Float64bits(float64(math.Float32frombits(uint32(x0 >> 32))))
	return v128{r0, r1}
}

//go:nosplit
func f64x2_convert_low_i32x4_s(a v128) v128 {
	x0 := a.lo
	r0 := math.Float64bits(float64(int32(uint32(x0))))
	r1 := math.Float64bits(float64(int32(uint32(x0 >> 32))))
	return v128{r0, r1}
}

//go:nosplit
func f64x2_convert_low_i32x4_u(a v128) v128 {
	x0 := a.lo
	r0 := math.Float64bits(float64(uint32(x0)))
	r1 := math.Float64bits(float64(uint32(x0 >> 32)))
	return v128{r0, r1}
}

//go:nosplit
func i16x8_mul(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := uint16(x0)
	ly0_0 := uint16(y0)
	var o0_0 uint16
	o0_0 = lx0_0 * ly0_0
	lx0_1 := uint16(x0 >> 16)
	ly0_1 := uint16(y0 >> 16)
	var o0_1 uint16
	o0_1 = lx0_1 * ly0_1
	lx0_2 := uint16(x0 >> 32)
	ly0_2 := uint16(y0 >> 32)
	var o0_2 uint16
	o0_2 = lx0_2 * ly0_2
	lx0_3 := uint16(x0 >> 48)
	ly0_3 := uint16(y0 >> 48)
	var o0_3 uint16
	o0_3 = lx0_3 * ly0_3
	r0 := uint64(o0_0) | uint64(o0_1)<<16 | uint64(o0_2)<<32 | uint64(o0_3)<<48
	lx1_0 := uint16(x1)
	ly1_0 := uint16(y1)
	var o1_0 uint16
	o1_0 = lx1_0 * ly1_0
	lx1_1 := uint16(x1 >> 16)
	ly1_1 := uint16(y1 >> 16)
	var o1_1 uint16
	o1_1 = lx1_1 * ly1_1
	lx1_2 := uint16(x1 >> 32)
	ly1_2 := uint16(y1 >> 32)
	var o1_2 uint16
	o1_2 = lx1_2 * ly1_2
	lx1_3 := uint16(x1 >> 48)
	ly1_3 := uint16(y1 >> 48)
	var o1_3 uint16
	o1_3 = lx1_3 * ly1_3
	r1 := uint64(o1_0) | uint64(o1_1)<<16 | uint64(o1_2)<<32 | uint64(o1_3)<<48
	return v128{r0, r1}
}

//go:nosplit
func i32x4_mul(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := uint32(x0)
	ly0_0 := uint32(y0)
	var o0_0 uint32
	o0_0 = lx0_0 * ly0_0
	lx0_1 := uint32(x0 >> 32)
	ly0_1 := uint32(y0 >> 32)
	var o0_1 uint32
	o0_1 = lx0_1 * ly0_1
	r0 := uint64(o0_0) | uint64(o0_1)<<32
	lx1_0 := uint32(x1)
	ly1_0 := uint32(y1)
	var o1_0 uint32
	o1_0 = lx1_0 * ly1_0
	lx1_1 := uint32(x1 >> 32)
	ly1_1 := uint32(y1 >> 32)
	var o1_1 uint32
	o1_1 = lx1_1 * ly1_1
	r1 := uint64(o1_0) | uint64(o1_1)<<32
	return v128{r0, r1}
}

//go:nosplit
func i16x8_q15mulr_sat_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	lx0_0 := int16(uint16(x0))
	ly0_0 := int16(uint16(y0))
	var o0_0 uint16
	p0_0 := (int32(lx0_0)*int32(ly0_0) + 0x4000) >> 15
	if p0_0 > math.MaxInt16 {
		p0_0 = math.MaxInt16
	}
	o0_0 = uint16(p0_0)
	lx0_1 := int16(uint16(x0 >> 16))
	ly0_1 := int16(uint16(y0 >> 16))
	var o0_1 uint16
	p0_1 := (int32(lx0_1)*int32(ly0_1) + 0x4000) >> 15
	if p0_1 > math.MaxInt16 {
		p0_1 = math.MaxInt16
	}
	o0_1 = uint16(p0_1)
	lx0_2 := int16(uint16(x0 >> 32))
	ly0_2 := int16(uint16(y0 >> 32))
	var o0_2 uint16
	p0_2 := (int32(lx0_2)*int32(ly0_2) + 0x4000) >> 15
	if p0_2 > math.MaxInt16 {
		p0_2 = math.MaxInt16
	}
	o0_2 = uint16(p0_2)
	lx0_3 := int16(uint16(x0 >> 48))
	ly0_3 := int16(uint16(y0 >> 48))
	var o0_3 uint16
	p0_3 := (int32(lx0_3)*int32(ly0_3) + 0x4000) >> 15
	if p0_3 > math.MaxInt16 {
		p0_3 = math.MaxInt16
	}
	o0_3 = uint16(p0_3)
	r0 := uint64(o0_0) | uint64(o0_1)<<16 | uint64(o0_2)<<32 | uint64(o0_3)<<48
	lx1_0 := int16(uint16(x1))
	ly1_0 := int16(uint16(y1))
	var o1_0 uint16
	p1_0 := (int32(lx1_0)*int32(ly1_0) + 0x4000) >> 15
	if p1_0 > math.MaxInt16 {
		p1_0 = math.MaxInt16
	}
	o1_0 = uint16(p1_0)
	lx1_1 := int16(uint16(x1 >> 16))
	ly1_1 := int16(uint16(y1 >> 16))
	var o1_1 uint16
	p1_1 := (int32(lx1_1)*int32(ly1_1) + 0x4000) >> 15
	if p1_1 > math.MaxInt16 {
		p1_1 = math.MaxInt16
	}
	o1_1 = uint16(p1_1)
	lx1_2 := int16(uint16(x1 >> 32))
	ly1_2 := int16(uint16(y1 >> 32))
	var o1_2 uint16
	p1_2 := (int32(lx1_2)*int32(ly1_2) + 0x4000) >> 15
	if p1_2 > math.MaxInt16 {
		p1_2 = math.MaxInt16
	}
	o1_2 = uint16(p1_2)
	lx1_3 := int16(uint16(x1 >> 48))
	ly1_3 := int16(uint16(y1 >> 48))
	var o1_3 uint16
	p1_3 := (int32(lx1_3)*int32(ly1_3) + 0x4000) >> 15
	if p1_3 > math.MaxInt16 {
		p1_3 = math.MaxInt16
	}
	o1_3 = uint16(p1_3)
	r1 := uint64(o1_0) | uint64(o1_1)<<16 | uint64(o1_2)<<32 | uint64(o1_3)<<48
	return v128{r0, r1}
}

//go:nosplit
func i8x16_add_sat_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ((x0 & 0x7f7f7f7f7f7f7f7f) + (y0 & 0x7f7f7f7f7f7f7f7f)) ^ ((x0 ^ y0) & 0x8080808080808080)
	t2_0 := (^(x0 ^ y0) & (x0 ^ t1_0)) & 0x8080808080808080
	t3_0 := (t2_0 >> 7) * 0x00000000000000ff
	t4_0 := x0 & 0x8080808080808080
	t5_0 := (t4_0 >> 7) * 0x00000000000000ff
	t6_0 := t5_0 ^ 0x7f7f7f7f7f7f7f7f
	t7_0 := (t6_0 & t3_0) | (t1_0 &^ t3_0)
	t1_1 := ((x1 & 0x7f7f7f7f7f7f7f7f) + (y1 & 0x7f7f7f7f7f7f7f7f)) ^ ((x1 ^ y1) & 0x8080808080808080)
	t2_1 := (^(x1 ^ y1) & (x1 ^ t1_1)) & 0x8080808080808080
	t3_1 := (t2_1 >> 7) * 0x00000000000000ff
	t4_1 := x1 & 0x8080808080808080
	t5_1 := (t4_1 >> 7) * 0x00000000000000ff
	t6_1 := t5_1 ^ 0x7f7f7f7f7f7f7f7f
	t7_1 := (t6_1 & t3_1) | (t1_1 &^ t3_1)
	return v128{t7_0, t7_1}
}

//go:nosplit
func i8x16_sub_sat_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ((x0 | 0x8080808080808080) - (y0 & 0x7f7f7f7f7f7f7f7f)) ^ ((x0 ^ ^y0) & 0x8080808080808080)
	t2_0 := ((x0 ^ y0) & (x0 ^ t1_0)) & 0x8080808080808080
	t3_0 := (t2_0 >> 7) * 0x00000000000000ff
	t4_0 := x0 & 0x8080808080808080
	t5_0 := (t4_0 >> 7) * 0x00000000000000ff
	t6_0 := t5_0 ^ 0x7f7f7f7f7f7f7f7f
	t7_0 := (t6_0 & t3_0) | (t1_0 &^ t3_0)
	t1_1 := ((x1 | 0x8080808080808080) - (y1 & 0x7f7f7f7f7f7f7f7f)) ^ ((x1 ^ ^y1) & 0x8080808080808080)
	t2_1 := ((x1 ^ y1) & (x1 ^ t1_1)) & 0x8080808080808080
	t3_1 := (t2_1 >> 7) * 0x00000000000000ff
	t4_1 := x1 & 0x8080808080808080
	t5_1 := (t4_1 >> 7) * 0x00000000000000ff
	t6_1 := t5_1 ^ 0x7f7f7f7f7f7f7f7f
	t7_1 := (t6_1 & t3_1) | (t1_1 &^ t3_1)
	return v128{t7_0, t7_1}
}

//go:nosplit
func i16x8_add_sat_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ((x0 & 0x7fff7fff7fff7fff) + (y0 & 0x7fff7fff7fff7fff)) ^ ((x0 ^ y0) & 0x8000800080008000)
	t2_0 := (^(x0 ^ y0) & (x0 ^ t1_0)) & 0x8000800080008000
	t3_0 := (t2_0 >> 15) * 0x000000000000ffff
	t4_0 := x0 & 0x8000800080008000
	t5_0 := (t4_0 >> 15) * 0x000000000000ffff
	t6_0 := t5_0 ^ 0x7fff7fff7fff7fff
	t7_0 := (t6_0 & t3_0) | (t1_0 &^ t3_0)
	t1_1 := ((x1 & 0x7fff7fff7fff7fff) + (y1 & 0x7fff7fff7fff7fff)) ^ ((x1 ^ y1) & 0x8000800080008000)
	t2_1 := (^(x1 ^ y1) & (x1 ^ t1_1)) & 0x8000800080008000
	t3_1 := (t2_1 >> 15) * 0x000000000000ffff
	t4_1 := x1 & 0x8000800080008000
	t5_1 := (t4_1 >> 15) * 0x000000000000ffff
	t6_1 := t5_1 ^ 0x7fff7fff7fff7fff
	t7_1 := (t6_1 & t3_1) | (t1_1 &^ t3_1)
	return v128{t7_0, t7_1}
}

//go:nosplit
func i16x8_sub_sat_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	t1_0 := ((x0 | 0x8000800080008000) - (y0 & 0x7fff7fff7fff7fff)) ^ ((x0 ^ ^y0) & 0x8000800080008000)
	t2_0 := ((x0 ^ y0) & (x0 ^ t1_0)) & 0x8000800080008000
	t3_0 := (t2_0 >> 15) * 0x000000000000ffff
	t4_0 := x0 & 0x8000800080008000
	t5_0 := (t4_0 >> 15) * 0x000000000000ffff
	t6_0 := t5_0 ^ 0x7fff7fff7fff7fff
	t7_0 := (t6_0 & t3_0) | (t1_0 &^ t3_0)
	t1_1 := ((x1 | 0x8000800080008000) - (y1 & 0x7fff7fff7fff7fff)) ^ ((x1 ^ ^y1) & 0x8000800080008000)
	t2_1 := ((x1 ^ y1) & (x1 ^ t1_1)) & 0x8000800080008000
	t3_1 := (t2_1 >> 15) * 0x000000000000ffff
	t4_1 := x1 & 0x8000800080008000
	t5_1 := (t4_1 >> 15) * 0x000000000000ffff
	t6_1 := t5_1 ^ 0x7fff7fff7fff7fff
	t7_1 := (t6_1 & t3_1) | (t1_1 &^ t3_1)
	return v128{t7_0, t7_1}
}

//go:nosplit
func i16x8_extadd_pairwise_i8x16_s(a v128) v128 {
	x0, x1 := a.lo, a.hi
	t1_0 := x0 & 0x00ff00ff00ff00ff
	t2_0 := (x0 >> 8) & 0x00ff00ff00ff00ff
	t3_0 := t1_0 | (((t1_0&0x0080008000800080)>>7)*0xff)<<8
	t4_0 := t2_0 | (((t2_0&0x0080008000800080)>>7)*0xff)<<8
	t5_0 := ((t3_0 & 0x7fff7fff7fff7fff) + (t4_0 & 0x7fff7fff7fff7fff)) ^ ((t3_0 ^ t4_0) & 0x8000800080008000)
	t1_1 := x1 & 0x00ff00ff00ff00ff
	t2_1 := (x1 >> 8) & 0x00ff00ff00ff00ff
	t3_1 := t1_1 | (((t1_1&0x0080008000800080)>>7)*0xff)<<8
	t4_1 := t2_1 | (((t2_1&0x0080008000800080)>>7)*0xff)<<8
	t5_1 := ((t3_1 & 0x7fff7fff7fff7fff) + (t4_1 & 0x7fff7fff7fff7fff)) ^ ((t3_1 ^ t4_1) & 0x8000800080008000)
	return v128{t5_0, t5_1}
}

//go:nosplit
func i16x8_extadd_pairwise_i8x16_u(a v128) v128 {
	x0, x1 := a.lo, a.hi
	t1_0 := x0 & 0x00ff00ff00ff00ff
	t2_0 := (x0 >> 8) & 0x00ff00ff00ff00ff
	t3_0 := t1_0 + t2_0
	t1_1 := x1 & 0x00ff00ff00ff00ff
	t2_1 := (x1 >> 8) & 0x00ff00ff00ff00ff
	t3_1 := t1_1 + t2_1
	return v128{t3_0, t3_1}
}

//go:nosplit
func i32x4_extadd_pairwise_i16x8_s(a v128) v128 {
	x0, x1 := a.lo, a.hi
	t1_0 := x0 & 0x0000ffff0000ffff
	t2_0 := (x0 >> 16) & 0x0000ffff0000ffff
	t3_0 := t1_0 | (((t1_0&0x0000800000008000)>>15)*0xffff)<<16
	t4_0 := t2_0 | (((t2_0&0x0000800000008000)>>15)*0xffff)<<16
	t5_0 := ((t3_0 & 0x7fffffff7fffffff) + (t4_0 & 0x7fffffff7fffffff)) ^ ((t3_0 ^ t4_0) & 0x8000000080000000)
	t1_1 := x1 & 0x0000ffff0000ffff
	t2_1 := (x1 >> 16) & 0x0000ffff0000ffff
	t3_1 := t1_1 | (((t1_1&0x0000800000008000)>>15)*0xffff)<<16
	t4_1 := t2_1 | (((t2_1&0x0000800000008000)>>15)*0xffff)<<16
	t5_1 := ((t3_1 & 0x7fffffff7fffffff) + (t4_1 & 0x7fffffff7fffffff)) ^ ((t3_1 ^ t4_1) & 0x8000000080000000)
	return v128{t5_0, t5_1}
}

//go:nosplit
func i32x4_extadd_pairwise_i16x8_u(a v128) v128 {
	x0, x1 := a.lo, a.hi
	t1_0 := x0 & 0x0000ffff0000ffff
	t2_0 := (x0 >> 16) & 0x0000ffff0000ffff
	t3_0 := t1_0 + t2_0
	t1_1 := x1 & 0x0000ffff0000ffff
	t2_1 := (x1 >> 16) & 0x0000ffff0000ffff
	t3_1 := t1_1 + t2_1
	return v128{t3_0, t3_1}
}

//go:nosplit
func i16x8_extmul_low_i8x16_u(a, b v128) v128 {
	x0 := a.lo
	y0 := b.lo
	r0 := uint64(uint16(uint16(uint8(x0))*uint16(uint8(y0)))) | uint64(uint16(uint16(uint8(x0>>8))*uint16(uint8(y0>>8))))<<16 | uint64(uint16(uint16(uint8(x0>>16))*uint16(uint8(y0>>16))))<<32 | uint64(uint16(uint16(uint8(x0>>24))*uint16(uint8(y0>>24))))<<48
	r1 := uint64(uint16(uint16(uint8(x0>>32))*uint16(uint8(y0>>32)))) | uint64(uint16(uint16(uint8(x0>>40))*uint16(uint8(y0>>40))))<<16 | uint64(uint16(uint16(uint8(x0>>48))*uint16(uint8(y0>>48))))<<32 | uint64(uint16(uint16(uint8(x0>>56))*uint16(uint8(y0>>56))))<<48
	return v128{r0, r1}
}

//go:nosplit
func i32x4_extmul_low_i16x8_u(a, b v128) v128 {
	x0 := a.lo
	y0 := b.lo
	r0 := uint64(uint32(uint32(uint16(x0))*uint32(uint16(y0)))) | uint64(uint32(uint32(uint16(x0>>16))*uint32(uint16(y0>>16))))<<32
	r1 := uint64(uint32(uint32(uint16(x0>>32))*uint32(uint16(y0>>32)))) | uint64(uint32(uint32(uint16(x0>>48))*uint32(uint16(y0>>48))))<<32
	return v128{r0, r1}
}

//go:nosplit
func i64x2_extmul_low_i32x4_u(a, b v128) v128 {
	x0 := a.lo
	y0 := b.lo
	r0 := uint64(uint64(uint64(uint32(x0)) * uint64(uint32(y0))))
	r1 := uint64(uint64(uint64(uint32(x0>>32)) * uint64(uint32(y0>>32))))
	return v128{r0, r1}
}

//go:nosplit
func i16x8_extmul_low_i8x16_s(a, b v128) v128 {
	x0 := a.lo
	y0 := b.lo
	r0 := uint64(uint16(int16(int8(uint8(x0)))*int16(int8(uint8(y0))))) | uint64(uint16(int16(int8(uint8(x0>>8)))*int16(int8(uint8(y0>>8)))))<<16 | uint64(uint16(int16(int8(uint8(x0>>16)))*int16(int8(uint8(y0>>16)))))<<32 | uint64(uint16(int16(int8(uint8(x0>>24)))*int16(int8(uint8(y0>>24)))))<<48
	r1 := uint64(uint16(int16(int8(uint8(x0>>32)))*int16(int8(uint8(y0>>32))))) | uint64(uint16(int16(int8(uint8(x0>>40)))*int16(int8(uint8(y0>>40)))))<<16 | uint64(uint16(int16(int8(uint8(x0>>48)))*int16(int8(uint8(y0>>48)))))<<32 | uint64(uint16(int16(int8(uint8(x0>>56)))*int16(int8(uint8(y0>>56)))))<<48
	return v128{r0, r1}
}

//go:nosplit
func i32x4_extmul_low_i16x8_s(a, b v128) v128 {
	x0 := a.lo
	y0 := b.lo
	r0 := uint64(uint32(int32(int16(uint16(x0)))*int32(int16(uint16(y0))))) | uint64(uint32(int32(int16(uint16(x0>>16)))*int32(int16(uint16(y0>>16)))))<<32
	r1 := uint64(uint32(int32(int16(uint16(x0>>32)))*int32(int16(uint16(y0>>32))))) | uint64(uint32(int32(int16(uint16(x0>>48)))*int32(int16(uint16(y0>>48)))))<<32
	return v128{r0, r1}
}

//go:nosplit
func i64x2_extmul_low_i32x4_s(a, b v128) v128 {
	x0 := a.lo
	y0 := b.lo
	r0 := uint64(uint64(int64(int32(uint32(x0))) * int64(int32(uint32(y0)))))
	r1 := uint64(uint64(int64(int32(uint32(x0>>32))) * int64(int32(uint32(y0>>32)))))
	return v128{r0, r1}
}

//go:nosplit
func i16x8_extmul_high_i8x16_u(a, b v128) v128 {
	x1 := a.hi
	y1 := b.hi
	r0 := uint64(uint16(uint16(uint8(x1))*uint16(uint8(y1)))) | uint64(uint16(uint16(uint8(x1>>8))*uint16(uint8(y1>>8))))<<16 | uint64(uint16(uint16(uint8(x1>>16))*uint16(uint8(y1>>16))))<<32 | uint64(uint16(uint16(uint8(x1>>24))*uint16(uint8(y1>>24))))<<48
	r1 := uint64(uint16(uint16(uint8(x1>>32))*uint16(uint8(y1>>32)))) | uint64(uint16(uint16(uint8(x1>>40))*uint16(uint8(y1>>40))))<<16 | uint64(uint16(uint16(uint8(x1>>48))*uint16(uint8(y1>>48))))<<32 | uint64(uint16(uint16(uint8(x1>>56))*uint16(uint8(y1>>56))))<<48
	return v128{r0, r1}
}

//go:nosplit
func i32x4_extmul_high_i16x8_u(a, b v128) v128 {
	x1 := a.hi
	y1 := b.hi
	r0 := uint64(uint32(uint32(uint16(x1))*uint32(uint16(y1)))) | uint64(uint32(uint32(uint16(x1>>16))*uint32(uint16(y1>>16))))<<32
	r1 := uint64(uint32(uint32(uint16(x1>>32))*uint32(uint16(y1>>32)))) | uint64(uint32(uint32(uint16(x1>>48))*uint32(uint16(y1>>48))))<<32
	return v128{r0, r1}
}

//go:nosplit
func i64x2_extmul_high_i32x4_u(a, b v128) v128 {
	x1 := a.hi
	y1 := b.hi
	r0 := uint64(uint64(uint64(uint32(x1)) * uint64(uint32(y1))))
	r1 := uint64(uint64(uint64(uint32(x1>>32)) * uint64(uint32(y1>>32))))
	return v128{r0, r1}
}

//go:nosplit
func i16x8_extmul_high_i8x16_s(a, b v128) v128 {
	x1 := a.hi
	y1 := b.hi
	r0 := uint64(uint16(int16(int8(uint8(x1)))*int16(int8(uint8(y1))))) | uint64(uint16(int16(int8(uint8(x1>>8)))*int16(int8(uint8(y1>>8)))))<<16 | uint64(uint16(int16(int8(uint8(x1>>16)))*int16(int8(uint8(y1>>16)))))<<32 | uint64(uint16(int16(int8(uint8(x1>>24)))*int16(int8(uint8(y1>>24)))))<<48
	r1 := uint64(uint16(int16(int8(uint8(x1>>32)))*int16(int8(uint8(y1>>32))))) | uint64(uint16(int16(int8(uint8(x1>>40)))*int16(int8(uint8(y1>>40)))))<<16 | uint64(uint16(int16(int8(uint8(x1>>48)))*int16(int8(uint8(y1>>48)))))<<32 | uint64(uint16(int16(int8(uint8(x1>>56)))*int16(int8(uint8(y1>>56)))))<<48
	return v128{r0, r1}
}

//go:nosplit
func i32x4_extmul_high_i16x8_s(a, b v128) v128 {
	x1 := a.hi
	y1 := b.hi
	r0 := uint64(uint32(int32(int16(uint16(x1)))*int32(int16(uint16(y1))))) | uint64(uint32(int32(int16(uint16(x1>>16)))*int32(int16(uint16(y1>>16)))))<<32
	r1 := uint64(uint32(int32(int16(uint16(x1>>32)))*int32(int16(uint16(y1>>32))))) | uint64(uint32(int32(int16(uint16(x1>>48)))*int32(int16(uint16(y1>>48)))))<<32
	return v128{r0, r1}
}

//go:nosplit
func i64x2_extmul_high_i32x4_s(a, b v128) v128 {
	x1 := a.hi
	y1 := b.hi
	r0 := uint64(uint64(int64(int32(uint32(x1))) * int64(int32(uint32(y1)))))
	r1 := uint64(uint64(int64(int32(uint32(x1>>32))) * int64(int32(uint32(y1>>32)))))
	return v128{r0, r1}
}

//go:nosplit
func i32x4_dot_i16x8_s(a, b v128) v128 {
	x0, x1 := a.lo, a.hi
	y0, y1 := b.lo, b.hi
	r0 := uint64(uint32(int32(int16(uint16(x0)))*int32(int16(uint16(y0)))+int32(int16(uint16(x0>>16)))*int32(int16(uint16(y0>>16))))) | uint64(uint32(int32(int16(uint16(x0>>32)))*int32(int16(uint16(y0>>32)))+int32(int16(uint16(x0>>48)))*int32(int16(uint16(y0>>48)))))<<32
	r1 := uint64(uint32(int32(int16(uint16(x1)))*int32(int16(uint16(y1)))+int32(int16(uint16(x1>>16)))*int32(int16(uint16(y1>>16))))) | uint64(uint32(int32(int16(uint16(x1>>32)))*int32(int16(uint16(y1>>32)))+int32(int16(uint16(x1>>48)))*int32(int16(uint16(y1>>48)))))<<32
	return v128{r0, r1}
}

//go:nosplit
func i64x2_abs(a v128) v128 {
	x0, x1 := a.lo, a.hi
	r0 := x0
	if int64(r0) < 0 {
		r0 = -r0
	}
	r1 := x1
	if int64(r1) < 0 {
		r1 = -r1
	}
	return v128{r0, r1}
}
