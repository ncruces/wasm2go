//go:build goexperiment.simd

// AMD64 SIMD helpers for the Wasm fixed-width SIMD (v128) instructions.
//
// A v128 is represented directly as archsimd.Int8x16 (an XMM register under
// GOEXPERIMENT=simd), so vectors stay in hardware XMM registers across chained
// operations and helper calls inline into native AVX/AVX2 instructions.
//
// Every helper is self-contained: wasm2go's resolveHelpers copies referenced
// helpers by name without transitive resolution, so helpers never call each
// other or use package-level declarations.

package helpers

import (
	"math"
	"simd/archsimd"
)

type v128 = archsimd.Int8x16

//go:nosplit
func load128[T uint32 | uint64](mem []byte, addr T) v128 {
	return archsimd.LoadUint8x16((*[16]byte)(mem[addr:])).AsInt8x16()
}

//go:nosplit
func store128[T uint32 | uint64](mem []byte, addr T, val v128) {
	val.AsUint8x16().Store((*[16]byte)(mem[addr:]))
}

//go:nosplit
func v128_load32_zero(x uint32) v128 {
	return archsimd.Uint32x4{}.SetElem(0, x).AsInt8x16()
}

//go:nosplit
func v128_load64_zero(x uint64) v128 {
	return archsimd.Uint64x2{}.SetElem(0, x).AsInt8x16()
}

//go:nosplit
func i8x16_splat(x int32) v128 {
	return archsimd.BroadcastInt8x16(int8(x))
}

//go:nosplit
func i16x8_splat(x int32) v128 {
	return archsimd.BroadcastInt16x8(int16(x)).AsInt8x16()
}

//go:nosplit
func i32x4_splat(x int32) v128 {
	return archsimd.BroadcastInt32x4(x).AsInt8x16()
}

//go:nosplit
func i64x2_splat(x int64) v128 {
	return archsimd.BroadcastInt64x2(x).AsInt8x16()
}

//go:nosplit
func f32x4_splat(x float32) v128 {
	return archsimd.BroadcastFloat32x4(x).AsInt8x16()
}

//go:nosplit
func f64x2_splat(x float64) v128 {
	return archsimd.BroadcastFloat64x2(x).AsInt8x16()
}

//go:nosplit
func i8x16_extract_lane_s(v v128, l int) int32 {
	return int32(v.GetElem(uint8(l & 15)))
}

//go:nosplit
func i8x16_extract_lane_u(v v128, l int) int32 {
	return int32(v.AsUint8x16().GetElem(uint8(l & 15)))
}

//go:nosplit
func i16x8_extract_lane_s(v v128, l int) int32 {
	return int32(v.AsInt16x8().GetElem(uint8(l & 7)))
}

//go:nosplit
func i16x8_extract_lane_u(v v128, l int) int32 {
	return int32(v.AsUint16x8().GetElem(uint8(l & 7)))
}

//go:nosplit
func i32x4_extract_lane(v v128, l int) int32 {
	return v.AsInt32x4().GetElem(uint8(l & 3))
}

//go:nosplit
func i64x2_extract_lane(v v128, l int) int64 {
	return v.AsInt64x2().GetElem(uint8(l & 1))
}

//go:nosplit
func f32x4_extract_lane(v v128, l int) float32 {
	return math.Float32frombits(v.AsUint32x4().GetElem(uint8(l & 3)))
}

//go:nosplit
func f64x2_extract_lane(v v128, l int) float64 {
	return math.Float64frombits(v.AsUint64x2().GetElem(uint8(l & 1)))
}

//go:nosplit
func i8x16_replace_lane(v v128, l int, x int32) v128 {
	return v.SetElem(uint8(l&15), int8(x))
}

//go:nosplit
func i16x8_replace_lane(v v128, l int, x int32) v128 {
	return v.AsInt16x8().SetElem(uint8(l&7), int16(x)).AsInt8x16()
}

//go:nosplit
func i32x4_replace_lane(v v128, l int, x int32) v128 {
	return v.AsInt32x4().SetElem(uint8(l&3), x).AsInt8x16()
}

//go:nosplit
func f32x4_replace_lane(v v128, l int, x float32) v128 {
	return v.AsUint32x4().SetElem(uint8(l&3), math.Float32bits(x)).AsInt8x16()
}

//go:nosplit
func i64x2_replace_lane(v v128, l int, x int64) v128 {
	return v.AsInt64x2().SetElem(uint8(l&1), x).AsInt8x16()
}

//go:nosplit
func f64x2_replace_lane(v v128, l int, x float64) v128 {
	return v.AsUint64x2().SetElem(uint8(l&1), math.Float64bits(x)).AsInt8x16()
}

//go:nosplit
func v128_any_true(v v128) int32 {
	if v.IsZero() {
		return 0
	}
	return 1
}

//go:nosplit
func i8x16_bitmask(v v128) int32 {
	return int32(v.Less(archsimd.Int8x16{}).ToBits())
}

//go:nosplit
func i8x16_shuffle(a, b, m v128) v128 {
	m0f := archsimd.BroadcastInt8x16(0x0f)
	ma := m.And(m0f).Or(m.AsUint16x8().ShiftAllLeft(3).AsInt8x16().AndNot(m0f))
	mb := ma.Xor(m0f.AsUint16x8().ShiftAllLeft(4).AsInt8x16())
	return a.PermuteOrZero(ma).Or(b.PermuteOrZero(mb))
}

//go:nosplit
func i8x16_swizzle(a, s v128) v128 {
	m := s.AsUint8x16().AddSaturated(archsimd.BroadcastUint8x16(0x70)).AsInt8x16()
	return a.PermuteOrZero(m)
}

//go:nosplit
func i8x16_all_true(v v128) int32 {
	if v.Equal(archsimd.Int8x16{}).ToInt8x16().IsZero() {
		return 1
	}
	return 0
}

//go:nosplit
func i16x8_all_true(v v128) int32 {
	if v.AsInt16x8().Equal(archsimd.Int16x8{}).ToInt16x8().IsZero() {
		return 1
	}
	return 0
}

