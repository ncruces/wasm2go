package main

import (
	"encoding/binary"
	"fmt"
	"go/ast"
	"go/token"
	"io"
)

func (t *translator) readOpcodeSimd(fn *funcCompiler) error {
	code, err := readLEB128(t.in)
	if err != nil {
		return err
	}

	// Vector memory access opcodes start with a memarg immediate.
	var offset uint64
	switch {
	case code <= 0x0b, 0x54 <= code && code <= 0x5d:
		_, err = readLEB128(t.in) // align
		if err == nil {
			offset, err = readLEB128(t.in)
		}
		if err != nil {
			return err
		}
	}

	switch code {
	case 0x00: // v128.load
		fn.push(fn.loadV128(offset))
	case 0x01: // v128.load8x8_s
		fn.push(fn.simdLoadExt("v128_load8x8_s", offset))
	case 0x02: // v128.load8x8_u
		fn.push(fn.simdLoadExt("v128_load8x8_u", offset))
	case 0x03: // v128.load16x4_s
		fn.push(fn.simdLoadExt("v128_load16x4_s", offset))
	case 0x04: // v128.load16x4_u
		fn.push(fn.simdLoadExt("v128_load16x4_u", offset))
	case 0x05: // v128.load32x2_s
		fn.push(fn.simdLoadExt("v128_load32x2_s", offset))
	case 0x06: // v128.load32x2_u
		fn.push(fn.simdLoadExt("v128_load32x2_u", offset))
	case 0x07: // v128.load8_splat
		fn.push(fn.simdCall("i8x16_splat", convert(fn.load8(offset), "int32")))
	case 0x08: // v128.load16_splat
		fn.push(fn.simdCall("i16x8_splat", convert(fn.load("uint16", offset), "int32")))
	case 0x09: // v128.load32_splat
		fn.push(fn.simdCall("i32x4_splat", convert(fn.load("uint32", offset), "int32")))
	case 0x0a: // v128.load64_splat
		fn.push(fn.simdCall("i64x2_splat", convert(fn.load("uint64", offset), "int64")))
	case 0x0b: // v128.store
		fn.emit(fn.storeV128(offset))

	case 0x0c: // v128.const
		lit, err := t.constV128()
		if err != nil {
			return err
		}
		fn.pushPure(lit)

	case 0x0d: // i8x16.shuffle
		var lanes [16]byte
		if _, err := io.ReadFull(t.in, lanes[:]); err != nil {
			return err
		}
		y := fn.pop()
		x := fn.pop()
		fn.pushPure(fn.simdCall("i8x16_shuffle", x, y, v128lit(lanes)))
	case 0x0e: // i8x16.swizzle
		fn.simdBin("i8x16_swizzle")

	case 0x0f: // i8x16.splat
		fn.simdUn("i8x16_splat")
	case 0x10: // i16x8.splat
		fn.simdUn("i16x8_splat")
	case 0x11: // i32x4.splat
		fn.simdUn("i32x4_splat")
	case 0x12: // i64x2.splat
		fn.simdUn("i64x2_splat")
	case 0x13: // f32x4.splat
		fn.simdUn("f32x4_splat")
	case 0x14: // f64x2.splat
		fn.simdUn("f64x2_splat")

	case 0x15: // i8x16.extract_lane_s
		err = fn.extractLane("i8x16_extract_lane_s")
	case 0x16: // i8x16.extract_lane_u
		err = fn.extractLane("i8x16_extract_lane_u")
	case 0x17: // i8x16.replace_lane
		err = fn.replaceLane("i8x16_replace_lane")
	case 0x18: // i16x8.extract_lane_s
		err = fn.extractLane("i16x8_extract_lane_s")
	case 0x19: // i16x8.extract_lane_u
		err = fn.extractLane("i16x8_extract_lane_u")
	case 0x1a: // i16x8.replace_lane
		err = fn.replaceLane("i16x8_replace_lane")
	case 0x1b: // i32x4.extract_lane
		err = fn.extractLane("i32x4_extract_lane")
	case 0x1c: // i32x4.replace_lane
		err = fn.replaceLane("i32x4_replace_lane")
	case 0x1d: // i64x2.extract_lane
		err = fn.extractLane("i64x2_extract_lane")
	case 0x1e: // i64x2.replace_lane
		err = fn.replaceLane("i64x2_replace_lane")
	case 0x1f: // f32x4.extract_lane
		err = fn.extractLane("f32x4_extract_lane")
	case 0x20: // f32x4.replace_lane
		err = fn.replaceLane("f32x4_replace_lane")
	case 0x21: // f64x2.extract_lane
		err = fn.extractLane("f64x2_extract_lane")
	case 0x22: // f64x2.replace_lane
		err = fn.replaceLane("f64x2_replace_lane")

	case 0x23: // i8x16.eq
		fn.simdBin("i8x16_eq")
	case 0x24: // i8x16.ne
		fn.simdBin("i8x16_ne")
	case 0x25: // i8x16.lt_s
		fn.simdBin("i8x16_lt_s")
	case 0x26: // i8x16.lt_u
		fn.simdBin("i8x16_lt_u")
	case 0x27: // i8x16.gt_s
		fn.simdBin("i8x16_gt_s")
	case 0x28: // i8x16.gt_u
		fn.simdBin("i8x16_gt_u")
	case 0x29: // i8x16.le_s
		fn.simdBin("i8x16_le_s")
	case 0x2a: // i8x16.le_u
		fn.simdBin("i8x16_le_u")
	case 0x2b: // i8x16.ge_s
		fn.simdBin("i8x16_ge_s")
	case 0x2c: // i8x16.ge_u
		fn.simdBin("i8x16_ge_u")

	case 0x2d: // i16x8.eq
		fn.simdBin("i16x8_eq")
	case 0x2e: // i16x8.ne
		fn.simdBin("i16x8_ne")
	case 0x2f: // i16x8.lt_s
		fn.simdBin("i16x8_lt_s")
	case 0x30: // i16x8.lt_u
		fn.simdBin("i16x8_lt_u")
	case 0x31: // i16x8.gt_s
		fn.simdBin("i16x8_gt_s")
	case 0x32: // i16x8.gt_u
		fn.simdBin("i16x8_gt_u")
	case 0x33: // i16x8.le_s
		fn.simdBin("i16x8_le_s")
	case 0x34: // i16x8.le_u
		fn.simdBin("i16x8_le_u")
	case 0x35: // i16x8.ge_s
		fn.simdBin("i16x8_ge_s")
	case 0x36: // i16x8.ge_u
		fn.simdBin("i16x8_ge_u")

	case 0x37: // i32x4.eq
		fn.simdBin("i32x4_eq")
	case 0x38: // i32x4.ne
		fn.simdBin("i32x4_ne")
	case 0x39: // i32x4.lt_s
		fn.simdBin("i32x4_lt_s")
	case 0x3a: // i32x4.lt_u
		fn.simdBin("i32x4_lt_u")
	case 0x3b: // i32x4.gt_s
		fn.simdBin("i32x4_gt_s")
	case 0x3c: // i32x4.gt_u
		fn.simdBin("i32x4_gt_u")
	case 0x3d: // i32x4.le_s
		fn.simdBin("i32x4_le_s")
	case 0x3e: // i32x4.le_u
		fn.simdBin("i32x4_le_u")
	case 0x3f: // i32x4.ge_s
		fn.simdBin("i32x4_ge_s")
	case 0x40: // i32x4.ge_u
		fn.simdBin("i32x4_ge_u")

	case 0x41: // f32x4.eq
		fn.simdBin("f32x4_eq")
	case 0x42: // f32x4.ne
		fn.simdBin("f32x4_ne")
	case 0x43: // f32x4.lt
		fn.simdBin("f32x4_lt")
	case 0x44: // f32x4.gt
		fn.simdBin("f32x4_gt")
	case 0x45: // f32x4.le
		fn.simdBin("f32x4_le")
	case 0x46: // f32x4.ge
		fn.simdBin("f32x4_ge")

	case 0x47: // f64x2.eq
		fn.simdBin("f64x2_eq")
	case 0x48: // f64x2.ne
		fn.simdBin("f64x2_ne")
	case 0x49: // f64x2.lt
		fn.simdBin("f64x2_lt")
	case 0x4a: // f64x2.gt
		fn.simdBin("f64x2_gt")
	case 0x4b: // f64x2.le
		fn.simdBin("f64x2_le")
	case 0x4c: // f64x2.ge
		fn.simdBin("f64x2_ge")

	case 0x4d: // v128.not
		fn.simdUn("v128_not")
	case 0x4e: // v128.and
		fn.simdBin("v128_and")
	case 0x4f: // v128.andnot
		fn.simdBin("v128_andnot")
	case 0x50: // v128.or
		fn.simdBin("v128_or")
	case 0x51: // v128.xor
		fn.simdBin("v128_xor")
	case 0x52: // v128.bitselect
		z := fn.pop()
		y := fn.pop()
		x := fn.pop()
		fn.pushPure(fn.simdCall("v128_bitselect", x, y, z))
	case 0x53: // v128.any_true
		fn.simdUn("v128_any_true")

	case 0x54: // v128.load8_lane
		err = fn.simdLoadLane("i8x16_replace_lane", 8, offset)
	case 0x55: // v128.load16_lane
		err = fn.simdLoadLane("i16x8_replace_lane", 16, offset)
	case 0x56: // v128.load32_lane
		err = fn.simdLoadLane("i32x4_replace_lane", 32, offset)
	case 0x57: // v128.load64_lane
		err = fn.simdLoadLane("i64x2_replace_lane", 64, offset)
	case 0x58: // v128.store8_lane
		err = fn.simdStoreLane("i8x16_extract_lane_u", 8, offset)
	case 0x59: // v128.store16_lane
		err = fn.simdStoreLane("i16x8_extract_lane_u", 16, offset)
	case 0x5a: // v128.store32_lane
		err = fn.simdStoreLane("i32x4_extract_lane", 32, offset)
	case 0x5b: // v128.store64_lane
		err = fn.simdStoreLane("i64x2_extract_lane", 64, offset)
	case 0x5c: // v128.load32_zero
		fn.push(fn.simdCall("v128_load32_zero", fn.load("uint32", offset)))
	case 0x5d: // v128.load64_zero
		fn.push(fn.simdCall("v128_load64_zero", fn.load("uint64", offset)))

	case 0x5e: // f32x4.demote_f64x2_zero
		fn.simdUn("f32x4_demote_f64x2_zero")
	case 0x5f: // f64x2.promote_low_f32x4
		fn.simdUn("f64x2_promote_low_f32x4")

	case 0x60: // i8x16.abs
		fn.simdUn("i8x16_abs")
	case 0x61: // i8x16.neg
		fn.simdUn("i8x16_neg")
	case 0x62: // i8x16.popcnt
		fn.simdUn("i8x16_popcnt")
	case 0x63: // i8x16.all_true
		fn.simdUn("i8x16_all_true")
	case 0x64: // i8x16.bitmask
		fn.simdUn("i8x16_bitmask")
	case 0x65: // i8x16.narrow_i16x8_s
		fn.simdBin("i8x16_narrow_i16x8_s")
	case 0x66: // i8x16.narrow_i16x8_u
		fn.simdBin("i8x16_narrow_i16x8_u")

	case 0x67: // f32x4.ceil
		fn.simdUn("f32x4_ceil")
	case 0x68: // f32x4.floor
		fn.simdUn("f32x4_floor")
	case 0x69: // f32x4.trunc
		fn.simdUn("f32x4_trunc")
	case 0x6a: // f32x4.nearest
		fn.simdUn("f32x4_nearest")

	case 0x6b: // i8x16.shl
		fn.simdBin("i8x16_shl")
	case 0x6c: // i8x16.shr_s
		fn.simdBin("i8x16_shr_s")
	case 0x6d: // i8x16.shr_u
		fn.simdBin("i8x16_shr_u")
	case 0x6e: // i8x16.add
		fn.simdBin("i8x16_add")
	case 0x6f: // i8x16.add_sat_s
		fn.simdBin("i8x16_add_sat_s")
	case 0x70: // i8x16.add_sat_u
		fn.simdBin("i8x16_add_sat_u")
	case 0x71: // i8x16.sub
		fn.simdBin("i8x16_sub")
	case 0x72: // i8x16.sub_sat_s
		fn.simdBin("i8x16_sub_sat_s")
	case 0x73: // i8x16.sub_sat_u
		fn.simdBin("i8x16_sub_sat_u")

	case 0x74: // f64x2.ceil
		fn.simdUn("f64x2_ceil")
	case 0x75: // f64x2.floor
		fn.simdUn("f64x2_floor")

	case 0x76: // i8x16.min_s
		fn.simdBin("i8x16_min_s")
	case 0x77: // i8x16.min_u
		fn.simdBin("i8x16_min_u")
	case 0x78: // i8x16.max_s
		fn.simdBin("i8x16_max_s")
	case 0x79: // i8x16.max_u
		fn.simdBin("i8x16_max_u")

	case 0x7a: // f64x2.trunc
		fn.simdUn("f64x2_trunc")

	case 0x7b: // i8x16.avgr_u
		fn.simdBin("i8x16_avgr_u")

	case 0x7c: // i16x8.extadd_pairwise_i8x16_s
		fn.simdUn("i16x8_extadd_pairwise_i8x16_s")
	case 0x7d: // i16x8.extadd_pairwise_i8x16_u
		fn.simdUn("i16x8_extadd_pairwise_i8x16_u")
	case 0x7e: // i32x4.extadd_pairwise_i16x8_s
		fn.simdUn("i32x4_extadd_pairwise_i16x8_s")
	case 0x7f: // i32x4.extadd_pairwise_i16x8_u
		fn.simdUn("i32x4_extadd_pairwise_i16x8_u")

	case 0x80: // i16x8.abs
		fn.simdUn("i16x8_abs")
	case 0x81: // i16x8.neg
		fn.simdUn("i16x8_neg")
	case 0x82: // i16x8.q15mulr_sat_s
		fn.simdBin("i16x8_q15mulr_sat_s")
	case 0x83: // i16x8.all_true
		fn.simdUn("i16x8_all_true")
	case 0x84: // i16x8.bitmask
		fn.simdUn("i16x8_bitmask")
	case 0x85: // i16x8.narrow_i32x4_s
		fn.simdBin("i16x8_narrow_i32x4_s")
	case 0x86: // i16x8.narrow_i32x4_u
		fn.simdBin("i16x8_narrow_i32x4_u")
	case 0x87: // i16x8.extend_low_i8x16_s
		fn.simdUn("i16x8_extend_low_i8x16_s")
	case 0x88: // i16x8.extend_high_i8x16_s
		fn.simdUn("i16x8_extend_high_i8x16_s")
	case 0x89: // i16x8.extend_low_i8x16_u
		fn.simdUn("i16x8_extend_low_i8x16_u")
	case 0x8a: // i16x8.extend_high_i8x16_u
		fn.simdUn("i16x8_extend_high_i8x16_u")
	case 0x8b: // i16x8.shl
		fn.simdBin("i16x8_shl")
	case 0x8c: // i16x8.shr_s
		fn.simdBin("i16x8_shr_s")
	case 0x8d: // i16x8.shr_u
		fn.simdBin("i16x8_shr_u")
	case 0x8e: // i16x8.add
		fn.simdBin("i16x8_add")
	case 0x8f: // i16x8.add_sat_s
		fn.simdBin("i16x8_add_sat_s")
	case 0x90: // i16x8.add_sat_u
		fn.simdBin("i16x8_add_sat_u")
	case 0x91: // i16x8.sub
		fn.simdBin("i16x8_sub")
	case 0x92: // i16x8.sub_sat_s
		fn.simdBin("i16x8_sub_sat_s")
	case 0x93: // i16x8.sub_sat_u
		fn.simdBin("i16x8_sub_sat_u")

	case 0x94: // f64x2.nearest
		fn.simdUn("f64x2_nearest")

	case 0x95: // i16x8.mul
		fn.simdBin("i16x8_mul")
	case 0x96: // i16x8.min_s
		fn.simdBin("i16x8_min_s")
	case 0x97: // i16x8.min_u
		fn.simdBin("i16x8_min_u")
	case 0x98: // i16x8.max_s
		fn.simdBin("i16x8_max_s")
	case 0x99: // i16x8.max_u
		fn.simdBin("i16x8_max_u")
	case 0x9b: // i16x8.avgr_u
		fn.simdBin("i16x8_avgr_u")
	case 0x9c: // i16x8.extmul_low_i8x16_s
		fn.simdBin("i16x8_extmul_low_i8x16_s")
	case 0x9d: // i16x8.extmul_high_i8x16_s
		fn.simdBin("i16x8_extmul_high_i8x16_s")
	case 0x9e: // i16x8.extmul_low_i8x16_u
		fn.simdBin("i16x8_extmul_low_i8x16_u")
	case 0x9f: // i16x8.extmul_high_i8x16_u
		fn.simdBin("i16x8_extmul_high_i8x16_u")

	case 0xa0: // i32x4.abs
		fn.simdUn("i32x4_abs")
	case 0xa1: // i32x4.neg
		fn.simdUn("i32x4_neg")
	case 0xa3: // i32x4.all_true
		fn.simdUn("i32x4_all_true")
	case 0xa4: // i32x4.bitmask
		fn.simdUn("i32x4_bitmask")
	case 0xa7: // i32x4.extend_low_i16x8_s
		fn.simdUn("i32x4_extend_low_i16x8_s")
	case 0xa8: // i32x4.extend_high_i16x8_s
		fn.simdUn("i32x4_extend_high_i16x8_s")
	case 0xa9: // i32x4.extend_low_i16x8_u
		fn.simdUn("i32x4_extend_low_i16x8_u")
	case 0xaa: // i32x4.extend_high_i16x8_u
		fn.simdUn("i32x4_extend_high_i16x8_u")
	case 0xab: // i32x4.shl
		fn.simdBin("i32x4_shl")
	case 0xac: // i32x4.shr_s
		fn.simdBin("i32x4_shr_s")
	case 0xad: // i32x4.shr_u
		fn.simdBin("i32x4_shr_u")
	case 0xae: // i32x4.add
		fn.simdBin("i32x4_add")
	case 0xb1: // i32x4.sub
		fn.simdBin("i32x4_sub")
	case 0xb5: // i32x4.mul
		fn.simdBin("i32x4_mul")
	case 0xb6: // i32x4.min_s
		fn.simdBin("i32x4_min_s")
	case 0xb7: // i32x4.min_u
		fn.simdBin("i32x4_min_u")
	case 0xb8: // i32x4.max_s
		fn.simdBin("i32x4_max_s")
	case 0xb9: // i32x4.max_u
		fn.simdBin("i32x4_max_u")
	case 0xba: // i32x4.dot_i16x8_s
		fn.simdBin("i32x4_dot_i16x8_s")
	case 0xbc: // i32x4.extmul_low_i16x8_s
		fn.simdBin("i32x4_extmul_low_i16x8_s")
	case 0xbd: // i32x4.extmul_high_i16x8_s
		fn.simdBin("i32x4_extmul_high_i16x8_s")
	case 0xbe: // i32x4.extmul_low_i16x8_u
		fn.simdBin("i32x4_extmul_low_i16x8_u")
	case 0xbf: // i32x4.extmul_high_i16x8_u
		fn.simdBin("i32x4_extmul_high_i16x8_u")

	case 0xc0: // i64x2.abs
		fn.simdUn("i64x2_abs")
	case 0xc1: // i64x2.neg
		fn.simdUn("i64x2_neg")
	case 0xc3: // i64x2.all_true
		fn.simdUn("i64x2_all_true")
	case 0xc4: // i64x2.bitmask
		fn.simdUn("i64x2_bitmask")
	case 0xc7: // i64x2.extend_low_i32x4_s
		fn.simdUn("i64x2_extend_low_i32x4_s")
	case 0xc8: // i64x2.extend_high_i32x4_s
		fn.simdUn("i64x2_extend_high_i32x4_s")
	case 0xc9: // i64x2.extend_low_i32x4_u
		fn.simdUn("i64x2_extend_low_i32x4_u")
	case 0xca: // i64x2.extend_high_i32x4_u
		fn.simdUn("i64x2_extend_high_i32x4_u")
	case 0xcb: // i64x2.shl
		fn.simdBin("i64x2_shl")
	case 0xcc: // i64x2.shr_s
		fn.simdBin("i64x2_shr_s")
	case 0xcd: // i64x2.shr_u
		fn.simdBin("i64x2_shr_u")
	case 0xce: // i64x2.add
		fn.simdBin("i64x2_add")
	case 0xd1: // i64x2.sub
		fn.simdBin("i64x2_sub")
	case 0xd5: // i64x2.mul
		fn.simdBin("i64x2_mul")
	case 0xd6: // i64x2.eq
		fn.simdBin("i64x2_eq")
	case 0xd7: // i64x2.ne
		fn.simdBin("i64x2_ne")
	case 0xd8: // i64x2.lt_s
		fn.simdBin("i64x2_lt_s")
	case 0xd9: // i64x2.gt_s
		fn.simdBin("i64x2_gt_s")
	case 0xda: // i64x2.le_s
		fn.simdBin("i64x2_le_s")
	case 0xdb: // i64x2.ge_s
		fn.simdBin("i64x2_ge_s")
	case 0xdc: // i64x2.extmul_low_i32x4_s
		fn.simdBin("i64x2_extmul_low_i32x4_s")
	case 0xdd: // i64x2.extmul_high_i32x4_s
		fn.simdBin("i64x2_extmul_high_i32x4_s")
	case 0xde: // i64x2.extmul_low_i32x4_u
		fn.simdBin("i64x2_extmul_low_i32x4_u")
	case 0xdf: // i64x2.extmul_high_i32x4_u
		fn.simdBin("i64x2_extmul_high_i32x4_u")

	case 0xe0: // f32x4.abs
		fn.simdUn("f32x4_abs")
	case 0xe1: // f32x4.neg
		fn.simdUn("f32x4_neg")
	case 0xe3: // f32x4.sqrt
		fn.simdUn("f32x4_sqrt")
	case 0xe4: // f32x4.add
		fn.simdBin("f32x4_add")
	case 0xe5: // f32x4.sub
		fn.simdBin("f32x4_sub")
	case 0xe6: // f32x4.mul
		fn.simdBin("f32x4_mul")
	case 0xe7: // f32x4.div
		fn.simdBin("f32x4_div")
	case 0xe8: // f32x4.min
		fn.simdBin("f32x4_min")
	case 0xe9: // f32x4.max
		fn.simdBin("f32x4_max")
	case 0xea: // f32x4.pmin
		fn.simdBin("f32x4_pmin")
	case 0xeb: // f32x4.pmax
		fn.simdBin("f32x4_pmax")

	case 0xec: // f64x2.abs
		fn.simdUn("f64x2_abs")
	case 0xed: // f64x2.neg
		fn.simdUn("f64x2_neg")
	case 0xef: // f64x2.sqrt
		fn.simdUn("f64x2_sqrt")
	case 0xf0: // f64x2.add
		fn.simdBin("f64x2_add")
	case 0xf1: // f64x2.sub
		fn.simdBin("f64x2_sub")
	case 0xf2: // f64x2.mul
		fn.simdBin("f64x2_mul")
	case 0xf3: // f64x2.div
		fn.simdBin("f64x2_div")
	case 0xf4: // f64x2.min
		fn.simdBin("f64x2_min")
	case 0xf5: // f64x2.max
		fn.simdBin("f64x2_max")
	case 0xf6: // f64x2.pmin
		fn.simdBin("f64x2_pmin")
	case 0xf7: // f64x2.pmax
		fn.simdBin("f64x2_pmax")

	case 0xf8: // i32x4.trunc_sat_f32x4_s
		fn.simdUn("i32x4_trunc_sat_f32x4_s")
	case 0xf9: // i32x4.trunc_sat_f32x4_u
		fn.simdUn("i32x4_trunc_sat_f32x4_u")
	case 0xfa: // f32x4.convert_i32x4_s
		fn.simdUn("f32x4_convert_i32x4_s")
	case 0xfb: // f32x4.convert_i32x4_u
		fn.simdUn("f32x4_convert_i32x4_u")
	case 0xfc: // i32x4.trunc_sat_f64x2_s_zero
		fn.simdUn("i32x4_trunc_sat_f64x2_s_zero")
	case 0xfd: // i32x4.trunc_sat_f64x2_u_zero
		fn.simdUn("i32x4_trunc_sat_f64x2_u_zero")
	case 0xfe: // f64x2.convert_low_i32x4_s
		fn.simdUn("f64x2_convert_low_i32x4_s")
	case 0xff: // f64x2.convert_low_i32x4_u
		fn.simdUn("f64x2_convert_low_i32x4_u")

	default:
		return fmt.Errorf("unsupported opcode (SIMD): 0xFD 0x%02X", code)
	}
	return err
}

