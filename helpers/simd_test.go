package helpers

import (
	"math"
	"testing"
)

// Independent scalar references for the SIMD helpers.
// These deliberately recompute lane semantics from the Wasm spec,
// rather than reusing helper code.

func clamp(x, lo, hi int) int {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func boolByte[T comparable](b bool, _ T) uint64 {
	if b {
		return math.MaxUint64
	}
	return 0
}

// Lane accessors. The oracle reasons in Wasm memory order: lane 0 is the
// low bits of the first word, 16 little-endian bytes overall.

func getLane(v v128, bits, i int) uint64 {
	w := v.lo
	if i*bits >= 64 {
		w, i = v.hi, i-64/bits
	}
	if bits == 64 {
		return w
	}
	return w >> (bits * i) & (uint64(1)<<bits - 1)
}

func putLane(v *v128, bits, i int, x uint64) {
	w := &v.lo
	if i*bits >= 64 {
		w, i = &v.hi, i-64/bits
	}
	if bits == 64 {
		*w = x
		return
	}
	mask := (uint64(1)<<bits - 1) << (bits * i)
	*w = *w&^mask | x<<(bits*i)&mask
}

func makeVec(bits int, lanes ...uint64) (v v128) {
	for i, x := range lanes {
		putLane(&v, bits, i, x)
	}
	return
}

// vecOf and bytesOf convert between a v128 and its 16 memory-order bytes.
func vecOf(b [16]uint8) (v v128) {
	for i, x := range b {
		putLane(&v, 8, i, uint64(x))
	}
	return
}

func bytesOf(v v128) (b [16]uint8) {
	for i := range b {
		b[i] = uint8(getLane(v, 8, i))
	}
	return
}

// Exhaustive i8x16 binary operations: all 256x256 inputs, in both
// lane-varying positions.
func TestI8x16Binary_exhaustive(t *testing.T) {
	ops := []struct {
		name string
		fn   func(a, b v128) v128
		ref  func(x, y uint8) uint8
	}{
		{"add", i8x16_add, func(x, y uint8) uint8 { return x + y }},
		{"sub", i8x16_sub, func(x, y uint8) uint8 { return x - y }},
		{"add_sat_s", i8x16_add_sat_s, func(x, y uint8) uint8 {
			return uint8(clamp(int(int8(x))+int(int8(y)), -128, 127))
		}},
		{"add_sat_u", i8x16_add_sat_u, func(x, y uint8) uint8 {
			return uint8(clamp(int(x)+int(y), 0, 255))
		}},
		{"sub_sat_s", i8x16_sub_sat_s, func(x, y uint8) uint8 {
			return uint8(clamp(int(int8(x))-int(int8(y)), -128, 127))
		}},
		{"sub_sat_u", i8x16_sub_sat_u, func(x, y uint8) uint8 {
			return uint8(clamp(int(x)-int(y), 0, 255))
		}},
		{"min_s", i8x16_min_s, func(x, y uint8) uint8 {
			return uint8(min(int(int8(x)), int(int8(y))))
		}},
		{"min_u", i8x16_min_u, func(x, y uint8) uint8 {
			return uint8(min(int(x), int(y)))
		}},
		{"max_s", i8x16_max_s, func(x, y uint8) uint8 {
			return uint8(max(int(int8(x)), int(int8(y))))
		}},
		{"max_u", i8x16_max_u, func(x, y uint8) uint8 {
			return uint8(max(int(x), int(y)))
		}},
		{"avgr_u", i8x16_avgr_u, func(x, y uint8) uint8 {
			return uint8((int(x) + int(y) + 1) / 2)
		}},
		{"eq", i8x16_eq, func(x, y uint8) uint8 { return uint8(boolByte(x == y, x)) }},
		{"ne", i8x16_ne, func(x, y uint8) uint8 { return uint8(boolByte(x != y, x)) }},
		{"lt_s", i8x16_lt_s, func(x, y uint8) uint8 { return uint8(boolByte(int8(x) < int8(y), x)) }},
		{"lt_u", i8x16_lt_u, func(x, y uint8) uint8 { return uint8(boolByte(x < y, x)) }},
		{"gt_s", i8x16_gt_s, func(x, y uint8) uint8 { return uint8(boolByte(int8(x) > int8(y), x)) }},
		{"gt_u", i8x16_gt_u, func(x, y uint8) uint8 { return uint8(boolByte(x > y, x)) }},
		{"le_s", i8x16_le_s, func(x, y uint8) uint8 { return uint8(boolByte(int8(x) <= int8(y), x)) }},
		{"le_u", i8x16_le_u, func(x, y uint8) uint8 { return uint8(boolByte(x <= y, x)) }},
		{"ge_s", i8x16_ge_s, func(x, y uint8) uint8 { return uint8(boolByte(int8(x) >= int8(y), x)) }},
		{"ge_u", i8x16_ge_u, func(x, y uint8) uint8 { return uint8(boolByte(x >= y, x)) }},
	}
	for _, op := range ops {
		t.Run(op.name, func(t *testing.T) {
			for x := 0; x < 256; x++ {
				for y0 := 0; y0 < 256; y0 += 16 {
					var ab, bb [16]uint8
					for i := range ab {
						ab[i] = uint8(x)
						bb[i] = uint8(y0 + i)
					}
					a, b := vecOf(ab), vecOf(bb)
					// x splat vs varying lanes, and vice versa.
					r := bytesOf(op.fn(a, b))
					s := bytesOf(op.fn(b, a))
					for i := range r {
						if want := op.ref(ab[i], bb[i]); r[i] != want {
							t.Fatalf("%s(%d, %d): got %d, want %d", op.name, int8(ab[i]), int8(bb[i]), r[i], want)
						}
						if want := op.ref(bb[i], ab[i]); s[i] != want {
							t.Fatalf("%s(%d, %d): got %d, want %d", op.name, int8(bb[i]), int8(ab[i]), s[i], want)
						}
					}
				}
			}
		})
	}
}

// Exhaustive i8x16 unary operations: all 256 inputs in every lane.
func TestI8x16Unary_exhaustive(t *testing.T) {
	ops := []struct {
		name string
		fn   func(a v128) v128
		ref  func(x uint8) uint8
	}{
		{"abs", i8x16_abs, func(x uint8) uint8 {
			if int8(x) < 0 {
				return uint8(-int8(x))
			}
			return x
		}},
		{"neg", i8x16_neg, func(x uint8) uint8 { return uint8(-int8(x)) }},
		{"popcnt", i8x16_popcnt, func(x uint8) uint8 {
			var n uint8
			for ; x != 0; x >>= 1 {
				n += x & 1
			}
			return n
		}},
	}
	for _, op := range ops {
		t.Run(op.name, func(t *testing.T) {
			for x0 := 0; x0 < 256; x0 += 16 {
				var ab [16]uint8
				for i := range ab {
					ab[i] = uint8(x0 + i)
				}
				r := bytesOf(op.fn(vecOf(ab)))
				for i := range r {
					if want := op.ref(ab[i]); r[i] != want {
						t.Fatalf("%s(%d): got %d, want %d", op.name, int8(ab[i]), r[i], want)
					}
				}
			}
		})
	}
}

// Shifts: all lane values (exhaustive for i8, boundary-dense for wider),
// with all shift counts 0..255 to pin count masking.
func TestShifts(t *testing.T) {
	type shiftOp struct {
		name string
		fn   func(a v128, y int32) v128
		bits int
		ref  func(x uint64, c int) uint64
	}
	sext := func(x uint64, bits int) int64 {
		return int64(x<<(64-bits)) >> (64 - bits)
	}
	ops := []shiftOp{
		{"i8x16_shl", i8x16_shl, 8, func(x uint64, c int) uint64 { return x << (c % 8) }},
		{"i8x16_shr_s", i8x16_shr_s, 8, func(x uint64, c int) uint64 { return uint64(sext(x, 8) >> (c % 8)) }},
		{"i8x16_shr_u", i8x16_shr_u, 8, func(x uint64, c int) uint64 { return x >> (c % 8) }},
		{"i16x8_shl", i16x8_shl, 16, func(x uint64, c int) uint64 { return x << (c % 16) }},
		{"i16x8_shr_s", i16x8_shr_s, 16, func(x uint64, c int) uint64 { return uint64(sext(x, 16) >> (c % 16)) }},
		{"i16x8_shr_u", i16x8_shr_u, 16, func(x uint64, c int) uint64 { return x >> (c % 16) }},
		{"i32x4_shl", i32x4_shl, 32, func(x uint64, c int) uint64 { return x << (c % 32) }},
		{"i32x4_shr_s", i32x4_shr_s, 32, func(x uint64, c int) uint64 { return uint64(sext(x, 32) >> (c % 32)) }},
		{"i32x4_shr_u", i32x4_shr_u, 32, func(x uint64, c int) uint64 { return x >> (c % 32) }},
		{"i64x2_shl", i64x2_shl, 64, func(x uint64, c int) uint64 { return x << (c % 64) }},
		{"i64x2_shr_s", i64x2_shr_s, 64, func(x uint64, c int) uint64 { return uint64(sext(x, 64) >> (c % 64)) }},
		{"i64x2_shr_u", i64x2_shr_u, 64, func(x uint64, c int) uint64 { return x >> (c % 64) }},
	}
	for _, op := range ops {
		t.Run(op.name, func(t *testing.T) {
			lanes := 128 / op.bits
			mask := uint64(1)<<op.bits - 1
			if op.bits == 64 {
				mask = math.MaxUint64
			}
			for _, vals := range laneDomains(op.bits) {
				var a v128
				for i := 0; i < lanes; i++ {
					putLane(&a, op.bits, i, vals[i%len(vals)])
				}
				for c := 0; c < 256; c++ {
					r := op.fn(a, int32(c))
					for i := 0; i < lanes; i++ {
						x := getLane(a, op.bits, i)
						if got, want := getLane(r, op.bits, i), op.ref(x, c)&mask; got != want {
							t.Fatalf("%s(%#x, %d): got %#x, want %#x", op.name, x, c, got, want)
						}
					}
				}
			}
		})
	}
}

// laneDomains returns groups of boundary-dense lane values for a lane width.
func laneDomains(bits int) [][]uint64 {
	var vals []uint64
	if bits == 8 {
		for x := 0; x < 256; x++ {
			vals = append(vals, uint64(x))
		}
	} else {
		mask := uint64(1)<<bits - 1
		if bits == 64 {
			mask = math.MaxUint64
		}
		sign := uint64(1) << (bits - 1)
		for _, base := range []uint64{0, 1, 2, 3, 5, sign >> 1, sign - 1, sign, sign + 1, mask - 2, mask - 1, mask, 0x55 & mask, 0xaa & mask, 0x5555 & mask, 0xaaaa & mask, 0x12345678 & mask, 0xfedcba9876543210 & mask} {
			for _, d := range []uint64{0, 1, mask} {
				vals = append(vals, (base+d)&mask)
			}
		}
		for k := 0; k < bits; k += 3 {
			vals = append(vals, uint64(1)<<k, (uint64(1)<<k)-1, mask^(uint64(1)<<k))
		}
	}
	// Group into vectors' worth of lanes.
	lanes := 128 / bits
	var groups [][]uint64
	for i := 0; i < len(vals); i += lanes {
		groups = append(groups, vals[i:min(i+lanes, len(vals))])
	}
	// One group with all values, cycled through lanes at every offset.
	for o := 0; o < lanes; o++ {
		groups = append(groups, append(vals[o:len(vals):len(vals)], vals[:o]...))
	}
	return groups
}

// Wider integer binary ops over boundary-dense domains.
func TestIntBinary_domains(t *testing.T) {
	type binOp struct {
		name string
		fn   func(a, b v128) v128
		bits int
		ref  func(x, y uint64) uint64
	}
	sext := func(x uint64, bits int) int64 {
		return int64(x<<(64-bits)) >> (64 - bits)
	}
	satS := func(x int64, bits int) uint64 {
		lo := int64(-1) << (bits - 1)
		hi := -lo - 1
		if x < lo {
			x = lo
		}
		if x > hi {
			x = hi
		}
		return uint64(x)
	}
	satU := func(x int64, bits int) uint64 {
		hi := int64(1)<<bits - 1
		if x < 0 {
			x = 0
		}
		if x > hi {
			x = hi
		}
		return uint64(x)
	}
	ops := []binOp{
		{"i16x8_add", i16x8_add, 16, func(x, y uint64) uint64 { return x + y }},
		{"i16x8_sub", i16x8_sub, 16, func(x, y uint64) uint64 { return x - y }},
		{"i16x8_mul", i16x8_mul, 16, func(x, y uint64) uint64 { return x * y }},
		{"i16x8_add_sat_s", i16x8_add_sat_s, 16, func(x, y uint64) uint64 { return satS(sext(x, 16)+sext(y, 16), 16) }},
		{"i16x8_add_sat_u", i16x8_add_sat_u, 16, func(x, y uint64) uint64 { return satU(int64(x)+int64(y), 16) }},
		{"i16x8_sub_sat_s", i16x8_sub_sat_s, 16, func(x, y uint64) uint64 { return satS(sext(x, 16)-sext(y, 16), 16) }},
		{"i16x8_sub_sat_u", i16x8_sub_sat_u, 16, func(x, y uint64) uint64 { return satU(int64(x)-int64(y), 16) }},
		{"i16x8_min_s", i16x8_min_s, 16, func(x, y uint64) uint64 { return uint64(uint16(min(sext(x, 16), sext(y, 16)))) }},
		{"i16x8_min_u", i16x8_min_u, 16, func(x, y uint64) uint64 { return min(x, y) }},
		{"i16x8_max_s", i16x8_max_s, 16, func(x, y uint64) uint64 { return uint64(uint16(max(sext(x, 16), sext(y, 16)))) }},
		{"i16x8_max_u", i16x8_max_u, 16, func(x, y uint64) uint64 { return max(x, y) }},
		{"i16x8_avgr_u", i16x8_avgr_u, 16, func(x, y uint64) uint64 { return (x + y + 1) / 2 }},
		{"i16x8_q15mulr_sat_s", i16x8_q15mulr_sat_s, 16, func(x, y uint64) uint64 {
			return satS((sext(x, 16)*sext(y, 16)+0x4000)>>15, 16)
		}},
		{"i16x8_eq", i16x8_eq, 16, func(x, y uint64) uint64 { return boolByte(x == y, x) & 0xffff }},
		{"i16x8_ne", i16x8_ne, 16, func(x, y uint64) uint64 { return boolByte(x != y, x) & 0xffff }},
		{"i16x8_lt_s", i16x8_lt_s, 16, func(x, y uint64) uint64 { return boolByte(sext(x, 16) < sext(y, 16), x) & 0xffff }},
		{"i16x8_lt_u", i16x8_lt_u, 16, func(x, y uint64) uint64 { return boolByte(x < y, x) & 0xffff }},
		{"i16x8_gt_s", i16x8_gt_s, 16, func(x, y uint64) uint64 { return boolByte(sext(x, 16) > sext(y, 16), x) & 0xffff }},
		{"i16x8_gt_u", i16x8_gt_u, 16, func(x, y uint64) uint64 { return boolByte(x > y, x) & 0xffff }},
		{"i16x8_le_s", i16x8_le_s, 16, func(x, y uint64) uint64 { return boolByte(sext(x, 16) <= sext(y, 16), x) & 0xffff }},
		{"i16x8_le_u", i16x8_le_u, 16, func(x, y uint64) uint64 { return boolByte(x <= y, x) & 0xffff }},
		{"i16x8_ge_s", i16x8_ge_s, 16, func(x, y uint64) uint64 { return boolByte(sext(x, 16) >= sext(y, 16), x) & 0xffff }},
		{"i16x8_ge_u", i16x8_ge_u, 16, func(x, y uint64) uint64 { return boolByte(x >= y, x) & 0xffff }},

		{"i32x4_add", i32x4_add, 32, func(x, y uint64) uint64 { return uint64(uint32(x + y)) }},
		{"i32x4_sub", i32x4_sub, 32, func(x, y uint64) uint64 { return uint64(uint32(x - y)) }},
		{"i32x4_mul", i32x4_mul, 32, func(x, y uint64) uint64 { return uint64(uint32(x * y)) }},
		{"i32x4_min_s", i32x4_min_s, 32, func(x, y uint64) uint64 { return uint64(uint32(min(sext(x, 32), sext(y, 32)))) }},
		{"i32x4_min_u", i32x4_min_u, 32, func(x, y uint64) uint64 { return min(x, y) }},
		{"i32x4_max_s", i32x4_max_s, 32, func(x, y uint64) uint64 { return uint64(uint32(max(sext(x, 32), sext(y, 32)))) }},
		{"i32x4_max_u", i32x4_max_u, 32, func(x, y uint64) uint64 { return max(x, y) }},
		{"i32x4_eq", i32x4_eq, 32, func(x, y uint64) uint64 { return boolByte(x == y, x) & 0xffffffff }},
		{"i32x4_ne", i32x4_ne, 32, func(x, y uint64) uint64 { return boolByte(x != y, x) & 0xffffffff }},
		{"i32x4_lt_s", i32x4_lt_s, 32, func(x, y uint64) uint64 { return boolByte(sext(x, 32) < sext(y, 32), x) & 0xffffffff }},
		{"i32x4_lt_u", i32x4_lt_u, 32, func(x, y uint64) uint64 { return boolByte(x < y, x) & 0xffffffff }},
		{"i32x4_gt_s", i32x4_gt_s, 32, func(x, y uint64) uint64 { return boolByte(sext(x, 32) > sext(y, 32), x) & 0xffffffff }},
		{"i32x4_gt_u", i32x4_gt_u, 32, func(x, y uint64) uint64 { return boolByte(x > y, x) & 0xffffffff }},
		{"i32x4_le_s", i32x4_le_s, 32, func(x, y uint64) uint64 { return boolByte(sext(x, 32) <= sext(y, 32), x) & 0xffffffff }},
		{"i32x4_le_u", i32x4_le_u, 32, func(x, y uint64) uint64 { return boolByte(x <= y, x) & 0xffffffff }},
		{"i32x4_ge_s", i32x4_ge_s, 32, func(x, y uint64) uint64 { return boolByte(sext(x, 32) >= sext(y, 32), x) & 0xffffffff }},
		{"i32x4_ge_u", i32x4_ge_u, 32, func(x, y uint64) uint64 { return boolByte(x >= y, x) & 0xffffffff }},

		{"i64x2_add", i64x2_add, 64, func(x, y uint64) uint64 { return x + y }},
		{"i64x2_sub", i64x2_sub, 64, func(x, y uint64) uint64 { return x - y }},
		{"i64x2_mul", i64x2_mul, 64, func(x, y uint64) uint64 { return x * y }},
		{"i64x2_eq", i64x2_eq, 64, func(x, y uint64) uint64 { return boolByte(x == y, x) }},
		{"i64x2_ne", i64x2_ne, 64, func(x, y uint64) uint64 { return boolByte(x != y, x) }},
		{"i64x2_lt_s", i64x2_lt_s, 64, func(x, y uint64) uint64 { return boolByte(int64(x) < int64(y), x) }},
		{"i64x2_gt_s", i64x2_gt_s, 64, func(x, y uint64) uint64 { return boolByte(int64(x) > int64(y), x) }},
		{"i64x2_le_s", i64x2_le_s, 64, func(x, y uint64) uint64 { return boolByte(int64(x) <= int64(y), x) }},
		{"i64x2_ge_s", i64x2_ge_s, 64, func(x, y uint64) uint64 { return boolByte(int64(x) >= int64(y), x) }},
	}
	for _, op := range ops {
		t.Run(op.name, func(t *testing.T) {
			lanes := 128 / op.bits
			groups := laneDomains(op.bits)
			for _, xs := range groups {
				for _, ys := range groups {
					var a, b v128
					for i := 0; i < lanes; i++ {
						putLane(&a, op.bits, i, xs[i%len(xs)])
						putLane(&b, op.bits, i, ys[i%len(ys)])
					}
					r := op.fn(a, b)
					for i := 0; i < lanes; i++ {
						x := getLane(a, op.bits, i)
						y := getLane(b, op.bits, i)
						mask := uint64(1)<<(op.bits-1)<<1 - 1
						if got, want := getLane(r, op.bits, i), op.ref(x, y)&mask; got != want {
							t.Fatalf("%s(%#x, %#x): got %#x, want %#x", op.name, x, y, got, want)
						}
					}
				}
			}
		})
	}
}

// Wider integer unary ops over boundary-dense domains.
func TestIntUnary_domains(t *testing.T) {
	type unOp struct {
		name string
		fn   func(a v128) v128
		bits int
		ref  func(x uint64) uint64
	}
	sext := func(x uint64, bits int) int64 {
		return int64(x<<(64-bits)) >> (64 - bits)
	}
	ops := []unOp{
		{"i16x8_abs", i16x8_abs, 16, func(x uint64) uint64 {
			if v := sext(x, 16); v < 0 {
				return uint64(uint16(-v))
			}
			return x
		}},
		{"i16x8_neg", i16x8_neg, 16, func(x uint64) uint64 { return uint64(uint16(-sext(x, 16))) }},
		{"i32x4_abs", i32x4_abs, 32, func(x uint64) uint64 {
			if v := sext(x, 32); v < 0 {
				return uint64(uint32(-v))
			}
			return x
		}},
		{"i32x4_neg", i32x4_neg, 32, func(x uint64) uint64 { return uint64(uint32(-sext(x, 32))) }},
		{"i64x2_abs", i64x2_abs, 64, func(x uint64) uint64 {
			if v := int64(x); v < 0 {
				return uint64(-v)
			}
			return x
		}},
		{"i64x2_neg", i64x2_neg, 64, func(x uint64) uint64 { return uint64(-int64(x)) }},
	}
	for _, op := range ops {
		t.Run(op.name, func(t *testing.T) {
			lanes := 128 / op.bits
			for _, xs := range laneDomains(op.bits) {
				var a v128
				for i := 0; i < lanes; i++ {
					putLane(&a, op.bits, i, xs[i%len(xs)])
				}
				r := op.fn(a)
				for i := 0; i < lanes; i++ {
					x := getLane(a, op.bits, i)
					if got, want := getLane(r, op.bits, i), op.ref(x); got != want {
						t.Fatalf("%s(%#x): got %#x, want %#x", op.name, x, got, want)
					}
				}
			}
		})
	}
}

// Narrowing: exhaustive over all 65536 i16 source lanes (both operands),
// and boundary domains for i32.
func TestNarrow(t *testing.T) {
	t.Run("i8x16_narrow_i16x8", func(t *testing.T) {
		for x0 := 0; x0 < 65536; x0 += 8 {
			var a, b v128
			for i := 0; i < 8; i++ {
				putLane(&a, 16, i, uint64(x0+i))
				putLane(&b, 16, i, uint64(65535-x0-i))
			}
			s := bytesOf(i8x16_narrow_i16x8_s(a, b))
			u := bytesOf(i8x16_narrow_i16x8_u(a, b))
			for i := 0; i < 16; i++ {
				src := a
				if i >= 8 {
					src = b
				}
				x := int(int16(getLane(src, 16, i%8)))
				if got, want := s[i], uint8(clamp(x, -128, 127)); got != want {
					t.Fatalf("narrow_s(%d) lane %d: got %d, want %d", x, i, got, want)
				}
				if got, want := u[i], uint8(clamp(x, 0, 255)); got != want {
					t.Fatalf("narrow_u(%d) lane %d: got %d, want %d", x, i, got, want)
				}
			}
		}
	})
	t.Run("i16x8_narrow_i32x4", func(t *testing.T) {
		for _, xs := range laneDomains(32) {
			var a, b v128
			for i := 0; i < 4; i++ {
				putLane(&a, 32, i, xs[i%len(xs)])
				putLane(&b, 32, i, ^xs[i%len(xs)])
			}
			s := i16x8_narrow_i32x4_s(a, b)
			u := i16x8_narrow_i32x4_u(a, b)
			for i := 0; i < 8; i++ {
				src := a
				if i >= 4 {
					src = b
				}
				x := int(int32(getLane(src, 32, i%4)))
				if got, want := getLane(s, 16, i), uint64(uint16(clamp(x, -32768, 32767))); got != want {
					t.Fatalf("narrow_s(%d) lane %d: got %d, want %d", x, i, got, want)
				}
				if got, want := getLane(u, 16, i), uint64(uint16(clamp(x, 0, 65535))); got != want {
					t.Fatalf("narrow_u(%d) lane %d: got %d, want %d", x, i, got, want)
				}
			}
		}
	})
}

// Widening conversions, pairwise adds, extended multiplies: lane-distinct
// patterns with sign coverage, verified against scalar recomputation.
func TestWiden(t *testing.T) {
	var ab, bb [16]uint8
	for i := range ab {
		ab[i] = uint8(0x11*i + 3) // distinct, sign bit varies
		bb[i] = uint8(0xf5 - 7*i)
	}
	a, b := vecOf(ab), vecOf(bb)

	for i := 0; i < 8; i++ {
		lo := i16x8_extend_low_i8x16_s(a)
		hi := i16x8_extend_high_i8x16_s(a)
		ul := i16x8_extend_low_i8x16_u(a)
		uh := i16x8_extend_high_i8x16_u(a)
		if got, want := int16(getLane(lo, 16, i)), int16(int8(ab[i])); got != want {
			t.Errorf("extend_low_s lane %d: got %d, want %d", i, got, want)
		}
		if got, want := int16(getLane(hi, 16, i)), int16(int8(ab[8+i])); got != want {
			t.Errorf("extend_high_s lane %d: got %d, want %d", i, got, want)
		}
		if got, want := getLane(ul, 16, i), uint64(ab[i]); got != want {
			t.Errorf("extend_low_u lane %d: got %d, want %d", i, got, want)
		}
		if got, want := getLane(uh, 16, i), uint64(ab[8+i]); got != want {
			t.Errorf("extend_high_u lane %d: got %d, want %d", i, got, want)
		}

		ml := i16x8_extmul_low_i8x16_s(a, b)
		mh := i16x8_extmul_high_i8x16_s(a, b)
		mlu := i16x8_extmul_low_i8x16_u(a, b)
		mhu := i16x8_extmul_high_i8x16_u(a, b)
		if got, want := int16(getLane(ml, 16, i)), int16(int8(ab[i]))*int16(int8(bb[i])); got != want {
			t.Errorf("extmul_low_s lane %d: got %d, want %d", i, got, want)
		}
		if got, want := int16(getLane(mh, 16, i)), int16(int8(ab[8+i]))*int16(int8(bb[8+i])); got != want {
			t.Errorf("extmul_high_s lane %d: got %d, want %d", i, got, want)
		}
		if got, want := getLane(mlu, 16, i), uint64(uint16(ab[i])*uint16(bb[i])); got != want {
			t.Errorf("extmul_low_u lane %d: got %d, want %d", i, got, want)
		}
		if got, want := getLane(mhu, 16, i), uint64(uint16(ab[8+i])*uint16(bb[8+i])); got != want {
			t.Errorf("extmul_high_u lane %d: got %d, want %d", i, got, want)
		}

		ep := i16x8_extadd_pairwise_i8x16_s(a)
		epu := i16x8_extadd_pairwise_i8x16_u(a)
		if got, want := int16(getLane(ep, 16, i)), int16(int8(ab[2*i]))+int16(int8(ab[2*i+1])); got != want {
			t.Errorf("extadd_pairwise_s lane %d: got %d, want %d", i, got, want)
		}
		if got, want := getLane(epu, 16, i), uint64(uint16(ab[2*i])+uint16(ab[2*i+1])); got != want {
			t.Errorf("extadd_pairwise_u lane %d: got %d, want %d", i, got, want)
		}
	}

	for i := 0; i < 4; i++ {
		x16 := func(v v128, l int) int32 { return int32(int16(getLane(v, 16, l))) }
		u16 := func(v v128, l int) uint32 { return uint32(getLane(v, 16, l)) }
		lo := i32x4_extend_low_i16x8_s(a)
		hi := i32x4_extend_high_i16x8_s(a)
		ul := i32x4_extend_low_i16x8_u(a)
		uh := i32x4_extend_high_i16x8_u(a)
		if got, want := int32(getLane(lo, 32, i)), x16(a, i); got != want {
			t.Errorf("i32 extend_low_s lane %d: got %d, want %d", i, got, want)
		}
		if got, want := int32(getLane(hi, 32, i)), x16(a, 4+i); got != want {
			t.Errorf("i32 extend_high_s lane %d: got %d, want %d", i, got, want)
		}
		if got, want := uint32(getLane(ul, 32, i)), u16(a, i); got != want {
			t.Errorf("i32 extend_low_u lane %d: got %d, want %d", i, got, want)
		}
		if got, want := uint32(getLane(uh, 32, i)), u16(a, 4+i); got != want {
			t.Errorf("i32 extend_high_u lane %d: got %d, want %d", i, got, want)
		}

		ml := i32x4_extmul_low_i16x8_s(a, b)
		mh := i32x4_extmul_high_i16x8_s(a, b)
		mlu := i32x4_extmul_low_i16x8_u(a, b)
		mhu := i32x4_extmul_high_i16x8_u(a, b)
		if got, want := int32(getLane(ml, 32, i)), x16(a, i)*x16(b, i); got != want {
			t.Errorf("i32 extmul_low_s lane %d: got %d, want %d", i, got, want)
		}
		if got, want := int32(getLane(mh, 32, i)), x16(a, 4+i)*x16(b, 4+i); got != want {
			t.Errorf("i32 extmul_high_s lane %d: got %d, want %d", i, got, want)
		}
		if got, want := uint32(getLane(mlu, 32, i)), u16(a, i)*u16(b, i); got != want {
			t.Errorf("i32 extmul_low_u lane %d: got %d, want %d", i, got, want)
		}
		if got, want := uint32(getLane(mhu, 32, i)), u16(a, 4+i)*u16(b, 4+i); got != want {
			t.Errorf("i32 extmul_high_u lane %d: got %d, want %d", i, got, want)
		}

		ep := i32x4_extadd_pairwise_i16x8_s(a)
		epu := i32x4_extadd_pairwise_i16x8_u(a)
		if got, want := int32(getLane(ep, 32, i)), x16(a, 2*i)+x16(a, 2*i+1); got != want {
			t.Errorf("i32 extadd_pairwise_s lane %d: got %d, want %d", i, got, want)
		}
		if got, want := uint32(getLane(epu, 32, i)), u16(a, 2*i)+u16(a, 2*i+1); got != want {
			t.Errorf("i32 extadd_pairwise_u lane %d: got %d, want %d", i, got, want)
		}

		d := i32x4_dot_i16x8_s(a, b)
		if got, want := int32(getLane(d, 32, i)), x16(a, 2*i)*x16(b, 2*i)+x16(a, 2*i+1)*x16(b, 2*i+1); got != want {
			t.Errorf("dot lane %d: got %d, want %d", i, got, want)
		}
	}

	for i := 0; i < 2; i++ {
		x32 := func(v v128, l int) int64 { return int64(int32(getLane(v, 32, l))) }
		u32 := func(v v128, l int) uint64 { return uint64(uint32(getLane(v, 32, l))) }
		lo := i64x2_extend_low_i32x4_s(a)
		hi := i64x2_extend_high_i32x4_s(a)
		ul := i64x2_extend_low_i32x4_u(a)
		uh := i64x2_extend_high_i32x4_u(a)
		if got, want := int64(getLane(lo, 64, i)), x32(a, i); got != want {
			t.Errorf("i64 extend_low_s lane %d: got %d, want %d", i, got, want)
		}
		if got, want := int64(getLane(hi, 64, i)), x32(a, 2+i); got != want {
			t.Errorf("i64 extend_high_s lane %d: got %d, want %d", i, got, want)
		}
		if got, want := getLane(ul, 64, i), u32(a, i); got != want {
			t.Errorf("i64 extend_low_u lane %d: got %d, want %d", i, got, want)
		}
		if got, want := getLane(uh, 64, i), u32(a, 2+i); got != want {
			t.Errorf("i64 extend_high_u lane %d: got %d, want %d", i, got, want)
		}

		ml := i64x2_extmul_low_i32x4_s(a, b)
		mh := i64x2_extmul_high_i32x4_s(a, b)
		mlu := i64x2_extmul_low_i32x4_u(a, b)
		mhu := i64x2_extmul_high_i32x4_u(a, b)
		if got, want := int64(getLane(ml, 64, i)), x32(a, i)*x32(b, i); got != want {
			t.Errorf("i64 extmul_low_s lane %d: got %d, want %d", i, got, want)
		}
		if got, want := int64(getLane(mh, 64, i)), x32(a, 2+i)*x32(b, 2+i); got != want {
			t.Errorf("i64 extmul_high_s lane %d: got %d, want %d", i, got, want)
		}
		if got, want := getLane(mlu, 64, i), u32(a, i)*u32(b, i); got != want {
			t.Errorf("i64 extmul_low_u lane %d: got %d, want %d", i, got, want)
		}
		if got, want := getLane(mhu, 64, i), u32(a, 2+i)*u32(b, 2+i); got != want {
			t.Errorf("i64 extmul_high_u lane %d: got %d, want %d", i, got, want)
		}
	}
}

// Bitwise ops, bitselect, any/all_true, bitmask.
func TestBitwise(t *testing.T) {
	var ab, bb, cb [16]uint8
	for i := range ab {
		ab[i] = uint8(0x35 * (i + 1))
		bb[i] = uint8(0xc9 * (i + 3))
		cb[i] = uint8(0x5a + 17*i)
	}
	a, b, c := vecOf(ab), vecOf(bb), vecOf(cb)
	not, and, andnot := bytesOf(v128_not(a)), bytesOf(v128_and(a, b)), bytesOf(v128_andnot(a, b))
	or, xor, sel := bytesOf(v128_or(a, b)), bytesOf(v128_xor(a, b)), bytesOf(v128_bitselect(a, b, c))
	for i := range ab {
		if got, want := not[i], ^ab[i]; got != want {
			t.Errorf("not lane %d: got %x, want %x", i, got, want)
		}
		if got, want := and[i], ab[i]&bb[i]; got != want {
			t.Errorf("and lane %d: got %x, want %x", i, got, want)
		}
		if got, want := andnot[i], ab[i]&^bb[i]; got != want {
			t.Errorf("andnot lane %d: got %x, want %x", i, got, want)
		}
		if got, want := or[i], ab[i]|bb[i]; got != want {
			t.Errorf("or lane %d: got %x, want %x", i, got, want)
		}
		if got, want := xor[i], ab[i]^bb[i]; got != want {
			t.Errorf("xor lane %d: got %x, want %x", i, got, want)
		}
		if got, want := sel[i], ab[i]&cb[i]|bb[i]&^cb[i]; got != want {
			t.Errorf("bitselect lane %d: got %x, want %x", i, got, want)
		}
	}

	if got := v128_any_true(v128{}); got != 0 {
		t.Errorf("any_true(0): got %d", got)
	}
	for i := range ab {
		var v v128
		putLane(&v, 8, i, 1)
		if got := v128_any_true(v); got != 1 {
			t.Errorf("any_true(lane %d): got %d", i, got)
		}
	}

	allOnes := i8x16_splat(-1)
	tests := []struct {
		name    string
		allTrue func(v128) int32
		bitmask func(v128) int32
		bits    int
	}{
		{"i8x16", i8x16_all_true, i8x16_bitmask, 8},
		{"i16x8", i16x8_all_true, i16x8_bitmask, 16},
		{"i32x4", i32x4_all_true, i32x4_bitmask, 32},
		{"i64x2", i64x2_all_true, i64x2_bitmask, 64},
	}
	for _, tc := range tests {
		lanes := 128 / tc.bits
		if got := tc.allTrue(allOnes); got != 1 {
			t.Errorf("%s all_true(ones): got %d", tc.name, got)
		}
		if got := tc.allTrue(v128{}); got != 0 {
			t.Errorf("%s all_true(0): got %d", tc.name, got)
		}
		if got := tc.bitmask(v128{}); got != 0 {
			t.Errorf("%s bitmask(0): got %d", tc.name, got)
		}
		if got, want := tc.bitmask(allOnes), int32(1)<<lanes-1; got != want {
			t.Errorf("%s bitmask(ones): got %#x, want %#x", tc.name, got, want)
		}
		for i := 0; i < lanes; i++ {
			// A single zero lane fails all_true.
			v := allOnes
			putLane(&v, tc.bits, i, 0)
			if got := tc.allTrue(v); got != 0 {
				t.Errorf("%s all_true(zero lane %d): got %d", tc.name, i, got)
			}
			// A nonzero lane without sign bit is true but not in the mask.
			v = v128{}
			putLane(&v, tc.bits, i, 1)
			if got := tc.bitmask(v); got != 0 {
				t.Errorf("%s bitmask(1 in lane %d): got %#x", tc.name, i, got)
			}
			// Only the sign bit contributes the lane's mask bit.
			putLane(&v, tc.bits, i, uint64(1)<<(tc.bits-1))
			if got, want := tc.bitmask(v), int32(1)<<i; got != want {
				t.Errorf("%s bitmask(sign lane %d): got %#x, want %#x", tc.name, i, got, want)
			}
		}
	}
}

// Splats, lane extract/replace: distinct lanes, all indices.
func TestLanes(t *testing.T) {
	var ab [16]uint8
	for i := range ab {
		ab[i] = uint8(0x10*i + 5)
	}
	a := vecOf(ab)

	if got, want := i8x16_splat(-2), i8x16_splat(0xfe); got != want {
		t.Errorf("i8x16_splat truncation: got %v, want %v", got, want)
	}
	for i := 0; i < 16; i++ {
		if got, want := i8x16_extract_lane_u(a, i), int32(ab[i]); got != want {
			t.Errorf("i8x16_extract_lane_u(%d): got %d, want %d", i, got, want)
		}
		if got, want := i8x16_extract_lane_s(a, i), int32(int8(ab[i])); got != want {
			t.Errorf("i8x16_extract_lane_s(%d): got %d, want %d", i, got, want)
		}
		r := bytesOf(i8x16_replace_lane(a, i, -3))
		for j := range r {
			want := ab[j]
			if j == i {
				want = 0xfd
			}
			if r[j] != want {
				t.Errorf("i8x16_replace_lane(%d) lane %d: got %d, want %d", i, j, r[j], want)
			}
		}
		if got, want := bytesOf(i8x16_splat(int32(int8(ab[i]))))[i], ab[i]; got != want {
			t.Errorf("i8x16_splat lane %d: got %d, want %d", i, got, want)
		}
	}
	for i := 0; i < 8; i++ {
		x := int32(int16(getLane(a, 16, i)))
		if got := i16x8_extract_lane_s(a, i); got != x {
			t.Errorf("i16x8_extract_lane_s(%d): got %d, want %d", i, got, x)
		}
		if got, want := i16x8_extract_lane_u(a, i), int32(uint16(x)); got != want {
			t.Errorf("i16x8_extract_lane_u(%d): got %d, want %d", i, got, want)
		}
		r := i16x8_replace_lane(a, i, -12345)
		for j := 0; j < 8; j++ {
			want := getLane(a, 16, j)
			if j == i {
				want = uint64(uint16(-12345 & 0xffff))
			}
			if got := getLane(r, 16, j); got != want {
				t.Errorf("i16x8_replace_lane(%d) lane %d: got %d, want %d", i, j, got, want)
			}
		}
		if got, want := getLane(i16x8_splat(-12345), 16, i), uint64(uint16(53191)); got != want {
			t.Errorf("i16x8_splat lane %d: got %d, want %d", i, got, want)
		}
	}
	for i := 0; i < 4; i++ {
		x := int32(getLane(a, 32, i))
		if got := i32x4_extract_lane(a, i); got != x {
			t.Errorf("i32x4_extract_lane(%d): got %d, want %d", i, got, x)
		}
		r := i32x4_replace_lane(a, i, -123456789)
		if got := int32(getLane(r, 32, i)); got != -123456789 {
			t.Errorf("i32x4_replace_lane(%d): got %d", i, got)
		}
		f := f32x4_extract_lane(a, i)
		if math.Float32bits(f) != uint32(getLane(a, 32, i)) {
			t.Errorf("f32x4_extract_lane(%d): got bits %#x", i, math.Float32bits(f))
		}
		fr := f32x4_replace_lane(a, i, float32(1.5))
		if got := getLane(fr, 32, i); got != uint64(math.Float32bits(1.5)) {
			t.Errorf("f32x4_replace_lane(%d): got bits %#x", i, got)
		}
		if got, want := getLane(i32x4_splat(-123456789), 32, i), uint64(uint32(-123456789+1<<32)); int32(got) != -123456789 {
			t.Errorf("i32x4_splat lane %d: got %d, want %d", i, got, want)
		}
		if got := getLane(f32x4_splat(-2.5), 32, i); got != uint64(math.Float32bits(-2.5)) {
			t.Errorf("f32x4_splat lane %d: got bits %#x", i, got)
		}
	}
	for i := 0; i < 2; i++ {
		x := int64(getLane(a, 64, i))
		if got := i64x2_extract_lane(a, i); got != x {
			t.Errorf("i64x2_extract_lane(%d): got %d, want %d", i, got, x)
		}
		r := i64x2_replace_lane(a, i, -1234567890123)
		if got := int64(getLane(r, 64, i)); got != -1234567890123 {
			t.Errorf("i64x2_replace_lane(%d): got %d", i, got)
		}
		f := f64x2_extract_lane(a, i)
		if math.Float64bits(f) != getLane(a, 64, i) {
			t.Errorf("f64x2_extract_lane(%d): got bits %#x", i, math.Float64bits(f))
		}
		fr := f64x2_replace_lane(a, i, 2.25)
		if got := getLane(fr, 64, i); got != math.Float64bits(2.25) {
			t.Errorf("f64x2_replace_lane(%d): got bits %#x", i, got)
		}
		if got := int64(getLane(i64x2_splat(-1234567890123), 64, i)); got != -1234567890123 {
			t.Errorf("i64x2_splat lane %d: got %d", i, got)
		}
		if got := getLane(f64x2_splat(-2.5), 64, i); got != math.Float64bits(-2.5) {
			t.Errorf("f64x2_splat lane %d: got bits %#x", i, got)
		}
	}
}

// Shuffle and swizzle: exhaustive index domains.
func TestShuffleSwizzle(t *testing.T) {
	var ab, bb [16]uint8
	for i := range ab {
		ab[i] = uint8(i + 1)
		bb[i] = uint8(i + 101)
	}
	a, b := vecOf(ab), vecOf(bb)
	// Swizzle: all 256 index values through every lane position.
	for s0 := 0; s0 < 256; s0 += 16 {
		for o := 0; o < 16; o++ {
			var s [16]uint8
			for i := range s {
				s[i] = uint8(s0 + (i+o)%16)
			}
			r := bytesOf(i8x16_swizzle(a, vecOf(s)))
			for i := range r {
				var want uint8
				if s[i] < 16 {
					want = ab[s[i]]
				}
				if r[i] != want {
					t.Fatalf("swizzle(%d) lane %d: got %d, want %d", s[i], i, r[i], want)
				}
			}
		}
	}
	// Shuffle: all 32 valid indices through every lane position.
	for s0 := 0; s0 < 32; s0++ {
		var m [16]uint8
		for i := range m {
			m[i] = uint8((s0 + i) % 32)
		}
		r := bytesOf(i8x16_shuffle(a, b, vecOf(m)))
		for i := range r {
			want := ab[m[i]&15]
			if m[i] >= 16 {
				want = bb[m[i]&15]
			}
			if r[i] != want {
				t.Fatalf("shuffle(%d) lane %d: got %d, want %d", m[i], i, r[i], want)
			}
		}
	}
}

// Vector loads: extension, splat, zero forms.
func TestLoads(t *testing.T) {
	buf := []byte{0x01, 0x82, 0x03, 0x84, 0x05, 0x86, 0x07, 0x88, 0x09, 0x8a, 0x0b, 0x8c, 0x0d, 0x8e, 0x0f, 0x90}
	v := load128(buf)
	if v != vecOf([16]uint8(buf)) {
		t.Errorf("load128: got %v, want %v", v, buf)
	}
	var out [16]byte
	store128(out[:], v)
	if out != [16]byte(buf) {
		t.Errorf("store128: got %v, want %v", out, buf)
	}
	defer func() {
		if recover() == nil {
			t.Error("load128 of short slice did not panic")
		}
	}()

	x := uint64(0x8878685848382818)
	s := v128_load8x8_s(x)
	u := v128_load8x8_u(x)
	for i := 0; i < 8; i++ {
		bv := uint8(x >> (8 * i))
		if got, want := int16(getLane(s, 16, i)), int16(int8(bv)); got != want {
			t.Errorf("load8x8_s lane %d: got %d, want %d", i, got, want)
		}
		if got, want := getLane(u, 16, i), uint64(bv); got != want {
			t.Errorf("load8x8_u lane %d: got %d, want %d", i, got, want)
		}
	}
	s = v128_load16x4_s(x)
	u = v128_load16x4_u(x)
	for i := 0; i < 4; i++ {
		hv := uint16(x >> (16 * i))
		if got, want := int32(getLane(s, 32, i)), int32(int16(hv)); got != want {
			t.Errorf("load16x4_s lane %d: got %d, want %d", i, got, want)
		}
		if got, want := getLane(u, 32, i), uint64(hv); got != want {
			t.Errorf("load16x4_u lane %d: got %d, want %d", i, got, want)
		}
	}
	s = v128_load32x2_s(x)
	u = v128_load32x2_u(x)
	for i := 0; i < 2; i++ {
		wv := uint32(x >> (32 * i))
		if got, want := int64(getLane(s, 64, i)), int64(int32(wv)); got != want {
			t.Errorf("load32x2_s lane %d: got %d, want %d", i, got, want)
		}
		if got, want := getLane(u, 64, i), uint64(wv); got != want {
			t.Errorf("load32x2_u lane %d: got %d, want %d", i, got, want)
		}
	}
	if got, want := v128_load32_zero(0x87654321), makeVec(32, 0x87654321, 0, 0, 0); got != want {
		t.Errorf("load32_zero: got %v, want %v", got, want)
	}
	if got, want := v128_load64_zero(x), makeVec(64, x, 0); got != want {
		t.Errorf("load64_zero: got %v, want %v", got, want)
	}

	load128(buf[:15]) // panics
}

// Float lane cases: NaNs, signed zeros, infinities, denormals, and exact values.
func f32cases() []uint32 {
	return []uint32{
		0x00000000, 0x80000000, // ±0
		0x3f800000, 0xbf800000, // ±1
		0x3fc00000, 0xbfc00000, // ±1.5
		0x40200000, 0xc0200000, // ±2.5
		0x3f000000, 0xbf000000, // ±0.5
		0x7f800000, 0xff800000, // ±inf
		0x7fc00000, 0xffc00000, // ±canonical NaN
		0x7fa00000, 0x7fc00001, 0xffabcdef, // other NaNs
		0x00000001, 0x80000001, // ±denormal
		0x007fffff, 0x807fffff, // ±largest denormal
		0x00800000, 0x80800000, // ±smallest normal
		0x7f7fffff, 0xff7fffff, // ±largest finite
		0x4effffff, 0xceffffff, // ±2147483520 (largest f32 < 2^31)
		0x4f000000, 0xcf000000, // ±2^31
		0x4f800000, 0xcf800000, // ±2^32
		0x3effffff,             // largest f32 < 0.5
		0x4b400001, 0x4b3fffff, // 2^23 x ±(1 ± ε)
		0x45678123, 0xc5678123, // arbitrary
	}
}

func f64cases() []uint64 {
	return []uint64{
		0x0000000000000000, 0x8000000000000000, // ±0
		0x3ff0000000000000, 0xbff0000000000000, // ±1
		0x3ff8000000000000, 0xbff8000000000000, // ±1.5
		0x4004000000000000, 0xc004000000000000, // ±2.5
		0x3fe0000000000000, 0xbfe0000000000000, // ±0.5
		0x7ff0000000000000, 0xfff0000000000000, // ±inf
		0x7ff8000000000000, 0xfff8000000000000, // ±canonical NaN
		0x7ff4000000000000, 0x7ff8000000000001, 0xfff123456789abcd, // other NaNs
		0x0000000000000001, 0x8000000000000001, // ±denormal
		0x000fffffffffffff, 0x800fffffffffffff, // ±largest denormal
		0x0010000000000000, 0x8010000000000000, // ±smallest normal
		0x7fefffffffffffff, 0xffefffffffffffff, // ±largest finite
		0x41dfffffffc00000, 0xc1dfffffffc00000, // ±(2^31 - 1)
		0x41e0000000000000, 0xc1e0000000000000, // ±2^31
		0x41f0000000000000, 0xc1f0000000000000, // ±2^32
		0x4330000000000001, 0x432fffffffffffff, // 2^52 x ±(1 ± ε)
		0x4157654321000000, 0xc157654321000000, // arbitrary
	}
}

// Float binary operations over the case cross product,
// against independent scalar references.
func TestF32x4Binary(t *testing.T) {
	canon := uint32(0x7fc00000)
	isNaN := func(x uint32) bool { return x&0x7fffffff > 0x7f800000 }
	refMin := func(x, y uint32) uint32 {
		a, b := math.Float32frombits(x), math.Float32frombits(y)
		switch {
		case isNaN(x) || isNaN(y):
			return canon
		case a < b:
			return x
		case b < a:
			return y
		default: // equal: -0 wins over +0
			return x | y
		}
	}
	refMax := func(x, y uint32) uint32 {
		a, b := math.Float32frombits(x), math.Float32frombits(y)
		switch {
		case isNaN(x) || isNaN(y):
			return canon
		case a > b:
			return x
		case b > a:
			return y
		default: // equal: +0 wins over -0
			return x & y
		}
	}
	ops := []struct {
		name   string
		fn     func(a, b v128) v128
		ref    func(x, y uint32) uint32
		anyNaN bool // NaN results need not be bit-exact
	}{
		{"add", f32x4_add, func(x, y uint32) uint32 {
			return math.Float32bits(math.Float32frombits(x) + math.Float32frombits(y))
		}, true},
		{"sub", f32x4_sub, func(x, y uint32) uint32 {
			return math.Float32bits(math.Float32frombits(x) - math.Float32frombits(y))
		}, true},
		{"mul", f32x4_mul, func(x, y uint32) uint32 {
			return math.Float32bits(math.Float32frombits(x) * math.Float32frombits(y))
		}, true},
		{"div", f32x4_div, func(x, y uint32) uint32 {
			return math.Float32bits(math.Float32frombits(x) / math.Float32frombits(y))
		}, true},
		{"min", f32x4_min, refMin, false},
		{"max", f32x4_max, refMax, false},
		{"pmin", f32x4_pmin, func(x, y uint32) uint32 {
			if math.Float32frombits(y) < math.Float32frombits(x) {
				return y
			}
			return x
		}, false},
		{"pmax", f32x4_pmax, func(x, y uint32) uint32 {
			if math.Float32frombits(x) < math.Float32frombits(y) {
				return y
			}
			return x
		}, false},
	}
	cases := f32cases()
	for _, op := range ops {
		t.Run(op.name, func(t *testing.T) {
			for _, x := range cases {
				for y0 := 0; y0 < len(cases); y0 += 4 {
					var a, b v128
					for i := 0; i < 4; i++ {
						putLane(&a, 32, i, uint64(x))
						putLane(&b, 32, i, uint64(cases[(y0+i)%len(cases)]))
					}
					r := op.fn(a, b)
					for i := 0; i < 4; i++ {
						y := uint32(getLane(b, 32, i))
						got := uint32(getLane(r, 32, i))
						want := op.ref(x, y)
						if got != want && !(op.anyNaN && isNaN(got) && isNaN(want)) {
							t.Fatalf("%s(%#x, %#x): got %#x, want %#x", op.name, x, y, got, want)
						}
					}
				}
			}
		})
	}
}

func TestF64x2Binary(t *testing.T) {
	canon := uint64(0x7ff8000000000000)
	isNaN := func(x uint64) bool { return x&0x7fffffffffffffff > 0x7ff0000000000000 }
	refMin := func(x, y uint64) uint64 {
		a, b := math.Float64frombits(x), math.Float64frombits(y)
		switch {
		case isNaN(x) || isNaN(y):
			return canon
		case a < b:
			return x
		case b < a:
			return y
		default:
			return x | y
		}
	}
	refMax := func(x, y uint64) uint64 {
		a, b := math.Float64frombits(x), math.Float64frombits(y)
		switch {
		case isNaN(x) || isNaN(y):
			return canon
		case a > b:
			return x
		case b > a:
			return y
		default:
			return x & y
		}
	}
	ops := []struct {
		name   string
		fn     func(a, b v128) v128
		ref    func(x, y uint64) uint64
		anyNaN bool
	}{
		{"add", f64x2_add, func(x, y uint64) uint64 {
			return math.Float64bits(math.Float64frombits(x) + math.Float64frombits(y))
		}, true},
		{"sub", f64x2_sub, func(x, y uint64) uint64 {
			return math.Float64bits(math.Float64frombits(x) - math.Float64frombits(y))
		}, true},
		{"mul", f64x2_mul, func(x, y uint64) uint64 {
			return math.Float64bits(math.Float64frombits(x) * math.Float64frombits(y))
		}, true},
		{"div", f64x2_div, func(x, y uint64) uint64 {
			return math.Float64bits(math.Float64frombits(x) / math.Float64frombits(y))
		}, true},
		{"min", f64x2_min, refMin, false},
		{"max", f64x2_max, refMax, false},
		{"pmin", f64x2_pmin, func(x, y uint64) uint64 {
			if math.Float64frombits(y) < math.Float64frombits(x) {
				return y
			}
			return x
		}, false},
		{"pmax", f64x2_pmax, func(x, y uint64) uint64 {
			if math.Float64frombits(x) < math.Float64frombits(y) {
				return y
			}
			return x
		}, false},
	}
	cases := f64cases()
	for _, op := range ops {
		t.Run(op.name, func(t *testing.T) {
			for _, x := range cases {
				for _, y := range cases {
					a := makeVec(64, x, y)
					b := makeVec(64, y, x)
					r := op.fn(a, b)
					for i, want := range []uint64{op.ref(x, y), op.ref(y, x)} {
						got := getLane(r, 64, i)
						if got != want && !(op.anyNaN && isNaN(got) && isNaN(want)) {
							t.Fatalf("%s lane %d: got %#x, want %#x", op.name, i, got, want)
						}
					}
				}
			}
		})
	}
}

// Float comparisons over the case cross product.
func TestFloatCompare(t *testing.T) {
	f32 := math.Float32frombits
	f64 := math.Float64frombits
	c32 := f32cases()
	for _, x := range c32 {
		for _, y := range c32 {
			a := makeVec(32, uint64(x), uint64(y), uint64(x), uint64(y))
			b := makeVec(32, uint64(y), uint64(x), uint64(y), uint64(x))
			checks := []struct {
				name string
				fn   func(a, b v128) v128
				want bool // for lane 0: x OP y
			}{
				{"eq", f32x4_eq, f32(x) == f32(y)},
				{"ne", f32x4_ne, f32(x) != f32(y)},
				{"lt", f32x4_lt, f32(x) < f32(y)},
				{"gt", f32x4_gt, f32(x) > f32(y)},
				{"le", f32x4_le, f32(x) <= f32(y)},
				{"ge", f32x4_ge, f32(x) >= f32(y)},
			}
			for _, c := range checks {
				r := c.fn(a, b)
				if got := getLane(r, 32, 0) == 0xffffffff; got != c.want {
					t.Fatalf("f32x4_%s(%#x, %#x): got %v, want %v", c.name, x, y, got, c.want)
				}
			}
		}
	}
	c64 := f64cases()
	for _, x := range c64 {
		for _, y := range c64 {
			a := makeVec(64, x, y)
			b := makeVec(64, y, x)
			checks := []struct {
				name string
				fn   func(a, b v128) v128
				want bool
			}{
				{"eq", f64x2_eq, f64(x) == f64(y)},
				{"ne", f64x2_ne, f64(x) != f64(y)},
				{"lt", f64x2_lt, f64(x) < f64(y)},
				{"gt", f64x2_gt, f64(x) > f64(y)},
				{"le", f64x2_le, f64(x) <= f64(y)},
				{"ge", f64x2_ge, f64(x) >= f64(y)},
			}
			for _, c := range checks {
				r := c.fn(a, b)
				if got := getLane(r, 64, 0) == math.MaxUint64; got != c.want {
					t.Fatalf("f64x2_%s(%#x, %#x): got %v, want %v", c.name, x, y, got, c.want)
				}
			}
		}
	}
}

// Float unary operations against scalar math references.
func TestFloatUnary(t *testing.T) {
	isNaN32 := func(x uint32) bool { return x&0x7fffffff > 0x7f800000 }
	isNaN64 := func(x uint64) bool { return x&0x7fffffffffffffff > 0x7ff0000000000000 }
	ops32 := []struct {
		name  string
		fn    func(v128) v128
		ref   func(x uint32) uint32
		exact bool // NaN results must be bit-exact
	}{
		{"abs", f32x4_abs, func(x uint32) uint32 { return x &^ 0x80000000 }, true},
		{"neg", f32x4_neg, func(x uint32) uint32 { return x ^ 0x80000000 }, true},
		{"sqrt", f32x4_sqrt, func(x uint32) uint32 {
			return math.Float32bits(float32(math.Sqrt(float64(math.Float32frombits(x)))))
		}, false},
		{"ceil", f32x4_ceil, func(x uint32) uint32 {
			return math.Float32bits(float32(math.Ceil(float64(math.Float32frombits(x)))))
		}, false},
		{"floor", f32x4_floor, func(x uint32) uint32 {
			return math.Float32bits(float32(math.Floor(float64(math.Float32frombits(x)))))
		}, false},
		{"trunc", f32x4_trunc, func(x uint32) uint32 {
			return math.Float32bits(float32(math.Trunc(float64(math.Float32frombits(x)))))
		}, false},
		{"nearest", f32x4_nearest, func(x uint32) uint32 {
			return math.Float32bits(float32(math.RoundToEven(float64(math.Float32frombits(x)))))
		}, false},
	}
	for _, op := range ops32 {
		for _, x := range f32cases() {
			a := makeVec(32, uint64(x), uint64(x^0x80000000), uint64(x), uint64(x))
			r := op.fn(a)
			for i, want := range []uint32{op.ref(x), op.ref(x ^ 0x80000000)} {
				got := uint32(getLane(r, 32, i))
				if got != want && (op.exact || !(isNaN32(got) && isNaN32(want))) {
					t.Fatalf("f32x4_%s(%#x): got %#x, want %#x", op.name, x, got, want)
				}
			}
		}
	}
	ops64 := []struct {
		name  string
		fn    func(v128) v128
		ref   func(x uint64) uint64
		exact bool
	}{
		{"abs", f64x2_abs, func(x uint64) uint64 { return x &^ 0x8000000000000000 }, true},
		{"neg", f64x2_neg, func(x uint64) uint64 { return x ^ 0x8000000000000000 }, true},
		{"sqrt", f64x2_sqrt, func(x uint64) uint64 {
			return math.Float64bits(math.Sqrt(math.Float64frombits(x)))
		}, false},
		{"ceil", f64x2_ceil, func(x uint64) uint64 {
			return math.Float64bits(math.Ceil(math.Float64frombits(x)))
		}, false},
		{"floor", f64x2_floor, func(x uint64) uint64 {
			return math.Float64bits(math.Floor(math.Float64frombits(x)))
		}, false},
		{"trunc", f64x2_trunc, func(x uint64) uint64 {
			return math.Float64bits(math.Trunc(math.Float64frombits(x)))
		}, false},
		{"nearest", f64x2_nearest, func(x uint64) uint64 {
			return math.Float64bits(math.RoundToEven(math.Float64frombits(x)))
		}, false},
	}
	for _, op := range ops64 {
		for _, x := range f64cases() {
			a := makeVec(64, x, x^0x8000000000000000)
			r := op.fn(a)
			for i, want := range []uint64{op.ref(x), op.ref(x ^ 0x8000000000000000)} {
				got := getLane(r, 64, i)
				if got != want && (op.exact || !(isNaN64(got) && isNaN64(want))) {
					t.Fatalf("f64x2_%s(%#x): got %#x, want %#x", op.name, x, got, want)
				}
			}
		}
	}
}

// Independent saturating truncation references.
func refTruncSatS(f float64) int32 {
	switch {
	case f != f:
		return 0
	case f <= math.MinInt32:
		return math.MinInt32
	case f >= math.MaxInt32:
		return math.MaxInt32
	}
	return int32(f)
}

func refTruncSatU(f float64) int32 {
	switch {
	case f != f, f <= 0:
		return 0
	case f >= math.MaxUint32:
		return -1 // all ones
	}
	return int32(uint32(f))
}

// Conversions between float and integer lanes,
// verified against independent scalar references and Go conversions.
func TestConversions(t *testing.T) {
	for _, x := range f32cases() {
		a := makeVec(32, uint64(x), uint64(x), uint64(x), uint64(x))
		f := math.Float32frombits(x)
		if got, want := int32(getLane(i32x4_trunc_sat_f32x4_s(a), 32, 1)), refTruncSatS(float64(f)); got != want {
			t.Errorf("trunc_sat_f32x4_s(%#x): got %d, want %d", x, got, want)
		}
		if got, want := int32(getLane(i32x4_trunc_sat_f32x4_u(a), 32, 2)), refTruncSatU(float64(f)); got != want {
			t.Errorf("trunc_sat_f32x4_u(%#x): got %d, want %d", x, got, want)
		}
		// promote_low: exact
		p := f64x2_promote_low_f32x4(a)
		for i := 0; i < 2; i++ {
			got := getLane(p, 64, i)
			want := math.Float64bits(float64(f))
			if got != want && !(f != f && math.Float64frombits(got) != math.Float64frombits(got)) {
				t.Errorf("promote_low(%#x): got %#x, want %#x", x, got, want)
			}
		}
	}
	for _, x := range f64cases() {
		a := makeVec(64, x, x)
		f := math.Float64frombits(x)
		ts := i32x4_trunc_sat_f64x2_s_zero(a)
		tu := i32x4_trunc_sat_f64x2_u_zero(a)
		for i := 0; i < 2; i++ {
			if got, want := int32(getLane(ts, 32, i)), refTruncSatS(f); got != want {
				t.Errorf("trunc_sat_f64x2_s_zero(%#x): got %d, want %d", x, got, want)
			}
			if got, want := int32(getLane(tu, 32, i)), refTruncSatU(f); got != want {
				t.Errorf("trunc_sat_f64x2_u_zero(%#x): got %d, want %d", x, got, want)
			}
		}
		for i := 2; i < 4; i++ {
			if getLane(ts, 32, i) != 0 || getLane(tu, 32, i) != 0 {
				t.Errorf("trunc_sat_f64x2_zero(%#x): high lanes not zero", x)
			}
		}
		// demote: rounded conversion; NaN stays NaN
		d := f32x4_demote_f64x2_zero(a)
		for i := 0; i < 2; i++ {
			got := uint32(getLane(d, 32, i))
			want := math.Float32bits(float32(f))
			if got != want && !(f != f && got&0x7fffffff > 0x7f800000) {
				t.Errorf("demote(%#x): got %#x, want %#x", x, got, want)
			}
		}
		if getLane(d, 32, 2) != 0 || getLane(d, 32, 3) != 0 {
			t.Errorf("demote(%#x): high lanes not zero", x)
		}
	}
	for _, xs := range laneDomains(32) {
		var a v128
		for i := 0; i < 4; i++ {
			putLane(&a, 32, i, xs[i%len(xs)])
		}
		cs := f32x4_convert_i32x4_s(a)
		cu := f32x4_convert_i32x4_u(a)
		ds := f64x2_convert_low_i32x4_s(a)
		du := f64x2_convert_low_i32x4_u(a)
		for i := 0; i < 4; i++ {
			x := uint32(getLane(a, 32, i))
			if got, want := uint32(getLane(cs, 32, i)), math.Float32bits(float32(int32(x))); got != want {
				t.Errorf("convert_i32x4_s(%d): got %#x, want %#x", int32(x), got, want)
			}
			if got, want := uint32(getLane(cu, 32, i)), math.Float32bits(float32(x)); got != want {
				t.Errorf("convert_i32x4_u(%d): got %#x, want %#x", x, got, want)
			}
			if i < 2 {
				if got, want := getLane(ds, 64, i), math.Float64bits(float64(int32(x))); got != want {
					t.Errorf("convert_low_i32x4_s(%d): got %#x, want %#x", int32(x), got, want)
				}
				if got, want := getLane(du, 64, i), math.Float64bits(float64(x)); got != want {
					t.Errorf("convert_low_i32x4_u(%d): got %#x, want %#x", x, got, want)
				}
			}
		}
	}
}