//go:nosplit
func i32x4_all_true(v v128) int32 {
	if v.AsInt32x4().Equal(archsimd.Int32x4{}).ToInt32x4().IsZero() {
		return 1
	}
	return 0
}

//go:nosplit
func i64x2_all_true(v v128) int32 {
	if v.AsInt64x2().Equal(archsimd.Int64x2{}).ToInt64x2().IsZero() {
		return 1
	}
	return 0
}

//go:nosplit
func v128_and(a, b v128) v128 {
	return a.And(b)
}

//go:nosplit
func v128_or(a, b v128) v128 {
	return a.Or(b)
}

//go:nosplit
func v128_xor(a, b v128) v128 {
	return a.Xor(b)
}

//go:nosplit
func v128_andnot(a, b v128) v128 {
	return a.AndNot(b)
}

//go:nosplit
func v128_not(a v128) v128 {
	return a.Not()
}

//go:nosplit
func v128_bitselect(a, b, c v128) v128 {
	return a.And(c).Or(b.AndNot(c))
}

//go:nosplit
func i8x16_add(a, b v128) v128 {
	return a.Add(b)
}

//go:nosplit
func i8x16_sub(a, b v128) v128 {
	return a.Sub(b)
}

//go:nosplit
func i8x16_neg(a v128) v128 {
	return archsimd.Int8x16{}.Sub(a)
}

//go:nosplit
func i8x16_abs(a v128) v128 {
	return a.Abs()
}

//go:nosplit
func i8x16_eq(a, b v128) v128 {
	return a.Equal(b).ToInt8x16()
}

//go:nosplit
func i8x16_ne(a, b v128) v128 {
	return a.Equal(b).ToInt8x16().Equal(archsimd.Int8x16{}).ToInt8x16()
}

//go:nosplit
func i8x16_lt_s(a, b v128) v128 {
	return a.Less(b).ToInt8x16()
}

//go:nosplit
func i8x16_lt_u(a, b v128) v128 {
	u := a.AsUint8x16()
	return u.Max(b.AsUint8x16()).Equal(u).ToInt8x16().Equal(archsimd.Int8x16{}).ToInt8x16()
}

//go:nosplit
func i8x16_gt_s(a, b v128) v128 {
	return a.Greater(b).ToInt8x16()
}

//go:nosplit
func i8x16_gt_u(a, b v128) v128 {
	v := b.AsUint8x16()
	return a.AsUint8x16().Max(v).Equal(v).ToInt8x16().Equal(archsimd.Int8x16{}).ToInt8x16()
}

//go:nosplit
func i8x16_le_s(a, b v128) v128 {
	return a.Min(b).Equal(a).ToInt8x16()
}

//go:nosplit
func i8x16_le_u(a, b v128) v128 {
	u := a.AsUint8x16()
	return u.Min(b.AsUint8x16()).Equal(u).ToInt8x16()
}

//go:nosplit
func i8x16_ge_s(a, b v128) v128 {
	return a.Max(b).Equal(a).ToInt8x16()
}

//go:nosplit
func i8x16_ge_u(a, b v128) v128 {
	u := a.AsUint8x16()
	return u.Max(b.AsUint8x16()).Equal(u).ToInt8x16()
}

//go:nosplit
func i8x16_min_s(a, b v128) v128 {
	return a.Min(b)
}

//go:nosplit
func i8x16_min_u(a, b v128) v128 {
	return a.AsUint8x16().Min(b.AsUint8x16()).AsInt8x16()
}

//go:nosplit
func i8x16_max_s(a, b v128) v128 {
	return a.Max(b)
}

//go:nosplit
func i8x16_max_u(a, b v128) v128 {
	return a.AsUint8x16().Max(b.AsUint8x16()).AsInt8x16()
}

//go:nosplit
func i8x16_shl(a v128, y int32) v128 {
	s := uint64(y & 7)
	return a.AsUint16x8().ShiftAllLeft(s).AsUint8x16().And(archsimd.BroadcastUint8x16(uint8(0xff << s))).AsInt8x16()
}

//go:nosplit
func i8x16_shr_u(a v128, y int32) v128 {
	s := uint64(y & 7)
	return a.AsUint16x8().ShiftAllRight(s).AsUint8x16().And(archsimd.BroadcastUint8x16(uint8(0xff >> s))).AsInt8x16()
}

//go:nosplit
func i8x16_shr_s(a v128, y int32) v128 {
	s := uint64(y & 7)
	w := a.AsInt16x8()
	m := archsimd.BroadcastUint32x4(0x00ff00ff).AsInt16x8()
	lo := w.ShiftAllLeft(8).ShiftAllRight(8 + s).And(m)
	hi := w.ShiftAllRight(s).AndNot(m)
	return lo.Or(hi).AsInt8x16()
}

//go:nosplit
func i8x16_add_sat_u(a, b v128) v128 {
	return a.AsUint8x16().AddSaturated(b.AsUint8x16()).AsInt8x16()
}

//go:nosplit
func i8x16_sub_sat_u(a, b v128) v128 {
	return a.AsUint8x16().SubSaturated(b.AsUint8x16()).AsInt8x16()
}

//go:nosplit
func i8x16_avgr_u(a, b v128) v128 {
	return a.AsUint8x16().Average(b.AsUint8x16()).AsInt8x16()
}

//go:nosplit
func i16x8_add(a, b v128) v128 {
	return a.AsInt16x8().Add(b.AsInt16x8()).AsInt8x16()
}

//go:nosplit
func i16x8_sub(a, b v128) v128 {
	return a.AsInt16x8().Sub(b.AsInt16x8()).AsInt8x16()
}

//go:nosplit
func i16x8_neg(a v128) v128 {
	return archsimd.Int16x8{}.Sub(a.AsInt16x8()).AsInt8x16()
}

//go:nosplit
func i16x8_abs(a v128) v128 {
	return a.AsInt16x8().Abs().AsInt8x16()
}

//go:nosplit
func i16x8_eq(a, b v128) v128 {
	return a.AsInt16x8().Equal(b.AsInt16x8()).ToInt16x8().AsInt8x16()
}