// Returns a call to a SIMD helper, registering it and the v128 type.
func (fn *funcCompiler) simdCall(name string, args ...ast.Expr) ast.Expr {
	fn.helpers.add("v128")
	fn.helpers.add(name)
	return &ast.CallExpr{Fun: newID(name), Args: args}
}

// Executes a pure unary SIMD helper call.
func (fn *funcCompiler) simdUn(name string) {
	fn.pushPure(fn.simdCall(name, fn.pop()))
}

// Executes a pure binary SIMD helper call.
func (fn *funcCompiler) simdBin(name string) {
	y := fn.pop()
	x := fn.pop()
	fn.pushPure(fn.simdCall(name, x, y))
}

// Returns an expression that loads a v128 from memory.
func (fn *funcCompiler) loadV128(offset uint64) ast.Expr {
	return fn.simdCall("load128", &ast.SliceExpr{
		X:   fn.memory.selector,
		Low: fn.popAddr(offset)})
}

// Returns a statement that stores a v128 to memory.
func (fn *funcCompiler) storeV128(offset uint64) ast.Stmt {
	val := fn.pop()
	addr := fn.popAddr(offset)
	return &ast.ExprStmt{X: fn.simdCall("store128",
		&ast.SliceExpr{
			X:   fn.memory.selector,
			Low: addr},
		val)}
}