//go:nosplit
func i16x8_ne(a, b v128) v128 {
	return a.AsInt16x8().Equal(b.AsInt16x8()).ToInt16x8().Equal(archsimd.Int16x8{}).ToInt16x8().AsInt8x16()
}

//go:nosplit
func i16x8_lt_s(a, b v128) v128 {
	return a.AsInt16x8().Less(b.AsInt16x8()).ToInt16x8().AsInt8x16()
}

//go:nosplit
func i16x8_lt_u(a, b v128) v128 {
	u := a.AsUint16x8()
	return u.Max(b.AsUint16x8()).Equal(u).ToInt16x8().Equal(archsimd.Int16x8{}).ToInt16x8().AsInt8x16()
}

//go:nosplit
func i16x8_gt_s(a, b v128) v128 {
	return a.AsInt16x8().Greater(b.AsInt16x8()).ToInt16x8().AsInt8x16()
}

//go:nosplit
func i16x8_gt_u(a, b v128) v128 {
	v := b.AsUint16x8()
	return a.AsUint16x8().Max(v).Equal(v).ToInt16x8().Equal(archsimd.Int16x8{}).ToInt16x8().AsInt8x16()
}

//go:nosplit
func i16x8_le_s(a, b v128) v128 {
	x := a.AsInt16x8()
	return x.Min(b.AsInt16x8()).Equal(x).ToInt16x8().AsInt8x16()
}

//go:nosplit
func i16x8_le_u(a, b v128) v128 {
	u := a.AsUint16x8()
	return u.Min(b.AsUint16x8()).Equal(u).ToInt16x8().AsInt8x16()
}

//go:nosplit
func i16x8_ge_s(a, b v128) v128 {
	x := a.AsInt16x8()
	return x.Max(b.AsInt16x8()).Equal(x).ToInt16x8().AsInt8x16()
}

//go:nosplit
func i16x8_ge_u(a, b v128) v128 {
	u := a.AsUint16x8()
	return u.Max(b.AsUint16x8()).Equal(u).ToInt16x8().AsInt8x16()
}

//go:nosplit
func i16x8_min_s(a, b v128) v128 {
	return a.AsInt16x8().Min(b.AsInt16x8()).AsInt8x16()
}

//go:nosplit
func i16x8_min_u(a, b v128) v128 {
	return a.AsUint16x8().Min(b.AsUint16x8()).AsInt8x16()
}

//go:nosplit
func i16x8_max_s(a, b v128) v128 {
	return a.AsInt16x8().Max(b.AsInt16x8()).AsInt8x16()
}

//go:nosplit
func i16x8_max_u(a, b v128) v128 {
	return a.AsUint16x8().Max(b.AsUint16x8()).AsInt8x16()
}

//go:nosplit
func i16x8_shl(a v128, y int32) v128 {
	return a.AsInt16x8().ShiftAllLeft(uint64(y & 15)).AsInt8x16()
}

//go:nosplit
func i16x8_shr_u(a v128, y int32) v128 {
	return a.AsUint16x8().ShiftAllRight(uint64(y & 15)).AsInt8x16()
}

//go:nosplit
func i16x8_shr_s(a v128, y int32) v128 {
	return a.AsInt16x8().ShiftAllRight(uint64(y & 15)).AsInt8x16()
}

//go:nosplit
func i16x8_add_sat_u(a, b v128) v128 {
	return a.AsUint16x8().AddSaturated(b.AsUint16x8()).AsInt8x16()
}

//go:nosplit
func i16x8_sub_sat_u(a, b v128) v128 {
	return a.AsUint16x8().SubSaturated(b.AsUint16x8()).AsInt8x16()
}

//go:nosplit
func i16x8_avgr_u(a, b v128) v128 {
	return a.AsUint16x8().Average(b.AsUint16x8()).AsInt8x16()
}

//go:nosplit
func i32x4_add(a, b v128) v128 {
	return a.AsInt32x4().Add(b.AsInt32x4()).AsInt8x16()
}

//go:nosplit
func i32x4_sub(a, b v128) v128 {
	return a.AsInt32x4().Sub(b.AsInt32x4()).AsInt8x16()
}

//go:nosplit
func i32x4_neg(a v128) v128 {
	return archsimd.Int32x4{}.Sub(a.AsInt32x4()).AsInt8x16()
}

//go:nosplit
func i32x4_abs(a v128) v128 {
	return a.AsInt32x4().Abs().AsInt8x16()
}

//go:nosplit
func i32x4_eq(a, b v128) v128 {
	return a.AsInt32x4().Equal(b.AsInt32x4()).ToInt32x4().AsInt8x16()
}

//go:nosplit
func i32x4_ne(a, b v128) v128 {
	return a.AsInt32x4().Equal(b.AsInt32x4()).ToInt32x4().Equal(archsimd.Int32x4{}).ToInt32x4().AsInt8x16()
}

//go:nosplit
func i32x4_lt_s(a, b v128) v128 {
	return a.AsInt32x4().Less(b.AsInt32x4()).ToInt32x4().AsInt8x16()
}

//go:nosplit
func i32x4_lt_u(a, b v128) v128 {
	u := a.AsUint32x4()
	return u.Max(b.AsUint32x4()).Equal(u).ToInt32x4().Equal(archsimd.Int32x4{}).ToInt32x4().AsInt8x16()
}

//go:nosplit
func i32x4_gt_s(a, b v128) v128 {
	return a.AsInt32x4().Greater(b.AsInt32x4()).ToInt32x4().AsInt8x16()
}

//go:nosplit
func i32x4_gt_u(a, b v128) v128 {
	v := b.AsUint32x4()
	return a.AsUint32x4().Max(v).Equal(v).ToInt32x4().Equal(archsimd.Int32x4{}).ToInt32x4().AsInt8x16()
}

//go:nosplit
func i32x4_le_s(a, b v128) v128 {
	x := a.AsInt32x4()
	return x.Min(b.AsInt32x4()).Equal(x).ToInt32x4().AsInt8x16()
}

//go:nosplit
func i32x4_le_u(a, b v128) v128 {
	u := a.AsUint32x4()
	return u.Min(b.AsUint32x4()).Equal(u).ToInt32x4().AsInt8x16()
}

//go:nosplit
func i32x4_ge_s(a, b v128) v128 {
	x := a.AsInt32x4()
	return x.Max(b.AsInt32x4()).Equal(x).ToInt32x4().AsInt8x16()
}

//go:nosplit
func i32x4_ge_u(a, b v128) v128 {
	u := a.AsUint32x4()
	return u.Max(b.AsUint32x4()).Equal(u).ToInt32x4().AsInt8x16()
}

//go:nosplit
func i32x4_min_s(a, b v128) v128 {
	return a.AsInt32x4().Min(b.AsInt32x4()).AsInt8x16()
}

//go:nosplit
func i32x4_min_u(a, b v128) v128 {
	return a.AsUint32x4().Min(b.AsUint32x4()).AsInt8x16()
}

//go:nosplit
func i32x4_max_s(a, b v128) v128 {
	return a.AsInt32x4().Max(b.AsInt32x4()).AsInt8x16()
}

//go:nosplit
func i32x4_max_u(a, b v128) v128 {
	return a.AsUint32x4().Max(b.AsUint32x4()).AsInt8x16()
}

//go:nosplit
func i32x4_shl(a v128, y int32) v128 {
	return a.AsInt32x4().ShiftAllLeft(uint64(y & 31)).AsInt8x16()
}

//go:nosplit
func i32x4_shr_u(a v128, y int32) v128 {
	return a.AsUint32x4().ShiftAllRight(uint64(y & 31)).AsInt8x16()
}

//go:nosplit
func i32x4_shr_s(a v128, y int32) v128 {
	return a.AsInt32x4().ShiftAllRight(uint64(y & 31)).AsInt8x16()
}

//go:nosplit
func i8x16_popcnt(a v128) v128 {
	lut := archsimd.Int64x2{}.SetElem(0, 0x0302020102010100).SetElem(1, 0x0403030203020201).AsInt8x16()
	m0f := archsimd.BroadcastInt8x16(0x0f)
	lo := a.And(m0f)
	hi := a.AsUint16x8().ShiftAllRight(4).AsInt8x16().And(m0f)
	return lut.PermuteOrZero(lo).Add(lut.PermuteOrZero(hi))
}

//go:nosplit
func i64x2_add(a, b v128) v128 {
	return a.AsInt64x2().Add(b.AsInt64x2()).AsInt8x16()
}

//go:nosplit
func i64x2_sub(a, b v128) v128 {
	return a.AsInt64x2().Sub(b.AsInt64x2()).AsInt8x16()
}

//go:nosplit
func i64x2_mul(a, b v128) v128 {
	x, y := a.AsInt64x2(), b.AsInt64x2()
	return archsimd.Int64x2{}.SetElem(0, x.GetElem(0)*y.GetElem(0)).SetElem(1, x.GetElem(1)*y.GetElem(1)).AsInt8x16()
}

//go:nosplit
func i64x2_eq(a, b v128) v128 {
	return a.AsInt64x2().Equal(b.AsInt64x2()).ToInt64x2().AsInt8x16()
}

//go:nosplit
func i64x2_ne(a, b v128) v128 {
	return a.AsInt64x2().Equal(b.AsInt64x2()).ToInt64x2().Equal(archsimd.Int64x2{}).ToInt64x2().AsInt8x16()
}

//go:nosplit
func i64x2_lt_s(a, b v128) v128 {
	return a.AsInt64x2().Less(b.AsInt64x2()).ToInt64x2().AsInt8x16()
}

//go:nosplit
func i64x2_gt_s(a, b v128) v128 {
	return a.AsInt64x2().Greater(b.AsInt64x2()).ToInt64x2().AsInt8x16()
}

//go:nosplit
func i64x2_le_s(a, b v128) v128 {
	return a.AsInt64x2().Greater(b.AsInt64x2()).ToInt64x2().Equal(archsimd.Int64x2{}).ToInt64x2().AsInt8x16()
}

//go:nosplit
func i64x2_ge_s(a, b v128) v128 {
	return b.AsInt64x2().Greater(a.AsInt64x2()).ToInt64x2().Equal(archsimd.Int64x2{}).ToInt64x2().AsInt8x16()
}

//go:nosplit
func i64x2_neg(a v128) v128 {
	return archsimd.Int64x2{}.Sub(a.AsInt64x2()).AsInt8x16()
}

//go:nosplit
func i64x2_shl(a v128, y int32) v128 {
	return a.AsInt64x2().ShiftAllLeft(uint64(y & 63)).AsInt8x16()
}

//go:nosplit
func i64x2_shr_u(a v128, y int32) v128 {
	return a.AsUint64x2().ShiftAllRight(uint64(y & 63)).AsInt8x16()
}

//go:nosplit
func i64x2_shr_s(a v128, y int32) v128 {
	s := uint(y) & 63
	x := a.AsInt64x2()
	return archsimd.Int64x2{}.SetElem(0, x.GetElem(0)>>s).SetElem(1, x.GetElem(1)>>s).AsInt8x16()
}

//go:nosplit
func i16x8_extend_low_i8x16_u(a v128) v128 {
	return a.AsUint8x16().ExtendLo8ToUint16().AsInt8x16()
}

//go:nosplit
func i32x4_extend_low_i16x8_u(a v128) v128 {
	return a.AsUint16x8().ExtendLo4ToUint32().AsInt8x16()
}

//go:nosplit
func i64x2_extend_low_i32x4_u(a v128) v128 {
	return a.AsUint32x4().ExtendLo2ToUint64().AsInt8x16()
}

//go:nosplit
func i16x8_extend_low_i8x16_s(a v128) v128 {
	return a.ExtendLo8ToInt16().AsInt8x16()
}

//go:nosplit
func i32x4_extend_low_i16x8_s(a v128) v128 {
	return a.AsInt16x8().ExtendLo4ToInt32().AsInt8x16()
}