// Returns an expression that loads 64 bits from memory,
// extending them to a v128 with the given helper.
func (fn *funcCompiler) simdLoadExt(name string, offset uint64) ast.Expr {
	return fn.simdCall(name, fn.load("uint64", offset))
}

// Executes an extract_lane operation.
func (fn *funcCompiler) extractLane(name string) error {
	lane, err := fn.in.ReadByte()
	if err != nil {
		return err
	}
	fn.pushPure(fn.simdCall(name, fn.pop(), laneLit(lane)))
	return nil
}

// Executes a replace_lane operation.
func (fn *funcCompiler) replaceLane(name string) error {
	lane, err := fn.in.ReadByte()
	if err != nil {
		return err
	}
	x := fn.pop()
	v := fn.pop()
	fn.pushPure(fn.simdCall(name, v, laneLit(lane), x))
	return nil
}

// Executes a load_lane operation.
func (fn *funcCompiler) simdLoadLane(name string, bits int, offset uint64) error {
	lane, err := fn.in.ReadByte()
	if err != nil {
		return err
	}
	v := fn.pop()
	var val ast.Expr
	switch bits {
	case 8:
		val = convert(fn.load8(offset), "int32")
	case 16:
		val = convert(fn.load("uint16", offset), "int32")
	case 32:
		val = convert(fn.load("uint32", offset), "int32")
	case 64:
		val = convert(fn.load("uint64", offset), "int64")
	}
	fn.push(fn.simdCall(name, v, laneLit(lane), val))
	return nil
}