//go:nosplit
func i64x2_extend_low_i32x4_s(a v128) v128 {
	return a.AsInt32x4().ExtendLo2ToInt64().AsInt8x16()
}

//go:nosplit
func i16x8_extend_high_i8x16_u(a v128) v128 {
	return a.AsUint8x16().ConcatShiftBytesRight(8, a.AsUint8x16()).ExtendLo8ToUint16().AsInt8x16()
}

//go:nosplit
func i32x4_extend_high_i16x8_u(a v128) v128 {
	return a.AsUint16x8().InterleaveHi(archsimd.Uint16x8{}).AsInt8x16()
}

//go:nosplit
func i64x2_extend_high_i32x4_u(a v128) v128 {
	return a.AsUint32x4().InterleaveHi(archsimd.Uint32x4{}).AsInt8x16()
}

//go:nosplit
func i16x8_extend_high_i8x16_s(a v128) v128 {
	return a.AsUint8x16().ConcatShiftBytesRight(8, a.AsUint8x16()).AsInt8x16().ExtendLo8ToInt16().AsInt8x16()
}

//go:nosplit
func i32x4_extend_high_i16x8_s(a v128) v128 {
	return a.AsInt16x8().InterleaveHi(a.AsInt16x8()).AsInt32x4().ShiftAllRight(16).AsInt8x16()
}

//go:nosplit
func i64x2_extend_high_i32x4_s(a v128) v128 {
	return a.AsInt32x4().PermuteScalars(2, 3, 2, 3).ExtendLo2ToInt64().AsInt8x16()
}

//go:nosplit
func i8x16_narrow_i16x8_s(a, b v128) v128 {
	cMax := archsimd.BroadcastUint32x4(0x007f007f).AsInt16x8()
	cMin := cMax.ShiftAllLeft(9).ShiftAllRight(2)
	lo := a.AsInt16x8().Max(cMin).Min(cMax).AsInt8x16()
	hi := b.AsInt16x8().Max(cMin).Min(cMax).AsInt8x16()
	m := archsimd.Int64x2{}.SetElem(0, 0x0e0c0a0806040200).AsInt8x16()
	return lo.PermuteOrZero(m).AsInt64x2().InterleaveLo(hi.PermuteOrZero(m).AsInt64x2()).AsInt8x16()
}

//go:nosplit
func i8x16_narrow_i16x8_u(a, b v128) v128 {
	z := archsimd.Int16x8{}
	cMax := archsimd.BroadcastInt32x4(0x00ff00ff).AsInt16x8()
	lo := a.AsInt16x8().Max(z).Min(cMax).AsInt8x16()
	hi := b.AsInt16x8().Max(z).Min(cMax).AsInt8x16()
	m := archsimd.Int64x2{}.SetElem(0, 0x0e0c0a0806040200).AsInt8x16()
	return lo.PermuteOrZero(m).AsInt64x2().InterleaveLo(hi.PermuteOrZero(m).AsInt64x2()).AsInt8x16()
}

//go:nosplit
func i16x8_narrow_i32x4_s(a, b v128) v128 {
	return a.AsInt32x4().SaturateToInt16Concat(b.AsInt32x4()).AsInt8x16()
}

//go:nosplit
func i16x8_narrow_i32x4_u(a, b v128) v128 {
	return a.AsInt32x4().SaturateToUint16Concat(b.AsInt32x4()).AsInt8x16()
}

//go:nosplit
func f64x2_add(a, b v128) v128 {
	return a.AsFloat64x2().Add(b.AsFloat64x2()).AsInt8x16()
}

//go:nosplit
func f32x4_add(a, b v128) v128 {
	return a.AsFloat32x4().Add(b.AsFloat32x4()).AsInt8x16()
}

//go:nosplit
func f64x2_sub(a, b v128) v128 {
	return a.AsFloat64x2().Sub(b.AsFloat64x2()).AsInt8x16()
}

//go:nosplit
func f32x4_sub(a, b v128) v128 {
	return a.AsFloat32x4().Sub(b.AsFloat32x4()).AsInt8x16()
}

//go:nosplit
func f64x2_mul(a, b v128) v128 {
	return a.AsFloat64x2().Mul(b.AsFloat64x2()).AsInt8x16()
}

//go:nosplit
func f32x4_mul(a, b v128) v128 {
	return a.AsFloat32x4().Mul(b.AsFloat32x4()).AsInt8x16()
}

//go:nosplit
func f64x2_div(a, b v128) v128 {
	return a.AsFloat64x2().Div(b.AsFloat64x2()).AsInt8x16()
}

//go:nosplit
func f32x4_div(a, b v128) v128 {
	return a.AsFloat32x4().Div(b.AsFloat32x4()).AsInt8x16()
}

//go:nosplit
func i16x8_bitmask(v v128) int32 {
	m := archsimd.Int64x2{}.SetElem(0, 0x0f0d0b0907050301).SetElem(1, -1).AsInt8x16()
	return int32(v.PermuteOrZero(m).Less(archsimd.Int8x16{}).ToBits())
}

//go:nosplit
func i32x4_bitmask(v v128) int32 {
	return int32(v.AsInt32x4().Less(archsimd.Int32x4{}).ToBits())
}

//go:nosplit
func i64x2_bitmask(v v128) int32 {
	return int32(v.AsInt64x2().Less(archsimd.Int64x2{}).ToBits())
}

//go:nosplit
func v128_load8x8_u(x uint64) v128 {
	return archsimd.Uint64x2{}.SetElem(0, x).AsUint8x16().ExtendLo8ToUint16().AsInt8x16()
}

//go:nosplit
func v128_load16x4_u(x uint64) v128 {
	return archsimd.Uint64x2{}.SetElem(0, x).AsUint16x8().ExtendLo4ToUint32().AsInt8x16()
}

//go:nosplit
func v128_load32x2_u(x uint64) v128 {
	return archsimd.Uint64x2{}.SetElem(0, x).AsUint32x4().ExtendLo2ToUint64().AsInt8x16()
}

//go:nosplit
func v128_load8x8_s(x uint64) v128 {
	return archsimd.Uint64x2{}.SetElem(0, x).AsInt8x16().ExtendLo8ToInt16().AsInt8x16()
}

//go:nosplit
func v128_load16x4_s(x uint64) v128 {
	return archsimd.Uint64x2{}.SetElem(0, x).AsInt16x8().ExtendLo4ToInt32().AsInt8x16()
}

//go:nosplit
func v128_load32x2_s(x uint64) v128 {
	return archsimd.Uint64x2{}.SetElem(0, x).AsInt32x4().ExtendLo2ToInt64().AsInt8x16()
}

//go:nosplit
func f32x4_eq(a, b v128) v128 {
	return a.AsFloat32x4().Equal(b.AsFloat32x4()).ToInt32x4().AsInt8x16()
}

//go:nosplit
func f32x4_ne(a, b v128) v128 {
	return a.AsFloat32x4().NotEqual(b.AsFloat32x4()).ToInt32x4().AsInt8x16()
}

//go:nosplit
func f32x4_lt(a, b v128) v128 {
	return a.AsFloat32x4().Less(b.AsFloat32x4()).ToInt32x4().AsInt8x16()
}

//go:nosplit
func f32x4_gt(a, b v128) v128 {
	return a.AsFloat32x4().Greater(b.AsFloat32x4()).ToInt32x4().AsInt8x16()
}

//go:nosplit
func f32x4_le(a, b v128) v128 {
	return a.AsFloat32x4().LessEqual(b.AsFloat32x4()).ToInt32x4().AsInt8x16()
}

//go:nosplit
func f32x4_ge(a, b v128) v128 {
	return a.AsFloat32x4().GreaterEqual(b.AsFloat32x4()).ToInt32x4().AsInt8x16()
}

//go:nosplit
func f32x4_min(a, b v128) v128 {
	x, y := a.AsFloat32x4(), b.AsFloat32x4()
	qnan := archsimd.BroadcastUint32x4(0x7fc00000)
	sign := qnan.ShiftAllLeft(9).AsInt8x16()
	m := x.Min(y).AsInt8x16().Or(a.Or(b).And(sign))
	nan := x.IsNaN().Or(y.IsNaN()).ToInt32x4().AsInt8x16()
	return m.AndNot(nan).Or(nan.And(qnan.AsInt8x16()))
}

//go:nosplit
func f32x4_max(a, b v128) v128 {
	x, y := a.AsFloat32x4(), b.AsFloat32x4()
	qnan := archsimd.BroadcastUint32x4(0x7fc00000)
	sign := qnan.ShiftAllLeft(9).AsInt8x16()
	m := x.Max(y).AsInt8x16().AndNot(sign.AndNot(a.And(b)))
	nan := x.IsNaN().Or(y.IsNaN()).ToInt32x4().AsInt8x16()
	return m.AndNot(nan).Or(nan.And(qnan.AsInt8x16()))
}

//go:nosplit
func f32x4_pmin(a, b v128) v128 {
	x, y := a.AsFloat32x4(), b.AsFloat32x4()
	return y.Merge(x, y.Less(x)).AsInt8x16()
}

//go:nosplit
func f32x4_pmax(a, b v128) v128 {
	x, y := a.AsFloat32x4(), b.AsFloat32x4()
	return y.Merge(x, x.Less(y)).AsInt8x16()
}

//go:nosplit
func f32x4_abs(a v128) v128 {
	return a.AsUint32x4().And(archsimd.BroadcastUint32x4(0x7fffffff)).AsInt8x16()
}

//go:nosplit
func f32x4_neg(a v128) v128 {
	return a.AsUint32x4().Xor(archsimd.BroadcastUint32x4(0x80000000)).AsInt8x16()
}

//go:nosplit
func f32x4_sqrt(a v128) v128 {
	return a.AsFloat32x4().Sqrt().AsInt8x16()
}

//go:nosplit
func f32x4_ceil(a v128) v128 {
	return a.AsFloat32x4().Ceil().AsInt8x16()
}

//go:nosplit
func f32x4_floor(a v128) v128 {
	return a.AsFloat32x4().Floor().AsInt8x16()
}

//go:nosplit
func f32x4_trunc(a v128) v128 {
	return a.AsFloat32x4().Trunc().AsInt8x16()
}

//go:nosplit
func f32x4_nearest(a v128) v128 {
	return a.AsFloat32x4().RoundToEven().AsInt8x16()
}

//go:nosplit
func f64x2_eq(a, b v128) v128 {
	return a.AsFloat64x2().Equal(b.AsFloat64x2()).ToInt64x2().AsInt8x16()
}

//go:nosplit
func f64x2_ne(a, b v128) v128 {
	return a.AsFloat64x2().NotEqual(b.AsFloat64x2()).ToInt64x2().AsInt8x16()
}

//go:nosplit
func f64x2_lt(a, b v128) v128 {
	return a.AsFloat64x2().Less(b.AsFloat64x2()).ToInt64x2().AsInt8x16()
}

//go:nosplit
func f64x2_gt(a, b v128) v128 {
	return a.AsFloat64x2().Greater(b.AsFloat64x2()).ToInt64x2().AsInt8x16()
}

//go:nosplit
func f64x2_le(a, b v128) v128 {
	return a.AsFloat64x2().LessEqual(b.AsFloat64x2()).ToInt64x2().AsInt8x16()
}

//go:nosplit
func f64x2_ge(a, b v128) v128 {
	return a.AsFloat64x2().GreaterEqual(b.AsFloat64x2()).ToInt64x2().AsInt8x16()
}

//go:nosplit
func f64x2_min(a, b v128) v128 {
	x, y := a.AsFloat64x2(), b.AsFloat64x2()
	qnan := archsimd.BroadcastUint64x2(0x7ff8000000000000)
	sign := qnan.ShiftAllLeft(12).AsInt8x16()
	m := x.Min(y).AsInt8x16().Or(a.Or(b).And(sign))
	nan := x.IsNaN().Or(y.IsNaN()).ToInt64x2().AsInt8x16()
	return m.AndNot(nan).Or(nan.And(qnan.AsInt8x16()))
}