// Executes a store_lane operation.
func (fn *funcCompiler) simdStoreLane(name string, bits int, offset uint64) error {
	lane, err := fn.in.ReadByte()
	if err != nil {
		return err
	}
	v := fn.pop()
	val := fn.simdCall(name, v, laneLit(lane))
	if bits == 8 {
		fn.emit(&ast.AssignStmt{
			Tok: token.ASSIGN,
			Lhs: []ast.Expr{fn.load8(offset)},
			Rhs: []ast.Expr{convert(val, "byte")}})
		return nil
	}
	sbits := fmt.Sprint(bits)
	addr := fn.popAddr(offset)
	fn.helpers.add("store" + sbits)
	fn.emit(&ast.ExprStmt{X: &ast.CallExpr{
		Fun:  newID("store" + sbits),
		Args: []ast.Expr{fn.memory.selector, addr, convert(val, "uint"+sbits)}}})
	return nil
}

// Returns a v128 composite literal: the 16 lane bytes as two
// little-endian 64-bit words, matching the helpers' representation.
func v128lit(lanes [16]byte) ast.Expr {
	lo := binary.LittleEndian.Uint64(lanes[0:8])
	hi := binary.LittleEndian.Uint64(lanes[8:16])
	return &ast.CompositeLit{Type: newID("v128"), Elts: []ast.Expr{
		&ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%#016x", lo)},
		&ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%#016x", hi)}}}
}

// Returns a lane index literal.
func laneLit(lane byte) ast.Expr {
	return &ast.BasicLit{Kind: token.INT, Value: fmt.Sprint(lane)}
}

// Reads a v128.const immediate, returning its composite literal.
func (t *translator) constV128() (ast.Expr, error) {
	var lanes [16]byte
	if _, err := io.ReadFull(t.in, lanes[:]); err != nil {
		return nil, err
	}
	t.helpers.add("v128")
	return v128lit(lanes), nil
}