//go:nosplit
func f64x2_max(a, b v128) v128 {
	x, y := a.AsFloat64x2(), b.AsFloat64x2()
	qnan := archsimd.BroadcastUint64x2(0x7ff8000000000000)
	sign := qnan.ShiftAllLeft(12).AsInt8x16()
	m := x.Max(y).AsInt8x16().AndNot(sign.AndNot(a.And(b)))
	nan := x.IsNaN().Or(y.IsNaN()).ToInt64x2().AsInt8x16()
	return m.AndNot(nan).Or(nan.And(qnan.AsInt8x16()))
}

//go:nosplit
func f64x2_pmin(a, b v128) v128 {
	x, y := a.AsFloat64x2(), b.AsFloat64x2()
	return y.Merge(x, y.Less(x)).AsInt8x16()
}

//go:nosplit
func f64x2_pmax(a, b v128) v128 {
	x, y := a.AsFloat64x2(), b.AsFloat64x2()
	return y.Merge(x, x.Less(y)).AsInt8x16()
}

//go:nosplit
func f64x2_abs(a v128) v128 {
	return a.AsUint64x2().And(archsimd.BroadcastUint64x2(0x7fffffffffffffff)).AsInt8x16()
}

//go:nosplit
func f64x2_neg(a v128) v128 {
	return a.AsUint64x2().Xor(archsimd.BroadcastUint64x2(0x8000000000000000)).AsInt8x16()
}

//go:nosplit
func f64x2_sqrt(a v128) v128 {
	return a.AsFloat64x2().Sqrt().AsInt8x16()
}

//go:nosplit
func f64x2_ceil(a v128) v128 {
	return a.AsFloat64x2().Ceil().AsInt8x16()
}

//go:nosplit
func f64x2_floor(a v128) v128 {
	return a.AsFloat64x2().Floor().AsInt8x16()
}

//go:nosplit
func f64x2_trunc(a v128) v128 {
	return a.AsFloat64x2().Trunc().AsInt8x16()
}

//go:nosplit
func f64x2_nearest(a v128) v128 {
	return a.AsFloat64x2().RoundToEven().AsInt8x16()
}

//go:nosplit
func f32x4_convert_i32x4_s(a v128) v128 {
	return a.AsInt32x4().ConvertToFloat32().AsInt8x16()
}

//go:nosplit
func f32x4_convert_i32x4_u(a v128) v128 {
	u := a.AsUint32x4()
	lo := u.And(archsimd.BroadcastUint32x4(0xffff)).AsInt32x4().ConvertToFloat32()
	hi := u.ShiftAllRight(16).AsInt32x4().ConvertToFloat32().Mul(archsimd.BroadcastFloat32x4(65536))
	return hi.Add(lo).AsInt8x16()
}

//go:nosplit
func i32x4_trunc_sat_f32x4_s(a v128) v128 {
	x := a.AsFloat32x4()
	i := x.Trunc().ConvertToInt32()
	ovf := x.GreaterEqual(archsimd.BroadcastFloat32x4(2147483648.0)).ToInt32x4()
	valid := x.Equal(x).ToInt32x4()
	return i.Xor(ovf).And(valid).AsInt8x16()
}

//go:nosplit
func i32x4_trunc_sat_f32x4_u(a v128) v128 {
	x := a.AsFloat32x4().Trunc()
	pos := x.AsInt32x4().And(x.Greater(archsimd.Float32x4{}).ToInt32x4()).AsFloat32x4()
	lo := pos.ConvertToInt32()
	hi := pos.Sub(archsimd.BroadcastFloat32x4(2147483648.0)).ConvertToInt32()
	return lo.Xor(hi.And(lo.ShiftAllRight(31))).Or(lo.And(hi).ShiftAllRight(31)).AsInt8x16()
}

//go:nosplit
func i32x4_trunc_sat_f64x2_s_zero(a v128) v128 {
	x := a.AsFloat64x2().Trunc()
	clamped := x.Min(archsimd.BroadcastFloat64x2(2147483647.0))
	clean := clamped.AsInt64x2().AndNot(x.IsNaN().ToInt64x2()).AsFloat64x2()
	return clean.ConvertToInt32().AsInt8x16()
}

//go:nosplit
func i32x4_trunc_sat_f64x2_u_zero(a v128) v128 {
	x := a.AsFloat64x2().Trunc()
	pos := x.AsInt64x2().And(x.Greater(archsimd.Float64x2{}).ToInt64x2()).AsFloat64x2().Min(archsimd.BroadcastFloat64x2(4294967295.0))
	biased := pos.Add(archsimd.BroadcastFloat64x2(4503599627370496.0)).AsInt8x16()
	m := archsimd.Int64x2{}.SetElem(0, 0x0b0a090803020100).SetElem(1, -1).AsInt8x16()
	return biased.PermuteOrZero(m)
}

//go:nosplit
func f32x4_demote_f64x2_zero(a v128) v128 {
	return a.AsFloat64x2().ConvertToFloat32().AsInt8x16()
}

//go:nosplit
func f64x2_promote_low_f32x4(a v128) v128 {
	return a.AsFloat32x4().ConvertToFloat64().GetLo().AsInt8x16()
}

//go:nosplit
func f64x2_convert_low_i32x4_s(a v128) v128 {
	return a.AsInt32x4().ConvertToFloat64().GetLo().AsInt8x16()
}

//go:nosplit
func f64x2_convert_low_i32x4_u(a v128) v128 {
	u := a.AsUint32x4().ExtendLo2ToUint64()
	magic := archsimd.BroadcastUint64x2(0x4330000000000000)
	return u.Or(magic).AsFloat64x2().Sub(magic.AsFloat64x2()).AsInt8x16()
}

//go:nosplit
func i16x8_mul(a, b v128) v128 {
	return a.AsInt16x8().Mul(b.AsInt16x8()).AsInt8x16()
}

//go:nosplit
func i32x4_mul(a, b v128) v128 {
	return a.AsInt32x4().Mul(b.AsInt32x4()).AsInt8x16()
}

//go:nosplit
func i16x8_q15mulr_sat_s(a, b v128) v128 {
	p := a.AsInt16x8().ExtendToInt32().Mul(b.AsInt16x8().ExtendToInt32())
	p = p.Add(archsimd.BroadcastInt32x8(0x4000)).ShiftAllRight(15)
	return p.GetLo().SaturateToInt16Concat(p.GetHi()).AsInt8x16()
}

//go:nosplit
func i8x16_add_sat_s(a, b v128) v128 {
	return a.AddSaturated(b)
}

//go:nosplit
func i8x16_sub_sat_s(a, b v128) v128 {
	return a.SubSaturated(b)
}

//go:nosplit
func i16x8_add_sat_s(a, b v128) v128 {
	return a.AsInt16x8().AddSaturated(b.AsInt16x8()).AsInt8x16()
}

//go:nosplit
func i16x8_sub_sat_s(a, b v128) v128 {
	return a.AsInt16x8().SubSaturated(b.AsInt16x8()).AsInt8x16()
}

//go:nosplit
func i16x8_extadd_pairwise_i8x16_s(a v128) v128 {
	return archsimd.BroadcastUint8x16(1).DotProductPairsSaturated(a).AsInt8x16()
}

//go:nosplit
func i16x8_extadd_pairwise_i8x16_u(a v128) v128 {
	x := a.AsUint16x8()
	return x.And(archsimd.BroadcastUint32x4(0x00ff00ff).AsUint16x8()).Add(x.ShiftAllRight(8)).AsInt8x16()
}

//go:nosplit
func i32x4_extadd_pairwise_i16x8_s(a v128) v128 {
	return a.AsInt16x8().DotProductPairs(archsimd.BroadcastInt32x4(0x00010001).AsInt16x8()).AsInt8x16()
}

//go:nosplit
func i32x4_extadd_pairwise_i16x8_u(a v128) v128 {
	x := a.AsUint32x4()
	return x.And(archsimd.BroadcastUint32x4(0x0000ffff)).Add(x.ShiftAllRight(16)).AsInt8x16()
}

//go:nosplit
func i16x8_extmul_low_i8x16_u(a, b v128) v128 {
	return a.AsUint8x16().ExtendLo8ToUint16().Mul(b.AsUint8x16().ExtendLo8ToUint16()).AsInt8x16()
}

//go:nosplit
func i32x4_extmul_low_i16x8_u(a, b v128) v128 {
	return a.AsUint16x8().ExtendLo4ToUint32().Mul(b.AsUint16x8().ExtendLo4ToUint32()).AsInt8x16()
}

//go:nosplit
func i64x2_extmul_low_i32x4_u(a, b v128) v128 {
	x := a.AsUint32x4().InterleaveLo(a.AsUint32x4())
	y := b.AsUint32x4().InterleaveLo(b.AsUint32x4())
	return x.MulEvenWiden(y).AsInt8x16()
}

//go:nosplit
func i16x8_extmul_low_i8x16_s(a, b v128) v128 {
	return a.ExtendLo8ToInt16().Mul(b.ExtendLo8ToInt16()).AsInt8x16()
}

//go:nosplit
func i32x4_extmul_low_i16x8_s(a, b v128) v128 {
	return a.AsInt16x8().ExtendLo4ToInt32().Mul(b.AsInt16x8().ExtendLo4ToInt32()).AsInt8x16()
}

//go:nosplit
func i64x2_extmul_low_i32x4_s(a, b v128) v128 {
	x := a.AsInt32x4().InterleaveLo(a.AsInt32x4())
	y := b.AsInt32x4().InterleaveLo(b.AsInt32x4())
	return x.MulEvenWiden(y).AsInt8x16()
}

//go:nosplit
func i16x8_extmul_high_i8x16_u(a, b v128) v128 {
	x := a.AsUint8x16().ConcatShiftBytesRight(8, a.AsUint8x16()).ExtendLo8ToUint16()
	y := b.AsUint8x16().ConcatShiftBytesRight(8, b.AsUint8x16()).ExtendLo8ToUint16()
	return x.Mul(y).AsInt8x16()
}

//go:nosplit
func i32x4_extmul_high_i16x8_u(a, b v128) v128 {
	x, y := a.AsUint16x8(), b.AsUint16x8()
	return x.Mul(y).InterleaveHi(x.MulHigh(y)).AsInt8x16()
}

//go:nosplit
func i64x2_extmul_high_i32x4_u(a, b v128) v128 {
	x := a.AsUint32x4().InterleaveHi(a.AsUint32x4())
	y := b.AsUint32x4().InterleaveHi(b.AsUint32x4())
	return x.MulEvenWiden(y).AsInt8x16()
}

//go:nosplit
func i16x8_extmul_high_i8x16_s(a, b v128) v128 {
	x := a.AsUint8x16().ConcatShiftBytesRight(8, a.AsUint8x16()).AsInt8x16().ExtendLo8ToInt16()
	y := b.AsUint8x16().ConcatShiftBytesRight(8, b.AsUint8x16()).AsInt8x16().ExtendLo8ToInt16()
	return x.Mul(y).AsInt8x16()
}

//go:nosplit
func i32x4_extmul_high_i16x8_s(a, b v128) v128 {
	x, y := a.AsInt16x8(), b.AsInt16x8()
	return x.Mul(y).InterleaveHi(x.MulHigh(y)).AsInt8x16()
}

//go:nosplit
func i64x2_extmul_high_i32x4_s(a, b v128) v128 {
	x := a.AsInt32x4().InterleaveHi(a.AsInt32x4())
	y := b.AsInt32x4().InterleaveHi(b.AsInt32x4())
	return x.MulEvenWiden(y).AsInt8x16()
}

//go:nosplit
func i32x4_dot_i16x8_s(a, b v128) v128 {
	return a.AsInt16x8().DotProductPairs(b.AsInt16x8()).AsInt8x16()
}

//go:nosplit
func i64x2_abs(a v128) v128 {
	x := a.AsInt64x2()
	m := x.Less(archsimd.Int64x2{}).ToInt64x2()
	return x.Xor(m).Sub(m).AsInt8x16()
}
